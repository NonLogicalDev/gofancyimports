package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"

	gofancyimports "github.com/nonlogicaldev/gofancyimports"
	"github.com/spf13/cobra"
)

// https://github.com/golang/tools/blob/6e9046bfcd34178dc116189817430a2ad1ee7b43/internal/imports/sortimports.go#L63

// NOTE:
//   node.
//    Decls[].
//      (*ast.GenDecl, _.Tok == token.IMPORT).
//    Specs[].
//      (*ast.ImportSpec).

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
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		sourcePath := args[0]

		src, err := os.ReadFile(sourcePath)
		if err != nil { return err }

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, sourcePath, src, parser.ParseComments)
		if err != nil { return err }

		importDeclRange, _ := gofancyimports.GatherImportDecls(fset, node.Decls, node.Comments)
		importBase := importDeclRange.Base
		if importDeclRange.Base > 0 {
			importDeclRange.Decls = OrganizeImports(importDeclRange.Decls)
			importDecls, newLines, importEndPos := gofancyimports.BuildImportDecls(fset, importDeclRange.Start, importDeclRange.Decls)

			importString := gofancyimports.PrintImportDecls(
				importBase, importEndPos, newLines, importDecls,
			)

			var output []byte

			f := fset.File(node.Package)
			output = append(output, src[:f.Offset(importDeclRange.Start)-1]...)
			output = append(output, importString...)
			output = append(output, src[f.Offset(importDeclRange.End)+1:]...)


			if !*flagWrite {
				fmt.Println(string(output))
				return nil
			}

			return os.WriteFile(sourcePath, output, 0x666)
		} else {
			fmt.Println("WEIRD:", sourcePath)
		}
		
		return nil
	}

	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}