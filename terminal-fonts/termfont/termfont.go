// Package termfont renders text as multi-height terminal banner fonts,
// segment-display flavored — something near a 14-segment readout.
//
// Render(height, text) returns a row-major byte buffer with no newlines
// plus the width of each row, so callers can blit row r via
// buf[r*width : (r+1)*width] anywhere inside their own frame data.
//
// Height 1 is the plain terminal font: the characters themselves.
// Heights 2 through 5 are ASCII art. Anything outside 1..5 is an error.
package termfont

import (
	"bytes"
	"errors"
	"fmt"
)

const (
	// MinHeight is the shortest supported font: the plain terminal font.
	MinHeight = 1
	// MaxHeight is the tallest supported font. Anything beyond errors.
	MaxHeight = 5
)

// gap is the blank column between adjacent glyphs at art heights.
const gap = 1

// Charset lists every rune the art heights (2-5) can draw. Lowercase
// letters fold onto their uppercase glyphs. Height 1 is broader: it
// passes through any printable ASCII.
const Charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 .,:;!?'\"-+=()/\\_"

var (
	// ErrInvalidHeight reports a height outside MinHeight..MaxHeight.
	ErrInvalidHeight = errors.New("invalid height")
	// ErrUnsupportedRune reports a rune the requested height cannot draw.
	ErrUnsupportedRune = errors.New("unsupported rune")
)

// Render draws text at the given height. It returns a byte buffer of
// exactly height*width bytes — height rows of width bytes each, space
// padded, no newlines — plus the width of each row. Empty text is a
// zero-width buffer. On error the buffer is nil and the width is zero.
func Render(height int, text string) ([]byte, int, error) {
	if height < MinHeight || height > MaxHeight {
		return nil, 0, fmt.Errorf("termfont: %w: %d (supported: %d..%d)", ErrInvalidHeight, height, MinHeight, MaxHeight)
	}
	if text == "" {
		return []byte{}, 0, nil
	}
	if height == 1 {
		for i, r := range text {
			if r < ' ' || r > '~' {
				return nil, 0, fmt.Errorf("termfont: %w: %q at index %d (height 1 draws printable ASCII only)", ErrUnsupportedRune, r, i)
			}
		}
		return []byte(text), len(text), nil
	}
	return compose(glyphSets[height], Charset, height, text)
}

// Lines is a convenience view of Render: the same buffer split into
// height strings of width bytes each.
func Lines(height int, text string) ([]string, error) {
	buf, width, err := Render(height, text)
	if err != nil {
		return nil, err
	}
	return splitRows(buf, height, width), nil
}

// compose blits one glyph per rune (lowercase folded) from the given
// table into a row-major buffer, one blank column between glyphs.
func compose(set map[rune][]string, charset string, height int, text string) ([]byte, int, error) {
	var glyphs [][]string
	width := 0
	for i, r := range text {
		g, ok := set[foldRune(r)]
		if !ok {
			return nil, 0, fmt.Errorf("termfont: %w: %q at index %d (height %d draws %q)", ErrUnsupportedRune, r, i, height, charset)
		}
		if len(glyphs) > 0 {
			width += gap
		}
		width += len(g[0])
		glyphs = append(glyphs, g)
	}

	buf := bytes.Repeat([]byte{' '}, height*width)
	x := 0
	for i, g := range glyphs {
		if i > 0 {
			x += gap
		}
		for row := 0; row < height; row++ {
			copy(buf[row*width+x:], g[row])
		}
		x += len(g[0])
	}
	return buf, width, nil
}

// splitRows slices a Render buffer into its height rows.
func splitRows(buf []byte, height, width int) []string {
	lines := make([]string, height)
	if width == 0 {
		return lines
	}
	for r := 0; r < height; r++ {
		lines[r] = string(buf[r*width : (r+1)*width])
	}
	return lines
}

// foldRune maps lowercase letters onto their uppercase glyphs.
func foldRune(r rune) rune {
	if r >= 'a' && r <= 'z' {
		return r - ('a' - 'A')
	}
	return r
}
