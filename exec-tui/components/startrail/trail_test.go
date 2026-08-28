package startrail

// Tests written FIRST: Trail is the persist-particle comet wake. It
// emits at Origin on each period, the specks stay where they were
// born, and they fade as life runs out. Follow moves the nozzle so a
// moving star leaves a trail behind it. The engine is ModePersist —
// not pool scatter, not flying exhaust. Paint is the fading sparkle
// (✦ * ˚ ·) by remaining life. NewOrbit is the tuner/viewer preview:
// the nozzle walks a circle so the tail shape is readable.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 48
	stageH = 24
)

var _ screenplay.Component = (*Trail)(nil)
var _ screenplay.Component = (*Orbit)(nil)

func sparkCount(sp sprite.Sprite) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if !sp.At(r, c).Transparent() {
				n++
			}
		}
	}
	return n
}

func TestTrailComponent(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: Start arms a persist engine and the first Update drops specks at the origin", func(t *testing.T) {
		tr := New(7)
		tr.Follow(particle.Vec2{X: 20, Y: 20}, particle.Vec2{X: 1, Y: 0})
		tr.Start(stageW, stageH)
		defer tr.Stop()
		if tr.Eng == nil {
			t.Fatal("Start must arm the engine")
		}
		if tr.Eng.Cfg.Mode != particle.ModePersist {
			t.Fatalf("mode %v, want ModePersist — the trail parks specks", tr.Eng.Cfg.Mode)
		}
		tr.Update(0.02)
		if len(tr.Eng.Particles) == 0 {
			t.Fatal("the first period must drop specks")
		}
		for i, p := range tr.Eng.Particles {
			if p.Vel != (particle.Vec2{}) {
				t.Fatalf("trail speck %d flies %+v", i, p.Vel)
			}
		}
		if sparkCount(tr.Render()) == 0 {
			t.Fatal("a live trail must paint")
		}
	})
	t.Run("happy: Follow moves the nozzle and the old specks stay put", func(t *testing.T) {
		tr := New(3)
		tr.Follow(particle.Vec2{X: 10, Y: 16}, particle.Vec2{X: 1, Y: 0})
		tr.Start(stageW, stageH)
		defer tr.Stop()
		tr.Update(0.02)
		if len(tr.Eng.Particles) == 0 {
			t.Fatal("need a first drop")
		}
		first := append([]particle.Particle(nil), tr.Eng.Particles...)
		tr.Follow(particle.Vec2{X: 30, Y: 16}, particle.Vec2{X: 1, Y: 0})
		tr.Update(0.05)
		if len(tr.Eng.Particles) <= len(first) {
			t.Fatal("the second drop must add specks")
		}
		for i, p := range first {
			if tr.Eng.Particles[i].Pos != p.Pos {
				t.Fatalf("speck %d drifted %+v -> %+v — persist specks live where they were created", i, p.Pos, tr.Eng.Particles[i].Pos)
			}
		}
	})
	t.Run("happy: specks die when life runs out and the paint thins", func(t *testing.T) {
		t.Cleanup(Reset)
		c := DefaultConfig()
		c.MinLife, c.MaxLife = 0.1, 0.15
		c.Period = 0.05
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		tr := New(5)
		tr.Follow(particle.Vec2{X: 24, Y: 20}, particle.Vec2{X: 0, Y: -1})
		tr.Start(stageW, stageH)
		defer tr.Stop()
		tr.Update(0.02)
		if sparkCount(tr.Render()) == 0 {
			t.Fatal("mid-life the trail must paint")
		}
		tr.Follow(particle.Vec2{X: 24, Y: 20}, particle.Vec2{X: 0, Y: -1})
		c.Period = 0
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		tr.Update(2)
		if len(tr.Eng.Particles) != 0 {
			t.Fatalf("after 2s the trail must have died, still %d specks", len(tr.Eng.Particles))
		}
		if sparkCount(tr.Render()) != 0 {
			t.Fatal("a dead trail must paint nothing")
		}
	})
	t.Run("unhappy: before Start and after Stop there is nothing, and dt<=0 holds", func(t *testing.T) {
		tr := New(1)
		if sparkCount(tr.Render()) != 0 {
			t.Fatal("before Start the trail is empty")
		}
		tr.Start(stageW, stageH)
		tr.Update(0.05)
		tr.Stop()
		if tr.Eng != nil {
			t.Fatal("Stop must drop the engine")
		}
		if sparkCount(tr.Render()) != 0 {
			t.Fatal("after Stop the trail is empty")
		}
		held := New(2)
		held.Follow(particle.Vec2{X: 12, Y: 12}, particle.Vec2{X: 1, Y: 0})
		held.Start(stageW, stageH)
		defer held.Stop()
		held.Update(0.02)
		n := len(held.Eng.Particles)
		held.Update(0)
		held.Update(-3)
		if len(held.Eng.Particles) != n {
			t.Fatal("dt<=0 must not emit")
		}
	})
}

func TestOrbit(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: an orbiting trail walks a circle and leaves a wake", func(t *testing.T) {
		o := NewOrbit(11)
		o.Start(stageW, stageH)
		defer o.Stop()
		o.Update(0.02)
		first := o.Trail.Origin
		o.Update(0.4)
		if o.Trail.Origin == first {
			t.Fatal("the orbit must move the nozzle")
		}
		d0 := math.Hypot(first.X-o.cx, first.Y-o.cy)
		d1 := math.Hypot(o.Trail.Origin.X-o.cx, o.Trail.Origin.Y-o.cy)
		if math.Abs(d0-d1) > 1 {
			t.Fatalf("orbit radius drifted %.2f -> %.2f", d0, d1)
		}
		if sparkCount(o.Render()) == 0 {
			t.Fatal("the circling trail must paint a wake")
		}
	})
	t.Run("unhappy: a stopped orbit paints nothing", func(t *testing.T) {
		o := NewOrbit(4)
		if sparkCount(o.Render()) != 0 {
			t.Fatal("before Start the orbit is empty")
		}
		o.Start(stageW, stageH)
		o.Stop()
		if sparkCount(o.Render()) != 0 {
			t.Fatal("after Stop the orbit is empty")
		}
	})
}
