// Package adjuststars is the sky-tuning scene: the whole starfield
// plays behind a small panel of eight numbers — a fly delay and a
// density for each of the four star layers. j/k pick a number, h/l
// change it, and the sky reacts live. The tuner is a screenplay.Scene,
// so the same lifecycle that runs the premiere runs the tool.
package adjuststars

import (
	"fmt"

	"github.com/theprimeagen/apollo-11/stars-lab/stars"

	"github.com/theprimeagen/apollo-11/screenplay-lab/cast"
	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

const (
	// Rows is the eight tunable numbers: delay and density per layer.
	Rows = 8
	// MinDelay/MaxDelay bound a layer's ticks-per-cell: 1 streaks
	// every tick, 30 barely crawls.
	MinDelay = 1
	MaxDelay = 30
	// MinDensity/MaxDensity bound a layer's stars per 1000 cells.
	MinDensity = 1
	MaxDensity = 300
	// StarFPS converts the tuner's seconds into stars.Field ticks.
	StarFPS = 30
)

// layerNames label the four star layers, dim to bright.
var layerNames = [4]string{"dust", "spark", "mid", "near"}

// Tuner is the scene: the eight knobs plus the sky's clock.
type Tuner struct {
	Delays    [4]int
	Densities [4]int
	Cursor    int
	clock     float64
}

// NewTuner builds an unseeded tuner; Start deals the opening values.
func NewTuner() *Tuner {
	return &Tuner{}
}

// Start seeds the knobs: the drift delays and the stock densities.
func (t *Tuner) Start() {
	if t == nil {
		return
	}
	t.Delays = stars.Drift.Delay
	t.Densities = stars.DefaultDensity
	t.Cursor = 0
	t.clock = 0
}

// Update runs the sky's clock. dt <= 0 holds.
func (t *Tuner) Update(dt float64) {
	if t == nil || dt <= 0 {
		return
	}
	t.clock += dt
}

// Stop is the curtain: nothing to free, the knobs keep their values.
func (t *Tuner) Stop() {}

// Move slides the cursor delta rows, clamped to the eight numbers.
func (t *Tuner) Move(delta int) {
	if t == nil {
		return
	}
	t.Cursor = clamp(t.Cursor+delta, 0, Rows-1)
}

// Nudge changes the selected number by delta, clamped to its bounds.
// Even rows are a layer's delay, odd rows its density.
func (t *Tuner) Nudge(delta int) {
	if t == nil {
		return
	}
	kind := t.Cursor / 2
	if t.Cursor%2 == 1 {
		t.Densities[kind] = clamp(t.Densities[kind]+delta, MinDensity, MaxDensity)
		return
	}
	t.Delays[kind] = clamp(t.Delays[kind]+delta, MinDelay, MaxDelay)
}

// Render paints this instant's sky across the whole screen, then the
// panel of eight numbers over its top-left corner.
func (t *Tuner) Render(scr *screenplay.Screen) {
	if t == nil || scr == nil {
		return
	}
	w, h := scr.Size()
	t.field(w, h).Paint(func(row, col int, ch rune, fg int) {
		cast.PutCell(scr, col, row, ch, fg, -1)
	})
	t.panel(scr)
}

// field is the sky the current knobs describe.
func (t *Tuner) field(w, h int) stars.Field {
	return stars.Field{
		Width:    w,
		Height:   h,
		Tick:     int(t.clock * StarFPS),
		Strategy: stars.Strategy{Name: "custom", Delay: t.Delays},
		Density:  t.Densities,
	}
}

// panel is the header and the eight rows, opaque over the sky.
func (t *Tuner) panel(scr *screenplay.Screen) {
	putText(scr, 0, 0, "adjust stars   j/k select  h/l change  q quit ", 245)
	for i := 0; i < Rows; i++ {
		kind := i / 2
		marker, fg := "  ", 250
		if i == t.Cursor {
			marker, fg = "> ", 214
		}
		label, value := "delay  ", t.Delays[kind]
		if i%2 == 1 {
			label, value = "density", t.Densities[kind]
		}
		row := fmt.Sprintf("%s%s %-5s %s %3d ",
			marker, string(stars.Glyphs[kind]), layerNames[kind], label, value)
		putText(scr, 0, 1+i, row, fg)
	}
}

// putText writes a run of cells, spaces included, so the panel always
// occludes the sky beneath it.
func putText(scr *screenplay.Screen, x, y int, text string, fg int) {
	for i, r := range []rune(text) {
		cast.PutCell(scr, x+i, y, r, fg, -1)
	}
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
