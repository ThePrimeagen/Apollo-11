// Package termfont renders text as multi-height terminal banner fonts.
//
// STUB: contract only — implementation follows the failing test suite.
package termfont

import "errors"

const (
	// MinHeight is the shortest supported font: the plain terminal font.
	MinHeight = 1
	// MaxHeight is the tallest supported font. Anything beyond errors.
	MaxHeight = 5
)

// Charset lists every rune the art heights (2-5) can draw.
const Charset = ""

var (
	// ErrInvalidHeight reports a height outside MinHeight..MaxHeight.
	ErrInvalidHeight = errors.New("invalid height")
	// ErrUnsupportedRune reports a rune the requested height cannot draw.
	ErrUnsupportedRune = errors.New("unsupported rune")
)

// Render draws text at the given height. It returns a row-major byte
// buffer of exactly height*width bytes (no newlines) plus the width of
// each row, so the rows can be blitted anywhere inside caller-owned data.
func Render(height int, text string) ([]byte, int, error) {
	return nil, 0, nil
}

// Lines is a convenience view of Render: the buffer split into rows.
func Lines(height int, text string) ([]string, error) {
	return nil, nil
}
