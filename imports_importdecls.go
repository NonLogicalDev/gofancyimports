package gofancyimports

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/NonLogicalDev/go.fancyimports/internal/astutils"
)

type (
	importDeclRange struct {
		Decls    []ImportDecl
		Comments commentRange

		Start token.Pos
		End   token.Pos
		Base  int
	}

	commentRange struct {
		Before []*ast.CommentGroup
		Inside []*ast.CommentGroup
		After  []*ast.CommentGroup
	}
)

func gatherImportDecls(fset *token.FileSet, nodeDecls []ast.Decl, nodeComments []*ast.CommentGroup) (importDeclRange, []ast.Decl) {
	var (
		nonImportDecls []ast.Decl
		importDecls    []ImportDecl
		lastDecl       ast.Decl
	)

	for _, decl := range nodeDecls {
		if gdecl, ok := decl.(*ast.GenDecl); ok && gdecl.Tok == token.IMPORT {
			currDecl := ImportDecl{
				Doc:  gdecl.Doc,
				spec: gdecl,
			}

			declRange := astutils.TruePosRange(gdecl)

			var (
				groups   []ImportSpecGroup
				groupIdx = 0
				prevSpec ast.Spec
			)
			for specIdx, spec := range gdecl.Specs {
				specRange := astutils.TruePosRange(spec)

				// Check if line difference is greater than one, and reset Group Index.
				if prevSpec != nil && fset.Position(spec.Pos()).Line-fset.Position(prevSpec.Pos()).Line > 1 {
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
						prevSpecRange := astutils.TruePosRange(prevSpec)
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
					lastDeclRange := astutils.TruePosRange(lastDecl)
					// Find floating comments that start after last declaration.
					if cg.Pos() > lastDeclRange.End && cg.Pos() < declRange.Start {
						currDecl.FloatingComments = append(currDecl.FloatingComments, cg)
					}
				}
				if prevSpec != nil && gdecl.Rparen != token.NoPos {
					// Find floating comments that are not attachable to any spec.
					prevSpecRange := astutils.TruePosRange(prevSpec)
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
		specPosRange := astutils.TruePosRange(d.spec)

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
	return importDeclRange{
		Decls:    importDecls,
		Comments: gatherComments(nodeComments, firstPos, lastPos),

		Start: firstPos,
		End:   lastPos,
		Base:  basePos,
	}, nonImportDecls
}

func gatherComments(comments []*ast.CommentGroup, startPos, endPos token.Pos) commentRange {
	result := commentRange{}

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

func findAllIndexes(s string, c string) []int {
	var r []int
	for i := 0; i < len(s); {
		if idx := strings.Index(s[i:], c); idx >= 0 {
			r = append(r, i+idx)
			i += idx + len(c)
		} else {
			break
		}
	}
	return r
}
