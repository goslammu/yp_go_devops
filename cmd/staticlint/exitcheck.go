package main

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// DirectExitCallAnalyzer checks the main() for direct calls of os.Exit().
var directExitCallAnalyzer = &analysis.Analyzer{
	Name: "DirectExitCallAnalyzer",
	Doc:  "checks the main() for direct calls of os.Exit()",
	Run:  run,
}

// run is the main logic implementation of searching for needed cases of direct calling os.Exit.
func run(pass *analysis.Pass) (interface{}, error) {
	if strings.Contains(pass.Pkg.Path(), ".test") {
		return nil, nil
	}

	for _, file := range pass.Files {
		if file.Name.Name != "main" {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			n, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}

			f, ok := n.Fun.(*ast.SelectorExpr)
			if !ok || f.Sel.Name != "Exit" {
				return true
			}

			e, ok := f.X.(*ast.Ident)
			if !ok || e.Name != "os" {
				return true
			}

			pass.Reportf(f.Pos(), "direct os.Exit() call in main()")

			return true
		})
	}

	return nil, nil
}
