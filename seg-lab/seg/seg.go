// Package seg renders segmented terminal characters.
//
// Unicode only encodes the ten seven-segment digits U+1FBF0 through
// U+1FBF9. Letters are composed: 7-segment for the ones that fit, and
// 14-segment box-drawing for the full alphabet. For sized writing, use
// package font (Small / Large).
package seg

import (
	"strings"
	"unicode"
)

// Style selects how a string is drawn.
type Style int

const (
	StyleUnicode Style = iota
	StyleSeven
	StyleFourteen
)

// Styles returns the viewer styles in cycle order, starting at unicode.
func Styles() []Style {
	return []Style{StyleUnicode, StyleSeven, StyleFourteen}
}

func (s Style) String() string {
	switch s {
	case StyleUnicode:
		return "unicode"
	case StyleSeven:
		return "7-seg"
	case StyleFourteen:
		return "14-seg"
	}
	return ""
}

const firstSegDigit = '\U0001FBF0' // SEGMENTED DIGIT ZERO

// UnicodeDigit returns the official segmented digit for r (U+1FBF0 + n).
func UnicodeDigit(r rune) (rune, bool) {
	if r < '0' || r > '9' {
		return 0, false
	}
	return firstSegDigit + (r - '0'), true
}

var sevenBlank = [3]string{"   ", "   ", "   "}

// sevenFont is the classic thin-stroke calculator glyph per rune.
// Letters that need diagonals or a third vertical (K M V W X) are absent.
var sevenFont = map[rune][3]string{
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
	'A': {" _ ", "|_|", "| |"},
	'B': {" _ ", "|_|", "|_|"},
	'b': {"   ", "|_ ", "|_|"},
	'C': {" _ ", "|  ", "|_ "},
	'c': {"   ", " _ ", "|_ "},
	'D': {" _ ", "| |", "|_|"},
	'd': {"   ", " _|", "|_|"},
	'E': {" _ ", "|_ ", "|_ "},
	'F': {" _ ", "|_ ", "|  "},
	'G': {" _ ", "|  ", "|_|"},
	'H': {"   ", "|_|", "| |"},
	'h': {"   ", "|_ ", "| |"},
	'I': {"   ", "  |", "  |"},
	'J': {"   ", "  |", "|_|"},
	'L': {"   ", "|  ", "|_ "},
	'N': {"   ", "|_ ", "| |"},
	'n': {"   ", "|_ ", "| |"},
	'O': {" _ ", "| |", "|_|"},
	'o': {"   ", " _ ", "|_|"},
	'P': {" _ ", "|_|", "|  "},
	'Q': {" _ ", "|_|", " _|"},
	'q': {" _ ", "|_|", "  |"},
	'R': {"   ", " _ ", "|  "},
	'r': {"   ", " _ ", "|  "},
	'S': {" _ ", "|_ ", " _|"},
	'T': {"   ", "|_ ", "|_ "},
	't': {"   ", "|_ ", "|_ "},
	'U': {"   ", "| |", "|_|"},
	'Y': {"   ", "|_|", " _|"},
	'y': {"   ", "|_|", " _|"},
	'Z': {" _ ", " _|", "|_ "},
	'-': {"   ", " _ ", "   "},
	'_': {"   ", "   ", " _ "},
}

// Seven returns the 3×3 seven-segment rows for r.
func Seven(r rune) ([3]string, bool) {
	if r == ' ' {
		return sevenBlank, true
	}
	if rows, ok := sevenFont[r]; ok {
		return rows, true
	}
	if rows, ok := sevenFont[unicode.ToUpper(r)]; ok {
		return rows, true
	}
	if rows, ok := sevenFont[unicode.ToLower(r)]; ok {
		return rows, true
	}
	return sevenBlank, false
}

// 14-segment bits. Layout:
//
//	    A A A
//	  F H I J B
//	    G   G     (G1 left, G2 right)
//	  E K L M C
//	    D D D
const (
	sA uint16 = 1 << iota
	sB
	sC
	sD
	sE
	sF
	sG1
	sG2
	sH
	sI
	sJ
	sK
	sL
	sM
)

var fourteenBlank = [5]string{"     ", "     ", "     ", "     ", "     "}

