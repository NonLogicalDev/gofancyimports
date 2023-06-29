# No-Compromise Deterministic GoLang Import Management

A mother of all tools to enforce deterministic order of imports across your golang codebase.
* ✅ Easy to use, configure or extend
* ✅ Deterministically orders toughest comment ridden imports
* ✅ Handles comments gracefully

This repo is the home for:
* `pkg/analyzer/autogroupimports` and `pkg/organizer/autogroup` 
	* ready to use, deterministic, highly configurable, pluggable, import order organizer
    * based on golang `Analyzer` framework
* `cmd/gofancyimports fix`
	* ready to use cli with full power of `pkg/organizer/autogroup` and same command line interface as `goimports`
* `gofancyimports`
  * the lower level library which allows manipulating import groups with ease for implementing your own group and comment aware import fixers

## Get the ready to use tool:

If all you need is an import sorting tool that will deterministically fix your import order to a consistent opinionated convention, grab a copy of the `gofancyimports` tool:

```
go install github.com/NonLogicalDev/gofancyimports/cmd/gofancyimports@latest
```

```
$ gofancyimports fix -h
Fixup single or multiple provided files

Usage:
  gofancyimports fix [flags]

Flags:
  -d, --diff                print diff
      --group-effect        group side effect imports
      --group-nodot         group no dot imports
  -h, --help                help for fix
  -l, --local stringArray   group local imports (comma separated prefixes)
  -w, --write               write the file back?
```

<table>
<tr>
<th colspan="2"><code>gofancyimports fix</code></th>
</tr>
<tr>
<th><code>Before</code></th>
<th><code>After</code></th>
</tr>
<tr>
<td>

```go
import (
	"github.com/sanity-io/litter"
	"flag"
)

import (
	_ "net/http/pprof"
	"os"
	"strconv"
	"gen/mocks/github.com/go-redis/redis"
	"github.com/go-redis/redis"
	"strings"
	"github.com/NonLogicalDev/gofancyimports/internal/stdlib"
)
```

</td>
<td>

```go
import (
	"flag"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"

	"gen/mocks/github.com/go-redis/redis"
	"github.com/NonLogicalDev/gofancyimports/internal/stdlib"
	"github.com/go-redis/redis"
	"github.com/sanity-io/litter"
)
```

</td>
</tr>
</table>

<table>
<tr>
<th colspan="2"><code>gofancyimports fix --group-nodot</code></th>
</tr>
<tr>
<th><code>Before</code></th>
<th><code>After</code></th>
</tr>
<tr>
<td>

```go
import (
	"github.com/sanity-io/litter"
	"flag"
)

import (
	_ "net/http/pprof"
	"os"
	"strconv"
	"gen/mocks/github.com/go-redis/redis"
	"github.com/go-redis/redis"
	"strings"
	"github.com/NonLogicalDev/gofancyimports/internal/stdlib"
)
```

</td>
<td>

```go
import (
	"flag"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"

	"gen/mocks/github.com/go-redis/redis"

	"github.com/go-redis/redis"
	"github.com/sanity-io/litter"
	"github.com/NonLogicalDev/gofancyimports/internal/stdlib"
)
```

</td>
</tr>
</table>


<table>
<tr>
<th colspan="2"><code>gofancyimports fix --group-no-dot --group-effect --local=github.com/NonLogicalDev</code></th>
</tr>
<tr>
<th><code>Before</code></th>
<th><code>After</code></th>
</tr>
<tr>
<td>

```go
import (
	"github.com/sanity-io/litter"
	"flag"
)

import (
	_ "net/http/pprof"
	"os"
	"strconv"
	"gen/mocks/github.com/go-redis/redis"
	"github.com/go-redis/redis"
	"strings"
	"github.com/NonLogicalDev/gofancyimports/internal/stdlib"
)
```

</td>
<td>

```go
import (
	"flag"
	"os"
	"strconv"
	"strings"

	"gen/mocks/github.com/go-redis/redis"

	"github.com/go-redis/redis"
	"github.com/sanity-io/litter"

	"github.com/NonLogicalDev/gofancyimports/internal/stdlib"

	_ "net/http/pprof"
)
```

</td>
</tr>
</table>

- - - - - -

# 🎓 Extending or implementing your own import fixer

## Background:

There are plenty of tools for formatting GoLang. Most of them do a fairly good job at
tidying up imports.  However they tend to not give a lot of power for deterministically
setting the order, or suffer from issues around dealing with comments.

The world of Go Imports formatting divides into three common approaches:

1. Trust the programmer's groupings, and don't muck around with them, only sort within groups:
	* [`go imports`][1]
2. Don't trust the programmer's grouping and impose a set of opinionated restrictive rules on how imports should be grouped.
	* [`go fumpt`][2] / [`gopls`][3]
3. Give a little bit of control via CLI parameters but not export the framework to build custom formatter.
	* [`gci`][4]

If your organization or project happens to use a convention that does not fit within the
group 2, and you wish to modify an existing tool like fumpt, it ends up being rather
difficult endeavor as these tools have not been designed with simple extensiblity in mind.
This project stems from a whish to make programmaticaly defining rules for organizing
imports simple and composable.

