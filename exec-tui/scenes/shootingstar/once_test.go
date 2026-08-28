package shootingstar

// Tests written FIRST: NewOnce is the shooting star as a component —
// one meteor, top mid-right to bottom mid-left, then gone. It does
// not carry a sky of its own: a scene casts it over whatever
// background it already has. After the crossing the star does not
// come back.

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
