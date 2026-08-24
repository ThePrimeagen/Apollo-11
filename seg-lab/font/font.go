// Package font draws a string at a height unit.
//
//	font.Render("HELLO WORLD", 1) // terminal default font (1 row)
//	font.Render("HELLO WORLD", 3) // constructed 14-seg, 3 rows
//	font.Render("HELLO WORLD", 5) // constructed 14-seg, 5 rows
//
// Height is the number of terminal rows. Height 1 is the string as-is.
// Height 2 cannot hold a 14-seg letter (needs a top, mid, and bottom)
// and returns ErrHeight. Heights 3–5 stamp the Segmented Alpha
// outlines onto a character grid. Anything else also returns ErrHeight.
package font

import (
	"errors"
	"math"
	"strings"
	"unicode"
)

// ErrHeight is returned when height is not 1, 3, 4, or 5.
var ErrHeight = errors.New("font: height must be 1, 3, 4, or 5")

// heightCell is the per-letter cell (width × rows) for constructed
// units. The row count is the height unit. Width is greater than
// rows so terminal cells (taller than wide) do not squash the letter.
var heightCell = map[int][2]int{
	3: {5, 3},
	4: {7, 4},
	5: {7, 5},
}

// GlyphSize is the per-letter cell (width × rows) for a height unit.
// Height 1 is one terminal cell — the default font.
func GlyphSize(height int) (w, rows int, err error) {
	if height == 1 {
		return 1, 1, nil
	}
	c, ok := heightCell[height]
	if !ok {
		return 0, 0, ErrHeight
	}
	return c[0], c[1], nil
}

// 14-segment bits. Same map as the TTF.
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

// Same frame as SegmentedAlpha.ttf (1000 em, Cascadia-width advance).
const (
	segL   = 70.0
	segR   = 516.0
	segT   = 770.0
	segB   = -50.0
	segMY  = 360.0
	segCX  = (segL + segR) / 2
	ttfTH  = 52.0
	ttfGap = 12.0
)

type point struct{ x, y float64 }

// Render draws text at height 1, 3, 4, or 5. Height 1 is the
// terminal default font. Heights 3–5 are constructed 14-seg of
// that many rows. Height 2 and anything else return ErrHeight.
// Empty text yields "".
func Render(text string, height int) (string, error) {
	w, h, err := GlyphSize(height)
	if err != nil {
		return "", err
	}
	if height == 1 {
		return text, nil
	}
	rs := []rune(text)
	if len(rs) == 0 {
		return "", nil
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
	return strings.Join(lines, "\n"), nil
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
	// Two pixel rows per display row → ▀▄█. Thickness is scaled up from
	// the TTF's 52/1000 hairline so a 5–9 row grid still reads as a bar.
	//
	// Crop to the letter frame, not the full em: the TTF's empty
	// descender/ascender padding was showing up as a ghost ▀ row.
	pixW, pixH := w, h*2
	th := (segT - segB) * 0.16
	g := ttfGap * (th / ttfTH)
	pad := th/2 + 10
	x0, x1 := segL-pad, segR+pad
	y0, y1 := segB-pad, segT+pad
	polys := segmentPolys(bits, th, g)

	on := make([][]bool, pixH)
	for py := 0; py < pixH; py++ {
		on[py] = make([]bool, pixW)
		for px := 0; px < pixW; px++ {
			hits := 0
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					x := x0 + (float64(px)+0.25+0.5*float64(sx))/float64(pixW)*(x1-x0)
					y := y1 - (float64(py)+0.25+0.5*float64(sy))/float64(pixH)*(y1-y0)
					if insideAny(x, y, polys) {
						hits++
					}
				}
			}
			on[py][px] = hits >= 2
		}
	}

	out := make([]string, h)
	for row := 0; row < h; row++ {
		buf := make([]rune, w)
		for col := 0; col < w; col++ {
			top, bot := on[row*2][col], on[row*2+1][col]
			switch {
			case top && bot:
				buf[col] = '█'
			case top:
				buf[col] = '▀'
			case bot:
				buf[col] = '▄'
			default:
				buf[col] = ' '
			}
		}
		out[row] = string(buf)
	}
	return out
}

func segmentPolys(bits uint16, th, g float64) [][]point {
	var out [][]point
	add := func(x1, y1, x2, y2 float64) {
		if p := thickLine(x1, y1, x2, y2, th); len(p) == 4 {
			out = append(out, p)
		}
	}
	on := func(seg uint16) bool { return bits&seg != 0 }
	if on(sA) {
		add(segL+g, segT, segR-g, segT)
	}
	if on(sD) {
		add(segL+g, segB, segR-g, segB)
	}
	if on(sG1) {
		add(segL+g, segMY, segCX-g, segMY)
	}
	if on(sG2) {
		add(segCX+g, segMY, segR-g, segMY)
	}
	if on(sF) {
		add(segL, segT-g, segL, segMY+g)
	}
	if on(sE) {
		add(segL, segMY-g, segL, segB+g)
	}
	if on(sB) {
		add(segR, segT-g, segR, segMY+g)
	}
	if on(sC) {
		add(segR, segMY-g, segR, segB+g)
	}
	if on(sI) {
		add(segCX, segT-g, segCX, segMY+g)
	}
	if on(sL) {
		add(segCX, segMY-g, segCX, segB+g)
	}
	if on(sH) {
		add(segL+g*2, segT-g*2, segCX-g, segMY+g)
	}
	if on(sJ) {
		add(segR-g*2, segT-g*2, segCX+g, segMY+g)
	}
	if on(sK) {
		add(segL+g*2, segB+g*2, segCX-g, segMY-g)
	}
	if on(sM) {
		add(segR-g*2, segB+g*2, segCX+g, segMY-g)
	}
	return out
}

func thickLine(x1, y1, x2, y2, w float64) []point {
	dx, dy := x2-x1, y2-y1
	length := math.Hypot(dx, dy)
	if length == 0 {
		return nil
	}
	nx, ny := -dy/length*w/2, dx/length*w/2
	return []point{
		{x1 + nx, y1 + ny},
		{x2 + nx, y2 + ny},
		{x2 - nx, y2 - ny},
		{x1 - nx, y1 - ny},
	}
}

func insideAny(x, y float64, polys [][]point) bool {
	for _, poly := range polys {
		if inside(point{x, y}, poly) {
			return true
		}
	}
	return false
}

func inside(p point, poly []point) bool {
	n := len(poly)
	if n < 3 {
		return false
	}
	pos, neg := 0, 0
	for i := 0; i < n; i++ {
		a, b := poly[i], poly[(i+1)%n]
		cross := (b.x-a.x)*(p.y-a.y) - (b.y-a.y)*(p.x-a.x)
		switch {
		case cross > 1e-9:
			pos++
		case cross < -1e-9:
			neg++
		}
	}
	return pos == 0 || neg == 0
}
