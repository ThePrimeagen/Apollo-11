package liftoff

// Tests written FIRST: the liftoff scene is 03. Inverse Walkthrough —
// the walkthrough played backwards inside one scene. The curtain rises
// on the landing's final frame: the north-facing lander parked on the
// huge moon horizon, engine cold, under a still sky. The booster
// ignites and throttles up (¼, ½, ¾, full), pad dust blows, and at
// lift-at the craft climbs on the landing's mirrored ease — a heavy
// crawl off the pad that rockets off the top. The moment the hull is
// fully gone the scene cuts, exactly like the walkthrough's own cuts:
// the horizon vanishes and the tilted-sideways west craft is revealed
// parked at center stage, tail fire on, under the very same sky. After
// FireOff seconds the fire cuts, and the craft bobbles on the parked
// sine ad infinitum — the scene never ends on its own.

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

// quietStars collects the star glyphs in the quiet corners of the
// stage — above the horizon band and clear of every hull and plume
// column — where the sky must hold perfectly still across the cut.
func quietStars(scr *screenplay.Screen) map[[2]int]string {
	out := map[[2]int]string{}
	for y := 0; y < 14; y++ {
		for x := 0; x < stageW; x++ {
			if x >= 20 && x <= 62 {
				continue
			}
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
	t.Run("happy: the bill is the one inverse-walkthrough scene", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the liftoff bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "inverse walkthrough" {
			t.Fatalf("the scene is %q, want inverse walkthrough", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the inverse walkthrough has no performer")
		}
	})
	t.Run("unhappy: a second scene is not hiding on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "inverse walkthrough" {
			t.Fatalf("the show opens on %d %q, want one inverse walkthrough", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the inverse walkthrough there is nothing left")
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
		row := hullRow(opening, "▟")
		if row < 0 {
			t.Fatal("at t=0 the north hull must already sit on the pad")
		}
		if hullRow(opening, "▌") >= 0 {
			t.Fatal("the ground phase must not wear the west-facing hull")
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
		sc.Cfg.FireOff = 0.4
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
		if hullRow(climbing, "▌") >= 0 {
			t.Fatal("the climb is flown by the north hull, not the west one")
		}
	})
	t.Run("happy: the cut reveals the tilted-sideways craft, fire on, then fire off, bobbling forever", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LiftAt = 0.2
		sc.Cfg.RiseSeconds = 0.4
		sc.Cfg.Fire25 = 0
		sc.Cfg.Fire50 = 0.05
		sc.Cfg.Fire75 = 0.1
		sc.Cfg.FireFull = 0.15
		sc.Cfg.DustStart = 0
		sc.Cfg.DustRun = 0
		sc.Cfg.FireOff = 0.8
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, sc.Cfg.CutSeconds()+0.02)
		reveal := paint(sc)
		if hullRow(reveal, "▌") < 0 {
			t.Fatal("past the cut the tilted-sideways west hull must be parked on stage")
		}
		if moonBGRows(reveal, stageW/2) != 0 || moonBGRows(reveal, 0) != 0 {
			t.Fatal("past the cut the moon horizon must be gone — the craft is up in space now")
		}
		base := hullRow(reveal, "▌")
		tick(sc, 0.4)
		lit := paint(sc)
		if !hotBraille(lit) {
			t.Fatal("the reveal must burn its tail fire before the cut-off")
		}
		tick(sc, 0.5)
		doused := paint(sc)
		if hotBraille(doused) {
			t.Fatal("FireOff seconds after the reveal the tail fire must be out")
		}
		if hullRow(doused, "▌") < 0 {
			t.Fatal("the doused craft must stay parked on stage")
		}
		tick(sc, 1.6)
		crest := hullRow(paint(sc), "▌")
		if crest != base-1 {
			t.Fatalf("a quarter period after the reveal the bobble must crest one cell up, row %d want %d", crest, base-1)
		}
		tick(sc, 2.5)
		if got := hullRow(paint(sc), "▌"); got != base {
			t.Fatalf("half a period after the reveal the bobble is back at center, row %d want %d", got, base)
		}
		tick(sc, 2.5)
		trough := hullRow(paint(sc), "▌")
		if trough != base+1 {
			t.Fatalf("three quarters in the bobble must dip one cell down, row %d want %d", trough, base+1)
		}
		tick(sc, 30.0)
		forever := paint(sc)
		if hullRow(forever, "▌") < 0 {
			t.Fatal("ad infinitum: half a minute later the craft must still hold the park")
		}
		if hotBraille(forever) {
			t.Fatal("ad infinitum: the fire stays out for the rest of the scene")
		}
	})
	t.Run("happy: the sky holds the very same stars across the cut", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LiftAt = 0.2
		sc.Cfg.RiseSeconds = 0.3
		sc.Cfg.DustStart = 0
		sc.Cfg.DustRun = 0
		sc.Cfg.FireOff = 1.0
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.4)
		before := quietStars(paint(sc))
		if len(before) == 0 {
			t.Fatal("test premise: the quiet corners must hold stars before the cut")
		}
		tick(sc, 0.2)
		after := paint(sc)
		if hullRow(after, "▌") < 0 {
			t.Fatal("test premise: the cut must already have played")
		}
		got := quietStars(after)
		for pos, ch := range before {
			if got[pos] != ch {
				t.Fatalf("the cut moved a star at (%d,%d): %q -> %q", pos[0], pos[1], ch, got[pos])
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
		if hullRow(paint(sc), "▌") < 0 {
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
		if hullRow(paint(sc), "▌") < 0 {
			t.Fatal("test premise: the cut must have played before the replay")
		}
		sc.Stop()
		sc.Start()
		opening := paint(sc)
		if hullRow(opening, "▟") < 0 {
			t.Fatal("play must rewind the craft onto the pad")
		}
		if hullRow(opening, "▌") >= 0 {
			t.Fatal("play must rewind the cut away")
		}
		sc.Stop()
	})
	t.Run("unhappy: changing knobs mid-flight does not move the cut", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 1.0)
		sc.Cfg.LiftAt = 0
		sc.Cfg.RiseSeconds = 0.05
		tick(sc, 0.3)
		mid := paint(sc)
		if hullRow(mid, "▌") >= 0 {
			t.Fatal("an in-flight show must keep the knobs it launched with")
		}
		if hullRow(mid, "▟") < 0 {
			t.Fatal("the in-flight craft must still be on its original climb")
		}
	})
	t.Run("unhappy: fire-off at 0 reveals a craft already dark", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LiftAt = 0.1
		sc.Cfg.RiseSeconds = 0.2
		sc.Cfg.DustStart = 0
		sc.Cfg.DustRun = 0
		sc.Cfg.FireOff = 0
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, sc.Cfg.CutSeconds()+0.02)
		_ = paint(sc)
		tick(sc, 0.4)
		reveal := paint(sc)
		if hullRow(reveal, "▌") < 0 {
			t.Fatal("test premise: the reveal must be on stage")
		}
		if hotBraille(reveal) {
			t.Fatal("fire-off at 0 must open the reveal with the tail fire already out")
		}
	})
	t.Run("unhappy: updates before the first render hold the curtain", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LiftAt = 0.1
		sc.Cfg.RiseSeconds = 0.2
		sc.Start()
		defer sc.Stop()
		sc.Update(10)
		opening := paint(sc)
		if hullRow(opening, "▟") < 0 {
			t.Fatal("time before the first render must not fly the show — the curtain opens on the pad")
		}
		if hullRow(opening, "▌") >= 0 {
			t.Fatal("time before the first render must not play the cut")
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
