package gofancyimports_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NonLogicalDev/gofancyimports"
	"github.com/NonLogicalDev/gofancyimports/pkg/organizer/autogroup"
	"github.com/NonLogicalDev/gofancyimports/pkg/types"
)

var _defaultTransform = autogroup.New()

func ExampleWithTransform() {
	transform := func(decls []types.ImportDeclaration) []types.ImportDeclaration {
		rootDecl := types.ImportDeclaration{}
		importGroupSTD := &types.ImportGroup{
			Doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "// stdlib"}}},
		}
		importRest := &types.ImportGroup{
			Doc: &ast.CommentGroup{List: []*ast.Comment{{Text: "// thirdparty"}}},
		}

		// iterate over import blocks
		for _, decl := range decls {
			rootDecl.DetachedComments = append(rootDecl.DetachedComments, decl.DetachedComments...)
			rootDecl.LeadingComments = append(rootDecl.DetachedComments, decl.LeadingComments...)

			// iterate over import groups (consecutive blocks of )
			for _, group := range decl.ImportGroups {

				// iterate over import specs
				for _, spec := range group.Specs {
					// check if
					if !strings.Contains(spec.Path.Value, ".") {
						importGroupSTD.Specs = append(importGroupSTD.Specs, spec)
					} else {
						importRest.Specs = append(importRest.Specs, spec)
					}
				}
			}
		}

		rootDecl.ImportGroups = append(rootDecl.ImportGroups, *importGroupSTD, *importRest)
		return []types.ImportDeclaration{rootDecl}
	}

	inputSRC := `
package main_test

import (
	"fmt"
	"sync"
	"github.com/stretchr/testify/assert"
)

import (
	"net/http"
	"github.com/stretchr/testify/require"
)

func TestSuit(t *testing.T) {}
`

	expectedSRC := `
package main_test

import (
	// stdlib
	"fmt"
	"sync"
	"net/http"

	// thirdparty
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuit(t *testing.T) {}
`

	outputSRC, _ := gofancyimports.RewriteImportsSource(
		"main_test.go", []byte(inputSRC),
		gofancyimports.WithTransform(transform),
	)

	if string(expectedSRC) == string(outputSRC) {
		fmt.Println("MATCHING")
	}

	// Output:
	// MATCHING
}

