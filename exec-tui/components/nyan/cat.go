package nyan

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// FlySpeed is cells per second across the stage.
	FlySpeed = 14
	// BobHz is how fast the pop-tart bounces; BobAmp is ±rows.
	BobHz  = 1.2
	BobAmp = 1
	// warmSeconds primes the plume so the first frame already has a trail.
	warmSeconds = 0.6
	warmFPS     = 20
)

// Cat is the nyan component: pop-tart hull plus a rainbow particle
// trail from the shared engine. Start builds both for a w×h stage;
// Stop drops them. Each Update pulls ActiveTrail() onto the engine
// so an in-process editor can retune the plume live.
type Cat struct {
	Park  bool
	Body  sprite.Sprite
	Eng   *particle.Engine
	seed  int64
	clock float64
	w, h  int
}

// NewCat binds a flying cat to its particle seed. Nothing is built
// until Start.
func NewCat(seed int64) *Cat {
	return &Cat{seed: seed}
}

// NewParked binds a cat that bobbles in place — the editor's preview.
func NewParked(seed int64) *Cat {
	return &Cat{seed: seed, Park: true}
}

// Start builds the hull and arms a fresh rainbow trail for a w×h
// stage. The clock carries across restarts so a resize never replays
// the fly-across.
func (c *Cat) Start(w, h int) {
	if c == nil {
		return
	}
	c.w, c.h = w, h
	c.Body = body()
	c.Eng = particle.New(c.seed, ActiveTrail().ParticleConfig())
	dt := 1.0 / float64(warmFPS)
	for t := 0.0; t < warmSeconds; t += dt {
		c.Eng.Update(dt)
	}
}

// Update moves the clock, applies the active trail knobs, and burns
// the particle engine. dt <= 0 holds.
func (c *Cat) Update(dt float64) {
	if c == nil || dt <= 0 {
		return
	}
	c.clock += dt
	if c.Eng != nil {
		c.Eng.Cfg = ActiveTrail().ParticleConfig()
		c.Eng.Update(dt)
	}
}

// Render composes trail first, hull second, into a stage-sized sprite
// so the pastry always wins the overlap at the nozzle. Before Start
// and after Stop there is nothing built, so the stage is empty.
func (c *Cat) Render() sprite.Sprite {
	if c == nil || c.Eng == nil || c.w < 1 || c.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(c.w, c.h)
	trail := paintTrail(c.Eng, ActiveTrail().BandWidth, c.clock)
	bodyRow, bodyCol := c.flight()
	oc := particle.CellOf(c.Eng.Cfg.Origin.X, c.Eng.Cfg.Origin.Y)
	trailCol := bodyCol + tartCol - oc.Col
	trailRow := bodyRow + tartRow - oc.Row
	sprite.Blit(stage, trailCol, trailRow, trail)
	sprite.Blit(stage, bodyCol, bodyRow, c.Body)
	return stage
}

// Stop drops the hull and the engine; a fresh Start rebuilds both.
func (c *Cat) Stop() {
	if c == nil {
		return
	}
	c.Body = sprite.Sprite{}
	c.Eng = nil
}

func (c *Cat) flight() (row, col int) {
	row = (c.h - BodyRows) / 2
	if row < 0 {
		row = 0
	}
	bob := int(math.Round(BobAmp * math.Sin(c.clock*2*math.Pi*BobHz)))
	row += bob
	if c.Park {
		col = (c.w - BodyCols) / 2
		if col < 8 {
			col = 8
		}
		return row, col
	}
	span := c.w + BodyCols + 16
	if span < 1 {
		span = 1
	}
	col = int(c.clock*FlySpeed)%span - 4
	return row, col
}
