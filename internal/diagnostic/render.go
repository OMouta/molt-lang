package diagnostic

import (
	"fmt"
	"strings"

	"molt/internal/source"
)

func Render(err DetailedError) string {
	return RenderDiagnostic(err.Diagnostic())
}

func RenderDiagnostic(diag Diagnostic) string {
	if err := diag.Validate(); err != nil {
		return fmt.Sprintf("invalid diagnostic: %v", err)
	}

	var b strings.Builder

	writeHeader(&b, diag.Kind, diag.Message, diag.Span)
	writeSnippet(&b, diag.Span)

	for _, note := range diag.Notes {
		b.WriteString("note: ")
		b.WriteString(note.Message)
		b.WriteByte('\n')

		if note.Span != nil {
			writeSnippet(&b, *note.Span)
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func writeHeader(b *strings.Builder, kind Kind, message string, span source.Span) {
	fmt.Fprintf(
		b,
		"%s:%d:%d: %s: %s\n",
		span.File.Path(),
		span.Start.Line,
		span.Start.Column,
		kind,
		message,
	)
}

func writeSnippet(b *strings.Builder, span source.Span) {
	if !span.IsValid() {
		return
	}

	lastLine := span.End.Line
	if span.End.Column == 1 && span.End.Offset > span.Start.Offset {
		lastLine--
	}

	if lastLine < span.Start.Line {
		lastLine = span.Start.Line
	}

	width := len(fmt.Sprintf("%d", lastLine))

	for line := span.Start.Line; line <= lastLine; line++ {
		text, ok := span.File.LineText(line)
		if !ok {
			continue
		}

		fmt.Fprintf(b, "%*d | %s\n", width, line, text)
		fmt.Fprintf(b, "%s | %s\n", strings.Repeat(" ", width), markerLine(text, span, line))
	}
}

func markerLine(text string, span source.Span, line int) string {
	lineLength := len(text)

	startColumn := 1
	if line == span.Start.Line {
		startColumn = span.Start.Column
	}

	endColumnExclusive := lineLength + 1
	switch {
	case span.IsSingleLine():
		endColumnExclusive = span.End.Column
	case line == span.End.Line:
		endColumnExclusive = span.End.Column
	case line < span.End.Line:
		endColumnExclusive = lineLength + 1
	}

	if span.End.Column == 1 && span.End.Offset > span.Start.Offset && line == span.End.Line {
		endColumnExclusive = 1
	}

	if startColumn < 1 {
		startColumn = 1
	}

	if startColumn > lineLength+1 {
		startColumn = lineLength + 1
	}

	if endColumnExclusive < startColumn {
		endColumnExclusive = startColumn
	}

	if endColumnExclusive > lineLength+1 {
		endColumnExclusive = lineLength + 1
	}

	width := endColumnExclusive - startColumn
	if width < 1 {
		width = 1
	}

	return strings.Repeat(" ", startColumn-1) + strings.Repeat("^", width)
}
