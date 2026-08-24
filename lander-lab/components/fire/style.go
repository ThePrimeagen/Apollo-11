package fire

import (
	"fmt"
	"strings"

	"github.com/theprimeagen/apollo-11/lander-lab/particle"
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

// Band is one rung of the heat → glyph ladder.
type Band struct {
	Min, Max int
	Glyph    rune
	Name     string
	FG, BG   int
	Eq       string
}

// Bands is the full style table, sparse → dense. Bright yellow is the
// ceiling; nothing uses xterm 231 (white).
func Bands() []Band {
	return []Band{
		{Min: 1, Max: 1, Glyph: '⠁', Name: "single braille", FG: 88, BG: -1, Eq: "H == 1"},
		{Min: 2, Max: 2, Glyph: '⠒', Name: "two-dot braille", FG: 124, BG: -1, Eq: "H == 2"},
		{Min: 3, Max: 4, Glyph: '⠶', Name: "four-dot braille", FG: 160, BG: -1, Eq: "3 <= H <= 4"},
		{Min: 5, Max: 8, Glyph: '░', Name: "quarter shade", FG: 166, BG: -1, Eq: "5 <= H <= 8"},
		{Min: 9, Max: 14, Glyph: '▒', Name: "half shade", FG: 202, BG: -1, Eq: "9 <= H <= 14"},
		{Min: 15, Max: 24, Glyph: '▄', Name: "half square", FG: 208, BG: 52, Eq: "15 <= H <= 24"},
		{Min: 25, Max: 39, Glyph: '▓', Name: "heavy shade", FG: 220, BG: 166, Eq: "25 <= H <= 39"},
		{Min: 40, Max: 1 << 30, Glyph: '█', Name: "solid bright yellow", FG: 226, BG: 220, Eq: "H >= 40"},
	}
}

// Style maps neighborhood heat onto a flame cell.
func Style(h int) sprite.Cell {
	if h <= 0 {
		return sprite.Cell{Ch: ' ', FG: -1, BG: -1}
	}
	for _, b := range Bands() {
		if h >= b.Min && h <= b.Max {
			return sprite.Cell{Ch: b.Glyph, FG: b.FG, BG: b.BG}
		}
	}
	last := Bands()[len(Bands())-1]
	return sprite.Cell{Ch: last.Glyph, FG: last.FG, BG: last.BG}
}

// Color is Style. Heat, not raw occupancy, is what the sprite uses.
func Color(n int) sprite.Cell { return Style(n) }

// Heat is occupancy of this cell plus every cardinal neighbour except
// the one we came from (opposite the travel direction).
//
//	H(c) = n(c) + n(N) + n(S) + n(E) + n(W) − n(incoming)
func Heat(occ map[particle.Cell]int, at particle.Cell, dir particle.Vec2) int {
	if occ == nil {
		return 0
	}
	idc, idr := incoming(dir)
	h := occ[at]
	for _, n := range []particle.Cell{
		{Col: at.Col, Row: at.Row - 1},
		{Col: at.Col, Row: at.Row + 1},
		{Col: at.Col - 1, Row: at.Row},
		{Col: at.Col + 1, Row: at.Row},
	} {
		if n.Col == at.Col+idc && n.Row == at.Row+idr {
			continue
		}
		h += occ[n]
	}
	return h
}

func incoming(dir particle.Vec2) (dc, dr int) {
	if abs(dir.X) >= abs(dir.Y) {
		if dir.X >= 0 {
			return -1, 0
		}
		return 1, 0
	}
	if dir.Y >= 0 {
		return 0, -1
	}
	return 0, 1
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// Guide is the printed decision table: every style and its equation.
func Guide() string {
	var b strings.Builder
	b.WriteString("H(c) = n(c) + n(north) + n(south) + n(east) + n(west) - n(incoming)\n")
	b.WriteString("incoming = neighbour opposite the travel direction\n")
	b.WriteString("colour climbs toward bright yellow (226/220); never white (231)\n\n")
	b.WriteString("| H | glyph | style | fg | bg | equation |\n")
	b.WriteString("|---|-------|-------|----|----|----------|\n")
	b.WriteString("| H <= 0 |   | empty | -1 | -1 | H <= 0 |\n")
	for _, band := range Bands() {
		max := fmt.Sprintf("%d", band.Max)
		if band.Max > 1000 {
			max = "∞"
		}
		fmt.Fprintf(&b, "| %d..%s | %c | %s | %d | %d | %s |\n",
			band.Min, max, band.Glyph, band.Name, band.FG, band.BG, band.Eq)
	}
	return b.String()
}
