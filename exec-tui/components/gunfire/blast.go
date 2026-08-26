package gunfire

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Blast is the one-shot shotgun component: four quiet particle
// engines sharing a muzzle. Start builds them for a w×h stage and
// holds fire; Fire is the trigger — flash, pellets, and sparks burst
// now and the smoke's fuse starts burning; Update flies the shot and
// re-reads the active blast each frame so an in-process tuner retunes
// it live; Render paints the whole shot onto one stage-sized sprite;
// Done reports a shot that has fully played out. A fresh Start rises
// idle — the old shot forgotten.
type Blast struct {
	Flash, Pellets, Sparks, Smoke *particle.Engine

	seed  int64
	w, h  int
	fired bool
	armed bool    // a smoke fuse is burning
	fuse  float64 // seconds of fuse left
}

// NewBlast binds a blast to its particle seed. Nothing is built until
// Start.
func NewBlast(seed int64) *Blast {
	return &Blast{seed: seed}
}

// Start builds the four engines for a w×h stage, all holding fire.
func (b *Blast) Start(w, h int) {
	if b == nil {
		return
	}
	b.w, b.h = w, h
	uw, uh := b.units()
	flash, pellets, sparks, smoke := ActiveBlast().Engines(uw, uh)
	b.Flash = particle.New(b.seed, flash)
	b.Pellets = particle.New(b.seed+1, pellets)
	b.Sparks = particle.New(b.seed+2, sparks)
	b.Smoke = particle.New(b.seed+3, smoke)
	b.fired, b.armed, b.fuse = false, false, 0
}

func (b *Blast) units() (w, h float64) {
	return float64(b.w)*particle.CellWidthUnits - 0.01,
		float64(b.h)*particle.CellHeightUnits - 0.01
}

// Fire is the trigger: the flash, the pellets, and the sparks burst
// at the muzzle right now, and the smoke's fuse is lit — it curls out
// SmokeDelay seconds later (immediately on a zero fuse). Firing again
// stacks another volley onto whatever is still flying. The trigger
// needs a stage: before Start it is refused, and the report says so.
func (b *Blast) Fire() bool {
	if b == nil || b.Flash == nil {
		return false
	}
	b.Flash.Burst()
	b.Pellets.Burst()
	b.Sparks.Burst()
	if delay := ActiveBlast().SmokeDelay; delay > 0 {
		b.armed, b.fuse = true, delay
	} else {
		b.Smoke.Burst()
	}
	b.fired = true
	return true
}

// Update pulls the active blast onto the engines, flies the shot dt
// seconds, and burns the smoke fuse — when it runs out, the smoke
// curls out once. dt <= 0 holds everything, fuse included.
func (b *Blast) Update(dt float64) {
	if b == nil || dt <= 0 || b.Flash == nil {
		return
	}
	uw, uh := b.units()
	flash, pellets, sparks, smoke := ActiveBlast().Engines(uw, uh)
	b.Flash.Cfg = flash
	b.Pellets.Cfg = pellets
	b.Sparks.Cfg = sparks
	b.Smoke.Cfg = smoke
	b.Flash.Update(dt)
	b.Pellets.Update(dt)
	b.Sparks.Update(dt)
	b.Smoke.Update(dt)
	if b.armed {
		b.fuse -= dt
		if b.fuse <= 0 {
			b.armed = false
			b.Smoke.Burst()
		}
	}
}

// Done reports a shot that has fully played out: the trigger was
// pulled, the fuse is burnt, and nothing is left flying. An idle,
// unstarted, or stopped blast has not performed — never done.
func (b *Blast) Done() bool {
	return b != nil && b.Flash != nil && b.fired && !b.armed &&
		len(b.Flash.Particles) == 0 && len(b.Pellets.Particles) == 0 &&
		len(b.Sparks.Particles) == 0 && len(b.Smoke.Particles) == 0
}

// Render paints the shot onto one stage-sized sprite. Before Start
// and after Stop there is nothing built, so the stage is empty.
func (b *Blast) Render() sprite.Sprite {
	if b == nil || b.Flash == nil || b.w < 1 || b.h < 1 {
		return sprite.Sprite{}
	}
	return paint(ActiveBlast(), b.w, b.h, b.Flash, b.Pellets, b.Sparks, b.Smoke)
}

// Stop drops the engines; a fresh Start rebuilds them idle.
func (b *Blast) Stop() {
	if b == nil {
		return
	}
	b.Flash, b.Pellets, b.Sparks, b.Smoke = nil, nil, nil, nil
}
