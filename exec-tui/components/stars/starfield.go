package stars

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

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
	Strategy  Strategy
	Tuned     bool
	clock     float64
	fly       Strategy
	density   [4]int
	cat       *Catalog
	w, h      int
	dockMin   int
	dockSec   float64
	slideSec  float64
	slideBody int
	slideHold float64
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

// Dock asks the sky to blank the right third of the stage — at least
// minCols, or one third if that is wider — one column at a time over
// seconds. Call before Start. Nil-safe.
func (f *Starfield) Dock(minCols int, seconds float64) *Starfield {
	if f == nil {
		return nil
	}
	f.dockMin = minCols
	f.dockSec = seconds
	return f
}

// SlideIn translates every star uniformly over seconds to match a
// westbound craft of bodyCols sliding in from the right wing to center
// stage. The offset uses the same ease-out cubic the ship flies, holds
// after the park, and stacks on top of the sky's own fly strategy.
// Call before Start. Nil-safe.
func (f *Starfield) SlideIn(seconds float64, bodyCols int) *Starfield {
	if f == nil {
		return nil
	}
	f.slideSec = seconds
	f.slideBody = bodyCols
	return f
}

// Hold waits seconds of the sky's own fly before the SlideIn
// translation starts. Call before Start. Nil-safe.
func (f *Starfield) Hold(seconds float64) *Starfield {
	if f == nil {
		return nil
	}
	f.slideHold = seconds
	return f
}

// SlideOffset is how many columns left the sky has translated at time
// t of a seconds-long slide on a width-wide stage for a bodyCols-wide
// craft. Zero before the slide or when the stage cannot move.
func SlideOffset(width, bodyCols int, t, seconds float64) int {
	if width < 1 || bodyCols < 1 || seconds <= 0 {
		return 0
	}
	park := (width - bodyCols) / 2
	dist := width - park
	if dist < 1 {
		return 0
	}
	if t <= 0 {
		return 0
	}
	if t+1e-9 >= seconds {
		return dist
	}
	p := t / seconds
	eased := 1 - math.Pow(1-p, 3)
	return int(math.Round(eased * float64(dist)))
}

// DockCols is how many columns from the right edge a dock occupies on
// a width-wide stage: the larger of one third and minCols, clipped to
// the stage.
func DockCols(width, minCols int) int {
	if width < 1 {
		return 0
	}
	n := width / 3
	if minCols > n {
		n = minCols
	}
	if n > width {
		n = width
	}
	return n
}

// wipeCols is how many of total columns have gone dark after t seconds
// of a duration-seconds wipe. Columns pop from the right, one at a time.
func wipeCols(total int, t, seconds float64) int {
	if total < 1 {
		return 0
	}
	if seconds <= 0 || t+1e-9 >= seconds {
		return total
	}
	if t <= 0 {
		return 0
	}
	n := int(float64(total) * t / seconds)
	if n > total {
		return total
	}
	return n
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
	cutoff := f.w
	if f.dockMin > 0 || f.dockSec > 0 {
		cutoff = f.w - wipeCols(DockCols(f.w, f.dockMin), f.clock, f.dockSec)
	}
	shift := SlideOffset(f.w, f.slideBody, f.clock-f.slideHold, f.slideSec)
	f.cat.Paint(int(f.clock*StarFPS), f.fly, func(row, col int, ch rune, fg int) {
		col = wrap(col-shift, f.w)
		if col >= cutoff {
			return
		}
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
