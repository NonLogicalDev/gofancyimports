package types

import (
	"go/ast"
	"strings"
)

// ImportTransform is a function that allows reordering merging and splitting
// existing []ImportDeclaration obtained from source.
type ImportTransform func(decls []ImportDeclaration) []ImportDeclaration

// ImportDeclaration represents a single import block. (i.e. the contents of the `import` statement)
type ImportDeclaration struct {
	// LeadingComments comments that are floating above this declaration,
	// in the middle of import blocks.
	LeadingComments []*ast.CommentGroup

	// DetachedComments are comments that are floating inside this declaration
	// unattached to specs (typically after the last import spec in a group).
	DetachedComments []*ast.CommentGroup

	// Doc is the doc comment for this import declaration.
	Doc *ast.CommentGroup

	// ImportGroups contains the list of underlying ast.ImportSpec-s.
	ImportGroups []ImportGroup
}

// ImportGroup maps to set of consecutive import specs delimited by
// whitespace and potentially having a doc comment.
//
// This type is the powerhouse of this package, allowing easy operation
// on sets of imports, delimited by whitespace.
//
// Contained within an ImportDeclaration.
type ImportGroup struct {
	Doc   *ast.CommentGroup
	Specs []*ast.ImportSpec
}

// MergeDeclarations returns two or more ImportDeclarations merged together
// by appending import groups and comment sections together.
func MergeDeclarations(decls []ImportDeclaration) ImportDeclaration {
	var merged ImportDeclaration

	var groups []ImportGroup
	for _, d := range decls {
		if merged.Doc == nil {
			merged.Doc = d.Doc
		}

		merged.LeadingComments = append(merged.LeadingComments, d.LeadingComments...)
		merged.DetachedComments = append(merged.DetachedComments, d.DetachedComments...)

		groups = append(groups, d.ImportGroups...)
	}

	merged.ImportGroups = groups
	return merged
}

// MergeGroups returns two or more ImportGroups merged together
// by appending their import specs and comments together and.
func MergeGroups(groups []ImportGroup) ImportGroup {
	var merged ImportGroup
	var docCommets []*ast.CommentGroup
	for _, g := range groups {
		if g.Doc != nil {
			docCommets = append(docCommets, g.Doc)
		}
		merged.Specs = append(merged.Specs, g.Specs...)
	}
	merged.Doc = MergeDocComments(docCommets)
	return merged
}

// MergeDocComments returns multiple doc comment groups merged into one.
func MergeDocComments(groups []*ast.CommentGroup) *ast.CommentGroup {
	var docText []string
	for _, g := range groups {
		for _, c := range g.List {
			docText = append(docText, c.Text)
		}
	}
	if len(docText) == 0 {
		return nil
	}
	return &ast.CommentGroup{
		List: []*ast.Comment{
			{Text: strings.Join(docText, "\n")},
		},
	}
}
