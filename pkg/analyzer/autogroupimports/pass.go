package autogroupimports

import (
	"bytes"
	"go/printer"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"

	gofancyimports "github.com/NonLogicalDev/gofancyimports"
	"github.com/NonLogicalDev/gofancyimports/pkg/organizer/autogroup"
)

const _name = "autogroupimports"
const _doc = `This analysis pass detects unconventionally formatted imports and provides fixes.

The convention it uses is as follows:

  1. Default imports go first
  2. Generated imports (not dots in first path entry) go second
  3. All third party imports go third
  4. (optionally) All imports from "localImportsPrefix" go last

Additionally all un-aliased imports whose base name (last path entry) does not match the package name will be aliased with correct package name.

# Transforms:

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

# Into:

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
`

var Analyzer = &analysis.Analyzer{
	Name: _name,
	Doc:  _doc,
	Run:  run,

	RunDespiteErrors: true,
}

var (
	argLocalPrefix string

	argNoDotGroup bool

	argSideEffectGroup bool

	_defaultPrintConfig = &printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
)

func init() {
	Analyzer.Flags.StringVar(&argLocalPrefix,
		"group-local-prefixes", "",
		"comma separated list of local prefixes")
	Analyzer.Flags.BoolVar(&argNoDotGroup,
		"group-nodot", false,
		"separate no dot imports that are not stdlib")
	Analyzer.Flags.BoolVar(&argSideEffectGroup,
		"group-effect", false,
		"separate side effect imports into separate group")
}

func run(pass *analysis.Pass) (interface{}, error) {
	localPrefixes := strings.Split(argLocalPrefix, ",")

	pkgInfo := map[string]*types.Package{}
	if pass.Pkg != nil {
		pkgInfo[pass.Pkg.Path()] = pass.Pkg
		for _, importPkg := range pass.Pkg.Imports() {
			pkgInfo[importPkg.Path()] = importPkg
		}
	}

	transform := autogroup.New(
		autogroup.WithLocalPrefixGroup(localPrefixes),
		autogroup.WithSpecFixups(autogroup.DefaultSpecFixups(pkgInfo)...),
		autogroup.WithNoDotGroupEnabled(argNoDotGroup),
		autogroup.WithSideEffectGroupEnabled(argSideEffectGroup),
	)
	for _, file := range pass.Files {
		b := bytes.NewBuffer(nil)
		err := _defaultPrintConfig.Fprint(b, pass.Fset, file)
		if err != nil {
			pass.Reportf(file.Pos(), "error while loading file: %v", err)
			continue
		}

		edits, err := gofancyimports.RewriteImportsAST(pass.Fset, file, b.Bytes(),
			gofancyimports.WithTransform(transform),
			gofancyimports.WithPrinterConfig(_defaultPrintConfig),
		)
		if err != nil {
			pass.Reportf(file.Pos(), "error while parsing imports: %v", err)
			continue
		}

		for _, edit := range edits {
			edit := edit
			pass.Report(analysis.Diagnostic{
				Pos: edit.Pos,
				End: edit.End,

				Category: "imports",
				Message:  "go imports are not properly formatted",

				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message:   "format imports",
						TextEdits: []analysis.TextEdit{*edit},
					},
				},
			})
		}
	}

	return nil, nil
}
