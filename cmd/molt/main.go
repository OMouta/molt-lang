package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"molt/internal/cli/exitcode"
	"molt/internal/diagnostic"
	"molt/internal/evaluator"
	"molt/internal/parser"
	"molt/internal/runtime"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		return runREPL(stdin, stdout, stderr, nil)
	}

	if args[0] == "fmt" {
		return runFmt(args[1:], stdin, stdout, stderr)
	}

	if isUnsupportedOption(args[0]) {
		printMainUsage(stderr)
		return exitcode.Usage
	}

	path := args[0]
	programArgs := args[1:]
	text, err := readProgramSource(path, stdin)
	if err != nil {
		if path == "-" {
			fmt.Fprintf(stderr, "failed to read source from stdin: %v\n", err)
			return exitcode.SourceIO
		}

		fmt.Fprintf(stderr, "failed to read source file %q: %v\n", path, err)
		return exitcode.SourceIO
	}

	program, err := parser.Parse(path, text)
	if err != nil {
		return reportError(err, stderr)
	}

	_, err = evaluator.NewWithContext(stdin, stdout, programArgs).EvalProgram(program, runtime.NewEnvironment(nil))
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

func isUnsupportedOption(arg string) bool {
	return strings.HasPrefix(arg, "-") && arg != "-"
}

func printMainUsage(stderr io.Writer) {
	fmt.Fprintln(stderr, "usage: molt [file|-] [args...]")
	fmt.Fprintln(stderr, "       molt fmt [--check] [path ...]")
}

func readProgramSource(path string, stdin io.Reader) (string, error) {
	if path == "-" {
		if stdin == nil {
			stdin = os.Stdin
		}

		text, err := io.ReadAll(stdin)
		if err != nil {
			return "", err
		}

		return string(text), nil
	}

	text, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(text), nil
}
