package armed

// Tests written FIRST: Armed is the composite component — the eagle,
// a shotgun on each talon, and the gunfire particle blast — as one
// performer. Scenes no longer wire three siblings; they hang one
// Armed on the cast. Delay / Cross / Path retune the flight; LeftGun
// / RightGun retune each talon (aim, shell count, rate). Rate > 0
// fires after the bird is on stage at that shots/sec (first shell
// waits one interval). Rate == 0 spaces shells evenly along the
// flight, the America schedule. Zero shells (or a zero rate on the
// rate schedule) stay mounted and never flame. Before Start and
// after Stop the stage is empty; dt <= 0 never moves the clock.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 27
	gunInk = 178
)

var _ screenplay.Component = (*Armed)(nil)

func tick(a *Armed, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		a.Update(dt)
	}
}

func cellsWith(sp sprite.Sprite, inks ...int) [][2]int {
	want := map[int]bool{}
	for _, n := range inks {
		want[n] = true
	}
	var out [][2]int
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			cell := sp.At(r, c)
			if want[cell.FG] || want[cell.BG] {
				out = append(out, [2]int{r, c})
			}
		}
	}
	return out
}

func leftmost(cells [][2]int) int {
	l := 1 << 30
	for _, rc := range cells {
		if rc[1] < l {
			l = rc[1]
		}
	}
	return l
}

func blastCells(sp sprite.Sprite) [][2]int {
	return cellsWith(sp, 226, 208, 196)
}

func TestArmedComposite(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: mid-crossing the same sprite holds the eagle and both talon guns", func(t *testing.T) {
		a := New().Delay(0.2).Cross(2).
			LeftGun(sprite.W, 1, 0).RightGun(sprite.E, 1, 0)
		a.Start(stageW, stageH)
		defer a.Stop()
		_ = a.Render()
		tick(a, 0.2+0.5)
		sp := a.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		if got := cellsWith(sp, eagle.SignatureInks()...); len(got) < 100 {
			t.Fatalf("only %d eagle cells — the bird must be on this sprite", len(got))
		}
		if got := cellsWith(sp, gunInk); len(got) == 0 {
			t.Fatal("the talon guns must be painted on the same sprite as the bird")
		}
	})
	t.Run("happy: the guns ride the bird leftward as one component", func(t *testing.T) {
		a := New().Delay(0.2).Cross(2).
			LeftGun(sprite.W, 0, 0).RightGun(sprite.E, 0, 0)
		a.Start(stageW, stageH)
		defer a.Stop()
		_ = a.Render()
		tick(a, 0.2+0.5)
		l1 := leftmost(cellsWith(a.Render(), gunInk))
		tick(a, 0.5)
		l2 := leftmost(cellsWith(a.Render(), gunInk))
		if l2 >= l1 {
			t.Fatalf("the guns must ride leftward: leftmost went %d -> %d", l1, l2)
		}
	})
	t.Run("unhappy: before Start and after Stop the stage is empty", func(t *testing.T) {
		a := New().LeftGun(sprite.W, 1, 2)
		if sp := a.Render(); sp.Width != 0 {
			t.Fatalf("before Start the composite is %dx%d, want empty", sp.Width, sp.Height)
		}
		a.Start(stageW, stageH)
		a.Stop()
		if sp := a.Render(); sp.Width != 0 {
			t.Fatalf("after Stop the composite is %dx%d, want empty", sp.Width, sp.Height)
		}
	})
}

func TestArmedRateFire(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: the bird flies in, then the guns fire on their own rate", func(t *testing.T) {
		a := New().Delay(0.2).Cross(2).
			LeftGun(sprite.W, 2, 2).RightGun(sprite.E, 0, 0)
		a.Start(stageW, stageH)
		defer a.Stop()
		_ = a.Render()
		tick(a, 0.2+0.2)
		if got := blastCells(a.Render()); len(got) != 0 {
			t.Fatalf("just after the bird enters the sky holds %d flame cells — the guns wait for their rate", len(got))
		}
		if len(cellsWith(a.Render(), eagle.SignatureInks()...)) == 0 {
			t.Fatal("test premise: the bird must already be on stage")
		}
		tick(a, 0.4)
		if got := blastCells(a.Render()); len(got) == 0 {
			t.Fatal("past 1/rate of air time the muzzle flame must be in the air")
		}
	})
	t.Run("unhappy: zero shells or a zero rate is a silent flyover — mounted guns, no flame", func(t *testing.T) {
		a := New().Delay(0.2).Cross(2).
			LeftGun(sprite.W, 0, 4).RightGun(sprite.E, 3, 0)
		a.Start(stageW, stageH)
		defer a.Stop()
		_ = a.Render()
		tick(a, 0.2+0.5)
		if len(cellsWith(a.Render(), gunInk)) == 0 {
			t.Fatal("a silent flyover must still mount the guns")
		}
		at := 0.7
		for target := 0.8; target <= 2.4; target += 0.4 {
			tick(a, target-at)
			at = target
			if got := blastCells(a.Render()); len(got) != 0 {
				t.Fatalf("at %.1fs a silent gun threw %d flame cells", target, len(got))
			}
		}
	})
	t.Run("unhappy: before the delay the armed bird is fully off stage — guns too", func(t *testing.T) {
		a := New().Delay(1.5).Cross(2).
			LeftGun(sprite.W, 1, 4).RightGun(sprite.E, 1, 4)
		a.Start(stageW, stageH)
		defer a.Stop()
		_ = a.Render()
		tick(a, 1.0)
		if got := cellsWith(a.Render(), gunInk); len(got) != 0 {
			t.Fatalf("before the delay %d gun cells are on stage — the guns wait with the bird", len(got))
		}
	})
}

func TestArmedProgressFire(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: rate zero spaces shells evenly along the crossing", func(t *testing.T) {
		a := New().Delay(0.2).Cross(2).
			LeftGun(sprite.W, 2, 0).RightGun(sprite.E, 2, 0)
		a.Start(stageW, stageH)
		defer a.Stop()
		_ = a.Render()
		tick(a, 0.2+0.3)
		if got := blastCells(a.Render()); len(got) != 0 {
			t.Fatalf("before the first scheduled shot the sky holds %d flame cells", len(got))
		}
		tick(a, 0.3)
		if got := blastCells(a.Render()); len(got) == 0 {
			t.Fatal("past the first scheduled shot the muzzle flame must be in the air")
		}
	})
	t.Run("unhappy: a nil Armed never panics", func(t *testing.T) {
		var a *Armed
		a.Delay(1).Cross(1).Path(0, 1).LeftGun(sprite.W, 1, 1).RightGun(sprite.E, 1, 1)
		a.Start(8, 8)
		a.Update(1)
		_ = a.Render()
		a.Stop()
	})
}
