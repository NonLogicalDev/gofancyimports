package autogroup

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path"
	"sort"
	"strconv"
	"strings"

	gofancyimports "github.com/NonLogicalDev/go.fancyimports"
	"github.com/NonLogicalDev/go.fancyimports/internal/stdlib"
)

func New(localPrefixes []string, specFixups ...func(s *ast.ImportSpec)) gofancyimports.ImportOrganizer {
	org := organizer{
		groupLocalPrefixes:    localPrefixes,
		applyImportSpecFixups: specFixups,
	}
	return org.organiseImports
}

func DefaultSpecFixups(pkgTypeInfo map[string]*types.Package) func(s *ast.ImportSpec) {
	fixupAlias := FixupDefaultImportAlias(pkgTypeInfo)
	return func(s *ast.ImportSpec) {
		FixupEmbedPackage(s)
		fixupAlias(s)
	}
}

func FixupNoOp(_ *ast.ImportSpec) {}

func FixupEmbedPackage(s *ast.ImportSpec) {
	specPath, _ := strconv.Unquote(s.Path.Value)

	// Special treatment for well known packages.
	if specPath == "embed" && s.Name != nil && s.Name.Name == "_" && s.Comment == nil {
		s.Comment = mkLineComment(token.NoPos, "enable resource embedding")
	}
}

func FixupDefaultImportAlias(pkgTypeInfo map[string]*types.Package) func(s *ast.ImportSpec) {
	if pkgTypeInfo == nil {
		return FixupNoOp
	}

	return func(s *ast.ImportSpec) {
		specPath, _ := strconv.Unquote(s.Path.Value)
		specPathBase := path.Base(specPath)

		// If pkgTypeInfo is provided, ensure that all imports have well-defined name.
		if len(pkgTypeInfo) > 0 {
			specTypeInfo, found := pkgTypeInfo[specPath]
			if found && specPathBase != specTypeInfo.Name() && s.Name == nil {
				s.Name = &ast.Ident{
					NamePos: s.Pos(),
					Name:    specTypeInfo.Name(),
				}
			}
		}
	}
}

type organizer struct {
	groupLocalPrefixes    []string
	groupSideEffecImports bool

	applyImportSpecFixups []func(s *ast.ImportSpec)
}

func (org *organizer) organiseImports(decls []gofancyimports.ImportDecl) []gofancyimports.ImportDecl {
	var (
		defaultGroups []gofancyimports.ImportDecl
		stickyGroups  []gofancyimports.ImportDecl
	)

	var floatingComments []*ast.CommentGroup

	for _, d := range decls {
		if len(d.Groups) == 0 {
			continue
		}

		// Gather Floating comments in one group.
		floatingComments = append(floatingComments, d.FloatingComments...)
		d.FloatingComments = nil

		if d.Doc == nil {
			defaultGroups = append(defaultGroups, d)
		} else {
			stickyGroups = append(stickyGroups, d)
		}
	}

	var resultGroups []gofancyimports.ImportDecl
	if len(defaultGroups) > 0 {
		mergedDefaultGroup := gofancyimports.MergeImportDecls(defaultGroups)
		mergedDefaultGroup.Groups = org.organizeImportGroups(mergedDefaultGroup.Groups)

		resultGroups = append(resultGroups, mergedDefaultGroup)
	}
	for _, group := range stickyGroups {
		group.Groups = org.organizeImportGroups(group.Groups)
		resultGroups = append(resultGroups, group)
	}

	// Add all floating comments to the first available group.
	if len(resultGroups) > 0 {
		resultGroups[0].FloatingComments = floatingComments
	}

	return resultGroups
}

func (org *organizer) organizeImportGroups(groups []gofancyimports.ImportSpecGroup) []gofancyimports.ImportSpecGroup {
	var (
		defaultGroups []gofancyimports.ImportSpecGroup

		defaultStdGroup        gofancyimports.ImportSpecGroup
		defaultNoDotGroup      gofancyimports.ImportSpecGroup
		defaultLocalGroup      gofancyimports.ImportSpecGroup
		defaultThridPartyGroup gofancyimports.ImportSpecGroup
		defaultEffectGropup    gofancyimports.ImportSpecGroup

		stickyGroups []gofancyimports.ImportSpecGroup
	)

	for _, g := range groups {
		if len(g.Specs) == 0 {
			continue
		}

		// Apply fixups.
		for _, s := range g.Specs {
			for _, fixupSpec := range org.applyImportSpecFixups {
				fixupSpec(s)
			}
		}

		// Split based on Import section doc comment.
		if g.Doc == nil {
			defaultGroups = append(defaultGroups, g)
		} else {
			stickyGroups = append(stickyGroups, g)
		}
	}

	var result []gofancyimports.ImportSpecGroup
	if len(defaultGroups) > 0 {
		defaultGroup := gofancyimports.MergeImportGroups(defaultGroups)
		for _, s := range defaultGroup.Specs {
			specPath, _ := strconv.Unquote(s.Path.Value)
			specPathParts := strings.Split(specPath, "/")

			if s.Name != nil && s.Name.Name == "_" && org.groupSideEffecImports {
				defaultEffectGropup.Specs = append(defaultEffectGropup.Specs, s)
			} else if stdlib.IsStdlib(specPath) {
				defaultStdGroup.Specs = append(defaultStdGroup.Specs, s)
			} else if strings.Index(specPathParts[0], ".") == -1 {
				defaultNoDotGroup.Specs = append(defaultNoDotGroup.Specs, s)
			} else if hasAnyPrefix(specPath, org.groupLocalPrefixes) {
				defaultLocalGroup.Specs = append(defaultLocalGroup.Specs, s)
			} else {
				defaultThridPartyGroup.Specs = append(defaultThridPartyGroup.Specs, s)
			}
		}

		if len(defaultStdGroup.Specs) > 0 {
			result = append(result, defaultStdGroup)
		}
		if len(defaultNoDotGroup.Specs) > 0 {
			result = append(result, defaultNoDotGroup)
		}
		if len(defaultThridPartyGroup.Specs) > 0 {
			result = append(result, defaultThridPartyGroup)
		}
		if len(defaultLocalGroup.Specs) > 0 {
			result = append(result, defaultLocalGroup)
		}
		if len(defaultEffectGropup.Specs) > 0 {
			result = append(result, defaultEffectGropup)
		}
	}

	result = append(result, stickyGroups...)

	for _, r := range result {
		sort.SliceStable(r.Specs, func(i, j int) bool {
			iPath := r.Specs[i].Path.Value
			jPath := r.Specs[j].Path.Value
			return iPath < jPath
		})
	}

	return result
}

func hasAnyPrefix(path string, prefixes []string) bool {
	for _, pref := range prefixes {
		if strings.HasPrefix(path, pref) {
			return true
		}
	}
	return false
}

func mkLineComment(pos token.Pos, text string) *ast.CommentGroup {
	return &ast.CommentGroup{List: []*ast.Comment{{
		Slash: pos,
		Text:  fmt.Sprintf("// %s", text),
	}}}
}
