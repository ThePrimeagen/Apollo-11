package shotgun

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Size is the atlas slot the eight shotgun frames live in.
const Size = sprite.Size1

// Palette is the gun's materials. fg == bg so a pixel is one solid
// color whether it lands in a cell's top half, bottom half, or both.
var Palette = []sprite.PaletteEntry{
	{ID: ".", Name: "empty", FG: -1, BG: -1},
	{ID: "W", Name: "wood", FG: 94, BG: 94},
	{ID: "S", Name: "steel", FG: 250, BG: 250},
	{ID: "D", Name: "dark", FG: 238, BG: 238},
	{ID: "B", Name: "barrel", FG: 245, BG: 245},
	{ID: "G", Name: "gold", FG: 178, BG: 178},
	{ID: "P", Name: "pump", FG: 137, BG: 137},
}

// east is the one 2D asset: a side-on stock-and-barrel gun pointing
// +X, drawn one art row per terminal cell. Every row is 32 pixels.
// The right half of the compass is this grid spun in the screen plane
// around the Y-axis coming out of it (east 0°, counterclockwise); the
// left half (W, NW, SW) mirrors the right half so the sights always
// face up.
var east = []string{
	"................................",
	"........SSSSBBBBBBBBBBBBBBBBBB..",
	"...WWWWWSSSSBBBBBBBBBBBBBBBBBBB.",
	"..WWWWWWSSGGPP..................",
	"..WWWWWWDSS.....................",
	"...WWWWWG.......................",
	"....WWW.........................",
	"................................",
}

// dup doubles every row: a terminal cell is two stacked square
// pixels, so one cell-height art row becomes two square-pixel rows.
// The spin happens in this square-pixel space — that is what keeps
// the gun the same length on screen whichever way it points.
func dup(rows ...string) []string {
	out := make([]string, 0, len(rows)*2)
	for _, row := range rows {
		out = append(out, row, row)
	}
	return out
}

// headingDeg is the counterclockwise angle, in degrees, that spins
// the east gun onto a right-half heading around the Y-axis coming out
// of the screen. The left half of the compass is never spun — it is
// mirrored — so those headings (and anything off the compass) are
// east (0°).
func headingDeg(h sprite.Heading) float64 {
	switch h {
	case sprite.E:
		return 0
	case sprite.NE:
		return 45
	case sprite.N:
		return 90
	case sprite.S:
		return 270
	case sprite.SE:
		return 315
	}
	return 0
}

// headingGrid is the square-pixel grid for one compass heading: E,
// NE, N, SE and S are the east gun spun in the screen plane; W, NW
// and SW are the E, NE and SE grids mirrored left-right, because a
// spin past vertical would hang the gun upside down and the sights
// must always face up.
func headingGrid(h sprite.Heading) []string {
	switch h {
	case sprite.W:
		return mirrorGrid(headingGrid(sprite.E))
	case sprite.NW:
		return mirrorGrid(headingGrid(sprite.NE))
	case sprite.SW:
		return mirrorGrid(headingGrid(sprite.SE))
	}
	return rotateGrid(dup(east...), headingDeg(h))
}

// mirrorGrid flips a pixel grid left-right.
func mirrorGrid(rows []string) []string {
	out := make([]string, len(rows))
	for i, row := range rows {
		r := []rune(row)
		for a, b := 0, len(r)-1; a < b; a, b = a+1, b-1 {
			r[a], r[b] = r[b], r[a]
		}
		out[i] = string(r)
	}
	return out
}

// padEven tops an odd-height grid up with one transparent row so the
// half-block compile always has a bottom pixel to pair with.
func padEven(rows []string) []string {
	if len(rows) == 0 || len(rows)%2 == 0 {
		return rows
	}
	return append(rows, strings.Repeat(".", utf8.RuneCountInString(rows[0])))
}

