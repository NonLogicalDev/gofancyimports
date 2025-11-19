package gofancyimports_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NonLogicalDev/gofancyimports"
	"github.com/NonLogicalDev/gofancyimports/pkg/organizer/autogroup"
	"github.com/NonLogicalDev/gofancyimports/pkg/types"
)

const _ExampleWithTransformInput = `
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

const _ExampleWithTransformOutput = `
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

	inputSRC := _ExampleWithTransformInput
	expectedSRC := _ExampleWithTransformOutput

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

func TestTestset00(t *testing.T) {
	runTestSetFromFolder(t, "testdata/testset_demo", ".in.go", ".out.go", func(testname TestName) types.ImportTransform {
		return autogroup.New()
	})
}

func TestTestset01(t *testing.T) {
	runTestSetFromFolder(t, "testdata/testset_exhaustive", ".go.in", ".go.out", func(testname TestName) types.ImportTransform {
		switch testname {
		case "noimports_hard_custom_transform":
			return func(decls []types.ImportDeclaration) []types.ImportDeclaration {
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
			}
		}

		return autogroup.New()
	})
}

type (
	TestName   = string
	TestConfig struct {
		name   string
		srcIn  string
		srcOut string
	}

	TestFileType int
)

const (
	TestFileInvalid TestFileType = iota
	TestFileInput
	TestFileOutput
)

var TestRerunCount = 3

func runTestSetFromFolder(t *testing.T, testsetpath string, suffixin string, suffixout string, transformpicker func(TestName) types.ImportTransform) {
	table := make(map[TestName]TestConfig)

	direntries, err := os.ReadDir(testsetpath)
	require.NoError(t, err)

	for _, de := range direntries {
		if de.IsDir() {
			continue
		}

		var (
			name string
			src  []byte
			typ  TestFileType

			err error
		)
		if tname := strings.TrimSuffix(de.Name(), suffixin); tname != de.Name() {
			src, err = os.ReadFile(filepath.Join(testsetpath, de.Name()))
			require.NoError(t, err)
			typ = TestFileInput
			name = tname
		}
		if tname := strings.TrimSuffix(de.Name(), suffixout); tname != de.Name() {
			src, err = os.ReadFile(filepath.Join(testsetpath, de.Name()))
			require.NoError(t, err)
			typ = TestFileOutput
			name = tname
		}
		if typ == TestFileInvalid {
			continue
		}

		v := table[name]
		v.name = name
		if typ == TestFileInput {
			v.srcIn = string(src)
		} else if typ == TestFileOutput {
			v.srcOut = string(src)
		}
		table[name] = v
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			transform := transformpicker(tt.name)

			actualSrc := tt.srcIn
			for i := 0; i < TestRerunCount; i++ {
				fset := token.NewFileSet()
				file, err := parser.ParseFile(fset, "input.go", actualSrc, parser.ParseComments)
				require.NoError(t, err)

				edits, err := gofancyimports.RewriteImportsAST(fset, file, []byte(actualSrc), gofancyimports.WithTransform(transform))
				require.NoError(t, err)

				if len(edits) > 0 {
					actualSrc = string(gofancyimports.ApplyTextEdit(fset, file, []byte(actualSrc), edits[0]))
				}
				if !assert.Equal(t, tt.srcOut, actualSrc) {
					t.Logf("actual src:\n%v", actualSrc)
					t.FailNow()
				}
			}
		})
	}
}
