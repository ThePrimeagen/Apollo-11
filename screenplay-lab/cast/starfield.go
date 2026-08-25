package cast

import (
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
	"github.com/theprimeagen/apollo-11/stars-lab/stars"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

// StarFPS converts the starfield's seconds into stars.Field ticks.
const StarFPS = 30

// Starfield is the sky. It sizes itself to whatever stage it paints and
// flies with the given strategy. Cast it first: everything else lands
// on top.
type Starfield struct {
	Strategy stars.Strategy
	clock    float64
}

// NewStarfield opens a sky flying in the given style.
func NewStarfield(s stars.Strategy) *Starfield {
	return &Starfield{Strategy: s}
}

// Advance accumulates time. dt <= 0 holds the sky.
func (f *Starfield) Advance(dt float64) {
	if f == nil || dt <= 0 {
		return
	}
	f.clock += dt
}

// Paint fills the stage with this instant's sky.
func (f *Starfield) Paint(st *screenplay.Stage) {
	if f == nil || st == nil {
		return
	}
	w, h := st.Size()
	field := stars.Field{
		Width:    w,
		Height:   h,
		Tick:     int(f.clock * StarFPS),
		Strategy: f.Strategy,
	}
	field.Paint(func(row, col int, ch rune, fg int) {
		st.Put(row, col, sprite.Cell{Ch: ch, FG: fg, BG: -1})
	})
}
