package main

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

type ImportDecl struct {
	// Comments that are floating above this declaration, yet in the middle of import blocks.
	FloatingComments []*ast.CommentGroup

	// Comments that are floating inside this declaration after all specs but before RParen.
	WidowComments []*ast.CommentGroup

	Doc *ast.CommentGroup
	Groups []ImportSpecGroup
	Spec *ast.GenDecl
}

type ImportSpecGroup struct {
	Doc *ast.CommentGroup
	Specs []*ast.ImportSpec
}

type posRange struct {
	start token.Pos
	end token.Pos
}

type commentRangeResult struct {
	before []*ast.CommentGroup
	inside []*ast.CommentGroup
	after  []*ast.CommentGroup
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

func ReGroupImports(groups []ImportSpecGroup) []ImportSpecGroup {
	var mergableGroups []ImportSpecGroup
	var nonMergeableGroups []ImportSpecGroup

	for _, g := range groups {
		if len(g.Specs) == 0 {
			continue
		}
		if g.Doc == nil {
			mergableGroups = append(mergableGroups, g)
		} else {
			nonMergeableGroups = append(nonMergeableGroups, g)
		}
	}
	defaultGroup := MergeImportGroups(mergableGroups)
	if len(defaultGroup.Specs) > 0 {
		return append([]ImportSpecGroup{defaultGroup}, nonMergeableGroups...)
	} else {
		return nonMergeableGroups
	}
}

func SplitSTDSpecs(specs []*ast.ImportSpec) ([]*ast.ImportSpec, []*ast.ImportSpec) {
	var stdImportSpecs []*ast.ImportSpec
	var defaultImportSpecs []*ast.ImportSpec

	for _, s := range specs {
		defaultImportSpecs = append(defaultImportSpecs, s)
	}
	return stdImportSpecs, defaultImportSpecs
}

func GatherComments(fset *token.FileSet, comments []*ast.CommentGroup, startPos, endPos token.Pos) commentRangeResult {
	result := commentRangeResult{}

	for _, c := range comments {
		if c.End() < startPos {
			result.before = append(result.before, c)
		} else if c.Pos() > endPos {
			result.after = append(result.after, c)
		} else {
			result.inside = append(result.inside, c)
		}
	}

	return result
}

func GatherImportDecls(fset *token.FileSet, cgs []*ast.CommentGroup, decls []ast.Decl) ([]ImportDecl, []ast.Decl) {
	var (
		nonImportDecls []ast.Decl
		importDecls []ImportDecl
		lastDecl ast.Decl
	)

	for _, decl := range decls {
		if gdecl, ok := decl.(*ast.GenDecl); ok && gdecl.Tok == token.IMPORT {
			currDecl := ImportDecl{
				Doc:  gdecl.Doc,
				Spec: gdecl,
			}

			declRange := TruePosRange(gdecl)

			var (
				groups []ImportSpecGroup
				groupIdx = 0
				prevSpec ast.Spec
			)
			for _, spec := range gdecl.Specs {
				specRange := TruePosRange(spec)

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
					for _, cg := range cgs {
						prevSpecRange := TruePosRange(prevSpec)
						if cg.Pos() > prevSpecRange.end && cg.Pos() < specRange.start {
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


			for _, cg := range cgs {
				if lastDecl != nil {
					lastDeclRange := TruePosRange(lastDecl)
					// Find floating comments that start after last declaration.
					if cg.Pos() > lastDeclRange.end && cg.Pos() < declRange.start {
						currDecl.FloatingComments = append(currDecl.FloatingComments, cg)
					}
				}
				if prevSpec != nil && gdecl.Rparen != token.NoPos {
					// Find floating comments that are not attachable to any spec.
					prevSpecRange := TruePosRange(prevSpec)
					if cg.Pos() < gdecl.Rparen && cg.Pos() > prevSpecRange.end {
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

	return importDecls, nonImportDecls
}


func BuildDecl(fset *token.FileSet, offset token.Pos, idecl ImportDecl) (ast.Decl, token.Pos, []int)  {
	var newLines []int

	// Add initial newline at the beginning of the group. (before comment)
	newLines = append(newLines, int(offset))
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

			cRange := TruePosRange(c)
			commentOffset += (cRange.end - cRange.start) + 2
		}
		idecl.Doc.List = newList
	}

	// Place the comment at the offset.
	if idecl.Doc != nil {
		AdjustCommentGroupPos(offset-idecl.Doc.Pos(), idecl.Doc)
		for _, c := range idecl.Doc.List {
			// Ensure all comments start from new line.
			newLines = append(newLines, int(c.Pos()))
			// All newlines inside the comments need to be preserved otherwise printer will not be happy.
			for _, idx := range FindAllIndexes(c.Text, "\n") {
				newLines = append(newLines, int(c.Pos()) + idx)
			}
		}
		offset = idecl.Doc.End() + 1
		idecl.Spec.Doc = idecl.Doc

	}

	// Ensure declaration starts from a new line.
	newLines = append(newLines, int(offset))
	AdjustGenDeclPos(offset-idecl.Spec.Pos(), idecl.Spec)


	offset = idecl.Spec.TokPos + 7 + 1
	if idecl.Spec.Lparen != 0 {
		offset = idecl.Spec.Lparen + 1
	} else {
		// Force add parenthesis if they don't exist.
		idecl.Spec.Lparen = offset
		offset++
	}

	idecl.Spec.Specs = nil
	groups := ReGroupImports(idecl.Groups)

	for _, g := range groups {
		if len(g.Specs) == 0 {
			continue
		}

		// Ensure there is a newline before each group.
		newLines = append(newLines, int(offset))
		offset++

		// And a spacer line to clearly delimit groups.
		newLines = append(newLines, int(offset))
		offset++

		for sIdx, s := range g.Specs {
			// Ensure specs don't have unexpected doc comment (these should come from the group).
			s.Doc = nil

			// For the first spec in the group, attach the doc comment.
			if sIdx == 0 {
				if g.Doc != nil {
					AdjustCommentGroupPos(offset-g.Doc.Pos(), g.Doc)
					offset = g.Doc.End() + 1
					s.Doc = g.Doc
				}
			}

			AdjustImportSpecPos(offset-s.Pos(), s)
			offset = s.End() + 1
			if s.Comment != nil {
				offset = s.Comment.End() + 1
			}
			idecl.Spec.Specs = append(idecl.Spec.Specs, s)
		}
	}

	idecl.Spec.Rparen = offset
	offset++

	return idecl.Spec, offset, newLines
}

func BuildImportDecls(fset *token.FileSet, imports []ImportDecl) ([]ast.Decl, []int) {
	var decls []ast.Decl
	if len(imports) == 0 {
		return nil, nil
	}

	offset := imports[0].Spec.Pos()
	offsetBase := fset.File(offset).Base()

	var newLines []int
	for _, d := range imports {
		decl, newOffset, nl := BuildDecl(fset, offset, d)
		newLines = append(newLines, nl...)
		decls = append(decls, decl)
		offset = newOffset
	}

	localisedLines := make([]int, len(newLines))
	for i, l := range newLines {
		localisedLines[i] = l - offsetBase
	}
	return decls, localisedLines
}

func TruePosRange(node ast.Node) posRange {
	var start, end token.Pos

	switch n := node.(type) {
	case *ast.GenDecl:
		if n.Doc != nil {
			start = n.Doc.Pos()
		} else {
			start = n.Pos()
		}

		if n.Lparen == token.NoPos {
			r := TruePosRange(n.Specs[0])
			end = r.end
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

func FindAllIndexes(s string, c string) []int {
	var r []int
	for i:=0; i<len(s); {
		if idx := strings.Index(s[i:], c); idx >= 0 {
			r = append(r, i+idx)
			i += idx + len(c)
		} else {
			break
		}
	}
	return r

}