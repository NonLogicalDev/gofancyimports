package autogroupimports

import (
	"strings"

	gofancyimports "github.com/NonLogicalDev/go.fancyimports"
	"golang.org/x/tools/go/analysis"

	"github.com/NonLogicalDev/go.fancyimports/pkg/organizer/autogroup"
)

const _name = "autogroupimports"
const _doc = `
some documentation string
`

var argLocalPrefix string

var Analyzer = &analysis.Analyzer{
	Name: _name,
	Doc:  _doc,
	Run:  run,

	RunDespiteErrors: true,
}

func init() {
	Analyzer.Flags.StringVar(&argLocalPrefix, "localPrefix", "", "comma separated list of local prefixes")
}

func run(pass *analysis.Pass) (interface{}, error) {
	localPrefixes := strings.Split(argLocalPrefix, ",")

	for _, file := range pass.Files {
		edit, err := gofancyimports.RewriteImportsLL(pass.Fset, file, autogroup.New(localPrefixes))
		if err != nil {
			return nil, err
		}
		if edit == nil {
			continue
		}

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

	return nil, nil
}
