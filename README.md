# Framework for Re-Formatting GoLang Imports according to your desire.

There are plenty of tools for formatting GoLang. Most of them do a fairly good
job at tidying up imports.  However they tend to think of imports rather
simplistically.

The world of Go Imports formatting divides into two most common approaches:

1. Trust the programmer's groupings, and don't muck around with them, only sort.
	* [`go imports`][1]
2. Don't trust the programmer's grouping and impose a set of opinionated restrictive rules on how imports should be grouped.
	* [`go fumpt`][2] / [`gopls`][2]

This project stems from a whish to make programmaticaly defining rules for organizing  imports eaiser. Go is blessed with a very well documented parser and AST implementation, but one of its biggest shortcomings is dealing with comments, and whitespace.

It is extremely tedious as I have discovered while writing this package. And it rings especially true when dealing with imports, because it is all about managing comments and whitespace.

> The difficulty of working with comments in Go AST mainly stems from the fact that Comments are do not quite behave like other AST nodes.
>
> Firstly they are not part of the tree unless they are referenced by another AST node as either Doc or Line End comment.
>
> And secondly they are very rigidly tied to the Byte offsets of corresponding files, meaning making changes to them or AST nodes to which they are attached reqires recaclulating their offset positions.

With advent of tools like [`go analysis`][3] which are very flexible about ispecting and modifying code, a tool for programmatically working with imports groupings is sorely needed in my opinion. A tool that exposes import groups as a concept to the rule writer, and allow them to reorder, merge and split them as they see fit.

If only you did not have to reinvent the bycicle (write logic for aligning comments, line breaks etc), but could operate on import groups and associated comments in a more structured  way...

## Solution `gofancyimports`

This framework takes away the difficulty from dealing with floating comments, and whitespace between import spec groupings, by exposing import declarations and groupings as simple structs that you can modify at will.  All of the complexity of rebuilding the imports to your spec represented by those structs is taken care of.

This framework can understand import clauses like this, all comments in this example are exposed structuraly and when rendered to string are properly positioned, no matter how you reorder the import groups.

```go

import "singleImport" // [Import Spec Comment: singleImport]

// [Import Decl Floating Comment: hoisted to Import Decl that follows]

// [Import Decl Doc Comment: for entire import block]
/*
	Multiline comments are understood and handled properly.
*/
import (
	"pkg1" // [Import Spec Comment: pkg1]
	"pkg2"

	// [Import Decl Widow comment 1: unattached to Import Specs, but exposed in enclosing Import Decl]

	// [Import Spec Group Doc Comment: (pkg3, pkg4)]
	/*
		Multiline comments are understood and handled properly.
	*/
	"pkg3"
	"pkg4"

	// [Import Decl Widow comment 2: unattached to Import Specs, but exposed in enclosing Import Decl]
)
```

This package mainly exposes following high level interface:

```go
type (
	// ImportOrganizer is a function that allows reordering merging
	// and splitting existing ImportDecl's obtained from source.
	ImportOrganizer func(decls []ImportDecl) []ImportDecl

	// ImportDecl represents a single import block, a 1:1 mapping.
	ImportDecl struct {
		// FloatingComments comments that are floating above
		// this declaration, yet in the middle of import blocks.
		FloatingComments []*ast.CommentGroup

		// WidowComments are comments that are floating inside this declaration unattached to specs.
		WidowComments []*ast.CommentGroup

		// Doc is the doc comment for this import gropup.
		Doc    *ast.CommentGroup
		Groups []ImportSpecGroup
	}

	// ImportSpecGroup maps to set of consecutive import specs delimited by
	// whitespace and potentially having a doc comment.
	//
	// This type is the powerhouse of this package, allowing easy operation
	// on sets of imports, delimited by whitespace.
	//
	// Contained within an ImportDecl.
	ImportSpecGroup struct {
		Doc   *ast.CommentGroup
		Specs []*ast.ImportSpec
	}
)

// RewriteImports takes same arguments as `go/parser.ParseFile` with an addition of `rewriter`
// and returns original source with imports grouping modified according to the rewriter.
func RewriteImports(filename string, src []byte, rewriter ImportOrganizer) ([]byte, error)
```

You can see the ease of use of this by having a look at:
* [go-fancy-imports/main.go](./cmd/go-fancy-imports/main.go)
* [go-fancy-imports/rules.go](./cmd/go-fancy-imports/rules.go)


# Appendix:

Good example of how easy using `go analysis` is:
* https://arslan.io/2020/07/07/using-go-analysis-to-fix-your-source-code/

[1]: https://pkg.go.dev/golang.org/x/tools/cmd/goimports
[2]: https://github.com/mvdan/gofumpt
[3]: https://pkg.go.dev/golang.org/x/tools/go/analysis