package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunExecutesSourceFileSuccessfully(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.molt")
	writeTestFile(t, path, "print(1 + 2)")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{path}, strings.NewReader(""), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "3\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "3\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunReportsParseDiagnostics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "parse_error.molt")
	writeTestFile(t, path, "f(1, 2")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{path}, strings.NewReader(""), &stdout, &stderr)

	if exit != 3 {
		t.Fatalf("exit code = %d, want 3", exit)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	output := stderr.String()
	if !strings.Contains(output, path+":1:7: parse error: expected ')' after list") {
		t.Fatalf("stderr = %q, want parse diagnostic header", output)
	}

	if !strings.Contains(output, "1 | f(1, 2") {
		t.Fatalf("stderr = %q, want source snippet", output)
	}
}

func TestRunReportsRuntimeDiagnostics(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "runtime_error.molt")
	writeTestFile(t, path, "xs = [1]\nxs[2]")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{path}, strings.NewReader(""), &stdout, &stderr)

	if exit != 4 {
		t.Fatalf("exit code = %d, want 4", exit)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	output := stderr.String()
	if !strings.Contains(output, path+":2:1: runtime error: list index 2 out of bounds") {
		t.Fatalf("stderr = %q, want runtime diagnostic header", output)
	}

	if !strings.Contains(output, "2 | xs[2]") {
		t.Fatalf("stderr = %q, want source snippet", output)
	}
}

func TestRunRejectsInvalidUsageAndMissingFiles(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if exit := run([]string{"--bad"}, strings.NewReader(""), &stdout, &stderr); exit != 1 {
		t.Fatalf("usage exit code = %d, want 1", exit)
	}

	if stderr.String() != "usage: molt [file|-] [args...]\n" {
		t.Fatalf("usage stderr = %q, want %q", stderr.String(), "usage: molt [file|-] [args...]\n")
	}

	stdout.Reset()
	stderr.Reset()

	if exit := run([]string{"missing-file.molt"}, strings.NewReader(""), &stdout, &stderr); exit != 2 {
		t.Fatalf("source io exit code = %d, want 2", exit)
	}

	if !strings.Contains(stderr.String(), `failed to read source file "missing-file.molt"`) {
		t.Fatalf("stderr = %q, want read failure", stderr.String())
	}
}

func TestRunExecutesProgramFromStdin(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exit := run([]string{"-", "hello", "world"}, strings.NewReader("print([stdin(), args()])"), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "[\"\", [\"hello\", \"world\"]]\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "[\"\", [\"hello\", \"world\"]]\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunReportsStdinReadFailures(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exit := run([]string{"-"}, errReader{err: errors.New("boom")}, &stdout, &stderr)

	if exit != 2 {
		t.Fatalf("exit code = %d, want 2", exit)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	if !strings.Contains(stderr.String(), "failed to read source from stdin: boom") {
		t.Fatalf("stderr = %q, want stdin read failure", stderr.String())
	}
}

func TestRunPassesCommandLineArgumentsToProgram(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "args.molt")
	writeTestFile(t, path, "print(args())")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{path, "alpha", "beta"}, strings.NewReader(""), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "[\"alpha\", \"beta\"]\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "[\"alpha\", \"beta\"]\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunExecutesReadFileBuiltin(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "data.txt")
	programPath := filepath.Join(dir, "read_file.molt")
	writeTestFile(t, dataPath, "hello from disk")
	writeTestFile(t, programPath, "print(read_file(\""+filepath.ToSlash(dataPath)+"\"))")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{programPath}, strings.NewReader(""), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "\"hello from disk\"\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "\"hello from disk\"\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunStartsREPLWhenNoFileIsProvided(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exit := run(nil, strings.NewReader("x = 1\nx + 2\n"), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "1\n3\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "1\n3\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunREPLSupportsMultilineInputAndContinuesAfterErrors(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	input := "" +
		"{\n" +
		"  1 + 2\n" +
		"}\n" +
		"len(1)\n" +
		"a < b < c\n" +
		"4 + 5\n"

	exit := run(nil, strings.NewReader(input), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "3\n9\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "3\n9\n")
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, `<repl>:1:1: runtime error: len expects list or string, got "number"`) {
		t.Fatalf("stderr = %q, want runtime diagnostic", errOut)
	}

	if !strings.Contains(errOut, `<repl>:1:7: parse error: chained relational operators are not allowed`) {
		t.Fatalf("stderr = %q, want parse diagnostic", errOut)
	}
}

func writeTestFile(t *testing.T, path, text string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

type errReader struct {
	err error
}

func (r errReader) Read(p []byte) (int, error) {
	return 0, r.err
}
