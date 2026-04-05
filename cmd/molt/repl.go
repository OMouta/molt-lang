package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
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

const replHelpText = "" +
	"REPL commands:\n" +
	":help         show this help\n" +
	":history      show submitted entries\n" +
	":load <path>  load and run a Molt file in this session\n" +
	":quit         exit the REPL\n" +
	":exit         exit the REPL\n"

func runREPL(stdin io.Reader, stdout, stderr io.Writer, args []string) int {
	reader := bufio.NewReader(inputOrStdin(stdin))
	eval := evaluator.NewWithContext(reader, stdout, args)
	env := runtime.NewEnvironment(nil)
	history := make([]string, 0, 16)
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
			if strings.HasPrefix(trimmed, ":") {
				quit := handleREPLCommand(trimmed, &history, eval, env, stdout, stderr)
				if quit {
					return 0
				}

				if errors.Is(err, io.EOF) {
					return 0
				}

				continue
			}
		}

		buffer.WriteString(line)

		consumed, done := processREPLBuffer(eval, env, buffer.String(), stdout, stderr, &history)
		if consumed {
			buffer.Reset()
		}

		if errors.Is(err, io.EOF) {
			if !done {
				processREPLBuffer(eval, env, buffer.String(), stdout, stderr, &history)
			}

			return 0
		}
	}
}

func processREPLBuffer(eval *evaluator.Evaluator, env *runtime.Environment, text string, stdout, stderr io.Writer, history *[]string) (consumed bool, done bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true, true
	}

	program, err := parser.Parse(replPath, text)
	if err != nil {
		if replNeedsMoreInput(text, err) {
			return false, false
		}

		appendREPLHistory(history, text)
		fmt.Fprintln(stderr, diagnostic.Render(err.(diagnostic.DetailedError)))
		return true, true
	}

	appendREPLHistory(history, text)
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

func handleREPLCommand(commandLine string, history *[]string, eval *evaluator.Evaluator, env *runtime.Environment, stdout, stderr io.Writer) bool {
	name, arg := splitREPLCommand(commandLine)

	switch name {
	case ":quit", ":exit":
		return true
	case ":help":
		fmt.Fprint(stdout, replHelpText)
		return false
	case ":history":
		printREPLHistory(history, stdout)
		return false
	case ":load":
		path := arg
		if path == "" {
			fmt.Fprintln(stderr, "repl command error: :load expects a path")
			return false
		}

		if unquoted, err := strconv.Unquote(path); err == nil {
			path = unquoted
		}

		appendREPLHistory(history, commandLine)
		loadREPLFile(path, eval, env, stdout, stderr)
		return false
	default:
		fmt.Fprintf(stderr, "repl command error: unknown command %q (try :help)\n", commandLine)
		return false
	}
}

func splitREPLCommand(commandLine string) (name, arg string) {
	commandLine = strings.TrimSpace(commandLine)
	if commandLine == "" {
		return "", ""
	}

	index := strings.IndexAny(commandLine, " \t")
	if index < 0 {
		return commandLine, ""
	}

	return commandLine[:index], strings.TrimSpace(commandLine[index+1:])
}

func loadREPLFile(path string, eval *evaluator.Evaluator, env *runtime.Environment, stdout, stderr io.Writer) {
	text, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(stderr, "repl command error: failed to read %q: %v\n", path, err)
		return
	}

	program, err := parser.Parse(path, string(text))
	if err != nil {
		fmt.Fprintln(stderr, diagnostic.Render(err.(diagnostic.DetailedError)))
		return
	}

	value, err := eval.EvalProgram(program, env)
	if err != nil {
		fmt.Fprintln(stderr, diagnostic.Render(err.(diagnostic.DetailedError)))
		return
	}

	if _, ok := value.(runtime.NilValue); !ok {
		fmt.Fprintln(stdout, runtime.ShowValue(value))
	}
}

func appendREPLHistory(history *[]string, entry string) {
	if history == nil {
		return
	}

	trimmed := strings.TrimSpace(entry)
	if trimmed == "" {
		return
	}

	*history = append(*history, trimmed)
}

func printREPLHistory(history *[]string, stdout io.Writer) {
	if history == nil || len(*history) == 0 {
		fmt.Fprintln(stdout, "history is empty")
		return
	}

	for index, entry := range *history {
		fmt.Fprintf(stdout, "%d | %s\n", index+1, entry)
	}
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
