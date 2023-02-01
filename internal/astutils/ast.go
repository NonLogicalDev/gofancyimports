package astutils

import (
	"go/ast"
	"go/token"
)

type PosRange struct {
	Start token.Pos
	End   token.Pos
}

func TruePosRange(node ast.Node) PosRange {
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
			end = r.End
		} else {
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

	return PosRange{start, end}
}

func ShiftImportSpecPos(delta token.Pos, spec *ast.ImportSpec) {
	if spec == nil {
		return
	}

	if spec.Name != nil {
		spec.Name.NamePos += delta
	}

	spec.Path.ValuePos += delta
	if spec.EndPos != 0 {
		spec.EndPos += delta
	}

	ShiftCommentGroupPos(delta, spec.Comment)
}

func ShiftGenDeclPos(delta token.Pos, spec *ast.GenDecl) {
	if spec == nil {
		return
	}

	spec.TokPos += delta
	if spec.Lparen > 0 {
		spec.Lparen += delta
	}
	if spec.Rparen > 0 {
		spec.Rparen += delta
	}
}

func ShiftCommentGroupPos(delta token.Pos, cg *ast.CommentGroup) {
	if cg == nil {
		return
	}

	for _, spec := range cg.List {
		spec.Slash += delta
	}
}

func FileSpliceLines(f *token.File, newLines []int) bool {
	if len(newLines) == 0 {
		return true
	}
	lines := fileGetLines(f)
	merged := mergeSortedListsUniq(lines, newLines, f.Size())
	return f.SetLines(merged)
}

func ConvertLinePosToOffsets(base int, lines []token.Pos) []int {
	result := make([]int, len(lines))
	for i, linePos := range lines {
		result[i] = int(linePos) - base
	}
	return result
}
