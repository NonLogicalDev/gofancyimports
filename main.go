package main

import (
	"fmt"
	"github.com/sanity-io/litter"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

//go:generate go run mkstdlib.go

// https://github.com/golang/tools/blob/6e9046bfcd34178dc116189817430a2ad1ee7b43/internal/imports/sortimports.go#L63

// NOTE:
//   node.
//    Decls[].
//      (*ast.GenDecl, _.Tok == token.IMPORT).
//    Specs[].
//      (*ast.ImportSpec).


func main() {
	sourcePath := os.Args[1]

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, sourcePath, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	importDecls, otherDecls := GatherImportDecls(fset, node.Decls)
	for i := range importDecls {
		lc := importDecls[len(importDecls)-1-i]
		f := fset.File(lc.Spec.Pos())

		startLine := f.Line(lc.Spec.Pos())
		endLine := f.Line(lc.Spec.End())

		for s := endLine + 1; s >= startLine; s-- {
			fmt.Println("MERGING: ", s)
			f.MergeLine(s)
		}
	}
	_ = otherDecls

	litter.Dump(importDecls[0].Spec)

	//Filter out groups without comments
	var mergableDecls []ImportDecl
	var nonMergableDecls []ImportDecl
	for _, d := range importDecls {
		if d.Doc == nil {
			mergableDecls = append(mergableDecls, d)
		} else {
			nonMergableDecls = append(nonMergableDecls, d)
		}
		d.Spec.Doc = nil

		//fmt.Print(s.Print(fset))
		//fmt.Println("---")
	}

	defaultGroup := MergeImportDecls(mergableDecls)
	fmt.Println(defaultGroup.Print(fset))

	newDecls := append(
		BuildImportDecls(fset, node, defaultGroup, nonMergableDecls),
		otherDecls...,
	)
	_ = newDecls

	node.Decls = newDecls
	node.Comments = nil

	fmt.Println("><>< ====== ><><")
	_ = printer.Fprint(os.Stdout, fset, node)
}
