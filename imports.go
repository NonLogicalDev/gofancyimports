package gofancyimports

import (
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/NonLogicalDev/go.fancyimports/internal/astutils"
	"golang.org/x/tools/go/analysis"
)

// https://github.com/golang/tools/blob/6e9046bfcd34178dc116189817430a2ad1ee7b43/internal/imports/sortimports.go#L63

type (
	// ImportOrganizer is a function that allows reordering merging
	// and splitting existing ImportDecl's obtained from source.
	ImportOrganizer func(decls []ImportDecl) []ImportDecl

	// ImportDecl represents a single import block, a 1:1 mapping.
	ImportDecl struct {
		// FloatingComments comments that are floating above
		// this declaration, yet in the middle of import blocks.
		FloatingComments []*ast.CommentGroup

		// WidowComments are comments that are floating inside this declaration unattached to specs.
		WidowComments []*ast.CommentGroup

		// Doc is the doc comment for this import gropup.
		Doc    *ast.CommentGroup
		Groups []ImportSpecGroup

		spec *ast.GenDecl
	}

	// ImportSpecGroup maps to set of consecutive import specs delimited by
	// whitespace and potentially having a doc comment.
	//
	// This type is the powerhouse of this package, allowing easy operation
	// on sets of imports, delimited by whitespace.
	//
	// Contained within an ImportDecl.
	ImportSpecGroup struct {
		Doc   *ast.CommentGroup
		Specs []*ast.ImportSpec
	}
)

// RewriteImports takes same arguments as `go/parser.ParseFile` with an addition of `rewriter`
// and returns original source with imports grouping modified according to the rewriter.
func RewriteImports(filename string, src []byte, rewriter ImportOrganizer) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	edit, err := RewriteImportsLL(fset, node, rewriter)
	if err != nil {
		return nil, err
	}

	if edit == nil {
		return nil, err
	}

	f := fset.File(node.Package)

	offsetStart := f.Offset(edit.Pos)
	offsetEnd := f.Offset(edit.End)
	fullFileSize := offsetStart + len(edit.NewText) + (len(src) - offsetEnd)

	output := make([]byte, 0, fullFileSize)
	output = append(output, src[:offsetStart]...)
	output = append(output, edit.NewText...)
	output = append(output, src[offsetEnd:]...)

	return output, err

}

func RewriteImportsLL(fset *token.FileSet, node *ast.File, rewriter ImportOrganizer) (*analysis.TextEdit, error) {
	importDeclRange, _ := gatherImportDecls(fset, node.Decls, node.Comments)

	// If importBase is not set, there are no import blocks.
	importBase := importDeclRange.Base
	if importBase <= 0 {
		return nil, nil
	}

	importDeclRange.Decls = rewriter(importDeclRange.Decls)
	importDecls, newLines, importEndPos := buildImportDecls(importDeclRange.Start, importDeclRange.Decls)

	importString := astutils.PrintImportDecls(
		importBase, importEndPos, newLines, importDecls,
	)

	return &analysis.TextEdit{
		Pos:     importDeclRange.Start,
		End:     importDeclRange.End,
		NewText: []byte(importString),
	}, nil
}

func NewImportDecl() ImportDecl {
	return ImportDecl{
		spec: &ast.GenDecl{
			Tok: token.IMPORT,
		},
	}
}

func MergeImportDecls(decls []ImportDecl) ImportDecl {
	var merged ImportDecl

	var groups []ImportSpecGroup
	for _, d := range decls {
		if merged.spec == nil {
			merged.spec = d.spec
			merged.Doc = d.Doc
		}

		merged.FloatingComments = append(merged.FloatingComments, d.FloatingComments...)
		merged.WidowComments = append(merged.WidowComments, d.WidowComments...)

		groups = append(groups, d.Groups...)
	}

	merged.Groups = groups
	return merged
}

func MergeImportGroups(groups []ImportSpecGroup) ImportSpecGroup {
	var merged ImportSpecGroup

	for _, g := range groups {
		if merged.Specs == nil {
			merged.Doc = g.Doc
		}
		merged.Specs = append(merged.Specs, g.Specs...)
	}

	return merged
}
