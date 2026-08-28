package liftoff

// Tests written FIRST: the liftoff scene is the landing played
// backwards, and nothing more. The curtain rises on the landing's
// final frame: the north-facing lander parked on the huge moon
// horizon, engine cold, under a still sky. The booster ignites and
// throttles up (¼, ½, ¾, full), pad dust blows both ways, and at
// lift-at the craft climbs on the landing's mirrored ease — a heavy
// crawl off the pad that rockets off the top. Then the scene simply
// holds: the moon floor and the still stars stay put until the
// screenplay cuts away. The west-facing craft never appears here —
// that is the bobble scene's job, on the next entry of the bill.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 27
)

func paint(sc screenplay.Scene) *screenplay.Screen {
	scr := screenplay.NewScreen(stageW, stageH)
	sc.Render(scr)
	return scr
}

func tick(sc screenplay.Scene, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		sc.Update(dt)
	}
}

func isMoonBG(scr *screenplay.Screen, x, y int) bool {
	c := scr.Cell(x, y)
	if c == nil || c.Style.Bg == nil {
		return false
	}
	ic, ok := c.Style.Bg.(ansi.IndexedColor)
	if !ok {
		return false
	}
	n := int(ic)
	return n == 251 || n == 247 || n == 243 || n == 240 || n == 249
}

func moonBGRows(scr *screenplay.Screen, col int) int {
	n := 0
	for y := 0; y < stageH; y++ {
		if isMoonBG(scr, col, y) {
			n++
		}
	}
	return n
}

func hotBraille(scr *screenplay.Screen) bool {
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			c := scr.Cell(x, y)
			if c == nil || !strings.ContainsAny(c.Content, "⠁⠒⠶") {
				continue
			}
			ic, ok := c.Style.Fg.(ansi.IndexedColor)
			if !ok || int(ic) < dust.GrayMin {
				return true
			}
		}
	}
	return false
}

// hullRow is the first stage row wearing the given hull glyph, or -1
// when the hull is not on stage.
func hullRow(scr *screenplay.Screen, glyph string) int {
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			if c := scr.Cell(x, y); c != nil && c.Content == glyph {
				return y
			}
		}
	}
	return -1
}

func offHullDust(scr *screenplay.Screen) (left, right bool) {
	hullCol := (stageW - lander.BodyCols) / 2
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			if x >= hullCol && x < hullCol+lander.BodyCols {
				continue
			}
			c := scr.Cell(x, y)
			if c == nil {
				continue
			}
			for _, r := range c.Content {
				if (r >= '⠀' && r <= '⣿') || r == '░' || r == '▒' {
					if x < hullCol {
						left = true
					} else {
						right = true
					}
				}
			}
		}
	}
	return left, right
}

func starCells(scr *screenplay.Screen) map[[2]int]string {
	out := map[[2]int]string{}
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			c := scr.Cell(x, y)
			if c == nil {
				continue
			}
			for _, g := range stars.Glyphs {
				if c.Content == string(g) {
					out[[2]int{x, y}] = c.Content
				}
			}
		}
	}
	return out
}

func TestLiftoffBill(t *testing.T) {
	t.Run("happy: the bill is the one liftoff scene", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the liftoff bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "liftoff" {
			t.Fatalf("the scene is %q, want liftoff", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the liftoff has no performer")
		}
	})
	t.Run("unhappy: a second scene is not hiding on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "liftoff" {
			t.Fatalf("the show opens on %d %q, want one liftoff", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the liftoff there is nothing left")
		}
	})
}

