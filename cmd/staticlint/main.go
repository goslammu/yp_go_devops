package main

import (
	"strings"

	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	atomicanalyzer "golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/pkgfact"
	printfanalyzer "golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"honnef.co/go/tools/analysis/facts/nilness"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	analyzersList := []*analysis.Analyzer{
		directExitCallAnalyzer, // Custom analyzer for searching of direct called os.Exit() functions.

		asmdecl.Analyzer, // all standart analyzers from "passes"
		assign.Analyzer,
		atomicanalyzer.Analyzer,
		atomicalign.Analyzer,
		buildssa.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		ctrlflow.Analyzer,
		bools.Analyzer,
		deepequalerrors.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		findcall.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		inspect.Analyzer,
		loopclosure.Analyzer,
		usesgenerics.Analyzer,
		unusedwrite.Analyzer,
		unusedresult.Analyzer,
		unsafeptr.Analyzer,
		unreachable.Analyzer,
		unmarshal.Analyzer,
		timeformat.Analyzer,
		tests.Analyzer,
		testinggoroutine.Analyzer,
		structtag.Analyzer,
		stringintconv.Analyzer,
		stdmethods.Analyzer,
		sortslice.Analyzer,
		sigchanyzer.Analyzer,
		shift.Analyzer,
		shadow.Analyzer,
		reflectvaluecompare.Analyzer,
		printfanalyzer.Analyzer,
		pkgfact.Analyzer,
		nilness.Analysis,
		lostcancel.Analyzer,

		errcheck.Analyzer,  // Public analyzer for searching of non-handled errors.
		bodyclose.Analyzer, // Public analyzer for searching of non-closed HTTP-bodies.
	}

	// Analyzers from staticcheck package.
	staticCheckers := map[string]bool{
		"S1002":  true, // Omit comparison with boolean constant.
		"ST1006": true, // Poorly chosen receiver name.
		"QF1006": true, // Lift "if+break" into loop condition.
	}

	// All "SA" analyzers from staticcheck package.
	for _, v := range staticcheck.Analyzers {
		if strings.Contains(v.Analyzer.Name, "SA") {
			analyzersList = append(analyzersList, v.Analyzer)
		}
	}

	for _, v := range quickfix.Analyzers {
		if staticCheckers[v.Analyzer.Name] {
			analyzersList = append(analyzersList, v.Analyzer)
		}
	}

	for _, v := range simple.Analyzers {
		if staticCheckers[v.Analyzer.Name] {
			analyzersList = append(analyzersList, v.Analyzer)
		}
	}

	for _, v := range stylecheck.Analyzers {
		if staticCheckers[v.Analyzer.Name] {
			analyzersList = append(analyzersList, v.Analyzer)
		}
	}

	multichecker.Main(
		analyzersList...,
	)
}
