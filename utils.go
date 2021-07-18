package main

import (
	"bytes"
	"fmt"
	"github.com/sanity-io/litter"
	"go/ast"
	"go/printer"
	"go/token"
	"os"
	"strings"
)

func AdjustImportSpecPos(delta token.Pos, spec *ast.ImportSpec) {
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

	AdjustCommentGroupPos(delta, spec.Comment)
}

func aa(spec *ast.GenDecl)  {

}

func AdjustGenDeclPos(delta token.Pos, spec *ast.GenDecl) {
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

func AdjustCommentGroupPos(delta token.Pos, cg *ast.CommentGroup) {
	if cg == nil {
		return
	}

	for _, spec := range cg.List {
		spec.Slash += delta
	}
}

func PrintNode(fset *token.FileSet, node interface{}, dump bool) {
	fmt.Println("> ======")
	_ = printer.Fprint(os.Stdout, fset, node)
	fmt.Println("")

	if dump {
		fmt.Println(">> ### ====== ======")
		litter.Dump(node)
		fmt.Println("<< ### ====== ======")
	}
}
func PrintNodeString(fset *token.FileSet, node interface{}) string {
	b := bytes.NewBuffer(nil)
	_ = printer.Fprint(b, fset, node)
	return b.String()
}

func PrintImportSpecs(fset *token.FileSet, gdecl *ast.GenDecl) {
	for _, spec := range gdecl.Specs {
		if ispec, ok := spec.(*ast.ImportSpec); ok {
			nodeString := PrintNodeString(fset, ispec)
			if nodeString == "\"os\"" || nodeString[len(nodeString)-1]=='\n' {
				PrintNode(fset, ispec, true)
			}
		}
	}
}

func (id *ImportDecl) Print(fset *token.FileSet) string {
	o := bytes.NewBuffer(nil)

	fmt.Fprintln(o, "docComment: >")
	if id.Doc != nil {
		fmt.Fprint(o, prefixLines("  ", id.Doc.Text()))
		fmt.Fprintln(o, "  ^^")
	} else {
		fmt.Fprint(o, prefixLines("  ", "[N/A]"))
	}

	fmt.Fprintln(o, "groups:")
	for _, g := range id.Groups {
		groupStr := prefixLines("  ", g.Print(fset))
		groupStr = "-" + groupStr[1:]
		fmt.Fprint(o, groupStr)
	}

	return o.String()
}

func (isg *ImportSpecGroup) Print(fset *token.FileSet) string {
	o := bytes.NewBuffer(nil)

	fmt.Fprintln(o, "docComment: >")
	if isg.Doc != nil {
		fmt.Fprint(o, prefixLines("  ", isg.Doc.Text()))
		fmt.Fprintln(o, "  ^^")
	} else {
		fmt.Fprint(o, prefixLines("  ", "[N/A]"))
	}

	fmt.Fprintln(o, "specs:")
	for _, s := range isg.Specs {
		fmt.Fprintln(o, "- >")
		fmt.Fprint(o, prefixLines("  ", PrintNodeString(fset, s)))
		fmt.Fprintln(o, "  ^^")
	}

	return o.String()
}

func prefixLines(prefix string, target string) string {
	o := bytes.NewBuffer(nil)
	lines := strings.Split(target, "\n")
	for _, line := range lines {
		fmt.Fprintf(o, "%s%s\n", prefix, line)
	}
	return o.String()
}

func FileGetLines(f *token.File) []int {
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
	lines := FileGetLines(f)
	merged := mergeSortedListsUniq(lines, newLines, f.Size())
	return f.SetLines(merged)
}

// lists must contain only integers i.e > 0.
func mergeSortedListsUniq(aList, bList []int, max int) []int {
	lenA := len(aList)
	lenB := len(bList)

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
