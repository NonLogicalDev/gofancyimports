package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"

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

	writeFile bool
	showDiff  bool

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

	return cmdFix.Command
}

func (c *fixCMD) RunE(cmd *cobra.Command, args []string) error {
	var errs error
	for _, srcPath := range args {
		srcOriginal, err := os.ReadFile(srcPath)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed reading file: %w", err))
			continue
		}

		err = c.runRewrite(srcPath, srcOriginal)
		if err != nil {
			errs = multierr.Append(errs, fmt.Errorf("failed fixing file: %w", err))
			continue
		}
	}

	return nil
}

func (c *fixCMD) runRewrite(srcPath string, srcOriginal []byte) error {
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

	return nil
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
