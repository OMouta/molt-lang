package main

import (
	"bufio"
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

const (
	replPath            = "<repl>"
	replPrimaryPrompt   = ">> "
	replSecondaryPrompt = ".. "
)

func runREPL(stdin io.Reader, stdout, stderr io.Writer, args []string) int {
	reader := bufio.NewReader(inputOrStdin(stdin))
	eval := evaluator.NewWithContext(reader, stdout, args)
	env := runtime.NewEnvironment(nil)
	var buffer strings.Builder
	interactive := replIsInteractive(stdin, stdout)

	for {
		if interactive {
			prompt := replPrimaryPrompt
			if strings.TrimSpace(buffer.String()) != "" {
				prompt = replSecondaryPrompt
			}

			if _, err := fmt.Fprint(stdout, prompt); err != nil {
				fmt.Fprintf(stderr, "internal error: %v\n", err)
				return exitcode.Internal
			}
		}

		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			fmt.Fprintf(stderr, "internal error: %v\n", err)
			return exitcode.Internal
		}

		if errors.Is(err, io.EOF) && line == "" {
			return 0
		}

		if buffer.Len() == 0 {
			trimmed := strings.TrimSpace(line)
			if trimmed == ":quit" || trimmed == ":exit" {
				return 0
			}
		}

		buffer.WriteString(line)

		consumed, done := processREPLBuffer(eval, env, buffer.String(), stdout, stderr)
		if consumed {
			buffer.Reset()
		}

		if errors.Is(err, io.EOF) {
			if !done {
				processREPLBuffer(eval, env, buffer.String(), stdout, stderr)
			}

			return 0
		}
	}
}

func processREPLBuffer(eval *evaluator.Evaluator, env *runtime.Environment, text string, stdout, stderr io.Writer) (consumed bool, done bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true, true
	}

	program, err := parser.Parse(replPath, text)
	if err != nil {
		if replNeedsMoreInput(text, err) {
			return false, false
		}

		fmt.Fprintln(stderr, diagnostic.Render(err.(diagnostic.DetailedError)))
		return true, true
	}

	value, err := eval.EvalProgram(program, env)
	if err != nil {
		fmt.Fprintln(stderr, diagnostic.Render(err.(diagnostic.DetailedError)))
		return true, true
	}

	if _, ok := value.(runtime.NilValue); !ok {
		fmt.Fprintln(stdout, runtime.ShowValue(value))
	}

	return true, true
}

func replNeedsMoreInput(text string, err error) bool {
	if hasUnclosedDelimiters(text) {
		return true
	}

	var parseErr diagnostic.ParseError
	if !errors.As(err, &parseErr) {
		return false
	}

	diag := parseErr.Diagnostic()
	if diag.Span.End.Offset < len(text) {
		return false
	}

	if diag.Message == "unterminated string literal" {
		return true
	}

	return strings.HasPrefix(diag.Message, "expected ")
}

func hasUnclosedDelimiters(text string) bool {
	var round, square, curly int
	inString := false
	inComment := false
	escaped := false

	for i := 0; i < len(text); i++ {
		ch := text[i]

		if inComment {
			if ch == '\n' {
				inComment = false
			}

			continue
		}

		if inString {
			if escaped {
				escaped = false
				continue
			}

			if ch == '\\' {
				escaped = true
				continue
			}

			if ch == '"' {
				inString = false
			}

			continue
		}

		switch ch {
		case '#':
			inComment = true
		case '"':
			inString = true
		case '(':
			round++
		case ')':
			if round > 0 {
				round--
			}
		case '[':
			square++
		case ']':
			if square > 0 {
				square--
			}
		case '{':
			curly++
		case '}':
			if curly > 0 {
				curly--
			}
		}
	}

	return inString || round > 0 || square > 0 || curly > 0
}

func replIsInteractive(stdin io.Reader, stdout io.Writer) bool {
	in, ok := stdin.(*os.File)
	if !ok {
		return false
	}

	out, ok := stdout.(*os.File)
	if !ok {
		return false
	}

	inInfo, err := in.Stat()
	if err != nil {
		return false
	}

	outInfo, err := out.Stat()
	if err != nil {
		return false
	}

	return inInfo.Mode()&os.ModeCharDevice != 0 && outInfo.Mode()&os.ModeCharDevice != 0
}

func inputOrStdin(stdin io.Reader) io.Reader {
	if stdin != nil {
		return stdin
	}

	return os.Stdin
}