// rotateGrid spins a 2D pixel grid counterclockwise by deg degrees
// around its canvas centre — the Y-axis coming out of the screen.
// 0/90/180/270 stay on exact pixel centres so a 180° spin is FlipH+FlipV.
// An empty or nil grid stays empty.
func rotateGrid(rows []string, deg float64) []string {
	if len(rows) == 0 {
		return nil
	}
	h := len(rows)
	w := utf8.RuneCountInString(rows[0])
	if w == 0 {
		return nil
	}
	src := make([][]rune, h)
	for r, row := range rows {
		src[r] = []rune(row)
		if len(src[r]) != w {
			return nil
		}
	}
	for deg < 0 {
		deg += 360
	}
	deg = math.Mod(deg, 360)

	var dst [][]rune
	switch deg {
	case 0:
		dst = cloneRunes(src)
	case 90:
		dst = make([][]rune, w)
		for r := 0; r < w; r++ {
			dst[r] = make([]rune, h)
			for c := 0; c < h; c++ {
				dst[r][c] = src[c][w-1-r]
			}
		}
	case 180:
		dst = make([][]rune, h)
		for r := 0; r < h; r++ {
			dst[r] = make([]rune, w)
			for c := 0; c < w; c++ {
				dst[r][c] = src[h-1-r][w-1-c]
			}
		}
	case 270:
		dst = make([][]rune, w)
		for r := 0; r < w; r++ {
			dst[r] = make([]rune, h)
			for c := 0; c < h; c++ {
				dst[r][c] = src[h-1-c][r]
			}
		}
	default:
		dst = rotateNearest(src, deg)
	}
	out := make([]string, len(dst))
	for i, row := range dst {
		out[i] = string(row)
	}
	return out
}

func cloneRunes(src [][]rune) [][]rune {
	out := make([][]rune, len(src))
	for i, row := range src {
		out[i] = append([]rune(nil), row...)
	}
	return out
}

func rotateNearest(src [][]rune, deg float64) [][]rune {
	h := len(src)
	w := len(src[0])
	rad := deg * math.Pi / 180
	cos, sin := math.Cos(rad), math.Sin(rad)
	cx := float64(w-1) / 2
	cy := float64(h-1) / 2
	rot := func(c, r float64) (float64, float64) {
		dc, dr := c-cx, r-cy
		return cx + dc*cos + dr*sin, cy - dc*sin + dr*cos
	}
	minC, minR := math.MaxFloat64, math.MaxFloat64
	maxC, maxR := -math.MaxFloat64, -math.MaxFloat64
	for _, c := range []float64{0, float64(w - 1)} {
		for _, r := range []float64{0, float64(h - 1)} {
			cc, rr := rot(c, r)
			if cc < minC {
				minC = cc
			}
			if cc > maxC {
				maxC = cc
			}
			if rr < minR {
				minR = rr
			}
			if rr > maxR {
				maxR = rr
			}
		}
	}
	outW := int(math.Floor(maxC-minC)) + 1
	outH := int(math.Floor(maxR-minR)) + 1
	if outW < 1 {
		outW = 1
	}
	if outH < 1 {
		outH = 1
	}
	dst := make([][]rune, outH)
	for r := 0; r < outH; r++ {
		dst[r] = make([]rune, outW)
		for c := 0; c < outW; c++ {
			dst[r][c] = '.'
		}
	}
	// Inverse: dest → source, nearest neighbour.
	invCos, invSin := math.Cos(-rad), math.Sin(-rad)
	inv := func(c, r float64) (float64, float64) {
		dc, dr := c-cx, r-cy
		return cx + dc*invCos + dr*invSin, cy - dc*invSin + dr*invCos
	}
	for r := 0; r < outH; r++ {
		for c := 0; c < outW; c++ {
			sc, sr := inv(float64(c)+minC, float64(r)+minR)
			ic := int(math.Round(sc))
			ir := int(math.Round(sr))
			if ir >= 0 && ir < h && ic >= 0 && ic < w {
				dst[r][c] = src[ir][ic]
			}
		}
	}
	return dst
}
