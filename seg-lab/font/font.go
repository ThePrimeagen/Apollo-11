// Package font draws a string in 14-segment terminal strokes.
//
// Unicode has no segmented letters. This package does not generate a TTF
// and does not call Python — pass a string and Small or Large.
package font

import (
	"strings"
	"unicode"
)

// Size is the writing scale.
type Size int

const (
	Small Size = iota
	Large
)

func (s Size) String() string {
	switch s {
	case Small:
		return "small"
	case Large:
		return "large"
	}
	return ""
}

// GlyphSize is the per-letter cell (width × height) for size.
func GlyphSize(size Size) (w, h int) {
	switch size {
	case Small:
		return 5, 5
	case Large:
		return 7, 9
	}
	return 0, 0
}

// 14-segment bits.
//
//	    A A A
//	  F H I J B
//	    G   G
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

var glyphs = map[rune]uint16{
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

// Render draws text at size. Empty text or an unknown size yield "".
func Render(text string, size Size) string {
	w, h := GlyphSize(size)
	if w == 0 || h == 0 {
		return ""
	}
	rs := []rune(text)
	if len(rs) == 0 {
		return ""
	}
	lines := make([]string, h)
	for i, r := range rs {
		g := paint(lookup(r), w, h)
		for row := 0; row < h; row++ {
			if i > 0 {
				lines[row] += " "
			}
			lines[row] += g[row]
		}
	}
	return strings.Join(lines, "\n")
}

func lookup(r rune) uint16 {
	if r == ' ' {
		return 0
	}
	if bits, ok := glyphs[unicode.ToUpper(r)]; ok {
		return bits
	}
	return 0
}

func paint(bits uint16, w, h int) []string {
	g := make([][]rune, h)
	for r := range g {
		g[r] = []rune(strings.Repeat(" ", w))
	}
	on := func(seg uint16) bool { return bits&seg != 0 }
	set := func(row, col int, ch rune) {
		if row >= 0 && row < h && col >= 0 && col < w {
			g[row][col] = ch
		}
	}
	mid := h / 2
	cx := w / 2
	last := w - 1

	if on(sA) {
		for c := 1; c < last; c++ {
			set(0, c, '─')
		}
	}
	if on(sD) {
		for c := 1; c < last; c++ {
			set(h-1, c, '─')
		}
	}
	if on(sG1) {
		for c := 1; c < cx; c++ {
			set(mid, c, '─')
		}
	}
	if on(sG2) {
		for c := cx + 1; c < last; c++ {
			set(mid, c, '─')
		}
	}
	if on(sG1) && on(sG2) {
		set(mid, cx, '─')
	}
	if on(sF) {
		for r := 0; r <= mid; r++ {
			set(r, 0, '│')
		}
	}
	if on(sE) {
		for r := mid; r < h; r++ {
			set(r, 0, '│')
		}
	}
	if on(sB) {
		for r := 0; r <= mid; r++ {
			set(r, last, '│')
		}
	}
	if on(sC) {
		for r := mid; r < h; r++ {
			set(r, last, '│')
		}
	}
	if on(sI) {
		for r := 1; r < mid; r++ {
			set(r, cx, '│')
		}
	}
	if on(sL) {
		for r := mid + 1; r < h-1; r++ {
			set(r, cx, '│')
		}
	}
	if on(sI) && on(sL) && !on(sG1) && !on(sG2) {
		set(mid, cx, '│')
	}

	// Diagonals live in the four quadrants around the center.
	if on(sH) {
		diag(set, 1, 1, mid-1, cx-1, '╲')
	}
	if on(sJ) {
		diag(set, 1, last-1, mid-1, cx+1, '╱')
	}
	if on(sK) {
		diag(set, h-2, 1, mid+1, cx-1, '╱')
	}
	if on(sM) {
		diag(set, h-2, last-1, mid+1, cx+1, '╲')
	}

	if on(sA) && on(sF) {
		set(0, 0, '┌')
	}
	if on(sA) && on(sB) {
		set(0, last, '┐')
	}
	if on(sD) && on(sE) {
		set(h-1, 0, '└')
	}
	if on(sD) && on(sC) {
		set(h-1, last, '┘')
	}
	if on(sG1) && (on(sF) || on(sE)) {
		set(mid, 0, '├')
	}
	if on(sG2) && (on(sB) || on(sC)) {
		set(mid, last, '┤')
	}

	out := make([]string, h)
	for i := range g {
		out[i] = string(g[i])
	}
	return out
}

func diag(set func(int, int, rune), r0, c0, r1, c1 int, ch rune) {
	n := abs(r1-r0) + 1
	if m := abs(c1-c0) + 1; m > n {
		n = m
	}
	if n <= 1 {
		set(r0, c0, ch)
		return
	}
	for i := 0; i < n; i++ {
		r := r0 + (r1-r0)*i/(n-1)
		c := c0 + (c1-c0)*i/(n-1)
		set(r, c, ch)
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
