package main

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/NonLogicalDev/gofancyimports"
	"github.com/NonLogicalDev/gofancyimports/cmd/reorganizer"
	"github.com/NonLogicalDev/gofancyimports/pkg/stdlib"
	"github.com/spf13/cobra"
)

/*
--------------------------------------------------------------------------------
Transforms:
--------------------------------------------------------------------------------

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

--------------------------------------------------------------------------------
Into:
--------------------------------------------------------------------------------

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

*/

var LocalPrefixes []string

func main() {
	_ = reorganizer.RunCmd(
		context.Background(),
		OrganizeImports,
		reorganizer.WithCommandName("gofancyimports"),
		reorganizer.WithArgHook(func(cmd *cobra.Command) func(cmd *cobra.Command, args []string) {
			localPrefixesFlag := cmd.PersistentFlags().
				StringArrayP("local", "l", nil, "local imports prefixes to put in a separate group")

			return func(cmd *cobra.Command, args []string) {
				LocalPrefixes = *localPrefixesFlag
			}
		},
	))
}

func OrganizeImports(decls []gofancyimports.ImportDecl) []gofancyimports.ImportDecl {
	var (
		defaultGroups []gofancyimports.ImportDecl
		stickyGroups  []gofancyimports.ImportDecl
	)

	for _, d := range decls {
		if len(d.Groups) == 0 {
			continue
		}

		if d.Doc == nil {
			defaultGroups = append(defaultGroups, d)
		} else {
			stickyGroups = append(stickyGroups, d)
		}
	}

	var resultGroups []gofancyimports.ImportDecl
	if len(defaultGroups) > 0 {
		mergedDefaultGroup := gofancyimports.MergeImportDecls(defaultGroups)
		mergedDefaultGroup.Groups = OgranizeImportGroups(mergedDefaultGroup.Groups)
		resultGroups = append(resultGroups, mergedDefaultGroup)
	}
	for _, group := range stickyGroups {
		group.Groups = OgranizeImportGroups(group.Groups)
		resultGroups = append(resultGroups, group)
	}

	return resultGroups
}

func OgranizeImportGroups(groups []gofancyimports.ImportSpecGroup) []gofancyimports.ImportSpecGroup {
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
			sPath, _ := strconv.Unquote(s.Path.Value)
			sPathParts := strings.Split(sPath, "/")
			if s.Name != nil && s.Name.Name == "_" {
				defaultEffectGropup.Specs = append(defaultEffectGropup.Specs, s)
			} else if stdlib.IsStdlib(sPath) {
				defaultStdGroup.Specs = append(defaultStdGroup.Specs, s)
			} else if strings.Index(sPathParts[0], ".") == -1 {
				defaultNoDotGroup.Specs = append(defaultNoDotGroup.Specs, s)
			} else if hasLocalPrefix(sPath) {
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
	}

	result = append(result, stickyGroups...)

	if len(defaultEffectGropup.Specs) > 0 {
		result = append(result, defaultEffectGropup)
	}

	for _, r := range result {
		sort.SliceStable(r.Specs, func(i, j int) bool {
			iPath := r.Specs[i].Path.Value
			jPath := r.Specs[j].Path.Value
			return iPath < jPath
		})
	}

	return result
}

func hasLocalPrefix(path string) bool {
	for _, pref := range LocalPrefixes {
		if strings.HasPrefix(path, pref) {
			return true
		}
	}
	return false
}
