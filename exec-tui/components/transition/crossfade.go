package transition

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Crossfade is the component. Start sizes both layers; Update runs
// the shared clock and both children; Render blends this instant.
type Crossfade struct {
	from, to    screenplay.Component
	delay, over float64
	clock       float64
	w, h        int
	staged      bool
}

// Between binds a crossfade from `from` to `to`. Nothing is built
// until Start. Nil-safe.
func Between(from, to screenplay.Component) *Crossfade {
	return &Crossfade{from: from, to: to}
}

// Delay holds From for seconds before the walk begins. seconds <= 0
// starts the walk on the first tick. Nil-safe.
func (c *Crossfade) Delay(seconds float64) *Crossfade {
	if c == nil {
		return nil
	}
	if seconds > 0 {
		c.delay = seconds
	}
	return c
}

// Over is how long the walk takes. seconds <= 0 snaps to To the
// moment the delay elapses. Nil-safe.
func (c *Crossfade) Over(seconds float64) *Crossfade {
	if c == nil {
		return nil
	}
	c.over = seconds
	return c
}

// Frac is how far the walk has come, 0 on From to 1 on To. Nil and
// unstarted fades sit at 0; a snap fade (Over <= 0) is 0 during the
// delay and 1 after.
func (c *Crossfade) Frac() float64 {
	if c == nil {
		return 0
	}
	if c.clock <= c.delay {
		return 0
	}
	if c.over <= 0 {
		return 1
	}
	p := (c.clock - c.delay) / c.over
	if p >= 1-1e-9 {
		return 1
	}
	if p < 0 {
		return 0
	}
	return p
}

// Start sizes both layers for a w×h stage. The fade clock is not
// touched: a resize never restarts the walk.
func (c *Crossfade) Start(w, h int) {
	if c == nil {
		return
	}
	c.w, c.h = w, h
	c.staged = true
	if c.from != nil {
		c.from.Start(w, h)
	}
	if c.to != nil {
		c.to.Start(w, h)
	}
}

// Update advances the fade and both layers. dt <= 0 holds still.
func (c *Crossfade) Update(dt float64) {
	if c == nil || !c.staged || dt <= 0 {
		return
	}
	c.clock += dt
	if c.from != nil {
		c.from.Update(dt)
	}
	if c.to != nil {
		c.to.Update(dt)
	}
}

// Render blends this instant's cells. Before Start and after Stop
// the stage is empty.
func (c *Crossfade) Render() sprite.Sprite {
	if c == nil || !c.staged || c.w < 1 || c.h < 1 {
		return sprite.Sprite{}
	}
	var from, to sprite.Sprite
	if c.from != nil {
		from = c.from.Render()
	}
	if c.to != nil {
		to = c.to.Render()
	}
	t := c.Frac()
	stage := sprite.New(c.w, c.h)
	for r := 0; r < c.h; r++ {
		for col := 0; col < c.w; col++ {
			stage.Set(r, col, Blend(from.At(r, col), to.At(r, col), t))
		}
	}
	return stage
}

// Stop drops both layers. The clock stays, so the next Start picks
// the walk up mid-fade.
func (c *Crossfade) Stop() {
	if c == nil {
		return
	}
	if c.from != nil {
		c.from.Stop()
	}
	if c.to != nil {
		c.to.Stop()
	}
	c.staged = false
}
