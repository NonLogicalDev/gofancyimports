package gofancyimports

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NonLogicalDev/gofancyimports/internal/astutils"
	"github.com/NonLogicalDev/gofancyimports/pkg/types"
)

func buildImportDecls(offset token.Pos, decls []types.ImportDeclaration) ([]ast.Decl, []token.Pos, token.Pos) {
	var astDecls []ast.Decl
	if len(decls) == 0 {
		return nil, nil, 0
	}

	var newLines []token.Pos
	for _, d := range decls {
		decl, nl, newOffset := buildImportDecl(offset, d)
		newLines = append(newLines, nl...)
		astDecls = append(astDecls, decl)
		offset = newOffset
	}
	return astDecls, newLines, offset
}

func buildImportDecl(offset token.Pos, decl types.ImportDeclaration) (ast.Decl, []token.Pos, token.Pos) {
	var newLines []token.Pos
	astDecl := &ast.GenDecl{
		Tok: token.IMPORT,
	}

	// Prepare Doc comment for the import group.
	var astCommentList []*ast.Comment
	for _, cg := range decl.LeadingComments {
		astCommentList = append(astCommentList, copyCommentList(cg.List)...)
	}
	if decl.Doc != nil {
		astCommentList = append(astCommentList, copyCommentList(decl.Doc.List)...)
	}
	for _, cg := range decl.DetachedComments {
		astCommentList = append(astCommentList, copyCommentList(cg.List)...)
	}
	astDecl.Doc = buildCombinedCommentGroup(offset, astCommentList)

	// Place the doc comment at the current offset if it exists and calculate newlines.
	if astDecl.Doc != nil {
		newLines = buildCommentListNewlines(astDecl.Doc.List, newLines)
		// Move offset to the next character after comment end.
		offset = astDecl.Doc.End()
	}

	// Place declaration at the offset.
	astutils.ShiftGenDeclPos(offset-astDecl.Pos(), astDecl)

	// Ensure declaration starts from a new line.
	newLines = append(newLines, offset)

	// After adjusting the position move offset past the import keyword.
	offset += token.Pos(len(token.IMPORT.String()))

	// Handle single import case (no parenthesis should be added).
	if len(decl.ImportGroups) == 1 && len(decl.ImportGroups[0].Specs) == 1 {
		g := decl.ImportGroups[0]
		s := g.Specs[0]
		astSpec := copyImportSpec(s)

		var astSpecDoc *ast.CommentGroup
		if g.Doc != nil {
			astSpecDoc = buildCombinedCommentGroup(offset, g.Doc.List)
		}

		offset, newLines = buildImportSpec(offset, astSpec, astSpecDoc, newLines)

		// Populate the import declaration with adjusted specs.
		astDecl.Specs = []ast.Spec{astSpec}

		// Make sure we have a newline at the end of the declaration.
		newLines = append(newLines, offset)
		offset++
	} else {
		// Force add parenthesis if it does not exist.
		astDecl.Lparen = offset
		offset++

		for i, g := range decl.ImportGroups {
			if len(g.Specs) == 0 {
				continue
			}

			if i != 0 {
				// And an extra spacer line to separate groups from each other.
				newLines = append(newLines, offset)
				offset++
			}

			for specIdx, s := range g.Specs {
				astSpec := copyImportSpec(s)
				var astSpecDoc *ast.CommentGroup
				if specIdx == 0 && g.Doc != nil {
					astSpecDoc = buildCombinedCommentGroup(offset, g.Doc.List)
				}
				offset, newLines = buildImportSpec(offset, astSpec, astSpecDoc, newLines)
				astDecl.Specs = append(astDecl.Specs, astSpec)

				// Make sure there is a newline at the end of the spec.
				newLines = append(newLines, offset)
				offset++
			}
		}

		astDecl.Rparen = offset
		offset = astDecl.End() + 1

		// Make sure we have a newline at the end of the declaration.
		newLines = append(newLines, offset)
		offset++
	}

	return astDecl, newLines, offset
}

