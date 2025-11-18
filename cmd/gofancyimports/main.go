package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/NonLogicalDev/gofancyimports"
	"github.com/NonLogicalDev/gofancyimports/pkg/organizer/autogroup"
	"github.com/NonLogicalDev/gofancyimports/pkg/types"
)

type rootCMD struct {
	*cobra.Command
}

type debugCMD struct {
	*cobra.Command
}

type fixCMD struct {
	*cobra.Command

	writeFile  bool
	showDiff   bool
	recursive  bool

	localPrefixes []string
	groupEffect   bool
	groupNoDot    bool
}

var cmdName = "gofancyimports"

func init() {
	if len(os.Args) > 0 {
		cmdName = filepath.Base(os.Args[0])
	}
}

func main() {
	cmd := rootCMD{
		Command: &cobra.Command{
			Use:  cmdName,
			Args: cobra.MinimumNArgs(1),
		},
	}
	cmd.AddCommand(makeDebugCommand())
	cmd.AddCommand(makeFixCommand())

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error:\n%s\n", err)
	}
}

func makeFixCommand() *cobra.Command {
	cmdFix := fixCMD{
		Command: &cobra.Command{
			Use:   "fix",
			Short: "Fixup single or multiple provided files",
		},

		writeFile: false,
		showDiff:  false,

		groupNoDot:    false,
		groupEffect:   false,
		localPrefixes: nil,
	}

	cmdFix.Command.RunE = cmdFix.RunE

	cmdFix.PersistentFlags().BoolVarP(&cmdFix.writeFile,
		"write", "w", false,
		"write the file back?")
	cmdFix.PersistentFlags().BoolVarP(&cmdFix.showDiff,
		"diff", "d", false,
		"print diff")

	cmdFix.PersistentFlags().StringArrayVarP(&cmdFix.localPrefixes,
		"local", "l", nil,
		"group local imports (comma separated prefixes)")
	cmdFix.PersistentFlags().BoolVar(&cmdFix.groupNoDot,
		"group-nodot", false,
		"group no dot imports")
	cmdFix.PersistentFlags().BoolVar(&cmdFix.groupEffect,
		"group-effect", false,
		"group side effect imports")
	cmdFix.PersistentFlags().BoolVarP(&cmdFix.recursive,
		"recursive", "r", false,
		"recurse into subdirectories when processing directories")

	return cmdFix.Command
}

func (c *fixCMD) RunE(cmd *cobra.Command, args []string) error {
	var errs error
	if len(args) == 0 {
		srcOriginal, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed reading stdin: %w", err)
		}

		err = c.runRewrite("stdin.go", srcOriginal, true)
		if err != nil {
			return fmt.Errorf("failed fixing file: %w", err)
		}

		return errs
	}

	for _, srcPath := range args {
		paths, err := discoverPaths(srcPath, c.recursive)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed to discover paths for %s: %w", srcPath, err))
			continue
		}

		for _, path := range paths {
			srcOriginal, err := os.ReadFile(path)
			if err != nil {
				errs = multierr.Append(errs, fmt.Errorf("failed reading file: %w", err))
				continue
			}

			err = c.runRewrite(path, srcOriginal, false)
			if err != nil {
				errs = multierr.Append(errs, fmt.Errorf("failed fixing file: %w", err))
				continue
			}
		}
	}

	return nil
}

// discoverPaths returns a list of Go file paths from the given path.
// If the path is a directory, it walks it to find all .go files.
// If recursive is true, it recursively walks subdirectories.
// If the path is a file, it returns it as a single-element slice.
func discoverPaths(path string, recursive bool) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %s: %w", path, err)
	}

	var paths []string

	if info.IsDir() {
		if recursive {
			// Recursively find all .go files in the directory
			err := filepath.Walk(path, func(walkPath string, walkInfo os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !walkInfo.IsDir() && strings.HasSuffix(walkPath, ".go") {
					paths = append(paths, walkPath)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk directory %s: %w", path, err)
			}
		} else {
			// Only find .go files in the immediate directory
			entries, err := os.ReadDir(path)
			if err != nil {
				return nil, fmt.Errorf("failed to read directory %s: %w", path, err)
			}
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
					paths = append(paths, filepath.Join(path, entry.Name()))
				}
			}
		}
	} else {
		// Regular file
		paths = append(paths, path)
	}

	return paths, nil
}

