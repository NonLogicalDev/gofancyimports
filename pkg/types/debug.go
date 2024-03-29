package types

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

func ImportDeclarationsToJSON(fset *token.FileSet, decls []ImportDeclaration) json.RawMessage {
	var msgs []json.RawMessage
	for _, decl := range decls {
		msgs = append(msgs, ImportDelarationToJSON(fset, decl))
	}
	return mustMarshalJson(msgs)
}

func ImportDelarationToJSON(fset *token.FileSet, id ImportDeclaration) json.RawMessage {
	r := struct {
		CommentDoc      json.RawMessage
		CommentLeading  []json.RawMessage
		CommentDetached []json.RawMessage

		Groups []json.RawMessage
	}{
		CommentDoc: CommentGroupToJSON(id.Doc),
	}
	for _, c := range id.DetachedComments {
		r.CommentDetached = append(r.CommentDetached, CommentGroupToJSON(c))
	}
	for _, c := range id.LeadingComments {
		r.CommentLeading = append(r.CommentLeading, CommentGroupToJSON(c))
	}
	for _, g := range id.ImportGroups {
		r.Groups = append(r.Groups, ImportGroupToJSON(fset, g))
	}
	return mustMarshalJson(r)
}

func ImportGroupToJSON(fset *token.FileSet, isg ImportGroup) json.RawMessage {
	r := struct {
		CommentDoc json.RawMessage
		Specs      [][]string
	}{
		CommentDoc: CommentGroupToJSON(isg.Doc),
	}
	for _, s := range isg.Specs {
		r.Specs = append(r.Specs, splitLines(renderASTNode(fset, s)))
	}
	return mustMarshalJson(r)
}

func CommentGroupToJSON(group *ast.CommentGroup) json.RawMessage {
	if group == nil {
		return nil
	}
	r := struct {
		CommentGroup []json.RawMessage
	}{}
	for _, c := range group.List {
		r.CommentGroup = append(r.CommentGroup, CommentToJSON(c))
	}
	return mustMarshalJson(r)
}

func CommentToJSON(c *ast.Comment) json.RawMessage {
	if c == nil {
		return nil
	}
	return mustMarshalJson(struct {
		Lines []string
	}{
		Lines: splitLines(c.Text),
	})
}

func mustMarshalJson(o interface{}) json.RawMessage {
	out, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	return out
}

func splitLines(in string) []string {
	return strings.Split(in, "\n")
}

func renderASTNode(fset *token.FileSet, node interface{}) string {
	b := bytes.NewBuffer(nil)
	_ = (&printer.Config{
		Mode:     printer.UseSpaces | printer.TabIndent,
		Tabwidth: 8,
	}).Fprint(b, fset, node)
	return b.String()
}
