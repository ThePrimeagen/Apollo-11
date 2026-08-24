// Package font draws a string in 14-segment LED bars.
//
// Unicode has no segmented letters. Pass a string and Small or Large.
// Bars are filled blocks with a gap at each joint — the same look as a
// calculator display, not a box-drawing wireframe.
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
		return 7, 7
	case Large:
		return 11, 13
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

// paint fills 14-segment bars the way the old TTF did: thick blocks
// with a one-cell gap at each joint, not a connected wireframe.
func paint(bits uint16, w, h int) []string {
	g := make([][]rune, h)
	for r := range g {
		g[r] = []rune(strings.Repeat(" ", w))
	}
	on := func(seg uint16) bool { return bits&seg != 0 }
	plot := func(row, col int) {
		if row >= 0 && row < h && col >= 0 && col < w {
			g[row][col] = '█'
		}
	}
	fill := func(r0, c0, r1, c1 int) {
		if r0 > r1 {
			r0, r1 = r1, r0
		}
		if c0 > c1 {
			c0, c1 = c1, c0
		}
		for r := r0; r <= r1; r++ {
			for c := c0; c <= c1; c++ {
				plot(r, c)
			}
		}
	}

	t := 1
	if h >= 11 {
		t = 2
	}
	mid := h / 2
	cx := w / 2
	last := w - 1
	// Inset horizontals so they do not meet the verticals — the LED gap.
	innerL := t + 1
	innerR := last - t - 1

	if on(sA) {
		fill(0, innerL, t-1, innerR)
	}
	if on(sD) {
		fill(h-t, innerL, h-1, innerR)
	}
	if on(sG1) {
		fill(mid-t/2, innerL, mid-t/2+t-1, cx-1)
	}
	if on(sG2) {
		fill(mid-t/2, cx+1, mid-t/2+t-1, innerR)
	}
	if on(sF) {
		fill(t, 0, mid-1, t-1)
	}
	if on(sE) {
		fill(mid+1, 0, h-1-t, t-1)
	}
	if on(sB) {
		fill(t, last-t+1, mid-1, last)
	}
	if on(sC) {
		fill(mid+1, last-t+1, h-1-t, last)
	}
	if on(sI) {
		fill(t, cx-t/2, mid-1, cx-t/2+t-1)
	}
	if on(sL) {
		fill(mid+1, cx-t/2, h-1-t, cx-t/2+t-1)
	}
	if on(sH) {
		diag(plot, t, t, mid-1, cx-1, t)
	}
	if on(sJ) {
		diag(plot, t, last-t, mid-1, cx+1, t)
	}
	if on(sK) {
		diag(plot, h-1-t, t, mid+1, cx-1, t)
	}
	if on(sM) {
		diag(plot, h-1-t, last-t, mid+1, cx+1, t)
	}

	out := make([]string, h)
	for i := range g {
		out[i] = string(g[i])
	}
	return out
}

func diag(plot func(int, int), r0, c0, r1, c1, thick int) {
	n := abs(r1-r0) + 1
	if m := abs(c1-c0) + 1; m > n {
		n = m
	}
	if n <= 1 {
		plot(r0, c0)
		return
	}
	if thick < 1 {
		thick = 1
	}
	for i := 0; i < n; i++ {
		r := r0 + (r1-r0)*i/(n-1)
		c := c0 + (c1-c0)*i/(n-1)
		for dt := 0; dt < thick; dt++ {
			plot(r, c+dt)
			plot(r+dt, c)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
