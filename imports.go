package gofancyimports

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/NonLogicalDev/gofancyimports/internal/astutils"
	"github.com/NonLogicalDev/gofancyimports/pkg/organizer/autogroup"
	"github.com/NonLogicalDev/gofancyimports/pkg/types"
)

type (
	rewriteConfig struct {
		transform  types.ImportTransform
		printerCfg *printer.Config
	}

	Option func(opt *rewriteConfig)
)

var (
	_defaultPrinterConfig = &printer.Config{
		Mode:     printer.UseSpaces | printer.TabIndent,
		Tabwidth: 8,
	}

	_defaultTransform = autogroup.New()
)

// WithPrinterConfig allows overriding a custom printer config.
func WithPrinterConfig(config *printer.Config) Option {
	return func(cfg *rewriteConfig) {
		if config != nil {
			cfg.printerCfg = config
		}
	}
}

// WithTransform allows overriding a custom import group transform.
func WithTransform(transform types.ImportTransform) Option {
	return func(cfg *rewriteConfig) {
		if transform != nil {
			cfg.transform = transform
		}
	}
}

// RewriteImportsSource takes same arguments as `go/parser.ParseFile` with an addition of `rewriter`
// and returns original source with imports grouping modified according to the rewriter.
func RewriteImportsSource(filename string, src []byte, opts ...Option) ([]byte, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed parsing file: %w", err)
	}

	edits, err := RewriteImportsAST(fset, node, src, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed rewriting AST: %w", err)
	}
	if len(edits) == 0 {
		return src, nil
	}

	// Import rewriter only returns one edit.
	edit := edits[0]
	output := ApplyTextEdit(fset, node, src, edit)
	return output, err
}

func RewriteImportsAST(fset *token.FileSet, node *ast.File, src []byte, opts ...Option) ([]*analysis.TextEdit, error) {
	config := rewriteConfig{
		transform:  _defaultTransform,
		printerCfg: _defaultPrinterConfig,
	}
	for _, apply := range opts {
		apply(&config)
	}

	importDeclRange, err := ParseImportDeclarations(fset, node)
	if err != nil {
		return nil, fmt.Errorf("while gathering declarations: %w", err)
	}

	// if there are no imports all together we shall insert our imports at a character right after package name.
	if importDeclRange.Pos == token.NoPos {
		importDeclRange.Pos = node.Name.End() + 1
		importDeclRange.End = importDeclRange.Pos
	}

	// Make sure that there are at least two newlines between imports and other declarations.
	var (
		f = fset.File(node.Package)

		addPaddingLeft string
		startOffset    = f.Offset(importDeclRange.Pos)

		addPaddingRight string
		endOffset       = f.Offset(importDeclRange.End)
	)
	if startOffset-1 > 0 && startOffset-1 < len(src) && src[startOffset-1] != '\n' {
		addPaddingLeft += "\n"
	}
	if startOffset-2 > 0 && startOffset-2 < len(src) && src[startOffset-1] != '\n' {
		addPaddingLeft += "\n"
	}
	if endOffset+1 > 0 && endOffset+1 < len(src) && src[endOffset+1] != '\n' {
		addPaddingRight += "\n"
	}
	if endOffset+2 > 0 && endOffset+2 < len(src) && src[endOffset+1] != '\n' {
		addPaddingRight += "\n"
	}

	transformedDecls := config.transform(importDeclRange.Statements)
	importDecls, newLines, newImportDeclRangeEnd := buildImportDecls(importDeclRange.Pos, transformedDecls)

	var importString string
	if importDecls != nil && newImportDeclRangeEnd != 0 {
		var err error
		importString, err = printImportDecls(
			f.Base(), int(newImportDeclRangeEnd)-f.Base(), newLines, importDecls, config.printerCfg,
		)
		if err != nil {
			return nil, fmt.Errorf("while serializing re-written import declarations: %w", err)
		}
	}
	if importString != "" {
		importString = addPaddingLeft + importString + addPaddingRight
	}
	importStringOriginal := string(src[f.Offset(importDeclRange.Pos):f.Offset(importDeclRange.End)])
	if importString == importStringOriginal {
		return nil, nil
	}
	return []*analysis.TextEdit{{
		Pos:     importDeclRange.Pos,
		End:     importDeclRange.End,
		NewText: []byte(importString),
	}}, nil
}

// ApplyTextEdit applies a single text edit to the source (for use in conjunction with RewriteImportsAST)
func ApplyTextEdit(fset *token.FileSet, node *ast.File, src []byte, edit *analysis.TextEdit) []byte {
	f := fset.File(node.Package)

	offsetStart := f.Offset(edit.Pos)
	offsetEnd := f.Offset(edit.End)
	fullFileSize := offsetStart + len(edit.NewText) + (len(src) - offsetEnd)

	output := make([]byte, 0, fullFileSize)
	output = append(output, src[:offsetStart]...)
	output = append(output, edit.NewText...)
	output = append(output, src[offsetEnd:]...)

	return output
}

func printImportDecls(
	importBase int,
	importSize int,
	newLines []token.Pos,
	importDecls []ast.Decl,
	printerCfg *printer.Config,
) (string, error) {
	if len(importDecls) == 0 {
		return "", nil
	}

	// Create a new fake file without line definitions, that we will use to render new imports into.
	fset := token.NewFileSet()
	f := fset.AddFile("imports.go", importBase, importSize)

	// Import lines generated from the import builder.
	if ok := astutils.FileSpliceLines(f, astutils.ConvertLinePosToOffsets(f.Base(), newLines)); !ok {
		return "", fmt.Errorf("can't set new lines generated from building imports")
	}

	fileNode := &ast.File{
		Name: &ast.Ident{
			Name:    "main",
			NamePos: token.Pos(importBase + len(token.PACKAGE.String()) + 1),
		},
		Decls: importDecls,
	}
	ast.Inspect(fileNode, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.CommentGroup:
			fileNode.Comments = append(fileNode.Comments, node)
		case *ast.ImportSpec:
			fileNode.Imports = append(fileNode.Imports, node)
		}
		return true
	})

	b := bytes.NewBuffer(nil)
	err := printerCfg.Fprint(b, fset, fileNode)
	if err != nil {
		return "", err
	}

	result := b.String()
	resultLines := strings.Split(result, "\n")

	// Cut off `package <name>` line
	result = strings.Join(resultLines[1:], "\n")

	return strings.Trim(result, "\n"), nil
}
