package bobble

// Tests written FIRST: the bobble scene is the west-facing lander
// alone — parked at center stage under the drifting sky, bobbling up
// and down on a sine, with or without its engine on. Lit, the tail
// fire burns for as long as the scene plays; Dark, the hull rides
// cold. The same scene slots into two screenplays: the walkthrough
// plays it engine off then engine on, the inverse walkthrough engine
// on then engine off — each flip is a cut on the bill, not a timer in
// here. Period and amplitude are knobs, so the ride can throb or
// undulate whichever way the operator sets it.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
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

// hullRow is the first stage row wearing the west hull glyph, or -1
// when the hull is not on stage.
func hullRow(scr *screenplay.Screen) int {
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			if c := scr.Cell(x, y); c != nil && c.Content == "▌" {
				return y
			}
		}
	}
	return -1
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

func hasMoonBG(scr *screenplay.Screen) bool {
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			c := scr.Cell(x, y)
			if c == nil || c.Style.Bg == nil {
				continue
			}
			if _, ok := c.Style.Bg.(ansi.IndexedColor); ok {
				return true
			}
		}
	}
	return false
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

func sameStars(a, b map[[2]int]string) bool {
	if len(a) != len(b) {
		return false
	}
	for pos, ch := range a {
		if b[pos] != ch {
			return false
		}
	}
	return true
}

func TestBobbleBill(t *testing.T) {
	t.Run("happy: the bill is the one bobble scene", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the bobble bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "bobble" {
			t.Fatalf("the scene is %q, want bobble", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the bobble has no performer")
		}
	})
	t.Run("unhappy: a second scene is not hiding on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "bobble" {
			t.Fatalf("the show opens on %d %q, want one bobble", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the bobble there is nothing left")
		}
	})
}