func TestASTRewrite(t *testing.T) {
	table := []struct {
		name        string
		inputSrc    string
		expectedSrc string
		expectedErr string
		transform   types.ImportTransform
	}{
		{
			name: "noimports_hard",
			inputSrc: `
package test;func main() {}
`,
			expectedSrc: `
package test;

// extra
import (
	// first import
	"fmt"
	"sync"

	// second import
	"net/http"
)

func main() {}
`,
			transform: func(decls []types.ImportDeclaration) []types.ImportDeclaration {
				return []types.ImportDeclaration{
					{
						Doc: &ast.CommentGroup{
							List: []*ast.Comment{
								{Text: "// extra"},
							},
						},
						ImportGroups: []types.ImportGroup{
							{
								Doc: &ast.CommentGroup{
									List: []*ast.Comment{
										{Text: "// first import"},
									},
								},
								Specs: []*ast.ImportSpec{
									{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"fmt"`,
										},
									},
									{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"sync"`,
										},
									},
								},
							},

							{
								Doc: &ast.CommentGroup{
									List: []*ast.Comment{
										{Text: "// second import"},
									},
								},
								Specs: []*ast.ImportSpec{
									{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"net/http"`,
										},
									},
								},
							},
						},
					},
				}
			},
		},
		{
			name: "basic",
			inputSrc: `
package test

import (
	"fmt"
	"sync"

	"net/http"
)

func main() {}
`,
			expectedSrc: `
package test

import (
	"fmt"
	"net/http"
	"sync"
)

func main() {}
`,
			transform: _defaultTransform,
		},
		{
			name: "basic_named_group",
			inputSrc: `
package test

import (
	"fmt"
	"sync"

	// http group
	"net/http"
)

func main() {}
`,
			expectedSrc: `
package test

import (
	"fmt"
	"sync"

	// http group
	"net/http"
)

func main() {}
`,
			transform: _defaultTransform,
		},
		{
			name: "basic_named_group_firstline",
			inputSrc: `
package test

import (
	// stdlib
	"fmt"
	"sync"

	// http group
	"net/http"
)

func main() {}
`,
			expectedSrc: `
package test

import (
	// stdlib
	"fmt"
	"sync"

	// http group
	"net/http"
)

func main() {}
`,
			transform: _defaultTransform,
		},
		{
			name: "comments_basics",
			inputSrc: `
package test

// before comment

import (
	"fmt"
	"sync"

	"net/http"
)

// after comment

func main() {}
`,
			expectedSrc: `
package test

// before comment

import (
	"fmt"
	"net/http"
	"sync"
)

// after comment

func main() {}
`,
			transform: _defaultTransform,
		},
		{
			name: "comments_doc",
			inputSrc: `
package test

// before comment

// doc comment
import (
	"fmt"
	"sync"

	"net/http"
)

// after comment

func main() {}
`,
			expectedSrc: `
package test

// before comment

// doc comment
import (
	"fmt"
	"net/http"
	"sync"
)

// after comment

func main() {}
`,
			transform: _defaultTransform,
		},
		{
			name: "comments_detached",
			inputSrc: `
package test

// before comment

// doc comment
import (
	"fmt"
	"sync"

	// detached comment 1

	"net/http"

	// detached comment 2
)

// after comment

func main() {}
`,
			expectedSrc: `
package test

// before comment

// doc comment
// detached comment 1
// detached comment 2
import (
	"fmt"
	"net/http"
	"sync"
)

// after comment

func main() {}
`,
			transform: _defaultTransform,
		},
		{
			name: "multiple_decls",
			inputSrc: `
package test

import (
	"fmt"
	"sync"

	"net/http"
)

import tplText "text/template"

import tplHtml "html/template"

func main() {}
`,
			expectedSrc: `
package test

import (
	"fmt"
	tplHtml "html/template"
	"net/http"
	"sync"
	tplText "text/template"
)

func main() {}
`,
			transform: _defaultTransform,
		},
		{
			name: "multiple_decls_pinned",
			inputSrc: `
package test

import (
	"fmt"
	"sync"

	"net/http"
)

import tplText "text/template"

// pinned declaration
import tplHtml "html/template"

func main() {}
`,
			expectedSrc: `
package test

import (
	"fmt"
	"net/http"
	"sync"
	tplText "text/template"
)

// pinned declaration
import tplHtml "html/template"

func main() {}
`,
			transform: _defaultTransform,
		},
		{
			name: "cgo_import",
			inputSrc: `
package test

/*
#cgo CFLAGS: -I${SRCDIR}/ctestlib
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/ctestlib
#cgo LDFLAGS: -L${SRCDIR}/ctestlib
#cgo LDFLAGS: -ltest

#include <test.h>
*/
import "C"

import (
	"fmt"
	"sync"
	"github.com/stretchr/testify/assert"

	"net/http"
	"github.com/stretchr/testify/require"
)

import tplText "text/template"

func main() {}
`,
			expectedSrc: `
package test

/*
#cgo CFLAGS: -I${SRCDIR}/ctestlib
#cgo LDFLAGS: -Wl,-rpath,${SRCDIR}/ctestlib
#cgo LDFLAGS: -L${SRCDIR}/ctestlib
#cgo LDFLAGS: -ltest

#include <test.h>
*/
import "C"

import (
	"fmt"
	"net/http"
	"sync"
	tplText "text/template"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func main() {}
`,
			transform: _defaultTransform,
		},
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			inputSrc := tt.inputSrc[1 : len(tt.inputSrc)-1]
			expectedSrc := tt.expectedSrc[1 : len(tt.expectedSrc)-1]

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "input.go", inputSrc, parser.ParseComments)
			if err != nil {
				require.NoError(t, err)
			}

			edits, err := gofancyimports.RewriteImportsAST(fset, file, []byte(inputSrc), gofancyimports.WithTransform(tt.transform))
			if tt.expectedErr != "" {
				assert.ErrorContains(t, err, tt.expectedErr)
				return
			}

			if err != nil {
				require.NoError(t, err)
			}
			actualSrc := inputSrc
			if len(edits) > 0 {
				actualSrc = string(gofancyimports.ApplyTextEdit(fset, file, []byte(inputSrc), edits[0]))
			}
			if !assert.Equal(t, expectedSrc, actualSrc) {
				t.Logf("actual src:\n%v", actualSrc)
			}
		})
	}
}
