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
		if file.Name.Name == "main" {
			ast.Inspect(file, func(node ast.Node) bool {
				if node, ok := node.(*ast.CallExpr); ok {
					if fun, ok := node.Fun.(*ast.SelectorExpr); ok {
						if expr, ok := fun.X.(*ast.Ident); ok {
							if expr.Name == "os" && fun.Sel.Name == "Exit" {
								pass.Reportf(fun.Pos(), "direct os.Exit() call in main()")
							}
						}
					}
				}

				return true
			})
		}
	}

	return nil, nil
}