func buildImportSpec(offset token.Pos, astSpec *ast.ImportSpec, astSpecDoc *ast.CommentGroup, newLines []token.Pos) (token.Pos, []token.Pos) {
	// For the first astDecl in the group, attach the doc comment.
	if astSpecDoc != nil {
		astSpec.Doc = copyCommentGroup(astSpecDoc)
		astutils.ShiftCommentGroupPos(offset-astSpec.Doc.Pos(), astSpec.Doc)
		newLines = buildCommentListNewlines(astSpec.Doc.List, newLines)
		offset = astSpec.Doc.End() + 1
	}

	// Place the identity value
	if astSpec.Name != nil {
		astSpec.Name.NamePos = offset
		offset += specSpan(astSpec.Name) + 1
	}

	// Place the path value
	astSpec.Path.ValuePos = offset
	offset += specSpan(astSpec.Path) + 1

	// Place the line comment at the end of the astSpec.
	if astSpec.Comment != nil {
		astutils.ShiftCommentGroupPos(offset-astSpec.Comment.Pos(), astSpec.Comment)
		offset += specSpan(astSpec.Comment) + 1
	}

	return offset, newLines
}

func buildCombinedCommentGroup(offset token.Pos, astCommentList []*ast.Comment) *ast.CommentGroup {
	// Adjust comment positions, so that they are relatively correct.
	// They will be shifted into correct place later.
	for _, c := range astCommentList {
		c.Slash = offset

		cRange := astutils.ASTNodeRangeWithComments(c)
		offset += (cRange.End - cRange.Pos) + 2
	}
	if len(astCommentList) == 0 {
		return nil
	}
	return &ast.CommentGroup{
		List: astCommentList,
	}
}

func buildCommentListNewlines(comments []*ast.Comment, newLines []token.Pos) []token.Pos {
	for _, c := range comments {
		// Ensure all comments start from new line.
		// c.Pos() reports correct offset since we have already aligned comments earlier.
		newLines = append(newLines, c.Pos())

		// All newlines inside the comments need to be preserved otherwise printer will not be happy.
		for _, idx := range findAllIndexes(c.Text, "\n") {
			newLines = append(newLines, c.Pos()+token.Pos(idx))
		}
	}
	return newLines
}

func copyComment(c *ast.Comment) *ast.Comment {
	if c == nil {
		return nil
	}
	return &ast.Comment{
		Text: c.Text,
	}
}
func copyCommentList(cs []*ast.Comment) (r []*ast.Comment) {
	r = make([]*ast.Comment, len(cs))
	for i, c := range cs {
		r[i] = copyComment(c)
	}
	return r
}

func copyCommentGroup(cg *ast.CommentGroup) *ast.CommentGroup {
	if cg == nil {
		return nil
	}
	return &ast.CommentGroup{List: copyCommentList(cg.List)}
}

func copyImportSpec(is *ast.ImportSpec) *ast.ImportSpec {
	if is == nil {
		return nil
	}
	return &ast.ImportSpec{
		Doc:     copyCommentGroup(is.Doc),
		Name:    copyIdent(is.Name),
		Path:    copyBasicLit(is.Path),
		Comment: copyCommentGroup(is.Comment),
	}
}

func copyBasicLit(l *ast.BasicLit) *ast.BasicLit {
	if l == nil {
		return nil
	}
	return &ast.BasicLit{
		Kind:  l.Kind,
		Value: l.Value,
	}
}
func copyIdent(id *ast.Ident) *ast.Ident {
	if id == nil {
		return nil
	}
	return &ast.Ident{
		Name: id.Name,
		Obj:  id.Obj,
	}
}

func specSpan(node ast.Node) token.Pos {
	return node.End() - node.Pos()
}

func findAllIndexes(s string, c string) []int {
	var r []int
	for i := 0; i < len(s); {
		idx := strings.Index(s[i:], c)
		if idx < 0 {
			break
		}
		r = append(r, i+idx)
		i += idx + len(c)
	}
	return r
}
