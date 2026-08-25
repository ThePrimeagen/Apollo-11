package stars

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

// StarFPS converts the starfield's seconds into sky ticks.
const StarFPS = 30

// Starfield is the sky as a scene component. Start scatters and caches
// the star catalog for its stage — a tuned sky samples the active
// config (stars.UseSky / the adjuststars file) right there — Update
// runs the fly clock, Render paints the cached catalog into a
// stage-sized sprite, and Stop deletes the catalog so a stopped sky
// holds no allocation. Start may come again: the sky re-scatters and
// flies on. Cast it first: everything else lands on top.
type Starfield struct {
	Strategy Strategy
	Tuned    bool
	clock    float64
	fly      Strategy
	density  [4]int
	cat      *Catalog
	w, h     int
}

// NewStarfield opens a sky flying in the given style.
func NewStarfield(s Strategy) *Starfield {
	return &Starfield{Strategy: s}
}

// NewTunedStarfield opens a sky that samples the active sky settings
// when it starts, so a tuned config file just works in any scene.
func NewTunedStarfield() *Starfield {
	return &Starfield{Tuned: true}
}

// Start scatters the catalog for a w×h stage. A tuned sky reads the
// active config here — the settings it opens with are the settings it
// flies until the next Start.
func (f *Starfield) Start(w, h int) {
	if f == nil {
		return
	}
	f.w, f.h = w, h
	f.fly = f.Strategy
	f.density = [4]int{}
	if f.Tuned {
		sky := ActiveSky()
		f.fly = sky.FlyStrategy()
		f.density = sky.DensityLayers()
	}
	f.cat = NewCatalog(w, h, f.density)
}

// Update accumulates time. dt <= 0 holds the sky.
func (f *Starfield) Update(dt float64) {
	if f == nil || dt <= 0 {
		return
	}
	f.clock += dt
}

// Render paints the cached catalog into a stage-sized sprite. Before
// Start and after Stop there is no catalog, so the stage is empty.
func (f *Starfield) Render() sprite.Sprite {
	if f == nil || f.cat == nil || f.w < 1 || f.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(f.w, f.h)
	f.cat.Paint(int(f.clock*StarFPS), f.fly, func(row, col int, ch rune, fg int) {
		stage.Set(row, col, sprite.Cell{Ch: ch, FG: fg, BG: -1})
	})
	return stage
}

// Stop deletes the catalog for the collector; a fresh Start rebuilds
// it. The clock carries on, so a restage never rewinds the sky.
func (f *Starfield) Stop() {
	if f == nil {
		return
	}
	f.cat = nil
}
