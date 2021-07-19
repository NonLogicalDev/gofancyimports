package main

import (
	"fmt"
	"github.com/NonLogicalDev/gofancyimports/internal/diff"
	"os"

	gofancyimports "github.com/NonLogicalDev/gofancyimports"
	"github.com/spf13/cobra"
)


func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cmd := cobra.Command{
		Use: "go-fancy-imports",

		Args: cobra.MinimumNArgs(1),
	}
	cmdFlags := cmd.PersistentFlags()

	var (
		flagWrite = cmdFlags.
			BoolP("write", "w", false, "write the file back?")
		showDiff  = cmdFlags.
			BoolP("diff", "d", false, "print diff")
		localPrefixes = cmdFlags.
			StringArrayP("local", "l", nil, "list of local prefixes")
	)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		sourcePath := args[0]
		LocalPrefixes = *localPrefixes

		src, err := os.ReadFile(sourcePath)
		if err != nil { return err }

		// OrganizeImports is the import rewriter defined in `rules.go`.
		newSrc, err := gofancyimports.RewriteImports(sourcePath, src, OrganizeImports)
		if err != nil { return err }

		// Print diff.
		if *showDiff {
			diffOut, err := diff.Diff("", src, newSrc)
			if err != nil { return err }

			diffOut, err = diff.ReplaceTempFilename(diffOut, sourcePath)
			if err != nil { return err }

			fmt.Println(string(diffOut))
			return nil
		}

		// Print source.
		if !*flagWrite {
			fmt.Println(string(newSrc))
			return nil
		}

		// Write back.
		err = os.WriteFile(sourcePath, newSrc, 0x666)
		if err != nil {
			return err
		}
		fmt.Println("Written:", sourcePath)
		return nil
	}

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