var fourteenFont = map[rune]uint16{
	'0': sA | sB | sC | sD | sE | sF,
	'1': sB | sC,
	'2': sA | sB | sG1 | sG2 | sE | sD,
	'3': sA | sB | sC | sD | sG1 | sG2,
	'4': sF | sG1 | sG2 | sB | sC,
	'5': sA | sF | sG1 | sG2 | sC | sD,
	'6': sA | sF | sE | sD | sC | sG1 | sG2,
	'7': sA | sB | sC,
	'8': sA | sB | sC | sD | sE | sF | sG1 | sG2,
	'9': sA | sF | sG1 | sG2 | sB | sC | sD,
	'A': sA | sF | sB | sG1 | sG2 | sE | sC,
	'B': sA | sB | sC | sD | sI | sL | sG1 | sG2,
	'C': sA | sF | sE | sD,
	'D': sA | sB | sC | sD | sI | sL,
	'E': sA | sF | sE | sD | sG1,
	'F': sA | sF | sE | sG1,
	'G': sA | sF | sE | sD | sC | sG2,
	'H': sF | sE | sB | sC | sG1 | sG2,
	'I': sA | sD | sI | sL,
	'J': sB | sC | sD | sE,
	'K': sF | sE | sG1 | sJ | sM,
	'L': sF | sE | sD,
	'M': sF | sE | sB | sC | sH | sJ,
	'N': sF | sE | sB | sC | sH | sM,
	'O': sA | sF | sE | sD | sC | sB,
	'P': sA | sF | sB | sG1 | sG2 | sE,
	'Q': sA | sF | sE | sD | sC | sB | sM,
	'R': sA | sF | sB | sG1 | sG2 | sE | sM,
	'S': sA | sF | sG1 | sG2 | sC | sD,
	'T': sA | sI | sL,
	'U': sF | sE | sD | sC | sB,
	'V': sF | sB | sK | sM,
	'W': sF | sE | sB | sC | sK | sM,
	'X': sH | sJ | sK | sM,
	'Y': sH | sJ | sL,
	'Z': sA | sJ | sK | sD,
	'-': sG1 | sG2,
	'+': sG1 | sG2 | sI | sL,
	'_': sD,
}

// Fourteen returns the 5×5 fourteen-segment rows for r.
func Fourteen(r rune) ([5]string, bool) {
	if r == ' ' {
		return fourteenBlank, true
	}
	bits, ok := fourteenFont[unicode.ToUpper(r)]
	if !ok {
		return fourteenBlank, false
	}
	return paint14(bits), true
}

func paint14(bits uint16) [5]string {
	g := [5][5]rune{
		{' ', ' ', ' ', ' ', ' '},
		{' ', ' ', ' ', ' ', ' '},
		{' ', ' ', ' ', ' ', ' '},
		{' ', ' ', ' ', ' ', ' '},
		{' ', ' ', ' ', ' ', ' '},
	}
	on := func(seg uint16) bool { return bits&seg != 0 }
	set := func(row, col int, ch rune) { g[row][col] = ch }

	if on(sA) {
		set(0, 1, '─')
		set(0, 2, '─')
		set(0, 3, '─')
	}
	if on(sD) {
		set(4, 1, '─')
		set(4, 2, '─')
		set(4, 3, '─')
	}
	if on(sG1) {
		set(2, 1, '─')
	}
	if on(sG2) {
		set(2, 3, '─')
	}
	if on(sG1) && on(sG2) {
		set(2, 2, '─')
	}
	if on(sF) {
		set(0, 0, '│')
		set(1, 0, '│')
	}
	if on(sE) {
		set(3, 0, '│')
		set(4, 0, '│')
	}
	if on(sF) && on(sE) {
		set(2, 0, '│')
	}
	if on(sB) {
		set(0, 4, '│')
		set(1, 4, '│')
	}
	if on(sC) {
		set(3, 4, '│')
		set(4, 4, '│')
	}
	if on(sB) && on(sC) {
		set(2, 4, '│')
	}
	if on(sI) {
		set(1, 2, '│')
	}
	if on(sL) {
		set(3, 2, '│')
	}
	if on(sI) && on(sL) && !on(sG1) && !on(sG2) {
		set(2, 2, '│')
	}
	if on(sH) {
		set(1, 1, '╲')
	}
	if on(sJ) {
		set(1, 3, '╱')
	}
	if on(sK) {
		set(3, 1, '╱')
	}
	if on(sM) {
		set(3, 3, '╲')
	}
	if on(sA) && on(sF) {
		set(0, 0, '┌')
	}
	if on(sA) && on(sB) {
		set(0, 4, '┐')
	}
	if on(sD) && on(sE) {
		set(4, 0, '└')
	}
	if on(sD) && on(sC) {
		set(4, 4, '┘')
	}
	if on(sG1) && (on(sF) || on(sE)) {
		set(2, 0, '├')
	}
	if on(sG2) && (on(sB) || on(sC)) {
		set(2, 4, '┤')
	}

	var rows [5]string
	for i := range g {
		rows[i] = string(g[i][:])
	}
	return rows
}

// Render draws text in style. Empty text or an unknown style yield "".
func Render(text string, style Style) string {
	switch style {
	case StyleUnicode:
		if text == "" {
			return ""
		}
		var b strings.Builder
		for _, r := range text {
			if d, ok := UnicodeDigit(r); ok {
				b.WriteRune(d)
			} else {
				b.WriteByte(' ')
			}
		}
		return b.String()
	case StyleSeven:
		return joinGlyphs(text, 3, func(r rune) []string {
			rows, _ := Seven(r)
			return rows[:]
		})
	case StyleFourteen:
		return joinGlyphs(text, 5, func(r rune) []string {
			rows, _ := Fourteen(r)
			return rows[:]
		})
	default:
		return ""
	}
}

func joinGlyphs(text string, height int, glyph func(rune) []string) string {
	rs := []rune(text)
	if len(rs) == 0 {
		return ""
	}
	lines := make([]string, height)
	for i, r := range rs {
		g := glyph(r)
		for row := 0; row < height; row++ {
			if i > 0 {
				lines[row] += " "
			}
			if row < len(g) {
				lines[row] += g[row]
			}
		}
	}
	return strings.Join(lines, "\n")
}
