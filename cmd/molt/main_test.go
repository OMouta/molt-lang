package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func stdImports(paths ...string) string {
	var builder strings.Builder
	for _, path := range paths {
		name := path[strings.LastIndex(path, ":")+1:]
		builder.WriteString("import ")
		builder.WriteString(name)
		builder.WriteString(` from "`)
		builder.WriteString(path)
		builder.WriteString("\"\n")
	}

	return builder.String()
}

func TestRunExecutesSourceFileSuccessfully(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ok.molt")
	writeTestFile(t, path, stdImports("std:io")+"io.print(1 + 2)")

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

func TestRunReportsThrownErrorDiagnosticsAtThrowSite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "throw_error.molt")
	writeTestFile(t, path, stdImports("std:errors")+""+
		"fn fail() = {\n"+
		"  errors.throw(errors.error(\"boom\", record { code: 7 }))\n"+
		"}\n"+
		"fail()\n",
	)

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
	if !strings.Contains(output, path+":3:3: runtime error: boom") {
		t.Fatalf("stderr = %q, want thrown diagnostic header", output)
	}

	if !strings.Contains(output, "3 |   errors.throw(errors.error(\"boom\", record { code: 7 }))") {
		t.Fatalf("stderr = %q, want throw-site snippet", output)
	}

	if !strings.Contains(output, `note: error data: record { code: 7 }`) {
		t.Fatalf("stderr = %q, want thrown data note", output)
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

	exit := run([]string{"-", "hello", "world"}, strings.NewReader(stdImports("std:io", "std:cli")+"io.print([io.stdin(), cli.args()])"), &stdout, &stderr)

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
	writeTestFile(t, path, stdImports("std:io", "std:cli")+"io.print(cli.args())")

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
	writeTestFile(t, programPath, stdImports("std:io")+"io.print(io.read_file(\""+filepath.ToSlash(dataPath)+"\"))")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{programPath}, strings.NewReader(""), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "hello from disk\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "hello from disk\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunExecutesWriteFileBuiltin(t *testing.T) {
	dir := t.TempDir()
	dataPath := filepath.Join(dir, "written.txt")
	programPath := filepath.Join(dir, "write_file.molt")
	writeTestFile(t, programPath, stdImports("std:io")+"io.write_file(\""+filepath.ToSlash(dataPath)+"\", \"hello from write_file\")")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{programPath}, strings.NewReader(""), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	data, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(data) != "hello from write_file" {
		t.Fatalf("written data = %q, want %q", string(data), "hello from write_file")
	}
}

func TestRunExecutesInputBuiltin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input.molt")
	writeTestFile(t, path, stdImports("std:io")+"io.print([io.input(), io.input(), io.stdin()])")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{path}, strings.NewReader("first\nsecond\nrest"), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "[\"first\", \"second\", \"rest\"]\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "[\"first\", \"second\", \"rest\"]\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunExecutesRelativeImports(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "lib.molt")
	mainPath := filepath.Join(dir, "main.molt")
	writeTestFile(t, libPath, "value = 41\nfn bump(x) = x + 1\nexport value\nexport bump")
	writeTestFile(t, mainPath, stdImports("std:io")+"import lib from \"./lib.molt\"\nio.print(lib.bump(lib.value))")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{mainPath}, strings.NewReader(""), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "42\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "42\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunKeepsPrivateImportedBindingsHidden(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "lib.molt")
	mainPath := filepath.Join(dir, "main.molt")
	writeTestFile(t, libPath, "hidden = 41\nfn value() = hidden\nexport value")
	writeTestFile(t, mainPath, stdImports("std:io")+"import lib from \"./lib.molt\"\nio.print(lib.hidden)")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{mainPath}, strings.NewReader(""), &stdout, &stderr)

	if exit != 4 {
		t.Fatalf("exit code = %d, want 4", exit)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	if !strings.Contains(stderr.String(), `record has no field "hidden"`) {
		t.Fatalf("stderr = %q, want hidden binding failure", stderr.String())
	}
}

func TestRunReportsImportCycleDiagnostics(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.molt")
	bPath := filepath.Join(dir, "b.molt")
	mainPath := filepath.Join(dir, "main.molt")
	writeTestFile(t, aPath, `import b from "./b.molt"`)
	writeTestFile(t, bPath, `import a from "./a.molt"`)
	writeTestFile(t, mainPath, `import a from "./a.molt"`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	exit := run([]string{mainPath}, strings.NewReader(""), &stdout, &stderr)

	if exit != 4 {
		t.Fatalf("exit code = %d, want 4", exit)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	output := stderr.String()
	want := "import cycle detected: " + filepath.ToSlash(aPath) + " -> " + filepath.ToSlash(bPath) + " -> " + filepath.ToSlash(aPath)
	if !strings.Contains(output, want) {
		t.Fatalf("stderr = %q, want cycle diagnostic", output)
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
		"[][0]\n" +
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
	if !strings.Contains(errOut, `<repl>:1:1: runtime error: list index 0 out of bounds`) {
		t.Fatalf("stderr = %q, want runtime diagnostic", errOut)
	}

	if !strings.Contains(errOut, `<repl>:1:7: parse error: chained relational operators are not allowed`) {
		t.Fatalf("stderr = %q, want parse diagnostic", errOut)
	}
}

func TestRunREPLSupportsInputBuiltin(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	input := "" +
		"{\n" +
		"import io from \"std:io\"\n" +
		"io.print([io.input(), io.input(), io.stdin()])\n" +
		"}\n" +
		"first\n" +
		"second\n" +
		"tail"

	exit := run(nil, strings.NewReader(input), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "[\"first\", \"second\", \"tail\"]\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "[\"first\", \"second\", \"tail\"]\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunREPLSupportsHelpAndHistoryCommands(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	input := "" +
		":help\n" +
		"x = 1\n" +
		"x + 2\n" +
		":history\n" +
		":quit\n"

	exit := run(nil, strings.NewReader(input), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	output := stdout.String()
	if !strings.Contains(output, "REPL commands:\n") {
		t.Fatalf("stdout = %q, want help output", output)
	}

	if !strings.Contains(output, ":load <path>  load and run a Molt file in this session\n") {
		t.Fatalf("stdout = %q, want :load help", output)
	}

	if !strings.Contains(output, "1\n3\n") {
		t.Fatalf("stdout = %q, want evaluated results", output)
	}

	if !strings.Contains(output, "1 | x = 1\n2 | x + 2\n") {
		t.Fatalf("stdout = %q, want history entries", output)
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunREPLSupportsLoadCommandAndSharedState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "seed.molt")
	writeTestFile(t, path, "x = 4")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	input := ":load " + filepath.ToSlash(path) + "\n" +
		"x + 2\n"

	exit := run(nil, strings.NewReader(input), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "4\n6\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "4\n6\n")
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunREPLLoadAndUnknownCommandFailuresDoNotKillSession(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	input := "" +
		":wat\n" +
		":load missing-file.molt\n" +
		"1 + 2\n"

	exit := run(nil, strings.NewReader(input), &stdout, &stderr)

	if exit != 0 {
		t.Fatalf("exit code = %d, want 0", exit)
	}

	if stdout.String() != "3\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "3\n")
	}

	errOut := stderr.String()
	if !strings.Contains(errOut, `repl command error: unknown command ":wat" (try :help)`) {
		t.Fatalf("stderr = %q, want unknown command error", errOut)
	}

	if !strings.Contains(errOut, `repl command error: failed to read "missing-file.molt"`) {
		t.Fatalf("stderr = %q, want load failure", errOut)
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
