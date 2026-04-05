package source

import (
	"fmt"
	"sort"
)

// Position identifies a byte offset together with 1-based line and column data.
type Position struct {
	Offset int
	Line   int
	Column int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Span represents a half-open source range [Start, End).
type Span struct {
	File  *File
	Start Position
	End   Position
}

func (s Span) IsValid() bool {
	return s.File != nil && s.Start.Offset <= s.End.Offset
}

func (s Span) IsSingleLine() bool {
	return s.Start.Line == s.End.Line
}

func (s Span) Len() int {
	return s.End.Offset - s.Start.Offset
}

// File stores source text together with precomputed line starts for efficient
// offset-to-line/column mapping.
type File struct {
	path       string
	text       string
	lineStarts []int
}

func NewFile(path, text string) *File {
	lineStarts := []int{0}

	for i := 0; i < len(text); i++ {
		if text[i] == '\n' {
			lineStarts = append(lineStarts, i+1)
		}
	}

	return &File{
		path:       path,
		text:       text,
		lineStarts: lineStarts,
	}
}

func (f *File) Path() string {
	return f.path
}

func (f *File) Text() string {
	return f.text
}

func (f *File) Len() int {
	return len(f.text)
}

func (f *File) LineCount() int {
	return len(f.lineStarts)
}

func (f *File) Position(offset int) (Position, error) {
	if offset < 0 || offset > len(f.text) {
		return Position{}, fmt.Errorf("offset %d out of range [0, %d]", offset, len(f.text))
	}

	lineIndex := sort.Search(len(f.lineStarts), func(i int) bool {
		return f.lineStarts[i] > offset
	}) - 1

	if lineIndex < 0 {
		lineIndex = 0
	}

	lineStart := f.lineStarts[lineIndex]

	return Position{
		Offset: offset,
		Line:   lineIndex + 1,
		Column: (offset - lineStart) + 1,
	}, nil
}

func (f *File) Span(startOffset, endOffset int) (Span, error) {
	if startOffset < 0 || endOffset < 0 {
		return Span{}, fmt.Errorf("span offsets must be non-negative: start=%d end=%d", startOffset, endOffset)
	}

	if startOffset > endOffset {
		return Span{}, fmt.Errorf("span start %d is after end %d", startOffset, endOffset)
	}

	start, err := f.Position(startOffset)
	if err != nil {
		return Span{}, err
	}

	end, err := f.Position(endOffset)
	if err != nil {
		return Span{}, err
	}

	return Span{
		File:  f,
		Start: start,
		End:   end,
	}, nil
}

func (f *File) MustSpan(startOffset, endOffset int) Span {
	span, err := f.Span(startOffset, endOffset)
	if err != nil {
		panic(err)
	}

	return span
}

func (f *File) Slice(span Span) (string, error) {
	if span.File != f {
		return "", fmt.Errorf("span belongs to a different source file")
	}

	if !span.IsValid() {
		return "", fmt.Errorf("invalid span")
	}

	return f.text[span.Start.Offset:span.End.Offset], nil
}

func (f *File) LineText(line int) (string, bool) {
	start, ok := f.lineStartOffset(line)
	if !ok {
		return "", false
	}

	end, _ := f.lineEndOffset(line)
	return f.text[start:end], true
}

func (f *File) lineStartOffset(line int) (int, bool) {
	if line < 1 || line > len(f.lineStarts) {
		return 0, false
	}

	return f.lineStarts[line-1], true
}

func (f *File) lineEndOffset(line int) (int, bool) {
	start, ok := f.lineStartOffset(line)
	if !ok {
		return 0, false
	}

	if line == len(f.lineStarts) {
		return len(f.text), true
	}

	end := f.lineStarts[line] - 1
	if end > start && f.text[end-1] == '\r' {
		end--
	}

	return end, true
}
