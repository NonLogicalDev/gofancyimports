package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/NonLogicalDev/gofancyimports/pkg/analyzer/autogroupimports"
)

func main() {
	singlechecker.Main(autogroupimports.Analyzer)
}
