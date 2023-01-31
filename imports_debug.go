//go:build debug

package gofancyimports

import (
	"bytes"
	"fmt"
	"go/token"
	"strings"

	"github.com/NonLogicalDev/go.fancyimports/internal/astutils"
)

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
		fmt.Fprint(o, prefixLines("  ", astutils.PrintNodeString(fset, s)))
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
