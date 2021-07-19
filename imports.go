package gofancyimports

import (
	"go/ast"
	"go/token"
	"strings"
)

// TODO:
//	 if   : declaration group is import and it has comment group, then preserve it.
//   else :
//	   preserve continious groups of imports if first element has a comment group attached. (Doc)
//

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

	ImportDecl struct {
		// Comments that are floating above this declaration, yet in the middle of import blocks.
		FloatingComments []*ast.CommentGroup

		// Comments that are floating inside this declaration unattached to specs.
		WidowComments []*ast.CommentGroup

		Doc *ast.CommentGroup
		Groups []ImportSpecGroup
		Spec *ast.GenDecl
	}

	ImportSpecGroup struct {
		Doc *ast.CommentGroup
		Specs []*ast.ImportSpec
	}
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
			Name: "main",
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

	return result
}

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
				Spec: gdecl,
			}

			declRange := truePosRange(gdecl)

			var (
				groups []ImportSpecGroup
				groupIdx = 0
				prevSpec ast.Spec
			)
			for _, spec := range gdecl.Specs {
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

				if prevSpec != nil {
					// Find floating comments that start after previous spec and end before this one.
					for _, cg := range nodeComments {
						prevSpecRange := truePosRange(prevSpec)
						if cg.Pos() > prevSpecRange.End && cg.Pos() < specRange.Start {
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
		if d.Spec.End() > lastPos {
			lastPos = d.Spec.End()
		}
		if d.Spec.Doc != nil && (firstPos == token.NoPos || d.Spec.Doc.Pos() < firstPos) {
			firstPos = d.Spec.Doc.Pos()
		} else if firstPos == token.NoPos || d.Spec.Pos() < firstPos {
			firstPos = d.Spec.Pos()
		}

		d.Spec.Doc = nil
	}

	return ImportDeclRange{
		Decls: importDecls,
		Comments: GatherComments(fset, nodeComments, firstPos, lastPos),

		Start: firstPos,
		End:   lastPos,
		Base:  fset.File(firstPos).Base(),
	}, nonImportDecls
}

func MergeImportDecls(decls []ImportDecl) ImportDecl {
	var merged ImportDecl

	var groups []ImportSpecGroup
	for _, d := range decls {
		if merged.Spec == nil {
			merged.Spec = d.Spec
			merged.Doc = d.Doc

			merged.FloatingComments = append(merged.FloatingComments, d.FloatingComments...)
			merged.WidowComments = append(merged.WidowComments, d.WidowComments...)
		}
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


func BuildImportDecl(fset *token.FileSet, offset token.Pos, idecl ImportDecl) (ast.Decl, []token.Pos, token.Pos)  {
	var newLines []token.Pos

	// Add initial newline at the beginning of the group. (before comment)
	newLines = append(newLines, offset)
	offset++

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
		idecl.Spec.Doc = idecl.Doc

	}

	// Ensure declaration starts from a new line.
	newLines = append(newLines, offset)
	adjustGenDeclPos(offset-idecl.Spec.Pos(), idecl.Spec)


	offset = idecl.Spec.TokPos + 7 + 1
	if idecl.Spec.Lparen != 0 {
		offset = idecl.Spec.Lparen + 1
	} else {
		// Force add parenthesis if they don't exist.
		idecl.Spec.Lparen = offset
		offset++
	}

	idecl.Spec.Specs = nil

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
			idecl.Spec.Specs = append(idecl.Spec.Specs, s)
		}
	}

	idecl.Spec.Rparen = offset
	offset++

	return idecl.Spec, newLines, offset
}

func BuildImportDecls(fset *token.FileSet, offset token.Pos, idecls []ImportDecl) ([]ast.Decl, []token.Pos, token.Pos) {
	var decls []ast.Decl
	if len(idecls) == 0 {
		return nil, nil, 0
	}

	var newLines []token.Pos
	for _, d := range idecls {
		decl, nl, newOffset := BuildImportDecl(fset, offset, d)
		newLines = append(newLines, nl...)
		decls = append(decls, decl)
		offset = newOffset
	}
	return decls, newLines, offset
}

func truePosRange(node ast.Node) posRange {
	var start, end token.Pos

	switch n := node.(type) {
	case *ast.GenDecl:
		if n.Doc != nil {
			start = n.Doc.Pos()
		} else {
			start = n.Pos()
		}

		if n.Lparen == token.NoPos {
			r := truePosRange(n.Specs[0])
			end = r.End
		} else{
			end = n.End()
		}
	case *ast.ImportSpec:
		if n.Doc != nil {
			start = n.Doc.Pos()
		} else {
			start = n.Pos()
		}
		if n.Comment != nil {
			end = n.Comment.End()
		} else {
			end = n.End()
		}
	default:
		start = n.Pos()
		end = n.End()
	}

	return posRange{start, end}
}