package main

import (
	"bytes"
	_ "embed" // enable resource embedding
	"fmt"
	"os"
	"path/filepath"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/cobra"

	gofancyimports "github.com/NonLogicalDev/gofancyimports"
	"github.com/NonLogicalDev/gofancyimports/pkg/organizer/autogroup"
)

type cmdRunE func(cmd *cobra.Command, args []string) error

type cmdFlags struct {
	writeFile bool
	showDiff  bool
	prefixes  []string
}

var cmdName = "gofancyimports"

func init() {
	if len(os.Args) > 0 {
		cmdName = filepath.Base(os.Args[0])
	}
}

func main() {
	cmd := &cobra.Command{
		Use:  cmdName,
		Args: cobra.MinimumNArgs(1),
	}

	var flg cmdFlags
	cmd.PersistentFlags().BoolVarP(&flg.writeFile,
		"write", "w", false,
		"write the file back?")
	cmd.PersistentFlags().BoolVarP(&flg.showDiff,
		"diff", "d", false,
		"print diff")
	cmd.PersistentFlags().StringArrayVarP(&flg.prefixes,
		"local", "l", nil,
		"local imports prefixes to put in a separate group")
	cmd.RunE = cmdRun(&flg)

	if err := cmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error:\n%s\n", err)
	}
}

func cmdRun(flg *cmdFlags) cmdRunE {
	return func(cmd *cobra.Command, args []string) error {
		for _, srcPath := range args {
			srcOriginal, err := os.ReadFile(srcPath)
			if err != nil {
				return fmt.Errorf("failed reading file: %w", err)
			}

			srcRewritten, err := gofancyimports.RewriteImportsSource(
				srcPath, srcOriginal,
				autogroup.New(flg.prefixes, autogroup.FixupEmbedPackage),
			)
			if err != nil {
				return fmt.Errorf("generating imports: %w", err)
			}

			diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				A:        difflib.SplitLines(string(srcOriginal)),
				FromFile: srcPath + ".orig",
				B:        difflib.SplitLines(string(srcRewritten)),
				ToFile:   srcPath,
				Context:  3,
			})
			if err != nil {
				return fmt.Errorf("generating diff: %w", err)
			}

			// Print diff.
			if flg.showDiff {
				if len(diff) != 0 {
					fmt.Printf("%s", diff)
				}
				continue
			}

			// Print source.
			if !flg.writeFile {
				fmt.Println(string(srcRewritten))
				continue
			}

			// Write back.
			if !bytes.Equal(srcOriginal, srcRewritten) {
				err = os.WriteFile(srcPath, srcRewritten, 0x666)
				if err != nil {
					return err
				}

				fmt.Println("Written:", srcPath)
			}
		}

		return nil

	}
}
