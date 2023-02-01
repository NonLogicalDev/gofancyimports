package astutils

import (
	"bytes"
	"fmt"
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
) (string, error) {
	fset := token.NewFileSet()

	// Create a new fake file without line definitions, that we will use to render new imports into.
	f := fset.AddFile("temp.go", importBase, int(importEndPos)-importBase)

	// Import lines generated from the import builder.
	if ok := FileSpliceLines(f, ConvertLinePosToOffsets(f.Base(), newLines)); !ok {
		return "", fmt.Errorf("can't set new lines generated from building imports")
	}

	fileNode := &ast.File{
		Name: &ast.Ident{
			Name:    "main",
			NamePos: token.Pos(importBase + 8),
		},
		Decls: importDecls,
	}
	ast.Inspect(fileNode, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.CommentGroup:
			fileNode.Comments = append(fileNode.Comments, node)
		case *ast.ImportSpec:
			fileNode.Imports = append(fileNode.Imports, node)
		}
		return true
	})

	var result string

	result = PrintNodeString(fset, fileNode)
	resultLines := strings.Split(result, "\n")

	// Cut off `package <name>` line
	result = strings.Join(resultLines[1:], "\n")

	return strings.Trim(result, "\n"), nil
}

func PrintNodeString(fset *token.FileSet, node interface{}) string {
	b := bytes.NewBuffer(nil)
	_ = printer.Fprint(b, fset, node)
	return b.String()
}
