package dust

// Tests written FIRST. Cloud is the dust-off component: two mirrored
// swirl engines kicking dust out of a shared floor point — leftward
// and rightward, 15° above horizontal — with a still gap of columns
// between the nozzles where nothing ever lands. It reads the active
// puff every update so an editor can retune it live.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// The cloud must be castable in any screenplay ensemble.
var _ screenplay.Component = (*Cloud)(nil)

// dustGlyph reports a cell painted by the dust ladder.
func dustGlyph(c sprite.Cell) bool {
	return (c.Ch >= '⠀' && c.Ch <= '⣿') || c.Ch == '░' || c.Ch == '▒'
}

// sides scans a stage for dust left of, inside, and right of the
// center strip of columns [lo, hi].
func sides(sp sprite.Sprite, lo, hi int) (left, center, right bool) {
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if !dustGlyph(sp.At(r, c)) {
				continue
			}
			switch {
			case c < lo:
				left = true
			case c > hi:
				right = true
			default:
				center = true
			}
		}
	}
	return left, center, right
}

func TestCloud(t *testing.T) {
	t.Run("happy: Start arms two mirrored engines and the first frame already has dust", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(11)
		cl.Start(80, 24)
		if cl.Left == nil || cl.Right == nil {
			t.Fatal("Start must build both engines")
		}
		gap := ActivePuff().Gap
		if got := cl.Right.Cfg.Origin.X - cl.Left.Cfg.Origin.X; math.Abs(got-gap) > 1e-9 {
			t.Fatalf("nozzles %v apart, want the gap %v", got, gap)
		}
		mid := (cl.Left.Cfg.Origin.X + cl.Right.Cfg.Origin.X) / 2
		if math.Abs(mid-40) > 0.5 {
			t.Fatalf("the gap must sit at stage center, midpoint %v", mid)
		}
		sp := cl.Render()
		if sp.Width != 80 || sp.Height != 24 {
			t.Fatalf("stage %dx%d, want 80x24", sp.Width, sp.Height)
		}
		if l, _, r := sides(sp, 38, 41); !l && !r {
			t.Fatal("a started cloud must already be dusty")
		}
	})
	t.Run("happy: dust blows out both sides and the center gap stays still", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(11)
		cl.Start(80, 24)
		sawLeft, sawRight := false, false
		for i := 0; i < 30; i++ {
			cl.Update(0.1)
			sp := cl.Render()
			l, c, r := sides(sp, 38, 41)
			if c {
				t.Fatalf("frame %d painted dust inside the still gap", i)
			}
			sawLeft = sawLeft || l
			sawRight = sawRight || r
		}
		if !sawLeft || !sawRight {
			t.Fatalf("dust must blow out both sides, left=%v right=%v", sawLeft, sawRight)
		}
	})
	t.Run("happy: the cloud follows the active puff live", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(3)
		cl.Start(60, 20)
		c := ActivePuff()
		c.Count = 5
		if err := UsePuff(c); err != nil {
			t.Fatalf("UsePuff: %v", err)
		}
		cl.Update(0.05)
		if cl.Left.Cfg.Count != 5 || cl.Right.Cfg.Count != 5 {
			t.Fatalf("engines run count %d/%d, want the active 5", cl.Left.Cfg.Count, cl.Right.Cfg.Count)
		}
	})
	t.Run("unhappy: before Start and after Stop the stage is empty", func(t *testing.T) {
		cl := NewCloud(2)
		if sp := cl.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted cloud must render nothing, got %dx%d", sp.Width, sp.Height)
		}
		cl.Start(40, 12)
		cl.Stop()
		if sp := cl.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped cloud must render nothing, got %dx%d", sp.Width, sp.Height)
		}
		if cl.Left != nil || cl.Right != nil {
			t.Fatal("Stop must drop both engines")
		}
	})
	t.Run("unhappy: a nil cloud skips every cue", func(t *testing.T) {
		var ghost *Cloud
		ghost.Start(10, 10)
		ghost.Update(0.1)
		if sp := ghost.Render(); sp.Width != 0 {
			t.Fatal("a nil cloud must render nothing")
		}
		ghost.Stop()
	})
}
