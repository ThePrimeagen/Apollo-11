// Package flag is the full-screened American flag as a scene
// component: thirteen stripes, the blue canton over the top seven,
// and all fifty stars, painted across every cell of the stage. The
// field is laid out on half rows, so the stripes are mathematically
// even at any stage height: each stripe spans 2h/13 half-rows give or
// take one, and a boundary that falls inside a cell paints an
// upper-half block over the lower stripe's background. The canton
// bottom lands exactly on the seventh stripe boundary, and only fully
// blue rows carry stars. The flag fades in from pure black — Start
// fixes the layout, and every frame each cell walks its own color
// ramp from black toward its finished ink, reaching full color at
// FadeSeconds. Only the colors move; the field itself holds perfectly
// still. The fade clock rides across restarts, so a resize repaints
// the layout without ever falling back to black.
package flag

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// The flag's xterm-256 inks. Black is where every fade begins;
// RedInk, WhiteInk and BlueInk are the finished stripes and canton;
// StarInk is the brighter white the fifty stars pop with.
const (
	Black    = 16
	RedInk   = 160
	WhiteInk = 255
	BlueInk  = 18
	StarInk  = 231
)

// StarGlyph is one star of the fifty.
const StarGlyph = '★'

// Stripes is the thirteen colonies; CantonStripes is how many of them
// the blue field covers.
const (
	Stripes       = 13
	CantonStripes = 7
)

// The fade ramps: each material walks black → finished ink, one
// brightness rung per fraction of the fade. Ramps end on the inks the
// finished flag wears.
var (
	redRamp   = []int{Black, 52, 88, 124, RedInk}
	whiteRamp = []int{Black, 233, 236, 239, 242, 245, 248, 252, WhiteInk}
	blueRamp  = []int{Black, 17, BlueInk}
	starRamp  = []int{Black, 233, 236, 239, 242, 245, 248, 252, StarInk}
)

// CantonCols is how many columns the blue field spans on a w-wide
// stage: two fifths of the fly, the union's traditional share.
func CantonCols(w int) int {
	if w < 1 {
		return 0
	}
	return int(math.Round(2 * float64(w) / 5))
}

// cantonHalfRows is how deep the blue field runs in half-rows: the
// top seven of the thirteen stripes, exactly.
func cantonHalfRows(h int) int {
	if h < 1 {
		return 0
	}
	return (2*CantonStripes*h + Stripes - 1) / Stripes
}

// CantonRows is how many whole rows of the stage are fully blue —
// the rows that can carry stars. A canton that ends mid-cell keeps
// its split row for the stripes.
func CantonRows(h int) int {
	return cantonHalfRows(h) / 2
}

// The cell materials the layout is made of.
const (
	kindRed = iota
	kindWhite
	kindBlue
)

// cellKind is one cell of the layout: the material of each half-row,
// and whether a star rides the cell.
type cellKind struct {
	up, low uint8
	star    bool
}

// Flag is the component. Start lays the field out for its stage,
// Update runs the fade clock, Render paints the layout at this
// instant's brightness, Stop drops the layout for the collector.
type Flag struct {
	fade   float64
	clock  float64
	kinds  [][]cellKind
	w, h   int
	staged bool
}

// New binds a flag that fades in from black over fadeSeconds. A fade
// of zero or less opens at full color. Nothing is built until Start.
func New(fadeSeconds float64) *Flag {
	return &Flag{fade: fadeSeconds}
}

// Start lays the stripes, canton and stars out for a w×h stage. The
// fade clock is not touched: a resize never restarts from black.
func (f *Flag) Start(w, h int) {
	if f == nil {
		return
	}
	f.w, f.h = w, h
	f.kinds = layout(w, h)
	f.staged = true
}

// Update advances the fade. dt <= 0 holds — time never runs backwards.
func (f *Flag) Update(dt float64) {
	if f == nil || dt <= 0 {
		return
	}
	f.clock += dt
}

