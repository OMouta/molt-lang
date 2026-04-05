package lexer

import (
	"testing"

	"molt/internal/source"
)

func TestCursorPeekingAdvancingAndMatching(t *testing.T) {
	file := source.NewFile("cursor.molt", "abc")
	cursor := newCursor(file)

	if got := cursor.peek(); got != 'a' {
		t.Fatalf("peek() = %q, want %q", got, 'a')
	}

	if got := cursor.peekN(2); got != 'c' {
		t.Fatalf("peekN(2) = %q, want %q", got, 'c')
	}

	if got := cursor.advance(); got != 'a' {
		t.Fatalf("advance() = %q, want %q", got, 'a')
	}

	if !cursor.match('b') {
		t.Fatalf("match('b') = false, want true")
	}

	if cursor.match('b') {
		t.Fatalf("match('b') = true, want false after consuming 'b'")
	}

	if got := cursor.advance(); got != 'c' {
		t.Fatalf("advance() = %q, want %q", got, 'c')
	}

	if !cursor.isAtEnd() {
		t.Fatalf("cursor should be at end")
	}
}

func TestCursorSpanUsesCurrentOffsets(t *testing.T) {
	file := source.NewFile("cursor.molt", "hello")
	cursor := newCursor(file)

	cursor.advance()
	cursor.advance()
	cursor.advance()

	span := cursor.span(1)
	if span.Start.Offset != 1 || span.End.Offset != 3 {
		t.Fatalf("span offsets = [%d, %d), want [1, 3)", span.Start.Offset, span.End.Offset)
	}
}
