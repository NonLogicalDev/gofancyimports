package gofancyimports

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"golang.org/x/tools/go/analysis"

	"github.com/NonLogicalDev/gofancyimports/internal/astutils"
)

// https://github.com/golang/tools/blob/6e9046bfcd34178dc116189817430a2ad1ee7b43/internal/imports/sortimports.go#L63

type (
	// ImportOrganizer is a function that allows reordering merging and splitting
	// existing ImportDecl-s obtained from source.
	ImportOrganizer func(decls []ImportDecl) []ImportDecl

	// ImportDecl represents a single import block.
	ImportDecl struct {
		// FloatingComments comments that are floating above this declaration,
		// in the middle of import blocks.
		FloatingComments []*ast.CommentGroup

		// WidowComments are comments that are floating inside this declaration
		// unattached to specs (typically after the last import spec in a group).
		WidowComments []*ast.CommentGroup

		// Doc is the doc comment for this import gropup.
		Doc *ast.CommentGroup

		// Groups contains the list of underlying ast.ImportSpec-s.
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

// RewriteImportsSource takes same arguments as `go/parser.ParseFile` with an addition of `rewriter`
// and returns original source with imports grouping modified according to the rewriter.
func RewriteImportsSource(filename string, src []byte, rewriter ImportOrganizer) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	edit, err := RewriteImportsAST(fset, node, src, rewriter)
	if err != nil {
		return nil, err
	}
	if edit == nil {
		return src, nil
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

func RewriteImportsAST(fset *token.FileSet, node *ast.File, src []byte, rewriter ImportOrganizer) (*analysis.TextEdit, error) {
	f := fset.File(node.Package)

	importDeclRange, nonImportDecls := gatherImportDecls(fset, node.Decls, node.Comments)
	invalidDecls := filterDeclsOverlappingRange(nonImportDecls, importDeclRange.Pos, importDeclRange.End)
	if len(invalidDecls) > 0 {
		return nil, fmt.Errorf("found %d non import declarations ovelapping imports", len(nonImportDecls))
	}

	// fmt.Println(string(DumpImportDeclList(fset, importDeclRange.Decls)))

	if importDeclRange.Pos == token.NoPos {
		return nil, nil
	}

	importStringOriginal := string(src[f.Offset(importDeclRange.Pos):f.Offset(importDeclRange.End)])

	importDeclRange.Decls = rewriter(importDeclRange.Decls)
	importDecls, newLines, importEndPos := buildImportDecls(importDeclRange.Pos, importDeclRange.Decls)

	importString, err := astutils.PrintImportDecls(
		f.Base(), importEndPos, newLines, importDecls,
	)
	if err != nil {
		return nil, fmt.Errorf("while serializing re-written import declarations: %w", err)
	}

	if importString == importStringOriginal {
		return nil, nil
	}
	return &analysis.TextEdit{
		Pos:     importDeclRange.Pos,
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