func TestBobbleScene(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: a lit bobble opens parked at center and burns its tail fire", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		if hullRow(opening) < 0 {
			t.Fatal("the bobble must open with the west hull already parked on stage")
		}
		tick(sc, 0.5)
		lit := paint(sc)
		if !hotBraille(lit) {
			t.Fatal("half a second in the tail fire must be burning")
		}
		if hullRow(lit) < 0 {
			t.Fatal("the hull holds the park while the fire burns")
		}
	})
	t.Run("happy: the ride obeys the period and amplitude knobs", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Engine = false
		sc.Cfg.PeriodSeconds = 4.0
		sc.Cfg.AmplitudeCells = 3
		sc.Start()
		defer sc.Stop()
		base := hullRow(paint(sc))
		tick(sc, 1.0)
		crest := hullRow(paint(sc))
		if crest != base-3 {
			t.Fatalf("a quarter of a 4s period in, the hull rides at %d, want three cells up at %d", crest, base-3)
		}
		tick(sc, 2.0)
		trough := hullRow(paint(sc))
		if trough != base+3 {
			t.Fatalf("three quarters in, the hull dips to %d, want three cells down at %d", trough, base+3)
		}
		tick(sc, 1.0)
		if got := hullRow(paint(sc)); got != base {
			t.Fatalf("a full period brings the hull home to %d, got %d", base, got)
		}
	})
	t.Run("happy: Dark parks a cold hull, Lit overrides a dark Active", func(t *testing.T) {
		t.Cleanup(Reset)
		dark := New(nil).Dark()
		dark.Start()
		defer dark.Stop()
		_ = paint(dark)
		tick(dark, 1.0)
		if hotBraille(paint(dark)) {
			t.Fatal("a dark bobble never lights its engine")
		}
		if hullRow(paint(dark)) < 0 {
			t.Fatal("a dark bobble still parks the hull on stage")
		}

		c := DefaultConfig()
		c.Engine = false
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		lit := New(nil).Lit()
		lit.Start()
		defer lit.Stop()
		_ = paint(lit)
		tick(lit, 0.5)
		if !hotBraille(paint(lit)) {
			t.Fatal("Lit must override a dark Active — the bill's word wins")
		}
	})
	t.Run("happy: the sky drifts on — the bobble plays in open space, not on the moon", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Engine = false
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		if hasMoonBG(opening) {
			t.Fatal("the bobble plays in open space — no moon floor")
		}
		before := starCells(opening)
		if len(before) == 0 {
			t.Fatal("the bobble plays under the stars")
		}
		tick(sc, 2.0)
		if sameStars(before, starCells(paint(sc))) {
			t.Fatal("the bobble sky must keep drifting — a parked craft, not a parked sky")
		}
	})
	t.Run("happy: a seeded sky opens on the frame the last scene left", func(t *testing.T) {
		sky := stars.NewContinuity()
		prior := stars.NewTunedStarfield().Seed(sky)
		prior.Start(stageW, stageH)
		for i := 0; i < 45; i++ {
			prior.Update(1.0 / 30)
		}
		want := starCells(func() *screenplay.Screen {
			scr := screenplay.NewScreen(stageW, stageH)
			scr.Blit(0, 0, prior.Render())
			return scr
		}())
		prior.Stop()

		sc := New(sky)
		sc.Cfg.Engine = false
		sc.Start()
		defer sc.Stop()
		got := starCells(paint(sc))
		hull := (stageW - lander.BodyCols) / 2
		matched := 0
		for pos, ch := range want {
			if pos[0] >= hull-1 && pos[0] <= hull+lander.BodyCols {
				continue
			}
			if g, ok := got[pos]; ok {
				if g != ch {
					t.Fatalf("the cut moved a star at (%d,%d): %q -> %q", pos[0], pos[1], ch, g)
				}
				matched++
			}
		}
		if matched == 0 {
			t.Fatal("the seeded bobble sky must open on the carried frame")
		}
	})
	t.Run("happy: Use is what New plays on the first Start", func(t *testing.T) {
		t.Cleanup(Reset)
		c := DefaultConfig()
		c.Engine = false
		c.PeriodSeconds = 4.0
		c.AmplitudeCells = 2
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		base := hullRow(paint(sc))
		tick(sc, 1.0)
		if got := hullRow(paint(sc)); got != base-2 {
			t.Fatalf("the first play must ride the active knobs, row %d want %d", got, base-2)
		}
		if hotBraille(paint(sc)) {
			t.Fatal("the first play must respect the active engine switch")
		}
	})
	t.Run("happy: Start after Stop replays with the current knobs", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Engine = false
		sc.Cfg.PeriodSeconds = 4.0
		sc.Cfg.AmplitudeCells = 1
		sc.Start()
		base := hullRow(paint(sc))
		tick(sc, 1.0)
		if got := hullRow(paint(sc)); got != base-1 {
			t.Fatal("test premise: the stock ride must crest one cell")
		}
		sc.Cfg.AmplitudeCells = 3
		sc.Stop()
		sc.Start()
		_ = paint(sc)
		tick(sc, 1.0)
		if got := hullRow(paint(sc)); got != base-3 {
			t.Fatalf("play must rebuild from the current knobs, row %d want %d", got, base-3)
		}
		sc.Stop()
	})
	t.Run("unhappy: changing knobs mid-ride waits for the replay", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Engine = false
		sc.Cfg.PeriodSeconds = 4.0
		sc.Cfg.AmplitudeCells = 1
		sc.Start()
		defer sc.Stop()
		base := hullRow(paint(sc))
		sc.Cfg.AmplitudeCells = 5
		tick(sc, 1.0)
		if got := hullRow(paint(sc)); got != base-1 {
			t.Fatalf("a ride in the air keeps the knobs it launched with, row %d want %d", got, base-1)
		}
	})
	t.Run("unhappy: amplitude 0 holds the park level — with or without the engine", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.AmplitudeCells = 0
		sc.Start()
		defer sc.Stop()
		base := hullRow(paint(sc))
		tick(sc, 3.0)
		if got := hullRow(paint(sc)); got != base {
			t.Fatalf("amplitude 0 must hold the park, row %d -> %d", base, got)
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
		held.Cfg.Engine = false
		held.Start()
		defer held.Stop()
		base := hullRow(paint(held))
		held.Update(0)
		held.Update(-3)
		if got := hullRow(paint(held)); got != base {
			t.Fatal("dt<=0 must hold the park")
		}
	})
	t.Run("unhappy: the bobble is not the fly-in — the hull is already parked at t=0", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Engine = false
		sc.Start()
		defer sc.Stop()
		if hullRow(paint(sc)) < 0 {
			t.Fatal("no fly-in here: the curtain rises on the parked craft")
		}
	})
}
