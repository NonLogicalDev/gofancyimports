package gofancyimports

import (
	"go/ast"
	"go/token"

	"github.com/NonLogicalDev/go.fancyimports/internal/astutils"
)

func buildImportDecls(offset token.Pos, idecls []ImportDecl) ([]ast.Decl, []token.Pos, token.Pos) {
	var decls []ast.Decl
	if len(idecls) == 0 {
		return nil, nil, 0
	}

	var newLines []token.Pos
	for _, d := range idecls {
		decl, nl, newOffset := buildImportDecl(offset, d)
		newLines = append(newLines, nl...)
		decls = append(decls, decl)
		offset = newOffset
	}

	return decls, newLines, offset
}

func buildImportDecl(offset token.Pos, idecl ImportDecl) (ast.Decl, []token.Pos, token.Pos) {
	var newLines []token.Pos

	if idecl.FloatingComments != nil || idecl.WidowComments != nil {
		if idecl.Doc == nil {
			idecl.Doc = &ast.CommentGroup{}
		}

		var newList []*ast.Comment
		curList := idecl.Doc.List
		for _, cg := range idecl.FloatingComments {
			newList = append(newList, cg.List...)
		}
		newList = append(newList, curList...)
		for _, cg := range idecl.WidowComments {
			newList = append(newList, cg.List...)
		}

		// Adjust comment positions, so that they are relatively correct.
		// They will be shifted into correct place later.
		commentOffset := token.Pos(1)
		for _, c := range newList {
			c.Slash = commentOffset

			cRange := astutils.TruePosRange(c)
			commentOffset += (cRange.End - cRange.Start) + 2
		}
		idecl.Doc.List = newList
	}

	// Place the comment at the offset.
	if idecl.Doc != nil {
		astutils.AdjustCommentGroupPos(offset-idecl.Doc.Pos(), idecl.Doc)
		for _, c := range idecl.Doc.List {
			// Ensure all comments start from new line.
			newLines = append(newLines, c.Pos())

			// All newlines inside the comments need to be preserved otherwise printer will not be happy.
			for _, idx := range findAllIndexes(c.Text, "\n") {
				newLines = append(newLines, c.Pos()+token.Pos(idx))
			}
		}
		offset = idecl.Doc.End() + 1
		idecl.spec.Doc = idecl.Doc
	}

	// Ensure declaration starts from a new line.
	newLines = append(newLines, offset)
	astutils.AdjustGenDeclPos(offset-idecl.spec.Pos(), idecl.spec)

	// After adjusting the position.
	offset += token.Pos(len(token.IMPORT.String())) + 1

	// Force add parenthesis if they don't exist.
	idecl.spec.Lparen = offset
	offset++

	// Zero out Specs of the import specs, will populate them later.
	idecl.spec.Specs = nil

	for _, g := range idecl.Groups {
		if len(g.Specs) == 0 {
			continue
		}

		// Ensure there is a newline before each group.
		newLines = append(newLines, offset)
		offset++

		// And a spacer line to clearly delimit groups.
		newLines = append(newLines, offset)
		offset++

		for sIdx, s := range g.Specs {
			// Ensure specs don't have unexpected doc comment (these should come from the group).
			s.Doc = nil

			// For the first spec in the group, attach the doc comment.
			if sIdx == 0 {
				if g.Doc != nil {
					astutils.AdjustCommentGroupPos(offset-g.Doc.Pos(), g.Doc)
					offset = g.Doc.End() + 1
					s.Doc = g.Doc
				}
			}

			astutils.AdjustImportSpecPos(offset-s.Pos(), s)
			offset = s.End() + 1
			if s.Comment != nil {
				offset = s.Comment.End() + 1
			}
			idecl.spec.Specs = append(idecl.spec.Specs, s)
		}
	}

	idecl.spec.Rparen = offset
	offset++

	return idecl.spec, newLines, offset
}
