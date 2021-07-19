package gofancyimports

import (
	"go/ast"
	"go/token"
)

type (
	ImportDeclRange struct{
		Decls    []ImportDecl
		Comments CommentRange

		Start token.Pos
		End   token.Pos
		Base  int
	}

	CommentRange struct {
		Before []*ast.CommentGroup
		Inside []*ast.CommentGroup
		After  []*ast.CommentGroup
	}
)

func GatherComments(fset *token.FileSet, comments []*ast.CommentGroup, startPos, endPos token.Pos) CommentRange {
	result := CommentRange{}

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

func GatherImportDecls(fset *token.FileSet, nodeDecls []ast.Decl, nodeComments []*ast.CommentGroup) (ImportDeclRange, []ast.Decl) {
	var (
		nonImportDecls []ast.Decl
		importDecls []ImportDecl
		lastDecl ast.Decl
	)

	for _, decl := range nodeDecls {
		if gdecl, ok := decl.(*ast.GenDecl); ok && gdecl.Tok == token.IMPORT {
			currDecl := ImportDecl{
				Doc:  gdecl.Doc,
				spec: gdecl,
			}

			declRange := truePosRange(gdecl)

			var (
				groups []ImportSpecGroup
				groupIdx = 0
				prevSpec ast.Spec
			)
			for specIdx, spec := range gdecl.Specs {
				specRange := truePosRange(spec)

				// Check if line difference is greater than one, and reset Group Index.
				if prevSpec != nil &&  fset.Position(spec.Pos()).Line - fset.Position(prevSpec.Pos()).Line > 1 {
					groupIdx++
				}
				shouldRecordComment := false
				if groupIdx > len(groups)-1 {
					shouldRecordComment = true
					groups = append(groups, ImportSpecGroup{})
				}

				currGroup := groups[groupIdx]

				for _, cg := range nodeComments {
					if prevSpec != nil {
						// Find floating comments that start after previous spec and end before this one.
						prevSpecRange := truePosRange(prevSpec)
						if cg.Pos() > prevSpecRange.End && cg.Pos() < specRange.Start {
							currDecl.WidowComments = append(currDecl.WidowComments, cg)
						}
					}
					// Catch comments after Open Paren.
					if specIdx == 0 && gdecl.Lparen != token.NoPos {
						if cg.Pos() > gdecl.Lparen && cg.Pos() < specRange.Start {
							currDecl.WidowComments = append(currDecl.WidowComments, cg)
						}
					}
				}


				if ispec, ok := spec.(*ast.ImportSpec); ok {
					if shouldRecordComment {
						currGroup.Doc = ispec.Doc
						ispec.Doc = nil
					}
					currGroup.Specs = append(currGroup.Specs, ispec)
				}

				groups[groupIdx] = currGroup
				prevSpec = spec
			}
			currDecl.Groups = groups

			for _, cg := range nodeComments {
				if lastDecl != nil {
					lastDeclRange := truePosRange(lastDecl)
					// Find floating comments that start after last declaration.
					if cg.Pos() > lastDeclRange.End && cg.Pos() < declRange.Start {
						currDecl.FloatingComments = append(currDecl.FloatingComments, cg)
					}
				}
				if prevSpec != nil && gdecl.Rparen != token.NoPos {
					// Find floating comments that are not attachable to any spec.
					prevSpecRange := truePosRange(prevSpec)
					if cg.Pos() < gdecl.Rparen && cg.Pos() > prevSpecRange.End {
						currDecl.WidowComments = append(currDecl.WidowComments, cg)
					}
				}
			}

			importDecls = append(importDecls, currDecl)
		} else {
			nonImportDecls = append(nonImportDecls, decl)
		}


		lastDecl = decl
	}

	var firstPos, lastPos token.Pos
	for _, d := range importDecls {
		specPosRange := truePosRange(d.spec)

		if specPosRange.End > lastPos {
			lastPos = specPosRange.End
		}
		if firstPos == token.NoPos || specPosRange.Start < firstPos {
			firstPos = specPosRange.Start
		}

		d.spec.Doc = nil
	}

	basePos := 0
	if firstPos != token.NoPos {
		basePos = fset.File(firstPos).Base()
	}
	return ImportDeclRange{
		Decls: importDecls,
		Comments: GatherComments(fset, nodeComments, firstPos, lastPos),

		Start: firstPos,
		End:   lastPos,
		Base:  basePos,
	}, nonImportDecls
}

func BuildImportDecl(fset *token.FileSet, offset token.Pos, addNewline bool, idecl ImportDecl) (ast.Decl, []token.Pos, token.Pos)  {
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

			cRange := truePosRange(c)
			commentOffset += (cRange.End - cRange.Start) + 2
		}
		idecl.Doc.List = newList
	}

	// Place the comment at the offset.
	if idecl.Doc != nil {
		adjustCommentGroupPos(offset-idecl.Doc.Pos(), idecl.Doc)
		for _, c := range idecl.Doc.List {
			// Ensure all comments start from new line.
			newLines = append(newLines, c.Pos())

			// All newlines inside the comments need to be preserved otherwise printer will not be happy.
			for _, idx := range findAllIndexes(c.Text, "\n") {
				newLines = append(newLines, c.Pos() + token.Pos(idx))
			}
		}
		offset = idecl.Doc.End() + 1
		idecl.spec.Doc = idecl.Doc
	}

	// Ensure declaration starts from a new line.
	newLines = append(newLines, offset)
	adjustGenDeclPos(offset-idecl.spec.Pos(), idecl.spec)

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
					adjustCommentGroupPos(offset-g.Doc.Pos(), g.Doc)
					offset = g.Doc.End() + 1
					s.Doc = g.Doc
				}
			}

			adjustImportSpecPos(offset-s.Pos(), s)
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

func BuildImportDecls(fset *token.FileSet, offset token.Pos, idecls []ImportDecl) ([]ast.Decl, []token.Pos, token.Pos) {
	var decls []ast.Decl
	if len(idecls) == 0 {
		return nil, nil, 0
	}

	var newLines []token.Pos
	for i, d := range idecls {
		decl, nl, newOffset := BuildImportDecl(fset, offset, i != 0, d)
		newLines = append(newLines, nl...)
		decls = append(decls, decl)
		offset = newOffset
	}
	return decls, newLines, offset
}
