// Package armed is the composite eagle: the bird, a shotgun on each
// talon, and the gunfire particle blast, as one scene component.
// Delay / Cross / Path retune the flight; LeftGun / RightGun retune
// each talon. Rate > 0 fires after the bird is on stage at that
// shots/sec; rate 0 spaces shells evenly along the crossing. New
// reads the Active knobs so a tuner and any scene share the same
// file. The flight clock rides across restarts, so a resize never
// replays the crossing.
package armed

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Armed is eagle + two talon shotguns + gunfire as one Component.
type Armed struct {
	bird        *eagle.Eagle
	left, right *gunner
	w, h        int
	staged      bool
}

// New binds an armed eagle from the Active knobs. Nothing is built
// until Start. Fluent setters override the stock flight and guns.
func New() *Armed {
	c := Active()
	a := &Armed{bird: eagle.New()}
	return a.Delay(c.Delay).Cross(c.Cross).Path(c.Start, c.End).
		LeftGun(c.LeftAim, c.LeftShots, c.LeftRate).
		RightGun(c.RightAim, c.RightShots, c.RightRate)
}

// Delay holds the bird off stage first. Nil-safe.
func (a *Armed) Delay(seconds float64) *Armed {
	if a == nil {
		return nil
	}
	a.bird.Delay(seconds)
	return a
}

// Cross is how long the flyover takes. Nil-safe.
func (a *Armed) Cross(seconds float64) *Armed {
	if a == nil {
		return nil
	}
	a.bird.Cross(seconds)
	return a
}

// Path sets where the flight begins and ends, as fractions of the
// full off-right-to-off-left span. Nil-safe.
func (a *Armed) Path(start, end float64) *Armed {
	if a == nil {
		return nil
	}
	a.bird.Path(start, end)
	return a
}

// LeftGun mounts a shotgun on the leading talon. Nil-safe.
func (a *Armed) LeftGun(aim sprite.Heading, shots int, rate float64) *Armed {
	if a == nil {
		return nil
	}
	a.left = newGunner(a.bird, eagle.Talons()[0], aim, shots, rate, false)
	return a
}

// LeftEven mounts a shotgun on the leading talon that spaces its
// shells evenly along the crossing — the America schedule. Nil-safe.
func (a *Armed) LeftEven(aim sprite.Heading, shots int) *Armed {
	if a == nil {
		return nil
	}
	a.left = newGunner(a.bird, eagle.Talons()[0], aim, shots, 0, true)
	return a
}

// RightGun mounts a shotgun on the trailing talon. Nil-safe.
func (a *Armed) RightGun(aim sprite.Heading, shots int, rate float64) *Armed {
	if a == nil {
		return nil
	}
	a.right = newGunner(a.bird, eagle.Talons()[1], aim, shots, rate, false)
	return a
}

// RightEven mounts a shotgun on the trailing talon that spaces its
// shells evenly along the crossing. Nil-safe.
func (a *Armed) RightEven(aim sprite.Heading, shots int) *Armed {
	if a == nil {
		return nil
	}
	a.right = newGunner(a.bird, eagle.Talons()[1], aim, shots, 0, true)
	return a
}

// Start sizes the bird and both guns for a w×h stage. The flight
// clock and spent shells are not touched: a resize never replays.
func (a *Armed) Start(w, h int) {
	if a == nil {
		return
	}
	a.w, a.h = w, h
	a.staged = true
	a.bird.Start(w, h)
	a.left.Start(w, h)
	a.right.Start(w, h)
}

// Update advances the flight and both guns. dt <= 0 holds still.
func (a *Armed) Update(dt float64) {
	if a == nil || !a.staged || dt <= 0 {
		return
	}
	a.bird.Update(dt)
	a.left.Update(dt)
	a.right.Update(dt)
}

// Render paints the bird, then each mounted gun and its blast, onto
// one stage-sized sprite. Before Start and after Stop the stage is
// empty.
func (a *Armed) Render() sprite.Sprite {
	if a == nil || !a.staged || a.w < 1 || a.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(a.w, a.h)
	sprite.Blit(stage, 0, 0, a.bird.Render())
	sprite.Blit(stage, 0, 0, a.left.Render())
	sprite.Blit(stage, 0, 0, a.right.Render())
	return stage
}

// Stop strikes the guns. The flight clock stays, so the next Start
// picks the crossing up mid-flight.
func (a *Armed) Stop() {
	if a == nil {
		return
	}
	a.left.Stop()
	a.right.Stop()
	a.bird.Stop()
	a.staged = false
}
