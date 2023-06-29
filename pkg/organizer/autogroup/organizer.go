package autogroup

import (
	"fmt"
	"go/ast"
	astTypes "go/types"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/NonLogicalDev/gofancyimports/internal/stdlib"
	"github.com/NonLogicalDev/gofancyimports/pkg/types"
)

type (
	organizer struct {
		config config
	}
	config struct {
		groupSideEffects  bool
		groupNoDotImports bool

		isLocalGroup  GroupMatcher
		isStdlibGroup GroupMatcher

		specFixups []SpecFixup
	}

	// Option represents configurable option for autogroup transform.
	Option func(conf *config)

	// SpecFixup is a function that modifies an ImportSpec in place.
	// Allows adding aliases and comments.
	SpecFixup func(s *ast.ImportSpec)

	// GroupMatcher is a function that determines group membership.
	GroupMatcher func(spec *ast.ImportSpec, path string) bool
)

// WithSpecFixups configures rules for adjusting import specs.
//
// Examples:
//   - FixupDefaultImportAlias - ensure import alias is added if last component of path does not match package name.
//   - FixupEmbedPackage - ensure a comment is added to side effect import of embed package to appease linters.
func WithSpecFixups(fixups ...SpecFixup) Option {
	return func(conf *config) {
		conf.specFixups = append(conf.specFixups, fixups...)
	}
}

// WithLocalPrefixGroup enables an extra local group of imports to differentiate between
// third party and project import based on import path prefixes.
func WithLocalPrefixGroup(prefixes []string) Option {
	return func(conf *config) {
		conf.isLocalGroup = func(spec *ast.ImportSpec, path string) bool {
			return hasAnyPrefix(path, prefixes)
		}
	}
}

// WithSideEffectGroupEnabled enables an extra side effect group for imports that are imported
// purely for side effects.
func WithSideEffectGroupEnabled(enable bool) Option {
	return func(conf *config) {
		conf.groupSideEffects = enable
	}
}

// WithNoDotGroupEnabled enables an extra group for imports that are not StdLib and do not have dots in first path component.
// Most typically this is useful for differentiating auto-generated imports.
func WithNoDotGroupEnabled(enable bool) Option {
	return func(conf *config) {
		conf.groupNoDotImports = enable
	}
}

// WithCustomStdlibMatcher allows overriding stdlib matcher.
//
// Stdlib lookup is messy and is a moving target with new Go releases. This is your escape
// hatch if built in lookup is not working for you to make sure that this library is still
// usable in the future.
func WithCustomStdlibMatcher(lookup GroupMatcher) Option {
	return func(conf *config) {
		conf.isStdlibGroup = lookup
	}
}

// WithCustomLocalGroupMatcher allows overriding local group matcher if prefix based lookup is not flexible enough.
func WithCustomLocalGroupMatcher(lookup GroupMatcher) Option {
	return func(conf *config) {
		conf.isLocalGroup = lookup
	}
}

func DefaultSpecFixups(pkgTypeInfo map[string]*astTypes.Package) []SpecFixup {
	return []SpecFixup{
		FixupEmbedPackage,
		FixupDefaultImportAlias(pkgTypeInfo),
	}
}

// FixupEmbedPackage ensures side effect only embed package has a comment to avoid getting flagged by linters.
func FixupEmbedPackage(s *ast.ImportSpec) {
	specPath, _ := strconv.Unquote(s.Path.Value)

	// Special treatment for well known packages.
	if specPath == "embed" && s.Name != nil && s.Name.Name == "_" && s.Comment == nil {
		s.Comment = makeLineComment("enable embedding")
	}
}

