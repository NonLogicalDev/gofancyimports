package gofancyimports

import (
	"bytes"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

func PrintImportDecls(
	importBase int,
	importEndPos token.Pos,
	newLines []token.Pos,
	importDecls []ast.Decl,
) string {
	fset := token.NewFileSet()

	// Create a new file without line definitions.
	f := fset.AddFile("temp.go", importBase, int(importEndPos)-importBase)

	// Import lines generated from the import builder.
	if ok := FileSpliceLines(f, ConvertLinePosToOffsets(f.Base(), newLines)); !ok {
		panic("can't set new lines generated from building imports")
	}

	node := &ast.File{
		Name: &ast.Ident{
			Name:    "main",
			NamePos: token.Pos(importBase + 8),
		},
		Decls: importDecls,
	}
	var (
		nodeComments []*ast.CommentGroup
		nodeImports  []*ast.ImportSpec
	)
	ast.Inspect(node, func(nodeAbstract ast.Node) bool {
		switch n := nodeAbstract.(type) {
		case *ast.CommentGroup:
			nodeComments = append(nodeComments, n)
		case *ast.ImportSpec:
			nodeImports = append(nodeImports, n)
		}
		return true
	})
	node.Comments = nodeComments
	node.Imports = nodeImports

	var result string
	result = PrintNodeString(fset, node)
	resultLines := strings.Split(result, "\n")
	result = strings.Join(resultLines[1:], "\n")

	return strings.Trim(result, "\n")
}

func PrintNodeString(fset *token.FileSet, node interface{}) string {
	b := bytes.NewBuffer(nil)
	_ = printer.Fprint(b, fset, node)
	return b.String()
}
