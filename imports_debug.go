package gofancyimports

import (
	"encoding/json"
	"go/token"
	"strings"

	"github.com/NonLogicalDev/gofancyimports/internal/astutils"
)

func DumpImportDeclList(fset *token.FileSet, decls []ImportDecl) json.RawMessage {
	var msgs []json.RawMessage
	for _, decl := range decls {
		msgs = append(msgs, DumpImportDecl(fset, decl))
	}
	out, _ := json.Marshal(msgs)
	return out
}

func DumpImportDecl(fset *token.FileSet, id ImportDecl) json.RawMessage {
	m := map[string]interface{}{}

	if id.Doc != nil {
		m["comment.doc"] = splitLines(id.Doc.Text())
	} else {
		m["comment.doc"] = nil
	}

	var commentWidow []interface{}
	for _, c := range id.WidowComments {
		commentWidow = append(commentWidow, splitLines(c.Text()))
	}
	m["comment.widow"] = commentWidow

	var commentFloat []interface{}
	for _, c := range id.FloatingComments {
		commentFloat = append(commentFloat, splitLines(c.Text()))
	}
	m["comment.float"] = commentFloat

	var groups []json.RawMessage
	for _, g := range id.Groups {
		groups = append(groups, g.JSON(fset))
	}
	m["importGroups"] = groups

	out, _ := json.Marshal(m)
	return out
}

func (isg *ImportSpecGroup) JSON(fset *token.FileSet) json.RawMessage {
	m := map[string]interface{}{}

	if isg.Doc != nil {
		m["comment.doc"] = splitLines(isg.Doc.Text())
	} else {
		m["comment.doc"] = nil
	}

	var specs []interface{}
	for _, s := range isg.Specs {
		specs = append(specs, splitLines(astutils.PrintNodeString(fset, s)))
	}
	m["specs"] = specs

	out, _ := json.Marshal(m)
	return out
}

func splitLines(in string) []string {
	return strings.Split(in, "\n")
}