// FixupDefaultImportAlias ensures import alias is added to imports if last component of
// path does not match package name.
func FixupDefaultImportAlias(pkgTypeInfo map[string]*astTypes.Package) func(s *ast.ImportSpec) {
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

func FixupNoOp(_ *ast.ImportSpec) {}

func New(opts ...Option) types.ImportTransform {
	org := organizer{
		config: config{
			isStdlibGroup: func(_ *ast.ImportSpec, path string) bool {
				return stdlib.IsStdlib(path)
			},
			isLocalGroup: func(_ *ast.ImportSpec, path string) bool {
				return false
			},
		},
	}
	for _, apply := range opts {
		apply(&org.config)
	}
	return org.organiseImports
}

func (org *organizer) organiseImports(decls []types.ImportDeclaration) []types.ImportDeclaration {
	var (
		cGroup        *types.ImportDeclaration
		defaultGroups []types.ImportDeclaration
		stickyGroups  []types.ImportDeclaration
	)

	var floatingComments []*ast.CommentGroup

	for _, d := range decls {
		d := d
		if len(d.ImportGroups) == 0 {
			continue
		}

		// Gather Floating comments in one group.
		floatingComments = append(floatingComments, d.LeadingComments...)
		d.LeadingComments = nil

		if len(d.ImportGroups[0].Specs) != 0 && d.ImportGroups[0].Specs[0].Path.Value == `"C"` {
			cGroup = &d
			continue
		}

		if d.Doc == nil {
			defaultGroups = append(defaultGroups, d)
		} else {
			stickyGroups = append(stickyGroups, d)
		}
	}

	var resultGroups []types.ImportDeclaration
	if cGroup != nil {
		resultGroups = append(resultGroups, *cGroup)
	}
	if len(defaultGroups) > 0 {
		mergedDefaultGroup := types.MergeDeclarations(defaultGroups)
		mergedDefaultGroup.ImportGroups = org.organizeImportGroups(mergedDefaultGroup.ImportGroups)

		resultGroups = append(resultGroups, mergedDefaultGroup)
	}
	for _, group := range stickyGroups {
		group.ImportGroups = org.organizeImportGroups(group.ImportGroups)
		resultGroups = append(resultGroups, group)
	}

	// Add all floating comments to the first available group.
	if len(resultGroups) > 0 {
		resultGroups[0].LeadingComments = floatingComments
	}

	return resultGroups
}

func (org *organizer) organizeImportGroups(groups []types.ImportGroup) []types.ImportGroup {
	var (
		defaultGroups []types.ImportGroup

		defaultStdGroup        types.ImportGroup
		defaultNoDotGroup      types.ImportGroup
		defaultLocalGroup      types.ImportGroup
		defaultThridPartyGroup types.ImportGroup
		defaultEffectGropup    types.ImportGroup

		stickyGroups []types.ImportGroup
	)

	for _, g := range groups {
		if len(g.Specs) == 0 {
			continue
		}

		// Apply fixups.
		for _, s := range g.Specs {
			for _, fixup := range org.config.specFixups {
				fixup(s)
			}
		}

		// Split based on Import section doc comment.
		if g.Doc == nil {
			defaultGroups = append(defaultGroups, g)
		} else {
			stickyGroups = append(stickyGroups, g)
		}
	}

	var result []types.ImportGroup
	if len(defaultGroups) > 0 {
		defaultGroup := types.MergeGroups(defaultGroups)
		for _, s := range defaultGroup.Specs {
			specPath, _ := strconv.Unquote(s.Path.Value)
			specPathParts := strings.Split(specPath, "/")

			if org.config.groupSideEffects && s.Name != nil && s.Name.Name == "_" {
				defaultEffectGropup.Specs = append(defaultEffectGropup.Specs, s)
			} else if org.config.isStdlibGroup(s, specPath) {
				defaultStdGroup.Specs = append(defaultStdGroup.Specs, s)
			} else if org.config.groupNoDotImports && !strings.Contains(specPathParts[0], ".") {
				defaultNoDotGroup.Specs = append(defaultNoDotGroup.Specs, s)
			} else if org.config.isLocalGroup(s, specPath) {
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

func makeLineComment(text string) *ast.CommentGroup {
	return &ast.CommentGroup{List: []*ast.Comment{{
		Text: fmt.Sprintf("// %s", text),
	}}}
}
