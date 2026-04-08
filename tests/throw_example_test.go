package integration_test

import (
	"strings"
	"testing"

	"molt/internal/diagnostic"
)

func TestThrowExampleProducesExpectedRuntimeDiagnostic(t *testing.T) {
	_, _, err := executeFile(t, "examples/errors/throw_error.molt")
	runtimeErr := expectRuntimeDiagnostic(t, err)

	diag := runtimeErr.Diagnostic()
	if diag.Message != "config file not found" {
		t.Fatalf("message = %q, want %q", diag.Message, "config file not found")
	}

	if diag.Span.Start.Line != 2 || diag.Span.Start.Column != 3 {
		t.Fatalf("span start = %d:%d, want 2:3", diag.Span.Start.Line, diag.Span.Start.Column)
	}

	if len(diag.Notes) != 1 {
		t.Fatalf("note count = %d, want 1", len(diag.Notes))
	}

	if diag.Notes[0].Message != `error data: record { path: "settings.json" }` {
		t.Fatalf("note = %q, want %q", diag.Notes[0].Message, `error data: record { path: "settings.json" }`)
	}

	rendered := diagnostic.Render(runtimeErr)
	if !strings.Contains(rendered, `note: error data: record { path: "settings.json" }`) {
		t.Fatalf("rendered diagnostic = %q, want data note", rendered)
	}
}

func expectRuntimeDiagnostic(t *testing.T, err error) diagnostic.RuntimeError {
	t.Helper()

	if err == nil {
		t.Fatal("expected runtime error, got nil")
	}

	runtimeErr, ok := err.(diagnostic.RuntimeError)
	if !ok {
		t.Fatalf("expected diagnostic.RuntimeError, got %T (%v)", err, err)
	}

	return runtimeErr
}
