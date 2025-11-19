package main

import (
	"fmt"

	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/NonLogicalDev/gofancyimports/pkg/analyzer/autogroupimports"
)

func main() {
	fmt.Println("Running testanalyser")
	singlechecker.Main(autogroupimports.Analyzer)
}