func (c *fixCMD) runRewrite(srcPath string, srcOriginal []byte, isStdin bool) error {
	transform := autogroup.New(
		autogroup.WithSpecFixups(autogroup.FixupEmbedPackage),
		autogroup.WithNoDotGroupEnabled(c.groupNoDot),
		autogroup.WithSideEffectGroupEnabled(c.groupEffect),
		autogroup.WithLocalPrefixGroup(c.localPrefixes),
	)
	srcRewritten, err := gofancyimports.RewriteImportsSource(
		srcPath, srcOriginal,
		gofancyimports.WithTransform(transform),
	)
	if err != nil {
		return fmt.Errorf("rewriting imports: %w", err)
	}

	// Print diff.
	if c.showDiff {
		diff, err := generateDiff(srcPath, srcOriginal, srcRewritten)
		if err != nil {
			return fmt.Errorf("generating diff: %w", err)
		}
		if len(diff) != 0 {
			fmt.Printf("%s", diff)
		}
		return nil
	}

	// Print source.
	if !c.writeFile {
		fmt.Println(string(srcRewritten))
		return nil
	}

	if c.writeFile && isStdin {
		return fmt.Errorf("can't write to stdin")
	}

	// Write back.
	if !bytes.Equal(srcOriginal, srcRewritten) {
		err = os.WriteFile(srcPath, srcRewritten, 0x666)
		if err != nil {
			return err
		}
		fmt.Println("Written:", srcPath)
	}

	return nil
}

func (c *debugCMD) RunE(cmd *cobra.Command, args []string) error {
	var errs error
	if len(args) == 0 {
		srcOriginal, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed reading stdin: %w", err)
		}

		err = c.runDebug("stdin.go", srcOriginal)
		if err != nil {
			return fmt.Errorf("failed fixing file: %w", err)
		}

		return errs
	}

	for _, srcPath := range args {
		srcOriginal, err := os.ReadFile(srcPath)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed reading file: %w", err))
			continue
		}

		err = c.runDebug(srcPath, srcOriginal)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed fixing file: %w", err))
			continue
		}
	}

	return errs
}

func makeDebugCommand() *cobra.Command {
	cmdDebug := debugCMD{
		Command: &cobra.Command{
			Use:   "debug",
			Short: "Print debug information in JSON format about discovered import groups in provided files",
		},
	}

	cmdDebug.Command.RunE = cmdDebug.RunE

	return cmdDebug.Command
}

func (c *debugCMD) runDebug(srcPath string, srcOriginal []byte) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, srcPath, srcOriginal, parser.ParseComments)
	if err != nil {
		return err
	}

	importDeclRange, err := debugImportDeclarationsFromFileToJSON(fset, node)
	if err != nil {
		return err
	}

	debugJSON, err := json.MarshalIndent(importDeclRange, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(debugJSON))
	return nil
}

func generateDiff(path string, origSource []byte, newSource []byte) (string, error) {
	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(origSource)),
		FromFile: path + ".orig",
		B:        difflib.SplitLines(string(newSource)),
		ToFile:   path,
		Context:  3,
	})
}

func debugImportDeclarationsFromFileToJSON(fset *token.FileSet, file *ast.File) (json.RawMessage, error) {
	declRange, err := gofancyimports.ParseImportDeclarations(fset, file)
	if err != nil {
		return nil, err
	}
	return types.ImportDeclarationsToJSON(fset, declRange.Statements), nil
}
