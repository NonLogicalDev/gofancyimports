package gofancyimports

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"os"

	"github.com/sanity-io/litter"
)

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
