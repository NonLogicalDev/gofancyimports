package main

import (
	"fmt"
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

	flagWrite := cmd.PersistentFlags().BoolP("write", "w", false, "write the file back?")
	localPrefixes := cmd.PersistentFlags().StringArrayP("local", "l", nil, "list of local prefixes")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		sourcePath := args[0]
		LocalPrefixes = *localPrefixes

		src, err := os.ReadFile(sourcePath)
		if err != nil { return err }

		newSrc, err := gofancyimports.RewriteImports(sourcePath, src, func(decls []gofancyimports.ImportDecl) []gofancyimports.ImportDecl {
			//fmt.Fprintln(os.Stderr, litter.Sdump(decls))
			return OrganizeImports(decls)
		})
		if err != nil { return err }

		if !*flagWrite {
			fmt.Println(string(newSrc))
			return nil
		}

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
