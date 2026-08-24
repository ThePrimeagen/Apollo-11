package termfont

// STUB: glyph tables land with the implementation.

// glyphSets maps an art height (2-5) to its rune glyph table. Every glyph
// is height rows of equal-width printable ASCII.
var glyphSets = map[int]map[rune][]string{}

// validateGlyphSet checks a table: complete rows, rectangular, ASCII-only.
func validateGlyphSet(height int, set map[rune][]string) error {
	return nil
}
