package diagnostic

import (
	"fmt"

	"molt/internal/source"
)

type Kind string

const (
	ParseErrorKind   Kind = "parse error"
	RuntimeErrorKind Kind = "runtime error"
)

type Note struct {
	Message string
	Span    *source.Span
}

type Diagnostic struct {
	Kind    Kind
	Message string
	Span    source.Span
	Notes   []Note
}

func (d Diagnostic) Validate() error {
	if d.Message == "" {
		return fmt.Errorf("diagnostic message cannot be empty")
	}

	if !d.Span.IsValid() {
		return fmt.Errorf("diagnostic span must be valid")
	}

	for i, note := range d.Notes {
		if note.Message == "" {
			return fmt.Errorf("diagnostic note %d has an empty message", i)
		}

		if note.Span != nil && !note.Span.IsValid() {
			return fmt.Errorf("diagnostic note %d has an invalid span", i)
		}
	}

	return nil
}

func (d Diagnostic) Location() string {
	return fmt.Sprintf("%s:%d:%d", d.Span.File.Path(), d.Span.Start.Line, d.Span.Start.Column)
}

type DetailedError interface {
	error
	Diagnostic() Diagnostic
}

type ParseError struct {
	diag Diagnostic
}

func NewParseError(message string, span source.Span, notes ...Note) ParseError {
	return ParseError{
		diag: Diagnostic{
			Kind:    ParseErrorKind,
			Message: message,
			Span:    span,
			Notes:   append([]Note(nil), notes...),
		},
	}
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%s: %s", e.diag.Location(), e.diag.Message)
}

func (e ParseError) Diagnostic() Diagnostic {
	return e.diag
}

type RuntimeError struct {
	diag Diagnostic
}

func NewRuntimeError(message string, span source.Span, notes ...Note) RuntimeError {
	return RuntimeError{
		diag: Diagnostic{
			Kind:    RuntimeErrorKind,
			Message: message,
			Span:    span,
			Notes:   append([]Note(nil), notes...),
		},
	}
}

func (e RuntimeError) Error() string {
	return fmt.Sprintf("%s: %s", e.diag.Location(), e.diag.Message)
}

func (e RuntimeError) Diagnostic() Diagnostic {
	return e.diag
}
