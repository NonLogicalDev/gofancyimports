# Framework for Re-Formatting GoLang Imports according to your desire.

There are plenty of tools for formatting GoLang. Most of them do a fairly good
job at tidying up imports.  However they tend to think of imports rather
simplistically.

The world of Go Imports formatting divides into two most common approaches:

1. Trust the programmer's groupings, and don't muck around with them, only sort.
	* [`go imports`][1]
2. Don't trust the programmer's grouping and impose a set of opinionated restrictive rules on how imports should be grouped.
	* [`go fumpt`][2] / [`gopls`][2]

This project stems from a whish to make programmaticaly defining rules for organizing  imports eaiser. Go is blessed with a very well documented parser and AST implementation, but one of its biggest shortcomings is dealing with comments, and whitespace. It is extremely tedious as I have discoverd while writing this package.

> The difficulty mainly stems from the fact that Comments are not pure AST nodes, and are very rigidly tied to the Byte offsets of corresponding files, meaning making changes to them or AST nodes to which they are attached reqires recaclulating their offset positions.

With advent of tools like `go analysis` which are very flexible about ispecting and modifying code, a tool for programmatically working with imports groupings is sorely needed in my opinion. A tool that exposes import groups as a concept to the rule writer, and allow them to reorder, merge and split them as they see fit.

This tool can understand import clauses like this, all comments in this example are exposed structuraly and when rendered to string are properly positioned, no matter how you reorder the import groups.

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

[1]: https://pkg.go.dev/golang.org/x/tools/cmd/goimports
[2]: https://github.com/mvdan/gofumpt