// Package adjuststars is the sky-tuning scene: the whole starfield
// plays behind a small panel of eight numbers — a fly delay and a
// density for each of the four star layers. j/k pick a number, h/l
// change it, and the sky reacts live. The tuner is a screenplay.Scene,
// so the same lifecycle that runs the premiere runs the tool.
package adjuststars

import (
	"github.com/theprimeagen/apollo-11/stars-lab/stars"

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
)

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
}

// Update runs the sky's clock. dt <= 0 holds.
func (t *Tuner) Update(dt float64) {
}

// Render paints this instant's sky across the whole screen, then the
// panel of eight numbers over its top-left corner.
func (t *Tuner) Render(scr *screenplay.Screen) {
}

// Stop is the curtain: nothing to free, the knobs keep their values.
func (t *Tuner) Stop() {
}

// Move slides the cursor delta rows, clamped to the eight numbers.
func (t *Tuner) Move(delta int) {
}

// Nudge changes the selected number by delta, clamped to its bounds.
func (t *Tuner) Nudge(delta int) {
}

// field is the sky the current knobs describe.
func (t *Tuner) field(w, h int) stars.Field {
	return stars.Field{}
}
