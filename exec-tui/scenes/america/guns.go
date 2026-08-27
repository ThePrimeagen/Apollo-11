package america

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/shotgun"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// gunner is one talon's armament as a scene component: a shotgun
// component painted on top of the claw, riding the eagle across the
// stage, firing the gunfire particle blast a fixed number of times —
// evenly spaced along the flight — with its barrel on one compass
// point. The gun and the flame are the reusable components
// (components/shotgun, components/gunfire); the gunner only mounts
// them on the bird. Spent shells stay spent across a resize, the same
// way the flight clock rides across restarts.
type gunner struct {
	bird  *eagle.Eagle
	gun   *shotgun.Gun
	aim   sprite.Heading
	shots int
	talon [2]int
	fired int
	w, h  int
}

// newGunner mounts a shotgun on one talon of the bird, aimed at h,
// with shots shells to spend across the crossing.
func newGunner(bird *eagle.Eagle, talon [2]int, h sprite.Heading, shots int) *gunner {
	return &gunner{bird: bird, gun: shotgun.New(), aim: h, shots: shots, talon: talon}
}

// Start builds the gun and its blast for a w×h stage. The spent-shell
// count is not touched: a resize never re-fires the crossing so far.
func (g *gunner) Start(w, h int) {
	if g == nil {
		return
	}
	g.w, g.h = w, h
	g.gun.Start(w, h)
	g.gun.Aim(g.aim)
}

// mount is where the gun sits this instant: the top-left stage cell
// of the gun frame centered on the talon, and whether the bird is on
// stage to carry it.
func (g *gunner) mount() (x, y int, on bool) {
	bx, by, on := g.bird.At()
	if !on {
		return 0, 0, false
	}
	body := g.gun.Frame(g.gun.Heading())
	return bx + g.talon[0] - body.Width/2, by + g.talon[1] - body.Height/2, true
}

// Update burns the flame and spends any shells whose moment has come:
// shell i fires when the flight passes (i+0.5)/shots of its path, so
// the shots spread evenly across the crossing. dt <= 0 holds.
func (g *gunner) Update(dt float64) {
	if g == nil || dt <= 0 {
		return
	}
	g.gun.Update(dt)
	if g.shots < 1 {
		return
	}
	p, flying := g.bird.Progress()
	if !flying {
		return
	}
	for g.fired < g.shots && p >= (float64(g.fired)+0.5)/float64(g.shots) {
		g.fire()
		g.fired++
	}
}

// fire pulls this talon's trigger: one shot from the mounted gun's
// barrel tip on this gun's own blast — the shotgun component owns the
// squeeze, the gunner only says when.
func (g *gunner) fire() {
	if gx, gy, on := g.mount(); on {
		_ = g.gun.FireFrom(gx, gy)
	}
}

// Render paints the mounted gun over the talon, then whatever flame
// is still in the air, onto a stage-sized sprite. When the bird is
// off stage the gun goes with it; the flame burns out on its own.
func (g *gunner) Render() sprite.Sprite {
	if g == nil || g.w < 1 || g.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(g.w, g.h)
	if gx, gy, on := g.mount(); on {
		sprite.Blit(stage, gx, gy, g.gun.Frame(g.gun.Heading()))
	}
	if g.gun.Blast != nil {
		sprite.Blit(stage, 0, 0, g.gun.Blast.Render())
	}
	return stage
}

// Stop strikes the gun and its blast; the spent shells stay spent.
func (g *gunner) Stop() {
	if g == nil {
		return
	}
	g.gun.Stop()
}
