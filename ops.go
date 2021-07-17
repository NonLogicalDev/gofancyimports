package main

import (
	"go/ast"
	"go/token"
)

// TODO:
//	 if   : declaration group is import and it has comment group, then preserve it.
//   else :
//	   preserve continious groups of imports if first element has a comment group attached. (Doc)
//

type ImportDecl struct {
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

func MergeImportDecls(decls []ImportDecl) ImportDecl {
	var merged ImportDecl

	var groups []ImportSpecGroup
	for _, d := range decls {
		if merged.Spec == nil {
			merged.Spec = d.Spec
			merged.Doc = d.Doc
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
		if g.Doc == nil {
			mergableGroups = append(mergableGroups, g)
		} else {
			nonMergeableGroups = append(nonMergeableGroups, g)
		}
	}
	defaultGroup := MergeImportGroups(mergableGroups)
	return append([]ImportSpecGroup{defaultGroup}, nonMergeableGroups...)
}

func SplitSTDSpecs(specs []*ast.ImportSpec) ([]*ast.ImportSpec, []*ast.ImportSpec) {
	var stdImportSpecs []*ast.ImportSpec
	var defaultImportSpecs []*ast.ImportSpec

	for _, s := range specs {
		defaultImportSpecs = append(defaultImportSpecs, s)
	}
	return stdImportSpecs, defaultImportSpecs
}


func GatherImportDecls(fset *token.FileSet, decls []ast.Decl) ([]ImportDecl, []ast.Decl) {
	var (
		nonImportDecls []ast.Decl
		importDecls []ImportDecl
	)

	for _, decl := range decls {
		if gdecl, ok := decl.(*ast.GenDecl); ok && gdecl.Tok == token.IMPORT {
			var (
				groups = []ImportSpecGroup{}
				groupIdx = 0
			)

			var prevSpec ast.Spec
			for _, spec := range gdecl.Specs {
				// Check if line difference is greater than one, and reset sticky Group.
				if prevSpec != nil &&  fset.Position(spec.Pos()).Line - fset.Position(prevSpec.Pos()).Line > 1 {
					groupIdx++
				}

				shouldRecordComment := false
				if groupIdx > len(groups)-1 {
					shouldRecordComment = true
					groups = append(groups, ImportSpecGroup{})
				}

				if ispec, ok := spec.(*ast.ImportSpec); ok {
					if shouldRecordComment {
						groups[groupIdx].Doc = ispec.Doc
						ispec.Doc = nil
					}
					groups[groupIdx].Specs = append(groups[groupIdx].Specs, ispec)
				}

				prevSpec = spec
			}

			importDecls = append(importDecls, ImportDecl{
				Doc:    gdecl.Doc,
				Spec:   gdecl,
				Groups: groups,
			})
		} else {
			nonImportDecls = append(nonImportDecls, decl)
		}
	}

	return importDecls, nonImportDecls
}


func BuildDecl(fset *token.FileSet, node *ast.File, offset token.Pos, idecl ImportDecl) (ast.Decl, token.Pos)  {
	f := fset.File(offset)
	var newLines []int

	// Place the comment at the offset.
	if idecl.Doc != nil {
		AdjustCommentGroupPos(offset-idecl.Doc.Pos(), idecl.Doc)
		offset = idecl.Doc.End() + 1
		idecl.Spec.Doc = idecl.Doc
	}

	AdjustGenDeclPos(offset-idecl.Spec.Pos(), idecl.Spec)

	offset = idecl.Spec.TokPos + 7 + 1
	if idecl.Spec.Lparen != 0 {
		offset = idecl.Spec.Lparen + 1
	}


	idecl.Spec.Specs = nil
	groups := ReGroupImports(idecl.Groups)
	for _, g := range groups {
		newLines = append(newLines, f.Offset(offset))
		newLines = append(newLines, f.Offset(offset+1))
		offset += 1

		first := true
		for _, s := range g.Specs {
			s.Doc = nil

			if first {
				if g.Doc != nil {
					AdjustCommentGroupPos(offset-g.Doc.Pos(), g.Doc)
					offset = g.Doc.End() + 1
					s.Doc = g.Doc
				}
				first = false
			}

			AdjustImportSpecPos(offset-s.Pos(), s)
			offset = s.End() + 1
			if s.Comment != nil {
				offset = s.Comment.End() + 1
			}
			idecl.Spec.Specs = append(idecl.Spec.Specs, s)
		}
	}

	if !FileSpliceLines(f, newLines) {
		panic("oops")
	}

	return idecl.Spec, offset
}

func BuildImportDecls(fset *token.FileSet, node *ast.File, defaultImports ImportDecl, namedImports []ImportDecl) []ast.Decl {
	var decls []ast.Decl

	offset := defaultImports.Spec.Pos()
	for _, d := range append([]ImportDecl{defaultImports}, namedImports...) {
		decl, newOffset := BuildDecl(fset, node, offset, d)
		decls = append(decls, decl)
		offset = newOffset
	}

	return decls
}
