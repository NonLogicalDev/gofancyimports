package gofancyimports

import (
	"fmt"
	"go/ast"
	"go/token"

	"github.com/NonLogicalDev/gofancyimports/internal/astutils"
	"github.com/NonLogicalDev/gofancyimports/pkg/types"
)

type (
	ImportDeclarationRange struct {
		Statements []types.ImportDeclaration
		Comments   ImportDeclarationComments

		Pos token.Pos
		End token.Pos
	}

	ImportDeclarationComments struct {
		Before []*ast.CommentGroup
		Inside []*ast.CommentGroup
		After  []*ast.CommentGroup
	}
)

func ParseImportDeclarations(fset *token.FileSet, node *ast.File) (ImportDeclarationRange, error) {
	var (
		nonImportDecls []ast.Decl
		importDecls    []types.ImportDeclaration
		lastDeclEnd    token.Pos
	)

	var firstPos, lastPos token.Pos
	for _, decl := range node.Decls {
		if importDecl, ok := decl.(*ast.GenDecl); ok && importDecl.Tok == token.IMPORT {
			importDecl, specPosRange := parseImportStatement(fset, importDecl, node.Comments, lastDeclEnd)
			importDecls = append(importDecls, importDecl)

			if specPosRange.End > lastPos {
				lastPos = specPosRange.End
			}
			if firstPos == token.NoPos || specPosRange.Pos < firstPos {
				firstPos = specPosRange.Pos
			}
		} else {
			nonImportDecls = append(nonImportDecls, decl)
		}
		lastDeclEnd = astutils.ASTNodeRangeWithComments(decl).End
	}

	invalidDecls := getDeclarationsInRange(nonImportDecls, firstPos, lastPos)
	if len(invalidDecls) > 0 {
		return ImportDeclarationRange{}, fmt.Errorf("found %d non import declarations ovelapping imports", len(nonImportDecls))
	}

	return ImportDeclarationRange{
		Statements: importDecls,
		Comments:   parseCommentsInRange(node.Comments, firstPos, lastPos),

		Pos: firstPos,
		End: lastPos,
	}, nil

}

func parseImportStatement(
	fset *token.FileSet,
	importDecl *ast.GenDecl,
	nodeComments []*ast.CommentGroup,
	nodeCommentOffset token.Pos,
) (types.ImportDeclaration, astutils.PosRange) {
	currDecl := types.ImportDeclaration{
		Doc: importDecl.Doc,
	}
	importDeclRange := astutils.ASTNodeRangeWithComments(importDecl)

	var (
		importGroups  []types.ImportGroup
		prevSpecRange astutils.PosRange
	)

	for specIdx, spec := range importDecl.Specs {
		specRange := astutils.ASTNodeRangeWithComments(spec)

		// Check if line difference is greater than one, and reset Group Index.
		shouldRecordComment := false
		if prevSpecRange.Pos == token.NoPos || lineDistance(fset, spec.Pos(), prevSpecRange.Pos) > 1 {
			shouldRecordComment = true
			importGroups = append(importGroups, types.ImportGroup{})
		}

		lastGroupIndex := len(importGroups) - 1
		currGroup := importGroups[lastGroupIndex]

		for _, cg := range nodeComments {
			// Find floating comments that start after previous spec and end before this one.
			if prevSpecRange.End != token.NoPos && cg.Pos() > prevSpecRange.End && cg.Pos() < specRange.Pos {
				currDecl.DetachedComments = append(currDecl.DetachedComments, cg)
			}

			// Catch comments after Open Paren that are not attached to any specs.
			if specIdx == 0 && importDecl.Lparen != token.NoPos {
				if cg.Pos() > importDecl.Lparen && cg.Pos() < specRange.Pos {
					currDecl.DetachedComments = append(currDecl.DetachedComments, cg)
				}
			}
		}

		if importSpec, ok := spec.(*ast.ImportSpec); ok {
			ispecCopy := copyImportSpec(importSpec)
			if shouldRecordComment {
				currGroup.Doc = ispecCopy.Doc
				ispecCopy.Doc = nil
			}
			currGroup.Specs = append(currGroup.Specs, ispecCopy)

			// Since we took off the doc comment, adjust the start of the spec range to after comment.
			prevSpecRange = astutils.ASTNodeRangeWithComments(importSpec)
			prevSpecRange.Pos = importSpec.Pos()
		} else {
			prevSpecRange = astutils.ASTNodeRangeWithComments(spec)
		}

		importGroups[lastGroupIndex] = currGroup
	}
	currDecl.ImportGroups = importGroups

	for _, cg := range nodeComments {
		// Find leading comments that start after last declaration.
		if nodeCommentOffset != token.NoPos && cg.Pos() > nodeCommentOffset && cg.Pos() < importDeclRange.Pos {
			currDecl.LeadingComments = append(currDecl.LeadingComments, cg)
		}

		if prevSpecRange.End != token.NoPos && importDecl.Rparen != token.NoPos {
			// Find floating comments that are not attachable to any spec.
			if cg.Pos() < importDecl.Rparen && cg.Pos() > prevSpecRange.End {
				currDecl.DetachedComments = append(currDecl.DetachedComments, cg)
			}
		}
	}
	return currDecl, astutils.ASTNodeRangeWithComments(importDecl)
}

func parseCommentsInRange(comments []*ast.CommentGroup, startPos, endPos token.Pos) ImportDeclarationComments {
	result := ImportDeclarationComments{}

	for _, c := range comments {
		if c.End() < startPos {
			result.Before = append(result.Before, c)
		} else if c.Pos() > endPos {
			result.After = append(result.After, c)
		} else {
			result.Inside = append(result.Inside, c)
		}
	}

	return result
}

func getDeclarationsInRange(decls []ast.Decl, pos, end token.Pos) (r []ast.Decl) {
	for _, decl := range decls {
		if (decl.Pos() > pos && decl.Pos() < end) || (decl.End() > pos && decl.End() < end) {
			r = append(r, decl)
		}
	}
	return r
}

func lineDistance(fset *token.FileSet, posA, posB token.Pos) int {
	return fset.Position(posA).Line - fset.Position(posB).Line
}