func TestLiftoffScene(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: the curtain rises on the landing's final frame — parked on the pad, engine cold", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		if moonBGRows(opening, stageW/2) != moon.HorizonCenterRows {
			t.Fatalf("center holds %d moon rows, want %d", moonBGRows(opening, stageW/2), moon.HorizonCenterRows)
		}
		if moonBGRows(opening, 0) != moon.HorizonEdgeRows {
			t.Fatalf("left edge holds %d moon rows, want %d", moonBGRows(opening, 0), moon.HorizonEdgeRows)
		}
		if hullRow(opening, "▟") < 0 {
			t.Fatal("at t=0 the north hull must already sit on the pad")
		}
		if hotBraille(opening) {
			t.Fatal("at t=0 the booster must still be cold")
		}
	})
	t.Run("happy: ignition on the pad, dust both ways, then the climb off the top", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LiftAt = 0.6
		sc.Cfg.RiseSeconds = 0.8
		sc.Cfg.Fire25 = 0.1
		sc.Cfg.Fire50 = 0.15
		sc.Cfg.Fire75 = 0.2
		sc.Cfg.FireFull = 0.25
		sc.Cfg.DustStart = 0.2
		sc.Cfg.DustRun = 0.6
		sc.Cfg.DustLoss = 0.05
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		if hotBraille(opening) {
			t.Fatal("before the first ignition offset the pad must be cold")
		}
		padRow := hullRow(opening, "▟")
		tick(sc, 0.55)
		holding := paint(sc)
		if got := hullRow(holding, "▟"); got != padRow {
			t.Fatalf("before lift-at the hull must hold the pad, row %d want %d", got, padRow)
		}
		if !hotBraille(holding) {
			t.Fatal("past full power the booster must burn on the pad")
		}
		if l, r := offHullDust(holding); !l || !r {
			t.Fatalf("the ignition must kick pad dust both ways, left=%v right=%v", l, r)
		}
		tick(sc, 0.6)
		climbing := paint(sc)
		got := hullRow(climbing, "▟")
		if got < 0 {
			t.Fatal("mid-climb the hull must still be on stage")
		}
		if got >= padRow {
			t.Fatalf("mid-climb the hull must have left the pad, row %d was %d", got, padRow)
		}
	})
	t.Run("happy: after the climb the scene holds the empty moon — no cut of its own", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LiftAt = 0.2
		sc.Cfg.RiseSeconds = 0.4
		sc.Cfg.DustStart = 0
		sc.Cfg.DustRun = 0
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, sc.Cfg.LiftAt+sc.Cfg.RiseSeconds+0.1)
		gone := paint(sc)
		if hullRow(gone, "▟") >= 0 {
			t.Fatal("past lift-at plus rise the hull must have cleared the top")
		}
		if moonBGRows(gone, stageW/2) != moon.HorizonCenterRows {
			t.Fatal("the moon floor stays after the craft has gone — the scene holds for the cut")
		}
		held := starCells(gone)
		if len(held) == 0 {
			t.Fatal("the held stage must still show stars")
		}
		tick(sc, 5.0)
		later := paint(sc)
		if hullRow(later, "▟") >= 0 || moonBGRows(later, stageW/2) != moon.HorizonCenterRows {
			t.Fatal("five seconds on, the scene must still hold the empty moon")
		}
		for pos, ch := range starCells(later) {
			if held[pos] != ch {
				t.Fatalf("the held sky crawled: star at (%d,%d) %q -> %q", pos[0], pos[1], held[pos], ch)
			}
		}
	})
	t.Run("unhappy: the west-facing hull never plays this scene", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LiftAt = 0.2
		sc.Cfg.RiseSeconds = 0.4
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		for i := 0; i < 8; i++ {
			tick(sc, 0.25)
			if hullRow(paint(sc), "▌") >= 0 {
				t.Fatal("the sideways craft belongs to the bobble scene, not the liftoff")
			}
		}
	})
	t.Run("happy: a seeded sky freezes on the frame the last scene left", func(t *testing.T) {
		sky := stars.NewContinuity()
		prior := stars.NewTunedStarfield().Seed(sky)
		prior.Start(stageW, stageH)
		for i := 0; i < 45; i++ {
			prior.Update(1.0 / 30)
		}
		_ = prior.Render()
		prior.Stop()

		sc := New(sky)
		sc.Start()
		defer sc.Stop()
		opening := starCells(paint(sc))
		tick(sc, 1.0)
		later := starCells(paint(sc))
		for pos, ch := range later {
			if opening[pos] != ch {
				t.Fatalf("the still liftoff sky crawled: star at (%d,%d) %q -> %q", pos[0], pos[1], opening[pos], ch)
			}
		}
		if len(later) == 0 {
			t.Fatal("the liftoff sky must show stars above the horizon")
		}
	})
	t.Run("happy: Use is what New plays on the first Start", func(t *testing.T) {
		t.Cleanup(Reset)
		c := DefaultConfig()
		c.LiftAt = 0
		c.RiseSeconds = 0.2
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.3)
		if hullRow(paint(sc), "▟") >= 0 {
			t.Fatal("the first play must already use the active knobs — a 0.2s climb is long gone")
		}
	})
	t.Run("happy: Start after Stop replays from the pad with the current knobs", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LiftAt = 0.1
		sc.Cfg.RiseSeconds = 0.2
		sc.Start()
		_ = paint(sc)
		tick(sc, 0.5)
		if hullRow(paint(sc), "▟") >= 0 {
			t.Fatal("test premise: the craft must be gone before the replay")
		}
		sc.Stop()
		sc.Start()
		if hullRow(paint(sc), "▟") < 0 {
			t.Fatal("play must rewind the craft onto the pad")
		}
		sc.Stop()
	})
	t.Run("unhappy: changing knobs mid-flight does not teleport the craft", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 1.0)
		sc.Cfg.LiftAt = 0
		sc.Cfg.RiseSeconds = 0.05
		tick(sc, 0.3)
		if hullRow(paint(sc), "▟") < 0 {
			t.Fatal("an in-flight craft must keep the knobs it launched with")
		}
	})
	t.Run("unhappy: updates before the first render hold the curtain", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LiftAt = 0.1
		sc.Cfg.RiseSeconds = 0.2
		sc.Start()
		defer sc.Stop()
		sc.Update(10)
		if hullRow(paint(sc), "▟") < 0 {
			t.Fatal("time before the first render must not fly the show — the curtain opens on the pad")
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics, and dt<=0 holds", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		sc.Update(1)
		sc.Stop()
		sc.Update(1)
		sc.Render(nil)
		sc.Stop()

		held := New(nil)
		held.Start()
		defer held.Stop()
		_ = paint(held)
		held.Update(0)
		held.Update(-3)
		if hullRow(paint(held), "▟") < 0 {
			t.Fatal("dt<=0 must hold the opening frame")
		}
	})
}
