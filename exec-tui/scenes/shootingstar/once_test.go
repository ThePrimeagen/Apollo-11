package shootingstar

// Tests written FIRST: NewOnce is the shooting star as a component —
// one meteor, top mid-right to bottom mid-left, then gone. It does
// not carry a sky of its own: a scene casts it over whatever
// background it already has. After the crossing the star does not
// come back. NewOnceWith is the same flyer on a scene's own knobs, so
// every scene the star appears in can tune its copy, and Retune swaps
// the knobs mid-flight for live editing.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/bigstar"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

var _ screenplay.Component = (*Flyer)(nil)

func flyerCore(sp sprite.Sprite) (x, y int, ok bool) {
	for y = 0; y < sp.Height; y++ {
		for x = 0; x < sp.Width; x++ {
			if sp.At(y, x).Ch == bigstar.CoreGlyph {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func TestOnceFlyer(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: one meteor enters top mid-right and falls toward bottom mid-left", func(t *testing.T) {
		f := NewOnce()
		if f == nil {
			t.Fatal("NewOnce must hand back a flyer")
		}
		f.Start(stageW, stageH)
		defer f.Stop()
		x0, y0, ok := flyerCore(f.Render())
		if !ok {
			t.Fatal("the once-meteor must open with the larger star already on stage")
		}
		if x0 < stageW/2 {
			t.Fatalf("the once-meteor must enter from the right, star at col %d", x0)
		}
		if y0 > stageH/2 {
			t.Fatalf("the once-meteor must enter from the top, star at row %d", y0)
		}
		if f.show == nil || f.show.cross.Start.X <= f.show.cross.End.X {
			t.Fatalf("the once-meteor must run right-to-left, %+v", f.show)
		}
		if f.show.cross.Start.Y >= f.show.cross.End.Y {
			t.Fatalf("the once-meteor must fall downward, %+v", f.show.cross)
		}
		const dt = 1.0 / 30
		for i := 0; i < 12; i++ {
			f.Update(dt)
		}
		x1, y1, ok := flyerCore(f.Render())
		if !ok {
			t.Fatal("the once-meteor must still be on stage a few frames in")
		}
		if x1 >= x0 {
			t.Fatalf("the once-meteor must travel right-to-left, col %d → %d", x0, x1)
		}
		if y1 < y0 {
			t.Fatalf("the once-meteor must fall, row %d → %d", y0, y1)
		}
	})
	t.Run("unhappy: after the crossing the star is gone and a second meteor never appears", func(t *testing.T) {
		f := NewOnce()
		f.Start(stageW, stageH)
		defer f.Stop()
		_ = f.Render()
		const dt = 1.0 / 30
		var seen bool
		for i := 0; i < 30; i++ {
			f.Update(dt)
			if _, _, ok := flyerCore(f.Render()); ok {
				seen = true
			}
		}
		if !seen {
			t.Fatal("test premise: the star must fly at least once")
		}
		for i := 0; i < 180; i++ {
			f.Update(dt)
		}
		if _, _, ok := flyerCore(f.Render()); ok {
			t.Fatal("after the crossing the star must leave the stage")
		}
		for i := 0; i < 90; i++ {
			f.Update(dt)
			if _, _, ok := flyerCore(f.Render()); ok {
				t.Fatal("a second meteor must not appear — NewOnce shoots once")
			}
		}
	})
}

func TestNewOnceWith(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: the flyer flies the given knobs, not the package active", func(t *testing.T) {
		tuned := DefaultConfig()
		tuned.Speed = 200
		if err := Use(tuned); err != nil {
			t.Fatal(err)
		}
		slow := DefaultConfig()
		slow.Speed = 4
		slow.Size = 3
		f := NewOnceWith(slow)
		if f == nil {
			t.Fatal("NewOnceWith must hand back a flyer")
		}
		f.Start(stageW, stageH)
		defer f.Stop()
		if f.show == nil || f.show.Cfg != slow {
			t.Fatalf("the flyer carries %+v, want the given knobs %+v", f.show.Cfg, slow)
		}
		if f.star == nil || f.star.Size != 3 {
			t.Fatal("the given size must reach the star")
		}
		const dt = 1.0 / 30
		for i := 0; i < 60; i++ {
			f.Update(dt)
		}
		if _, _, ok := flyerCore(f.Render()); !ok {
			t.Fatal("at speed 4 the star must still be crossing — the active knobs must not leak in")
		}
		fast := DefaultConfig()
		fast.Speed = 400
		g := NewOnceWith(fast)
		g.Start(stageW, stageH)
		defer g.Stop()
		for i := 0; i < 30; i++ {
			g.Update(dt)
		}
		if _, _, ok := flyerCore(g.Render()); ok {
			t.Fatal("at speed 400 the one crossing must already be over")
		}
	})
	t.Run("happy: NewOnce is NewOnceWith on the active knobs", func(t *testing.T) {
		Reset()
		tuned := DefaultConfig()
		tuned.Size = 3
		if err := Use(tuned); err != nil {
			t.Fatal(err)
		}
		f := NewOnce()
		if f.show == nil || f.show.Cfg != tuned {
			t.Fatalf("NewOnce carries %+v, want the active knobs %+v", f.show.Cfg, tuned)
		}
	})
	t.Run("unhappy: a zero-value config parks the star without a panic", func(t *testing.T) {
		f := NewOnceWith(Config{})
		f.Start(stageW, stageH)
		defer f.Stop()
		const dt = 1.0 / 30
		for i := 0; i < 30; i++ {
			f.Update(dt)
		}
		_ = f.Render()
	})
}

func TestFlyerRetune(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: a retune mid-flight swaps the knobs the flyer reads", func(t *testing.T) {
		slow := DefaultConfig()
		slow.Speed = 2
		f := NewOnceWith(slow)
		f.Start(stageW, stageH)
		defer f.Stop()
		const dt = 1.0 / 30
		for i := 0; i < 15; i++ {
			f.Update(dt)
		}
		if _, _, ok := flyerCore(f.Render()); !ok {
			t.Fatal("test premise: a slow star is still crossing")
		}
		fast := DefaultConfig()
		fast.Speed = 400
		f.Retune(fast)
		if f.show.Cfg != fast {
			t.Fatalf("after Retune the flyer carries %+v, want %+v", f.show.Cfg, fast)
		}
		for i := 0; i < 30; i++ {
			f.Update(dt)
		}
		if _, _, ok := flyerCore(f.Render()); ok {
			t.Fatal("retuned to speed 400 the crossing must already be over")
		}
	})
	t.Run("unhappy: a nil flyer and a flyer without a show hold still, no panic", func(t *testing.T) {
		var ghost *Flyer
		ghost.Retune(DefaultConfig())
		bare := &Flyer{}
		bare.Retune(DefaultConfig())
		f := NewOnceWith(DefaultConfig())
		f.Retune(DefaultConfig())
	})
}
