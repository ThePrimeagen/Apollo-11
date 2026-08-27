package armed

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/shotgun"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// gunner is one talon's armament inside the composite: a shotgun
// painted on the claw, riding the eagle. Rate > 0 fires after the
// bird is on stage at that shots/sec (first shell waits one
// interval). Rate == 0 spaces shells evenly along the flight.
type gunner struct {
	bird  *eagle.Eagle
	gun   *shotgun.Gun
	aim   sprite.Heading
	shots int
	rate  float64
	even  bool
	talon [2]int
	fired int
	air   float64
	w, h  int
}

func newGunner(bird *eagle.Eagle, talon [2]int, h sprite.Heading, shots int, rate float64, even bool) *gunner {
	return &gunner{bird: bird, gun: shotgun.New(), aim: h, shots: shots, rate: rate, even: even, talon: talon}
}

func (g *gunner) Start(w, h int) {
	if g == nil {
		return
	}
	g.w, g.h = w, h
	g.gun.Start(w, h)
	g.gun.Aim(g.aim)
}

func (g *gunner) mount() (x, y int, on bool) {
	bx, by, on := g.bird.At()
	if !on {
		return 0, 0, false
	}
	body := g.gun.Frame(g.gun.Heading())
	return bx + g.talon[0] - body.Width/2, by + g.talon[1] - body.Height/2, true
}

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
	if g.even {
		for g.fired < g.shots && p >= (float64(g.fired)+0.5)/float64(g.shots) {
			g.fire()
			g.fired++
		}
		return
	}
	if g.rate <= 0 {
		return
	}
	g.air += dt
	interval := 1.0 / g.rate
	for g.fired < g.shots && g.air >= interval*float64(g.fired+1) {
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

func (g *gunner) Stop() {
	if g == nil {
		return
	}
	g.gun.Stop()
}
