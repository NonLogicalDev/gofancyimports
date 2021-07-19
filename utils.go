package gofancyimports

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"sort"
	"strings"
)

type posRange struct {
	Start token.Pos
	End   token.Pos
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

func adjustImportSpecPos(delta token.Pos, spec *ast.ImportSpec) {
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

	adjustCommentGroupPos(delta, spec.Comment)
}

func adjustGenDeclPos(delta token.Pos, spec *ast.GenDecl) {
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

func adjustCommentGroupPos(delta token.Pos, cg *ast.CommentGroup) {
	if cg == nil {
		return
	}

	for _, spec := range cg.List {
		spec.Slash += delta
	}
}


func prefixLines(prefix string, target string) string {
	o := bytes.NewBuffer(nil)
	lines := strings.Split(target, "\n")
	for _, line := range lines {
		fmt.Fprintf(o, "%s%s\n", prefix, line)
	}
	return o.String()
}

// lists must contain only integers i.e > 0.
func mergeSortedListsUniq(aList, bList []int, max int) []int {
	lenA := len(aList)
	lenB := len(bList)

	sort.Ints(aList)
	sort.Ints(bList)

	result := make([]int, 0, lenA+lenB)

	a, b, prevValue := 0, 0, 0
	for ; a < lenA || b < lenB ; {
		value := 0

		// If within bounds of both lists.
		if a < lenA && b < lenB {
			if aList[a] > bList[b] {
				value = bList[b]
				b++
			} else if aList[a] < bList[b] {
				value = aList[a]
				a++
			} else { // Equality
				value = aList[a]
				a++
				b++
			}
		} else {
			if a >= lenA {
				value = bList[b]
				b++
			} else if b >= lenB {
				value = aList[a]
				a++
			}
		}

		// Ensure no duplicate elements.
		if value != prevValue && value < max {
			result = append(result, value)
			prevValue = value
		}
	}

	return result
}

func findAllIndexes(s string, c string) []int {
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

func fileGetLines(f *token.File) []int {
	lines := make([]int, f.LineCount())
	for i := 0; i < f.LineCount(); i++ {
		lines[i] = f.Offset(f.LineStart(i+1))
	}
	return lines
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
