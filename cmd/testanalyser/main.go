package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/NonLogicalDev/go.fancyimports/pkg/analyzer/autogroupimports"
)

func main() {
	singlechecker.Main(autogroupimports.Analyzer)
}
