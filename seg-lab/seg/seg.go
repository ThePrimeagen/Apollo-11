// Package seg renders segmented terminal characters.
//
// Unicode only encodes the ten seven-segment digits U+1FBF0–U+1FBF9
// (🯰🯱🯲🯳🯴🯵�+1FBF0–U+1FBF9
// (🯰🯱🯲🯳🯴🯵🯶🯷🯸🯹). There are no segmented letter codepoints, so A–Z are
// composed: 7-segment for the letters that fit a calculator display, and
// 14-segment (box-drawing) for the full alphabet.
package seg

// Style selects how a string is drawn.
type Style int

const (
	StyleUnicode Style = iota
	StyleSeven
	StyleFourteen
)

// Styles returns the viewer styles in cycle order, starting at unicode.
func Styles() []Style {
	return nil
}

func (s Style) String() string { return "" }

// UnicodeDigit returns the official segmented digit for r (U+1FBF0 + n).
func UnicodeDigit(r rune) (rune, bool) { return 0, false }

// Seven returns the 3×3 seven-segment rows for r.
func Seven(r rune) ([3]string, bool) { return [3]string{"   ", "   ", "   "}, false }

// Fourteen returns the 5×5 fourteen-segment rows for r.
func Fourteen(r rune) ([5]string, bool) {
	return [5]string{"     ", "     ", "     ", "     ", "     "}, false
}

// Render draws text in style. Empty text or an unknown style yield "".
func Render(text string, style Style) string { return "" }
