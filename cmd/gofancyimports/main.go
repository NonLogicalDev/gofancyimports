package main

import (
	"fmt"
	"os"

	"github.com/NonLogicalDev/go.fancyimports"
	"github.com/NonLogicalDev/go.fancyimports/internal/diff"
	"github.com/NonLogicalDev/go.fancyimports/pkg/organizer/autogroup"
	"github.com/spf13/cobra"
)

type cobraRunFuncE func(cmd *cobra.Command, args []string) error

func main() {
	cmd := &cobra.Command{
		Use:  "gofancyimports",
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

type cmdFlags struct {
	writeFile bool
	showDiff  bool
	prefixes  []string
}

func cmdRun(flg *cmdFlags) cobraRunFuncE {
	return func(cmd *cobra.Command, args []string) error {
		for _, srcPath := range args {
			srcOriginal, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}

			srcRewritten, err := gofancyimports.RewriteImports(srcPath, srcOriginal, autogroup.New(flg.prefixes))
			if err != nil {
				return err
			}

			diffOut, err := diff.Diff("", srcOriginal, srcRewritten)
			if err != nil {
				return err
			}
			diffOut, err = diff.ReplaceTempFilename(diffOut, srcPath)
			if err != nil {
				return err
			}

			// Print diff.
			if flg.showDiff {
				fmt.Println(string(diffOut))
				continue
			}

			// Print source.
			if !flg.writeFile {
				fmt.Println(">>> ", srcPath)
				fmt.Println(string(srcRewritten))
				continue
			}

			// Write back.
			err = os.WriteFile(srcPath, srcRewritten, 0x666)
			if err != nil {
				return err
			}
			fmt.Println("Written:", srcPath)
		}

		return nil

	}
}
