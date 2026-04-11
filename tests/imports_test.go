package integration_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"molt/internal/diagnostic"
	"molt/internal/evaluator"
	"molt/internal/parser"
	"molt/internal/runtime"
)

func TestImportsExecuteRelativeModulesEndToEnd(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "math.molt")
	mainPath := filepath.Join(dir, "main.molt")

	if err := os.WriteFile(libPath, []byte("base = 40\nfn add2(x) = x + 2\nexport base\nexport add2"), 0o644); err != nil {
		t.Fatalf("WriteFile lib failed: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte("import math from \"./math.molt\"\nmath.add2(math.base)"), 0o644); err != nil {
		t.Fatalf("WriteFile main failed: %v", err)
	}

	data, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("ReadFile main failed: %v", err)
	}

	program, err := parser.Parse(mainPath, string(data))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	var output bytes.Buffer
	value, err := evaluator.New(&output).EvalProgram(program, runtime.NewEnvironment(nil))
	if err != nil {
		t.Fatalf("EvalProgram failed: %v", err)
	}

	expectShownValue(t, value, "42")
	if output.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", output.String())
	}
}

func TestImportsCacheModuleExecutionEndToEnd(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "stateful.molt")
	mainPath := filepath.Join(dir, "main.molt")

	if err := os.WriteFile(libPath, []byte(stdImports("std:collections")+
		"xs = []\n"+
		"fn tick() = {\n"+
		"  collections.push(xs, 1)\n"+
		"  collections.len(xs)\n"+
		"}\n"+
		"export tick"), 0o644); err != nil {
		t.Fatalf("WriteFile lib failed: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte(""+
		"import m1 from \"./stateful.molt\"\n"+
		"a = m1.tick()\n"+
		"import m2 from \"./stateful.molt\"\n"+
		"b = m2.tick()\n"+
		"[a, b]"), 0o644); err != nil {
		t.Fatalf("WriteFile main failed: %v", err)
	}

	data, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("ReadFile main failed: %v", err)
	}

	program, err := parser.Parse(mainPath, string(data))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	value, err := evaluator.New(nil).EvalProgram(program, runtime.NewEnvironment(nil))
	if err != nil {
		t.Fatalf("EvalProgram failed: %v", err)
	}

	expectShownValue(t, value, "[1, 2]")
}

func TestImportsOnlyExposeExplicitExportsEndToEnd(t *testing.T) {
	dir := t.TempDir()
	libPath := filepath.Join(dir, "lib.molt")
	mainPath := filepath.Join(dir, "main.molt")

	if err := os.WriteFile(libPath, []byte(""+
		"helper = 40\n"+
		"fn add2(x) = helper + x\n"+
		"export add2"), 0o644); err != nil {
		t.Fatalf("WriteFile lib failed: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte(""+
		"import lib from \"./lib.molt\"\n"+
		"lib.add2(2)"), 0o644); err != nil {
		t.Fatalf("WriteFile main failed: %v", err)
	}

	data, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("ReadFile main failed: %v", err)
	}

	program, err := parser.Parse(mainPath, string(data))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	value, err := evaluator.New(nil).EvalProgram(program, runtime.NewEnvironment(nil))
	if err != nil {
		t.Fatalf("EvalProgram failed: %v", err)
	}

	expectShownValue(t, value, "42")

	if err := os.WriteFile(mainPath, []byte(""+
		"import lib from \"./lib.molt\"\n"+
		"lib.helper"), 0o644); err != nil {
		t.Fatalf("WriteFile main failed: %v", err)
	}

	data, err = os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("ReadFile main failed: %v", err)
	}

	program, err = parser.Parse(mainPath, string(data))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	_, err = evaluator.New(nil).EvalProgram(program, runtime.NewEnvironment(nil))
	runtimeErr, ok := err.(diagnostic.RuntimeError)
	if !ok {
		t.Fatalf("expected runtime error, got %T (%v)", err, err)
	}

	if runtimeErr.Diagnostic().Message != `record has no field "helper"` {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, `record has no field "helper"`)
	}
}

func TestImportsReportCycleDiagnosticsEndToEnd(t *testing.T) {
	dir := t.TempDir()
	aPath := filepath.Join(dir, "a.molt")
	bPath := filepath.Join(dir, "b.molt")
	mainPath := filepath.Join(dir, "main.molt")

	if err := os.WriteFile(aPath, []byte(`import b from "./b.molt"`), 0o644); err != nil {
		t.Fatalf("WriteFile a failed: %v", err)
	}

	if err := os.WriteFile(bPath, []byte(`import a from "./a.molt"`), 0o644); err != nil {
		t.Fatalf("WriteFile b failed: %v", err)
	}

	if err := os.WriteFile(mainPath, []byte(`import a from "./a.molt"`), 0o644); err != nil {
		t.Fatalf("WriteFile main failed: %v", err)
	}

	data, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("ReadFile main failed: %v", err)
	}

	program, err := parser.Parse(mainPath, string(data))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	_, err = evaluator.New(nil).EvalProgram(program, runtime.NewEnvironment(nil))
	runtimeErr, ok := err.(diagnostic.RuntimeError)
	if !ok {
		t.Fatalf("expected runtime error, got %T (%v)", err, err)
	}

	want := "import cycle detected: " + filepath.ToSlash(aPath) + " -> " + filepath.ToSlash(bPath) + " -> " + filepath.ToSlash(aPath)
	if runtimeErr.Diagnostic().Message != want {
		t.Fatalf("message = %q, want %q", runtimeErr.Diagnostic().Message, want)
	}
}
