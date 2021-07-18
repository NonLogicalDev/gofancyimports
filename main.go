package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

//go:generate go run mkstdlib.go

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
	sourcePath := os.Args[1]
	sourcePath = "_example/exhaustive.go"

	src, err := os.ReadFile(sourcePath)
	checkErr(err)

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, sourcePath, src, parser.ParseComments)
	checkErr(err)

	f := fset.File(node.Package)

	importDeclGroups, otherDecls := GatherImportDecls(fset, node.Comments, node.Decls)
	for i := range importDeclGroups {
		lc := importDeclGroups[len(importDeclGroups)-1-i]
		f := fset.File(lc.Spec.Pos())

		startLine := f.Line(lc.Spec.Pos())
		endLine := f.Line(lc.Spec.End())

		for s := endLine + 1; s >= startLine; s-- {
			if s < f.LineCount() {
				f.MergeLine(s)
			}
		}
	}
	_ = otherDecls

	// Filter out groups without comments.
	var lastPos, firstPos token.Pos
	{
		var (
			mergableDecls []ImportDecl
			nonMergableDecls []ImportDecl
		)
		for _, d := range importDeclGroups {
			if d.Spec.End() > lastPos {
				lastPos = d.Spec.End()
			}
			if d.Spec.Doc != nil && (firstPos == token.NoPos || d.Spec.Doc.Pos() < firstPos) {
				firstPos = d.Spec.Doc.Pos()
			} else if firstPos == token.NoPos || d.Spec.Pos() < firstPos {
				firstPos = d.Spec.Pos()
			}

			if d.Doc == nil {
				mergableDecls = append(mergableDecls, d)
			} else {
				nonMergableDecls = append(nonMergableDecls, d)
			}

			d.Spec.Doc = nil
		}
		if len(mergableDecls) > 0 {
			defaultGroup := MergeImportDecls(mergableDecls)
			importDeclGroups = append([]ImportDecl{defaultGroup}, nonMergableDecls...)
		} else {
			importDeclGroups = nonMergableDecls
		}
	}

	commentRange := GatherComments(fset, node.Comments, firstPos, lastPos)

	origLines := FileGetLines(f)
	lastLine := 0
	for ; lastLine < len(origLines)  ; lastLine++ {
		if origLines[lastLine] > int(firstPos) {
			break
		}
	}
	origLines = origLines[:lastLine]

	importDecls, newLines := BuildImportDecls(fset, importDeclGroups)
	endPos := importDecls[len(importDecls)-1].End()
	newFileSize := int(endPos)-f.Base()

	nFS := token.NewFileSet()
	nF := nFS.AddFile(sourcePath, 1, newFileSize)

	if ok := FileSpliceLines(nF, origLines); !ok {
		panic("whoa")
	}
	if ok := FileSpliceLines(nF, newLines); !ok {
		panic("whoa")
	}

	var result string
	{
		node.Decls = importDecls
		var comments []*ast.CommentGroup
		for _, d := range importDecls {
			ast.Inspect(d, func(node ast.Node) bool {
				if cg, ok := node.(*ast.CommentGroup); ok {
					comments = append(comments, cg)
				}
				return true
			})
		}
		node.Comments = append(commentRange.before, comments...)

		result = PrintNodeString(nFS, node)
	}
	{
		node.Decls = otherDecls
		node.Comments = commentRange.after
		node.Doc = nil

		output := PrintNodeString(fset, node)
		outputLines := strings.Split(output, "\n")

		// Cut out the package line.
		result += strings.Join(outputLines[1:], "\n")
	}

	fmt.Println(result)
}
