package diagnostic

import (
	"testing"

	"molt/internal/source"
)

func TestParseAndRuntimeErrorsExposeStructuredDiagnostics(t *testing.T) {
	file := source.NewFile("sample.molt", "x = 1")
	span := file.MustSpan(0, 1)

	parseErr := NewParseError("unexpected token", span)
	if got := parseErr.Diagnostic().Kind; got != ParseErrorKind {
		t.Fatalf("parse error kind = %q, want %q", got, ParseErrorKind)
	}

	runtimeErr := NewRuntimeError("division by zero", span)
	if got := runtimeErr.Diagnostic().Kind; got != RuntimeErrorKind {
		t.Fatalf("runtime error kind = %q, want %q", got, RuntimeErrorKind)
	}

	if parseErr.Diagnostic().Span != span {
		t.Fatalf("parse error span did not round-trip")
	}

	if runtimeErr.Diagnostic().Span != span {
		t.Fatalf("runtime error span did not round-trip")
	}
}

func TestRenderSingleLineDiagnostic(t *testing.T) {
	file := source.NewFile("single.molt", "x = 1 +")
	err := NewParseError("expected expression after '+'", file.MustSpan(6, 7))

	got := Render(err)
	want := "" +
		"single.molt:1:7: parse error: expected expression after '+'\n" +
		"1 | x = 1 +\n" +
		"  |       ^"

	if got != want {
		t.Fatalf("rendered diagnostic mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderMultiLineDiagnostic(t *testing.T) {
	file := source.NewFile("multi.molt", "warp(\n  x,\n  y\n")
	err := NewParseError("unterminated call expression", file.MustSpan(4, len(file.Text())))

	got := Render(err)
	want := "" +
		"multi.molt:1:5: parse error: unterminated call expression\n" +
		"1 | warp(\n" +
		"  |     ^\n" +
		"2 |   x,\n" +
		"  | ^^^^\n" +
		"3 |   y\n" +
		"  | ^^^"

	if got != want {
		t.Fatalf("rendered diagnostic mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderDiagnosticWithNotesAndMultiLineRange(t *testing.T) {
	file := source.NewFile(
		"notes.molt",
		"fn main() = {\n  x = 1\n  y = x +\n}\n",
	)

	primary := file.MustSpan(12, 31)
	noteSpan := file.MustSpan(3, 7)
	err := NewParseError(
		"block body ended with an incomplete expression",
		primary,
		Note{
			Message: "function body starts here",
			Span:    &noteSpan,
		},
		Note{
			Message: "add an expression after '+'",
		},
	)

	got := Render(err)
	want := "" +
		"notes.molt:1:13: parse error: block body ended with an incomplete expression\n" +
		"1 | fn main() = {\n" +
		"  |             ^\n" +
		"2 |   x = 1\n" +
		"  | ^^^^^^^\n" +
		"3 |   y = x +\n" +
		"  | ^^^^^^^^^\n" +
		"note: function body starts here\n" +
		"1 | fn main() = {\n" +
		"  |    ^^^^\n" +
		"note: add an expression after '+'"

	if got != want {
		t.Fatalf("rendered diagnostic mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
