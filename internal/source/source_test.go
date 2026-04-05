package source

import "testing"

func TestPositionTracksByteOffsetsAcrossLines(t *testing.T) {
	file := NewFile("sample.molt", "alpha\nbeta\r\ngamma")

	tests := []struct {
		offset int
		want   Position
	}{
		{offset: 0, want: Position{Offset: 0, Line: 1, Column: 1}},
		{offset: 5, want: Position{Offset: 5, Line: 1, Column: 6}},
		{offset: 6, want: Position{Offset: 6, Line: 2, Column: 1}},
		{offset: 11, want: Position{Offset: 11, Line: 2, Column: 6}},
		{offset: 12, want: Position{Offset: 12, Line: 3, Column: 1}},
		{offset: 13, want: Position{Offset: 13, Line: 3, Column: 2}},
		{offset: len(file.Text()), want: Position{Offset: len(file.Text()), Line: 3, Column: 6}},
	}

	for _, tc := range tests {
		got, err := file.Position(tc.offset)
		if err != nil {
			t.Fatalf("Position(%d) returned error: %v", tc.offset, err)
		}

		if got != tc.want {
			t.Fatalf("Position(%d) = %+v, want %+v", tc.offset, got, tc.want)
		}
	}
}

func TestSpanSupportsSingleAndMultiLineRanges(t *testing.T) {
	file := NewFile("sample.molt", "abc\ndefg\nhi")

	singleLine := file.MustSpan(1, 3)
	if !singleLine.IsSingleLine() {
		t.Fatalf("single-line span reported multi-line")
	}

	if singleLine.Start.Line != 1 || singleLine.Start.Column != 2 {
		t.Fatalf("unexpected single-line span start: %+v", singleLine.Start)
	}

	if singleLine.End.Line != 1 || singleLine.End.Column != 4 {
		t.Fatalf("unexpected single-line span end: %+v", singleLine.End)
	}

	multiLine := file.MustSpan(2, 8)
	if multiLine.IsSingleLine() {
		t.Fatalf("multi-line span reported single-line")
	}

	if multiLine.Start.Line != 1 || multiLine.End.Line != 2 {
		t.Fatalf("unexpected multi-line span lines: start=%d end=%d", multiLine.Start.Line, multiLine.End.Line)
	}

	snippet, err := file.Slice(multiLine)
	if err != nil {
		t.Fatalf("Slice returned error: %v", err)
	}

	if snippet != "c\ndefg" {
		t.Fatalf("Slice returned %q, want %q", snippet, "c\ndefg")
	}
}

func TestLineTextTrimsLineTerminators(t *testing.T) {
	file := NewFile("sample.molt", "one\r\ntwo\nthree")

	tests := []struct {
		line int
		want string
	}{
		{line: 1, want: "one"},
		{line: 2, want: "two"},
		{line: 3, want: "three"},
	}

	for _, tc := range tests {
		got, ok := file.LineText(tc.line)
		if !ok {
			t.Fatalf("LineText(%d) returned !ok", tc.line)
		}

		if got != tc.want {
			t.Fatalf("LineText(%d) = %q, want %q", tc.line, got, tc.want)
		}
	}
}
