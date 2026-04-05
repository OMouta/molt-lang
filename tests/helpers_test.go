package integration_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"molt/internal/evaluator"
	"molt/internal/parser"
	"molt/internal/runtime"
)

func repoRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}

	return filepath.Dir(wd)
}

func repoPath(t *testing.T, relative string) string {
	t.Helper()
	return filepath.Join(repoRoot(t), relative)
}

func executeProgram(t *testing.T, path, input string) (runtime.Value, string, error) {
	t.Helper()

	program, err := parser.Parse(path, input)
	if err != nil {
		return nil, "", err
	}

	var output bytes.Buffer
	value, err := evaluator.New(&output).EvalProgram(program, runtime.NewEnvironment(nil))
	return value, output.String(), err
}

func mustExecuteProgram(t *testing.T, path, input string) (runtime.Value, string) {
	t.Helper()

	value, output, err := executeProgram(t, path, input)
	if err != nil {
		t.Fatalf("executeProgram failed: %v", err)
	}

	return value, output
}

func executeFile(t *testing.T, relative string) (runtime.Value, string, error) {
	t.Helper()

	path := repoPath(t, relative)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	return executeProgram(t, path, string(data))
}

func mustExecuteFile(t *testing.T, relative string) (runtime.Value, string) {
	t.Helper()

	value, output, err := executeFile(t, relative)
	if err != nil {
		t.Fatalf("executeFile failed: %v", err)
	}

	return value, output
}

func runCLIExample(t *testing.T, relative string) (string, string) {
	t.Helper()

	command := exec.Command("go", "run", "./cmd/molt", relative)
	command.Dir = repoRoot(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		t.Fatalf("go run failed: %v\nstderr:\n%s", err, stderr.String())
	}

	return stdout.String(), stderr.String()
}

func expectShownValue(t *testing.T, value runtime.Value, want string) {
	t.Helper()

	if got := runtime.ShowValue(value); got != want {
		t.Fatalf("value = %q, want %q", got, want)
	}
}

func expectStringValue(t *testing.T, value runtime.Value, want string) {
	t.Helper()

	stringValue, ok := value.(*runtime.StringValue)
	if !ok {
		t.Fatalf("value type = %T, want *runtime.StringValue", value)
	}

	if stringValue.Value != want {
		t.Fatalf("string value = %q, want %q", stringValue.Value, want)
	}
}
