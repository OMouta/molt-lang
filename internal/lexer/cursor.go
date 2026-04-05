package lexer

import "molt/internal/source"

type cursor struct {
	file   *source.File
	text   string
	offset int
}

func newCursor(file *source.File) cursor {
	return cursor{
		file: file,
		text: file.Text(),
	}
}

func (c *cursor) isAtEnd() bool {
	return c.offset >= len(c.text)
}

func (c *cursor) peek() byte {
	return c.peekN(0)
}

func (c *cursor) peekN(distance int) byte {
	index := c.offset + distance
	if index < 0 || index >= len(c.text) {
		return 0
	}

	return c.text[index]
}

func (c *cursor) advance() byte {
	if c.isAtEnd() {
		return 0
	}

	ch := c.text[c.offset]
	c.offset++
	return ch
}

func (c *cursor) match(expected byte) bool {
	if c.peek() != expected {
		return false
	}

	c.offset++
	return true
}

func (c *cursor) span(start int) source.Span {
	return c.file.MustSpan(start, c.offset)
}