// Render paints every cell of the stage at the fade's current
// brightness. Before Start and after Stop the stage is empty.
func (f *Flag) Render() sprite.Sprite {
	if f == nil || !f.staged || f.w < 1 || f.h < 1 {
		return sprite.Sprite{}
	}
	frac := 1.0
	if f.fade > 0 {
		frac = f.clock / f.fade
		if frac > 1 {
			frac = 1
		}
		if frac < 0 {
			frac = 0
		}
	}
	stage := sprite.New(f.w, f.h)
	for r := 0; r < f.h; r++ {
		for c := 0; c < f.w; c++ {
			stage.Set(r, c, cellFor(f.kinds[r][c], frac))
		}
	}
	return stage
}

// Stop drops the layout. The fade clock stays, so the next Start
// carries on from the same brightness.
func (f *Flag) Stop() {
	if f == nil {
		return
	}
	f.kinds = nil
	f.staged = false
}

// shade picks the ramp rung for a fade fraction: 0 is black, 1 is the
// finished ink.
func shade(ramp []int, frac float64) int {
	return ramp[int(math.Round(frac*float64(len(ramp)-1)))]
}

// rampFor is the fade ramp a material walks.
func rampFor(kind uint8) []int {
	switch kind {
	case kindWhite:
		return whiteRamp
	case kindBlue:
		return blueRamp
	default:
		return redRamp
	}
}

// cellFor is one cell of the flag at a fade fraction. Halves that
// fade to the same color collapse into a plain field cell; a stripe
// boundary inside the cell paints the upper half over the lower
// half's background.
func cellFor(k cellKind, frac float64) sprite.Cell {
	if k.star {
		return sprite.Cell{Ch: StarGlyph, FG: shade(starRamp, frac), BG: shade(blueRamp, frac)}
	}
	up := shade(rampFor(k.up), frac)
	low := shade(rampFor(k.low), frac)
	if up == low {
		return sprite.Cell{Ch: ' ', FG: -1, BG: up}
	}
	return sprite.Cell{Ch: '▀', FG: up, BG: low}
}

// halfStripe is the stripe one half-row belongs to: half-rows divide
// 13 ways as evenly as integer math allows, so no stripe is ever more
// than one half-row taller than another.
func halfStripe(hr, h int) int {
	return hr * Stripes / (2 * h)
}

// fieldKind is the material of one half-row at one column: canton
// blue over the union's corner — cut exactly at the seventh stripe
// boundary — then red and white stripes, red first.
func fieldKind(hr, c, h, cw int) uint8 {
	if c < cw && hr*Stripes < 2*CantonStripes*h {
		return kindBlue
	}
	if halfStripe(hr, h)%2 == 0 {
		return kindRed
	}
	return kindWhite
}

// layout fixes which materials every cell of a w×h stage wears, half
// a row at a time, and settles the fifty stars onto the fully blue
// rows on their staggered grid.
func layout(w, h int) [][]cellKind {
	kinds := make([][]cellKind, h)
	cw := CantonCols(w)
	for r := 0; r < h; r++ {
		kinds[r] = make([]cellKind, w)
		for c := 0; c < w; c++ {
			kinds[r][c] = cellKind{
				up:  fieldKind(2*r, c, h, cw),
				low: fieldKind(2*r+1, c, h, cw),
			}
		}
	}
	cb := CantonRows(h)
	for _, rc := range starCells(cw, cb) {
		if rc[0] >= 0 && rc[0] < cb && rc[1] >= 0 && rc[1] < cw {
			kinds[rc[0]][rc[1]].star = true
		}
	}
	return kinds
}

// starCells is the fifty-star stagger inside the canton's cw columns
// and its ch fully blue rows: nine rows alternating six and five, the
// odd rows offset half a step. On a canton too small to keep them
// apart, stars land on shared cells and simply merge.
func starCells(cw, ch int) [][2]int {
	var out [][2]int
	if cw < 1 || ch < 1 {
		return out
	}
	for i := 0; i < 9; i++ {
		row := int(math.Round(float64(ch) * float64(i+1) / 10))
		cols := []int{1, 3, 5, 7, 9, 11}
		if i%2 == 1 {
			cols = []int{2, 4, 6, 8, 10}
		}
		for _, k := range cols {
			col := int(math.Round(float64(cw) * float64(k) / 12))
			out = append(out, [2]int{row, col})
		}
	}
	return out
}
