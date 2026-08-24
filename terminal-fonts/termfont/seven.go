package termfont

// Seven-segment numbers at variable height. Unlike the banner font,
// these are true seven-segment shapes: straight "_" and "|" segments
// only — no diagonals, no slashed zero, no slanted one — and every
// digit occupies the same display cell width. Height 2 has no room for
// a top-bar row, so 0 and 7 carry a "~" overline to stay unambiguous;
// every other digit survives on its middle and bottom bars alone.

import (
	"fmt"
	"strings"
)

// SevenCharset lists every rune the seven-segment display can show:
// the ten digits plus the clock/calculator marks a real display has.
const SevenCharset = "0123456789 .:-"

// RenderSeven draws text as seven-segment digits at the given height,
// under the same row-major buffer contract as Render: height*width
// bytes, no newlines, plus the width of each row. Height 1 prints the
// characters plainly, but the charset stays SevenCharset — a numeric
// display shows no letters at any height.
func RenderSeven(height int, text string) ([]byte, int, error) {
	if height < MinHeight || height > MaxHeight {
		return nil, 0, fmt.Errorf("termfont: %w: %d (supported: %d..%d)", ErrInvalidHeight, height, MinHeight, MaxHeight)
	}
	if text == "" {
		return []byte{}, 0, nil
	}
	if height == 1 {
		for i, r := range text {
			if !strings.ContainsRune(SevenCharset, r) {
				return nil, 0, fmt.Errorf("termfont: %w: %q at index %d (seven-segment draws %q)", ErrUnsupportedRune, r, i, SevenCharset)
			}
		}
		return []byte(text), len(text), nil
	}
	return compose(sevenSets[height], SevenCharset, height, text)
}

// LinesSeven is a convenience view of RenderSeven: the same buffer
// split into height strings of width bytes each.
func LinesSeven(height int, text string) ([]string, error) {
	buf, width, err := RenderSeven(height, text)
	if err != nil {
		return nil, err
	}
	return splitRows(buf, height, width), nil
}

// sevenSets maps an art height (2-5) to its seven-segment glyph table.
var sevenSets = map[int]map[rune][]string{
	2: sevenGlyphs2,
	3: sevenGlyphs3,
	4: sevenGlyphs4,
	5: sevenGlyphs5,
}

var sevenGlyphs2 = map[rune][]string{
	'0': {"|~|", "|_|"},
	'1': {"  |", "  |"},
	'2': {" _|", "|_ "},
	'3': {" _|", " _|"},
	'4': {"|_|", "  |"},
	'5': {"|_ ", " _|"},
	'6': {"|_ ", "|_|"},
	'7': {"~~|", "  |"},
	'8': {"|_|", "|_|"},
	'9': {"|_|", " _|"},
	' ': {"   ", "   "},
	'.': {" ", "."},
	':': {".", "."},
	'-': {" _ ", "   "},
}

var sevenGlyphs3 = map[rune][]string{
	'0': {" _ ", "| |", "|_|"},
	'1': {"   ", "  |", "  |"},
	'2': {" _ ", " _|", "|_ "},
	'3': {" _ ", " _|", " _|"},
	'4': {"   ", "|_|", "  |"},
	'5': {" _ ", "|_ ", " _|"},
	'6': {" _ ", "|_ ", "|_|"},
	'7': {" _ ", "  |", "  |"},
	'8': {" _ ", "|_|", "|_|"},
	'9': {" _ ", "|_|", " _|"},
	' ': {"   ", "   ", "   "},
	'.': {" ", " ", "."},
	':': {" ", ".", "."},
	'-': {"   ", " _ ", "   "},
}

var sevenGlyphs4 = map[rune][]string{
	'0': {" _ ", "| |", "| |", "|_|"},
	'1': {"   ", "  |", "  |", "  |"},
	'2': {" _ ", "  |", " _|", "|_ "},
	'3': {" _ ", "  |", " _|", " _|"},
	'4': {"   ", "| |", "|_|", "  |"},
	'5': {" _ ", "|  ", "|_ ", " _|"},
	'6': {" _ ", "|  ", "|_ ", "|_|"},
	'7': {" _ ", "  |", "  |", "  |"},
	'8': {" _ ", "| |", "|_|", "|_|"},
	'9': {" _ ", "| |", "|_|", " _|"},
	' ': {"   ", "   ", "   ", "   "},
	'.': {" ", " ", " ", "."},
	':': {" ", ".", " ", "."},
	'-': {"   ", "   ", " _ ", "   "},
}

var sevenGlyphs5 = map[rune][]string{
	'0': {" __ ", "|  |", "|  |", "|  |", "|__|"},
	'1': {"    ", "   |", "   |", "   |", "   |"},
	'2': {" __ ", "   |", " __|", "|   ", "|__ "},
	'3': {" __ ", "   |", " __|", "   |", " __|"},
	'4': {"    ", "|  |", "|__|", "   |", "   |"},
	'5': {" __ ", "|   ", "|__ ", "   |", " __|"},
	'6': {" __ ", "|   ", "|__ ", "|  |", "|__|"},
	'7': {" __ ", "   |", "   |", "   |", "   |"},
	'8': {" __ ", "|  |", "|__|", "|  |", "|__|"},
	'9': {" __ ", "|  |", "|__|", "   |", " __|"},
	' ': {"    ", "    ", "    ", "    ", "    "},
	'.': {" ", " ", " ", " ", "."},
	':': {" ", ".", " ", ".", " "},
	'-': {"    ", "    ", " __ ", "    ", "    "},
}
