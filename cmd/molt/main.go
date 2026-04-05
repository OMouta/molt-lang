package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"molt/internal/cli/exitcode"
	"molt/internal/diagnostic"
	"molt/internal/evaluator"
	"molt/internal/parser"
	"molt/internal/runtime"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "usage: molt <file>")
		return exitcode.Usage
	}

	path := args[0]
	text, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "failed to read source file %q: %v\n", path, err)
		return exitcode.SourceIO
	}

	program, err := parser.Parse(path, string(text))
	if err != nil {
		return reportError(err, stderr)
	}

	_, err = evaluator.New(stdout).EvalProgram(program, runtime.NewEnvironment(nil))
	if err != nil {
		return reportError(err, stderr)
	}

	return exitcode.Success
}

func reportError(err error, stderr io.Writer) int {
	var parseErr diagnostic.ParseError
	if errors.As(err, &parseErr) {
		fmt.Fprintln(stderr, diagnostic.Render(parseErr))
		return exitcode.Diagnostics
	}

	var runtimeErr diagnostic.RuntimeError
	if errors.As(err, &runtimeErr) {
		fmt.Fprintln(stderr, diagnostic.Render(runtimeErr))
		return exitcode.Runtime
	}

	fmt.Fprintf(stderr, "internal error: %v\n", err)
	return exitcode.Internal
}