Lucky for us Go is blessed with a very well documented parser and AST implementation,
however one of its biggest shortcomings is dealing with comments, and whitespace,
especially around imports, because beyond the bare basics it ALL all about managing
comments and whitespace.  With advent of tools like [`go analysis`][3] which are very
flexible about inspecting and modifying code, a compatibel tool for programmatically
working with imports groupings is sorely needed to provide a simple way of implementing an
organization wide import style. 


## Solution `gofancyimports`

A tool that exposes import groups as a concept to the rule writer, and allow them to
reorder, merge and split them deterministically.

The biggest selling point of this library is that you don't have to become an AST expert
to write an import transform using this library, everything is handled sensibly and
automatically, you just provide a function that takes existing import gropuings (nicely
abstracted) and transform it into a list of groupings you desire. All comments will be
hoisted and adjusted for you.

This framework takes away the difficulty from dealing with floating comments, and
whitespace between import spec groupings, by exposing import declarations and groupings as
simple structs that you can freely modify.  All of the complexity of rebuilding the
imports to your spec represented by those structs is taken care of.

This framework can understand import clauses like this (See `Fig 1`). For example: all comments in the
below figure are structurally parsed and when transformed are properly positioned, no
matter how you reorder the import groups, all complexity of recomputing offsets is
completely abstracted away.

<table>
<tr>
<th>Fig 1</th>
</tr>
<tr>
<td>


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

</td>
</tr>
</table>

This package mainly exposes the following high level interface (See `Fig 2`). The
implementation of a custom import fixer boils down to implementation of a simple function
that transforms one list of `[]ImportDelaration` that was parsed from file to another list
of `[]ImportDelaration`.

You can reorder add or remove entries from those `ImportDeclarations`. 
No comments will be lost and all comments will be magically and appropriately placed.

<table>
<tr>
<th>Fig 2</th>
</tr>
<tr>
<td>

```go
// imports.go
// ----------------------------------------------------------------------

// WithTransform allows overriding a custom import group transform.
func WithTransform(transform types.ImportTransform) Option // ...

// RewriteImportsSource takes same arguments as `go/parser.ParseFile` with an addition of `rewriter`
// and returns original source with imports grouping modified according to the rewriter.
func RewriteImportsSource(filename string, src []byte, opts ...Option) ([]byte, error) // ...

// pkg/types/types.go
// ----------------------------------------------------------------------

// ImportTransform is a function that allows reordering merging and splitting
// existing ImportDeclaration-s obtained from source.
type ImportTransform func(decls []ImportDeclaration) []ImportDeclaration

// ImportDeclaration represents a single import block. (i.e. the contents of the `import` statement)
type ImportDeclaration struct {
	// LeadingComments comments that are floating above this declaration,
	// in the middle of import blocks.
	LeadingComments []*ast.CommentGroup

	// DetachedComments are comments that are floating inside this declaration
	// unattached to specs (typically after the last import spec in a group).
	DetachedComments []*ast.CommentGroup

	// Doc is the doc comment for this import declaration.
	Doc *ast.CommentGroup

	// ImportGroups contains the list of underlying ast.ImportSpec-s.
	ImportGroups []ImportGroup
}

// ImportGroup maps to set of consecutive import specs delimited by
// whitespace and potentially having a doc comment.
//
// This type is the powerhouse of this package, allowing easy operation
// on sets of imports, delimited by whitespace.
//
// Contained within an ImportDeclaration.
type ImportGroup struct {
	Doc   *ast.CommentGroup
	Specs []*ast.ImportSpec
}
```

</td>
</tr>
</table>

You can see the ease of use of this by having a look at:
* [gofancyimports/main.go](./cmd/gofancyimports/main.go)
  * Implementation of CLI that is using configurable autogroup transform
* [pkg/organizer/autogroup](./pkg/organizer/autogroup/organizer.go)
  * Implementation of Autogroup transform that is implemented using `gofancyimports` framework

# Appendix

## Appendix (Go Analysis Integration):

Good example of how easy using `go analysis` is:
* https://arslan.io/2020/07/07/using-go-analysis-to-fix-your-source-code/

## Appendix (AST Comments)

The difficulty of working with comments in Go AST mainly stems from the fact that
Comments are do not quite behave like other AST nodes.

Firstly they are not part of the tree unless they are referenced by another AST node as
either Doc or Line End comment.

And secondly they are very rigidly tied to the Byte offsets of corresponding files,
meaning making changes to them or AST nodes to which they are attached requires
recalculating their offset positions manually.

## Appendix (Misc)

* https://github.com/golang/tools/blob/6e9046bfcd34178dc116189817430a2ad1ee7b43/internal/imports/sortimports.go#L63

- - - - - - - - - - - - - - - - - - - - - - - - - - - - -
[1]: https://pkg.go.dev/golang.org/x/tools/cmd/goimports
[2]: https://github.com/mvdan/gofumpt
[3]: https://github.com/golang/tools/tree/master/gopls
[4]: https://github.com/daixiang0/gci
[5]: https://pkg.go.dev/golang.org/x/tools/go/analysis
