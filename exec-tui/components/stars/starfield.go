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
	still     bool
	dockMin   int
	dockSec   float64
	slideSec  float64
	slideBody int
	slideHold float64
	slowBy    float64
	slowSec   float64
	seed      *Continuity
	base      Continuity
	adopted   bool
}

// Continuity is one show's sky odometer: how many seconds of fly and
// how many columns of translation the audience has watched so far.
// Seed every scene's starfield with the same Continuity and each new
// sky opens on the exact frame the last one left on screen — no jump,
// no skip at the cut. The catalog scatter is already deterministic per
// stage, so the clock and the shift are the only state a cut loses.
type Continuity struct {
	clock float64
	shift int
}

// NewContinuity is a fresh odometer at zero: the first seeded sky
// opens at its own beginning.
func NewContinuity() *Continuity {
	return &Continuity{}
}

// Seed hands the sky the show's continuity: on its first Start it
// adopts the clock and translation recorded there, and every tick it
// records its own totals back, ready for the next scene's sky. A nil
// continuity leaves the sky unseeded. Call before Start. Nil-safe.
func (f *Starfield) Seed(c *Continuity) *Starfield {
	if f == nil {
		return nil
	}
	f.seed = c
	return f
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

// Still parks the sky: whatever the strategy or the tuned config say,
// the fly clock freezes — an unseeded sky holds the home it scattered
// to, a seeded one holds the exact frame the seed carried in. Call
// before Start. Nil-safe.
func (f *Starfield) Still() *Starfield {
	if f == nil {
		return nil
	}
	f.still = true
	return f
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

// Slow eases the sky's fly speed down by `by` over `seconds`.
// by=0.6 over 5s means the stars finish 60% slower (40% of their
// opening speed) and hold that crawl. A still sky stays still.
// Call before Start. Nil-safe.
func (f *Starfield) Slow(by, seconds float64) *Starfield {
	if f == nil {
		return nil
	}
	f.slowBy = by
	f.slowSec = seconds
	return f
}

// BrakeClock is how many seconds of fly the sky has actually burned
// after t seconds of a brake that cuts speed by `by` over `seconds`.
// Speed goes from 1 to (1-by) linearly; after the window it holds
// the reduced speed. by=0.6 over 5s → 3.5s of fly at the window,
// then 40% speed from there.
func BrakeClock(t, by, seconds float64) float64 {
	if t < 0 {
		t = 0
	}
	if by < 0 {
		by = 0
	}
	if by > 1 {
		by = 1
	}
	if seconds <= 0 {
		return t
	}
	if t <= seconds {
		return t - by*t*t/(2*seconds)
	}
	return seconds*(1-by/2) + (1-by)*(t-seconds)
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
// flies until the next Start. A seeded sky adopts the continuity on
// its first Start only, so a resize restage never double-counts it.
func (f *Starfield) Start(w, h int) {
	if f == nil {
		return
	}
	f.w, f.h = w, h
	if f.seed != nil && !f.adopted {
		f.base = *f.seed
		f.adopted = true
	}
	f.fly = f.Strategy
	f.density = [4]int{}
	if f.Tuned {
		sky := ActiveSky()
		f.fly = sky.FlyStrategy()
		f.density = sky.DensityLayers()
	}
	f.cat = NewCatalog(w, h, f.density)
}

// Update accumulates time and records the sky's running totals into
// the continuity, so the next scene's sky can pick up exactly here.
// dt <= 0 holds the sky.
func (f *Starfield) Update(dt float64) {
	if f == nil || dt <= 0 {
		return
	}
	f.clock += dt
	if f.seed != nil && f.adopted {
		f.seed.clock, f.seed.shift = f.totals()
	}
}

// totals is the sky the audience sees this instant: the adopted base
// plus everything this sky has done itself — the (brake-eased) fly
// clock and the slide translation. A Still sky burns no fly of its
// own: it holds the frame the base describes.
func (f *Starfield) totals() (fly float64, shift int) {
	fly = f.base.clock
	if !f.still {
		local := f.clock
		if f.slowSec > 0 {
			local = BrakeClock(f.clock, f.slowBy, f.slowSec)
		}
		fly += local
	}
	shift = f.base.shift + SlideOffset(f.w, f.slideBody, f.clock-f.slideHold, f.slideSec)
	return fly, shift
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
	fly, shift := f.totals()
	f.cat.Paint(int(fly*StarFPS), f.fly, func(row, col int, ch rune, fg int) {
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
