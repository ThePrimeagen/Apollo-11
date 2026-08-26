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

// liveDust is how many specks both engines hold this instant.
func liveDust(cl *Cloud) int {
	return len(cl.Left.Particles) + len(cl.Right.Particles)
}

// noDust asserts a stage with not one dust glyph on it.
func noDust(t *testing.T, cl *Cloud, when string) {
	t.Helper()
	sp := cl.Render()
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if dustGlyph(sp.At(r, c)) {
				t.Fatalf("%s: dust still painted at (%d,%d)", when, r, c)
			}
		}
	}
}

// TestCloudFade: Fade makes the cloud a one-shot burst. It opens on
// the full warm kick, then counts its particles down — linearly, from
// however many the kick holds at its max to zero over the fade
// window — and then the stage is clear for good. FadeSeconds is that
// window: two seconds.
func TestCloudFade(t *testing.T) {
	t.Run("happy: a fading cloud opens on the same full kick as a classic one", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		if FadeSeconds != 2.0 {
			t.Fatalf("FadeSeconds is %v, want the two-second countdown", FadeSeconds)
		}
		classic := NewCloud(11)
		classic.Start(80, 24)
		fading := NewCloud(11).Fade(FadeSeconds)
		fading.Start(80, 24)
		got, want := liveDust(fading), liveDust(classic)
		if want == 0 {
			t.Fatal("test premise: a warmed cloud opens dusty")
		}
		if got != want {
			t.Fatalf("a fading cloud opens with %d specks, the classic kick holds %d — the countdown starts from the max, not below it", got, want)
		}
	})
	t.Run("happy: the kick counts its particles down to zero over the fade window", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(7).Fade(FadeSeconds)
		cl.Start(80, 24)
		leftMax, rightMax := len(cl.Left.Particles), len(cl.Right.Particles)
		if leftMax == 0 || rightMax == 0 {
			t.Fatal("test premise: both engines open with dust")
		}
		const dt = 1.0 / 30
		clock := 0.0
		checkpoints := map[int]int{} // frame -> live count at 0.5s marks
		for i := 1; i <= 61; i++ {
			cl.Update(dt)
			clock += dt
			frac := 1 - clock/FadeSeconds
			if frac < 0 {
				frac = 0
			}
			bound := int(math.Round(float64(leftMax)*frac)) +
				int(math.Round(float64(rightMax)*frac)) + 2
			if got := liveDust(cl); got > bound {
				t.Fatalf("%.2fs into the fade %d specks live, the countdown allows %d", clock, got, bound)
			}
			if i == 15 || i == 30 || i == 45 {
				checkpoints[i] = liveDust(cl)
			}
		}
		if checkpoints[15] >= leftMax+rightMax {
			t.Fatalf("half a second in, %d specks live — the countdown must already be under the opening %d", checkpoints[15], leftMax+rightMax)
		}
		if checkpoints[30] >= checkpoints[15] || checkpoints[45] >= checkpoints[30] {
			t.Fatalf("the countdown must fall through its checkpoints: %d, %d, %d", checkpoints[15], checkpoints[30], checkpoints[45])
		}
		if checkpoints[45] == 0 {
			t.Fatal("1.5s in the kick must still hold a few specks — a countdown, not a cliff")
		}
		if got := liveDust(cl); got != 0 {
			t.Fatalf("past the fade window %d specks still live, want zero", got)
		}
		noDust(t, cl, "past the fade window")
	})
	t.Run("happy: the countdown lets the blown fringe drift — mid-fade dust still shows far from the nozzles", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(7).Fade(FadeSeconds)
		cl.Start(80, 24)
		for i := 0; i < 30; i++ { // 1.0s: halfway down the countdown
			cl.Update(1.0 / 30)
		}
		// The nozzles sit at columns 36 and 44; the kick's far fringe
		// must still be in the air well beyond them. Culling the
		// oldest specks first would erase exactly that fringe and
		// contract the kick into a churn at the floor.
		farLeft, _, farRight := sides(cl.Render(), 28, 52)
		if !farLeft || !farRight {
			t.Fatalf("halfway down the countdown the blown fringe must still drift out both sides, left=%v right=%v", farLeft, farRight)
		}
	})
	t.Run("happy: mid-fade the dust still blows out both sides and spares the gap", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(11).Fade(FadeSeconds)
		cl.Start(80, 24)
		for i := 0; i < 9; i++ { // 0.3s: a third of the way down
			cl.Update(1.0 / 30)
		}
		l, c, r := sides(cl.Render(), 38, 41)
		if !l || !r {
			t.Fatalf("mid-fade dust must still blow both sides, left=%v right=%v", l, r)
		}
		if c {
			t.Fatal("mid-fade the still gap must stay still")
		}
	})
	t.Run("unhappy: a classic cloud never counts down", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(11)
		cl.Start(80, 24)
		for i := 0; i < 90; i++ { // 3s, well past any fade window
			cl.Update(1.0 / 30)
		}
		if liveDust(cl) == 0 {
			t.Fatal("a cloud without Fade must keep kicking forever")
		}
	})
	t.Run("unhappy: Fade(<=0) keeps the endless kick and Fade on nil stays nil", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(11).Fade(0)
		cl.Start(80, 24)
		for i := 0; i < 90; i++ {
			cl.Update(1.0 / 30)
		}
		if liveDust(cl) == 0 {
			t.Fatal("Fade(0) must mean no countdown, not an instant one")
		}
		var ghost *Cloud
		if ghost.Fade(FadeSeconds) != nil {
			t.Fatal("Fade must return the nil receiver")
		}
	})
	t.Run("happy: Fade after Start begins the countdown from that instant, not from the curtain", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(11)
		cl.Start(80, 24)
		for i := 0; i < 45; i++ { // 1.5s of endless kick
			cl.Update(1.0 / 30)
		}
		held := liveDust(cl)
		if held == 0 {
			t.Fatal("test premise: the endless kick must still be dusty")
		}
		cl.Fade(FadeSeconds)
		for i := 0; i < 15; i++ { // 0.5s into a fade that started just now
			cl.Update(1.0 / 30)
		}
		got := liveDust(cl)
		if got == 0 || got >= held {
			t.Fatalf("0.5s after a late Fade %d specks, held %d — the countdown must start from the cue, not from Start", got, held)
		}
		for i := 0; i < 50; i++ {
			cl.Update(1.0 / 30)
		}
		if liveDust(cl) != 0 {
			t.Fatalf("past the late fade window %d specks still live, want zero", liveDust(cl))
		}
		noDust(t, cl, "past a late fade")
	})
	t.Run("unhappy: a cloud that has not been asked to Fade yet keeps kicking past FadeSeconds", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		cl := NewCloud(11)
		cl.Start(80, 24)
		for i := 0; i < int(FadeSeconds*30)+15; i++ {
			cl.Update(1.0 / 30)
		}
		if liveDust(cl) == 0 {
			t.Fatal("without Fade the kick must still be running past FadeSeconds")
		}
	})
}
