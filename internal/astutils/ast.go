package astutils

import (
	"go/ast"
	"go/token"
)

type PosRange struct {
	Pos token.Pos
	End token.Pos
}

// ASTNodeRangeWithComments returns a position range for an AST note, start and end
// locations with doc and line comments attached to the node accounted for.
func ASTNodeRangeWithComments(node ast.Node) PosRange {
	var start, end token.Pos

	switch n := node.(type) {
	case *ast.GenDecl:
		if n.Doc != nil {
			start = n.Doc.Pos()
		} else {
			start = n.Pos()
		}

		if n.Lparen == token.NoPos {
			r := ASTNodeRangeWithComments(n.Specs[0])
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

// ShiftImportSpecPos moves an import spec and all of its dependent AST notes by a given
// delta in the fileset.
func ShiftImportSpecPos(delta token.Pos, spec *ast.ImportSpec) {
	if spec == nil {
		return
	}

	if spec.Name != nil {
		spec.Name.NamePos += delta
	}
	if spec.Path != nil {
		spec.Path.ValuePos += delta
	}
	if spec.EndPos != 0 {
		spec.EndPos += delta
	}

	// ShiftCommentGroupPos(delta, spec.Doc)
	ShiftCommentGroupPos(delta, spec.Comment)
}

// ShiftGenDeclPos moves a declaration spec and all of its dependent AST notes by a given
// delta in the fileset.
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

// ShiftCommentGroupPos moves a comment group and all of its dependent AST notes by a given
// delta in the fileset.
func ShiftCommentGroupPos(delta token.Pos, cg *ast.CommentGroup) {
	if cg == nil {
		return
	}

	for _, spec := range cg.List {
		spec.Slash += delta
	}
}

// FileSpliceLines adds new lines into a file.
func FileSpliceLines(f *token.File, newLines []int) bool {
	if len(newLines) == 0 {
		return true
	}
	lines := fileGetLines(f)
	merged := mergeSortedListsUniq(lines, newLines, f.Size())
	return f.SetLines(merged)
}

// ConvertLinePosToOffsets adjusts file lines by base offset for splicing into a file via SetLines call.
// Note: it is often easier to track newlines in terms of global positions in the file set, however
// SetLines excepts an array of integer offsets for lines that are relative to the file base.
func ConvertLinePosToOffsets(base int, lines []token.Pos) []int {
	result := make([]int, len(lines))
	for i, linePos := range lines {
		result[i] = int(linePos) - base
	}
	return result
}
