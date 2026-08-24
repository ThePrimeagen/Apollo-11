package termfont

// STUB: seven-segment contract only — implementation follows the failing
// test suite.

// SevenCharset lists every rune the seven-segment display can show.
const SevenCharset = ""

// sevenSets maps an art height (2-5) to its seven-segment glyph table.
var sevenSets = map[int]map[rune][]string{}

// RenderSeven draws text as seven-segment digits at the given height,
// under the same row-major buffer contract as Render.
func RenderSeven(height int, text string) ([]byte, int, error) {
	return nil, 0, nil
}

// LinesSeven is a convenience view of RenderSeven: the buffer rows.
func LinesSeven(height int, text string) ([]string, error) {
	return nil, nil
}
