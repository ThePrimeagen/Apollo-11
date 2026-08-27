package gunfire

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Blast is the one-shot muzzle flame on an eight-point compass: one
// shared white-hot core and one flame engine per direction, all
// sharing a muzzle, each burning its own tune and colors. Start
// builds all nine for a w×h stage and holds fire; Fire is the
// trigger — the core and every heading's flame burst now, the way
// the flame tuner plays all eight courses at once, and the fuse to
// Doom's second flash frame is lit against the whole rose; Update
// flies every direction and re-reads the active blast each frame so
// an in-process tuner retunes it live; Render paints the whole burn
// onto one stage-sized sprite; Done reports a blast with nothing
// left burning anywhere. A fresh Start rises idle.
type Blast struct {
	Core   *particle.Engine
	Flames [8]*particle.Engine // one per compass point, sprite.Headings order

	seed  int64
	w, h  int
	fired bool
	armed bool    // the second-frame fuse is burning
	fuse  float64 // seconds of fuse left
	aimed int     // -1 whole rose (Fire), >=0 one heading (FireAt)
}

// NewBlast binds a blast to its particle seed. Nothing is built until
// Start.
func NewBlast(seed int64) *Blast {
	return &Blast{seed: seed}
}

// headingIndex is h's slot in sprite.Headings, or -1 off the compass.
func headingIndex(h sprite.Heading) int {
	for i, hh := range sprite.Headings {
		if h == hh {
			return i
		}
	}
	return -1
}

// FlameAt is the flame engine one compass point fires. Nil blasts and
// headings off the compass hand back nothing.
func (b *Blast) FlameAt(h sprite.Heading) *particle.Engine {
	if b == nil {
		return nil
	}
	i := headingIndex(h)
	if i < 0 {
		return nil
	}
	return b.Flames[i]
}

// Start builds the core and all eight flames for a w×h stage, every
// one holding fire.
func (b *Blast) Start(w, h int) {
	if b == nil {
		return
	}
	b.w, b.h = w, h
	uw, uh := b.units()
	core, flames := ActiveBlast().Engines(uw, uh)
	b.Core = particle.New(b.seed, core)
	for i := range flames {
		b.Flames[i] = particle.New(b.seed+1+int64(i), flames[i])
	}
	b.fired, b.armed, b.fuse, b.aimed = false, false, 0, -1
}

func (b *Blast) units() (w, h float64) {
	return float64(b.w)*particle.CellWidthUnits - 0.01,
		float64(b.h)*particle.CellHeightUnits - 0.01
}

func (b *Blast) sync() {
	if b == nil || b.Core == nil {
		return
	}
	uw, uh := b.units()
	core, flames := ActiveBlast().Engines(uw, uh)
	b.Core.Cfg = core
	for i := range b.Flames {
		if b.Flames[i] != nil {
			b.Flames[i].Cfg = flames[i]
		}
	}
}

// Fire is the trigger: the core and every compass heading's flame
// burst at the muzzle right now — the whole rose, the way the flame
// tuner plays all eight courses at once — and, when the config plays
// Doom's second flash frame, the fuse to the dimmer re-pulse is lit
// against every heading. Firing again stacks another shot onto
// whatever is still burning. The trigger needs a stage: before Start
// it is refused, and the report says so.
func (b *Blast) Fire() bool {
	if b == nil || b.Core == nil {
		return false
	}
	b.sync()
	c := ActiveBlast()
	b.aimed = -1
	b.Core.Burst()
	for i := range b.Flames {
		if b.Flames[i] != nil {
			b.Flames[i].Burst()
		}
	}
	if c.PulseDelay > 0 && c.PulseFrac > 0 {
		b.armed, b.fuse = true, c.PulseDelay
	}
	b.fired = true
	return true
}

// FireAt is the shotgun trigger: the core and only heading h's flame
// burst now, using that heading's shot from the active config. Headings
// off the compass and a blast with no stage are refused.
func (b *Blast) FireAt(h sprite.Heading) bool {
	if b == nil || b.Core == nil {
		return false
	}
	i := headingIndex(h)
	if i < 0 {
		return false
	}
	b.sync()
	c := ActiveBlast()
	b.aimed = i
	b.Core.Burst()
	if b.Flames[i] != nil {
		b.Flames[i].Burst()
	}
	if c.PulseDelay > 0 && c.PulseFrac > 0 {
		b.armed, b.fuse = true, c.PulseDelay
	}
	b.fired = true
	return true
}

// pulse is Doom's second flash frame: one dimmer re-burst of the core
// and every heading, each at frac of its full count.
func (b *Blast) pulse(frac float64) {
	engines := []*particle.Engine{b.Core}
	if b.aimed >= 0 && b.aimed < len(b.Flames) {
		engines = append(engines, b.Flames[b.aimed])
	} else {
		for i := range b.Flames {
			engines = append(engines, b.Flames[i])
		}
	}
	for _, e := range engines {
		if e == nil {
			continue
		}
		full := e.Cfg.Count
		e.Cfg.Count = int(float64(full)*frac + 0.5)
		e.Burst()
		e.Cfg.Count = full
	}
}

// Update pulls the active blast onto every engine, burns the whole
// compass dt seconds, and burns the fuse — when it runs out, the
// second frame pulses once against every heading. dt <= 0 holds
// everything, fuse included.
func (b *Blast) Update(dt float64) {
	if b == nil || dt <= 0 || b.Core == nil {
		return
	}
	uw, uh := b.units()
	core, flames := ActiveBlast().Engines(uw, uh)
	b.Core.Cfg = core
	b.Core.Update(dt)
	for i := range b.Flames {
		b.Flames[i].Cfg = flames[i]
		b.Flames[i].Update(dt)
	}
	if b.armed {
		b.fuse -= dt
		if b.fuse <= 0 {
			b.armed = false
			b.pulse(ActiveBlast().PulseFrac)
		}
	}
}

// Done reports a blast that has fully burnt out: the trigger was
// pulled, the fuse is spent, and nothing is left burning on any
// heading. An idle, unstarted, or stopped blast has not performed —
// never done.
func (b *Blast) Done() bool {
	if b == nil || b.Core == nil || !b.fired || b.armed {
		return false
	}
	if len(b.Core.Particles) != 0 {
		return false
	}
	for _, e := range b.Flames {
		if len(e.Particles) != 0 {
			return false
		}
	}
	return true
}

// Render paints the burn onto one stage-sized sprite. Before Start
// and after Stop there is nothing built, so the stage is empty.
func (b *Blast) Render() sprite.Sprite {
	if b == nil || b.Core == nil || b.w < 1 || b.h < 1 {
		return sprite.Sprite{}
	}
	return paint(ActiveBlast(), b.w, b.h, b.Core, b.Flames[:])
}

// Stop drops every engine; a fresh Start rebuilds them idle.
func (b *Blast) Stop() {
	if b == nil {
		return
	}
	b.Core = nil
	for i := range b.Flames {
		b.Flames[i] = nil
	}
}
