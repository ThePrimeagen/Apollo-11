package moonwalk

// Tests written FIRST: the moonwalk is now a screenplay.Scene, cut
// into three beats so 03. Mario can play it as a bill. BeatRun is
// the crate climb — he sprints in and hops one, two, three high,
// then holds on the top stack. BeatPole is the flagpole — the leap
// onto the gold ball, the hold, the slide while the flag flies up,
// then the bow. BeatBoard is the exit — the camera pans to the
// lunar module, he runs over, jumps the hatch, and stays gone.
// Active/Use/Reset let the screenplay play the saved knobs.

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

var _ screenplay.Scene = (*Show)(nil)

const (
	showW = 84
	showH = 30
)

func paintShow(sc screenplay.Scene) *screenplay.Screen {
	scr := screenplay.NewScreen(showW, showH)
	sc.Render(scr)
	return scr
}

func tickShow(sc screenplay.Scene, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		sc.Update(dt)
	}
}

func hasPole(v string) bool {
	return strings.ContainsRune(v, '│') && strings.ContainsRune(v, '●')
}

func TestMoonwalkShowBeats(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: BeatRun opens running on the ground and holds on the top crate", func(t *testing.T) {
		sc := New(BeatRun)
		sc.Start()
		defer sc.Stop()
		opening := paintShow(sc)
		if !hasPole(opening.Render()) {
			t.Fatal("the run beat must stage the flagpole")
		}
		pose, _, y := timelineAt(sc.Cfg, showW, showH, 0)
		if !isRun(pose) || y != groundedY(showH) {
			t.Fatalf("the run beat opens running on the ground, got %q y %d", pose, y)
		}
		r := routeFor(sc.Cfg, showW, showH)
		tickShow(sc, r.leapAt+2)
		held := paintShow(sc)
		if strings.Contains(held.Render(), "38;5;26m") || strings.Contains(held.Render(), "48;5;26m") {
			t.Fatal("the run beat must never hoist the flag")
		}
		// Held on the third stack: a stand pose at yC, never on the pole.
		pose, _, y = timelineAt(sc.Cfg, showW, showH, r.leapAt-clockEps)
		if pose != astro.PoseStand || y != r.yC {
			t.Fatalf("the run beat must hold standing on stack three, got %q y %d want stand y %d", pose, y, r.yC)
		}
	})
	t.Run("happy: BeatPole opens on the leap and the flag rises while he slides", func(t *testing.T) {
		sc := New(BeatPole)
		sc.Start()
		defer sc.Stop()
		_ = paintShow(sc)
		r := routeFor(sc.Cfg, showW, showH)
		pose, _, _ := timelineAt(sc.Cfg, showW, showH, r.leapAt)
		if pose != astro.PoseJump {
			t.Fatalf("the pole beat opens on the leap, got %q", pose)
		}
		if _, visible := flagAt(sc.Cfg, showW, showH, r.leapAt); visible {
			t.Fatal("the flag must not exist at the start of the leap")
		}
		tickShow(sc, (r.panAt-r.leapAt)+1)
		early, _ := flagAt(sc.Cfg, showW, showH, r.slideAt+0.05)
		late, visible := flagAt(sc.Cfg, showW, showH, r.panAt-clockEps)
		if !visible {
			t.Fatal("by the bow the flag must be flying")
		}
		if late >= early {
			t.Fatalf("the flag must rise while he slides: top went %d -> %d", early, late)
		}
	})
	t.Run("happy: BeatBoard pans to the module, he jumps the hatch, and stays gone", func(t *testing.T) {
		sc := New(BeatBoard)
		sc.Start()
		defer sc.Stop()
		_ = paintShow(sc)
		r := routeFor(sc.Cfg, showW, showH)
		if cameraAt(sc.Cfg, showW, showH, r.panAt) != 0 {
			t.Fatal("the board beat opens before the pan moves")
		}
		tickShow(sc, r.cycle-r.panAt+1)
		end := paintShow(sc)
		if cameraAt(sc.Cfg, showW, showH, r.cycle-clockEps) != sc.Cfg.PanCols {
			t.Fatal("the board beat must finish the pan onto the module")
		}
		pose, _, _ := timelineAt(sc.Cfg, showW, showH, r.cycle-clockEps)
		if pose != PoseGone {
			t.Fatalf("once aboard he stays gone, got %q", pose)
		}
		if !strings.ContainsRune(end.Render(), '▟') {
			t.Fatal("the lunar module must be on stage after the pan")
		}
	})
	t.Run("happy: a faster runner finishes the climb sooner on the active knobs", func(t *testing.T) {
		t.Cleanup(Reset)
		fast := DefaultConfig()
		fast.RunSpeed = DefaultConfig().RunSpeed * 2
		if err := Use(fast); err != nil {
			t.Fatal(err)
		}
		slowR := routeFor(DefaultConfig(), showW, showH)
		sc := New(BeatRun)
		if sc.Cfg.RunSpeed != fast.RunSpeed {
			t.Fatalf("New must copy Active, run speed %v want %v", sc.Cfg.RunSpeed, fast.RunSpeed)
		}
		fastR := routeFor(sc.Cfg, showW, showH)
		if fastR.leapAt >= slowR.leapAt {
			t.Fatal("doubling the ground speed must shorten the climb")
		}
	})
	t.Run("unhappy: BeatRun never reaches the pole, however long it sits", func(t *testing.T) {
		sc := New(BeatRun)
		sc.Start()
		defer sc.Stop()
		_ = paintShow(sc)
		tickShow(sc, 20)
		r := routeFor(sc.Cfg, showW, showH)
		pose, x, _ := timelineAt(sc.Cfg, showW, showH, r.leapAt-clockEps)
		if isPole(pose) {
			t.Fatal("the run beat must freeze before the leap — he never grabs the pole")
		}
		if x >= r.grabX {
			t.Fatalf("the run beat walked onto the pole at x %d (grab %d)", x, r.grabX)
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		for _, beat := range []Beat{BeatRun, BeatPole, BeatBoard} {
			sc := New(beat)
			sc.Start()
			sc.Update(1)
			sc.Stop()
		}
	})
}

func TestMoonwalkActiveKnobs(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: Use then Active then Reset round-trips the stock show", func(t *testing.T) {
		c := DefaultConfig()
		c.RunSpeed = 40
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		if Active().RunSpeed != 40 {
			t.Fatalf("Active must keep the used knobs, got %+v", Active())
		}
		Reset()
		if Active() != DefaultConfig() {
			t.Fatalf("Reset must restore stock, got %+v", Active())
		}
	})
	t.Run("unhappy: a degenerate config is rejected and Active is unchanged", func(t *testing.T) {
		t.Cleanup(Reset)
		before := Active()
		bad := DefaultConfig()
		bad.RunSpeed = 0
		if err := Use(bad); err == nil {
			t.Fatal("a zero run speed must be rejected")
		}
		if Active() != before {
			t.Fatalf("a rejected Use must leave Active alone, got %+v", Active())
		}
	})
}
