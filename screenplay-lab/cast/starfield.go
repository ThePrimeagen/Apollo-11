package cast

import (
	"github.com/theprimeagen/apollo-11/stars-lab/stars"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

// StarFPS converts the starfield's seconds into stars.Field ticks.
const StarFPS = 30

// Starfield is the sky. It sizes itself to whatever screen it renders
// to and flies with the given strategy — or, when tuned, with whatever
// sky settings are active (stars.UseSky / the adjuststars config).
// Cast it first: everything else lands on top.
type Starfield struct {
	Strategy stars.Strategy
	Tuned    bool
	clock    float64
}

// NewStarfield opens a sky flying in the given style.
func NewStarfield(s stars.Strategy) *Starfield {
	return &Starfield{Strategy: s}
}

// NewTunedStarfield opens a sky that follows the active sky settings
// on every frame, so a tuned config file just works in any scene.
func NewTunedStarfield() *Starfield {
	return &Starfield{Tuned: true}
}

// Update accumulates time. dt <= 0 holds the sky.
func (f *Starfield) Update(dt float64) {
	if f == nil || dt <= 0 {
		return
	}
	f.clock += dt
}

// Render fills the screen with this instant's sky.
func (f *Starfield) Render(scr *screenplay.Screen) {
	if f == nil || scr == nil {
		return
	}
	w, h := scr.Size()
	field := stars.Field{
		Width:    w,
		Height:   h,
		Tick:     int(f.clock * StarFPS),
		Strategy: f.Strategy,
	}
	if f.Tuned {
		sky := stars.ActiveSky()
		field.Strategy = sky.FlyStrategy()
		field.Density = sky.DensityLayers()
	}
	field.Paint(func(row, col int, ch rune, fg int) {
		PutCell(scr, col, row, ch, fg, -1)
	})
}
