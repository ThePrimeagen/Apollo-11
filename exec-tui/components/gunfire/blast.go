package gunfire

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Blast is the one-shot muzzle flame: two quiet particle engines
// sharing a muzzle — the white-hot core and the red flame. Start
// builds them for a w×h stage and holds fire; Fire is the trigger —
// both burst now and the fuse to Doom's second flash frame is lit;
// Update flies the flame and re-reads the active blast each frame so
// an in-process tuner retunes it live; Render paints the burn onto
// one stage-sized sprite; Done reports a flame that has burnt out. A
// fresh Start rises idle — the old shot forgotten.
type Blast struct {
	Core, Flame *particle.Engine

	seed  int64
	w, h  int
	fired bool
	armed bool    // the second-frame fuse is burning
	fuse  float64 // seconds of fuse left
}

// NewBlast binds a blast to its particle seed. Nothing is built until
// Start.
func NewBlast(seed int64) *Blast {
	return &Blast{seed: seed}
}

// Start builds both engines for a w×h stage, holding fire.
func (b *Blast) Start(w, h int) {
	if b == nil {
		return
	}
	b.w, b.h = w, h
	uw, uh := b.units()
	core, flame := ActiveBlast().Engines(uw, uh)
	b.Core = particle.New(b.seed, core)
	b.Flame = particle.New(b.seed+1, flame)
	b.fired, b.armed, b.fuse = false, false, 0
}

func (b *Blast) units() (w, h float64) {
	return float64(b.w)*particle.CellWidthUnits - 0.01,
		float64(b.h)*particle.CellHeightUnits - 0.01
}

// Fire is the trigger: the core and the flame burst at the muzzle
// right now, and — when the config plays Doom's second flash frame —
// the fuse to the dimmer re-pulse is lit. Firing again stacks another
// flame onto whatever is still burning. The trigger needs a stage:
// before Start it is refused, and the report says so.
func (b *Blast) Fire() bool {
	if b == nil || b.Core == nil {
		return false
	}
	b.Core.Burst()
	b.Flame.Burst()
	c := ActiveBlast()
	if c.PulseDelay > 0 && c.PulseFrac > 0 {
		b.armed, b.fuse = true, c.PulseDelay
	}
	b.fired = true
	return true
}

// pulse is Doom's second flash frame: one dimmer re-burst, each layer
// at PulseFrac of its full count.
func (b *Blast) pulse(frac float64) {
	for _, e := range []*particle.Engine{b.Core, b.Flame} {
		full := e.Cfg.Count
		e.Cfg.Count = int(float64(full)*frac + 0.5)
		e.Burst()
		e.Cfg.Count = full
	}
}

// Update pulls the active blast onto the engines, burns the flame dt
// seconds, and burns the fuse — when it runs out, the second frame
// pulses once. dt <= 0 holds everything, fuse included.
func (b *Blast) Update(dt float64) {
	if b == nil || dt <= 0 || b.Core == nil {
		return
	}
	uw, uh := b.units()
	core, flame := ActiveBlast().Engines(uw, uh)
	b.Core.Cfg = core
	b.Flame.Cfg = flame
	b.Core.Update(dt)
	b.Flame.Update(dt)
	if b.armed {
		b.fuse -= dt
		if b.fuse <= 0 {
			b.armed = false
			b.pulse(ActiveBlast().PulseFrac)
		}
	}
}

// Done reports a flame that has fully burnt out: the trigger was
// pulled, the fuse is spent, and nothing is left burning. An idle,
// unstarted, or stopped blast has not performed — never done.
func (b *Blast) Done() bool {
	return b != nil && b.Core != nil && b.fired && !b.armed &&
		len(b.Core.Particles) == 0 && len(b.Flame.Particles) == 0
}

// Render paints the burn onto one stage-sized sprite. Before Start
// and after Stop there is nothing built, so the stage is empty.
func (b *Blast) Render() sprite.Sprite {
	if b == nil || b.Core == nil || b.w < 1 || b.h < 1 {
		return sprite.Sprite{}
	}
	return paint(ActiveBlast(), b.w, b.h, b.Core, b.Flame)
}

// Stop drops both engines; a fresh Start rebuilds them idle.
func (b *Blast) Stop() {
	if b == nil {
		return
	}
	b.Core, b.Flame = nil, nil
}
