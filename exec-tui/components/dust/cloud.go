package dust

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// warmSeconds primes the kick so the first frame is already dusty.
	warmSeconds = 1.0
	warmFPS     = 20
)

// Cloud is the dust-off component: two mirrored swirl engines kicking
// dust out of a shared floor point, leftward and rightward. Start
// builds both for a w×h stage; each Update re-reads the active puff so
// an in-process editor can retune the kick live; Render paints both
// engines' dust onto one stage-sized sprite.
type Cloud struct {
	Left, Right *particle.Engine

	seed int64
	w, h int
}

// NewCloud binds a kick to its particle seed. Nothing is built until Start.
func NewCloud(seed int64) *Cloud {
	return &Cloud{seed: seed}
}

// Start builds both engines for a w×h stage and warms them so the
// curtain rises on dust already in the air.
func (c *Cloud) Start(w, h int) {
	if c == nil {
		return
	}
	c.w, c.h = w, h
	uw, uh := c.units()
	left, right := ActivePuff().Engines(uw, uh)
	c.Left = particle.New(c.seed, left)
	c.Right = particle.New(c.seed+1, right)
	dt := 1.0 / float64(warmFPS)
	for t := 0.0; t < warmSeconds; t += dt {
		c.Left.Update(dt)
		c.Right.Update(dt)
	}
}

func (c *Cloud) units() (w, h float64) {
	return float64(c.w)*particle.CellWidthUnits - 0.01,
		float64(c.h)*particle.CellHeightUnits - 0.01
}

// Update pulls the active puff onto both engines and burns them.
// dt <= 0 holds.
func (c *Cloud) Update(dt float64) {
	if c == nil || dt <= 0 || c.Left == nil || c.Right == nil {
		return
	}
	uw, uh := c.units()
	left, right := ActivePuff().Engines(uw, uh)
	c.Left.Cfg = left
	c.Right.Cfg = right
	c.Left.Update(dt)
	c.Right.Update(dt)
}

// Render paints both engines onto one stage-sized sprite. Before Start
// and after Stop there is nothing built, so the stage is empty.
func (c *Cloud) Render() sprite.Sprite {
	if c == nil || c.Left == nil || c.Right == nil || c.w < 1 || c.h < 1 {
		return sprite.Sprite{}
	}
	return paint(ActivePuff(), c.w, c.h, c.Left, c.Right)
}

// Stop drops both engines; a fresh Start rebuilds them.
func (c *Cloud) Stop() {
	if c == nil {
		return
	}
	c.Left, c.Right = nil, nil
}
