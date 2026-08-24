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
		{Min: 1, Max: 5, Glyph: '⠁', Name: "single braille", FG: 88, BG: -1, Eq: "1 <= H <= 5"},
		{Min: 6, Max: 10, Glyph: '⠒', Name: "two-dot braille", FG: 124, BG: -1, Eq: "6 <= H <= 10"},
		{Min: 11, Max: 20, Glyph: '⠶', Name: "four-dot braille", FG: 160, BG: -1, Eq: "11 <= H <= 20"},
		{Min: 21, Max: 40, Glyph: '░', Name: "quarter shade", FG: 166, BG: -1, Eq: "21 <= H <= 40"},
		{Min: 41, Max: 70, Glyph: '▒', Name: "half shade", FG: 202, BG: -1, Eq: "41 <= H <= 70"},
		{Min: 71, Max: 120, Glyph: '▄', Name: "half square", FG: 208, BG: 52, Eq: "71 <= H <= 120"},
		{Min: 121, Max: 195, Glyph: '▓', Name: "heavy shade", FG: 220, BG: 166, Eq: "121 <= H <= 195"},
		{Min: 200, Max: 1 << 30, Glyph: '█', Name: "solid bright yellow", FG: 226, BG: 220, Eq: "H >= 200"},
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
