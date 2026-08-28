package explorer

// Tests written FIRST: the explorer scene is the Big E — the moon-sized
// Internet Explorer logo as its own component, under the blinky-star
// background as its own component, plus one shooting star that falls
// once from top mid-right to bottom mid-left and does not come back.
// The stars fly the twinkle mode: the sky holds where it scattered
// and some stars fade in and out on the knobs the scene's config
// carries. Assemble pushes the scene's knobs onto the stars package,
// so Play (Stop then Start) rebuilds the breathing from whatever the
// knobs hold now, and a tuner can retune it live between plays.

import (
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/bigstar"
	"github.com/theprimeagen/apollo-11/exec-tui/components/ie"
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

func fgIndex(scr *screenplay.Screen, x, y int) int {
	c := scr.Cell(x, y)
	if c == nil {
		return -1
	}
	ic, ok := c.Style.Fg.(ansi.IndexedColor)
	if !ok {
		return -1
	}
	return int(ic)
}

// inkCells counts stage cells wearing the given foreground ink.
func inkCells(scr *screenplay.Screen, ink int) int {
	n := 0
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			if fgIndex(scr, x, y) == ink {
				n++
			}
		}
	}
	return n
}

func meteorCell(scr *screenplay.Screen) (x, y int, ok bool) {
	for y = 0; y < stageH; y++ {
		for x = 0; x < stageW; x++ {
			c := scr.Cell(x, y)
			if c != nil && c.Content == string(bigstar.CoreGlyph) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

// skyCells is the twinkling field with the shooting star's own cells
// masked out — the trail reuses the sky glyphs, so a composite would
// look like stars shapeshifting as the meteor passes.
func skyCells(sc *Show, scr *screenplay.Screen) map[[2]int]string {
	mask := map[[2]int]bool{}
	if sc != nil && sc.meteor != nil {
		sp := sc.meteor.Render()
		for y := 0; y < sp.Height; y++ {
			for x := 0; x < sp.Width; x++ {
				cell := sp.At(y, x)
				if cell.Ch != 0 && cell.Ch != ' ' {
					mask[[2]int{x, y}] = true
				}
			}
		}
	}
	out := starCells(scr)
	for pos := range mask {
		delete(out, pos)
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

func TestExplorerBill(t *testing.T) {
	t.Run("happy: the bill is the one explorer scene", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the explorer bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "explorer" {
			t.Fatalf("the scene is %q, want explorer", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the explorer has no performer")
		}
	})
	t.Run("unhappy: a second scene is not hiding on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "explorer" {
			t.Fatalf("the show opens on %d %q, want one explorer", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the explorer there is nothing left")
		}
	})
}

func TestExplorerScene(t *testing.T) {
	t.Cleanup(reset)
	t.Run("happy: the curtain rises on the big logo under the stars", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		if inkCells(opening, ie.BlueInk) == 0 {
			t.Fatal("the blue e must be on stage from the first frame")
		}
		if inkCells(opening, ie.GoldInk) == 0 {
			t.Fatal("the golden swoosh must be on stage from the first frame")
		}
		if len(starCells(opening)) == 0 {
			t.Fatal("the logo plays under the stars")
		}
		x, y, ok := meteorCell(opening)
		if !ok {
			t.Fatal("one shooting star must already be on stage")
		}
		if x < stageW/2 {
			t.Fatalf("the shooting star must enter from the right, col %d", x)
		}
		if y > stageH/2 {
			t.Fatalf("the shooting star must enter from the top, row %d", y)
		}
	})
	t.Run("happy: one shooting star falls top mid-right to bottom mid-left, then is gone", func(t *testing.T) {
		reset()
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		x0, y0, ok := meteorCell(paint(sc))
		if !ok {
			t.Fatal("need the shooting star on stage to watch it fly")
		}
		tick(sc, 0.4)
		x1, y1, ok := meteorCell(paint(sc))
		if !ok {
			t.Fatal("the shooting star must still be on stage a beat later")
		}
		if x1 >= x0 {
			t.Fatalf("the shooting star must travel right-to-left, col %d → %d", x0, x1)
		}
		if y1 < y0 {
			t.Fatalf("the shooting star must fall, row %d → %d", y0, y1)
		}
		tick(sc, 6)
		if _, _, ok := meteorCell(paint(sc)); ok {
			t.Fatal("after the crossing the shooting star must leave the stage")
		}
		tick(sc, 3)
		if _, _, ok := meteorCell(paint(sc)); ok {
			t.Fatal("a second shooting star must not appear — the Big E fires once")
		}
	})
	t.Run("happy: the sky twinkles — stars fade while the sky holds its scatter", func(t *testing.T) {
		reset()
		sc := New(nil)
		sc.Cfg.MinCycleSeconds, sc.Cfg.MaxCycleSeconds = 4, 4
		sc.Cfg.MinFadeSeconds, sc.Cfg.MaxFadeSeconds = 1, 1
		sc.Start()
		defer sc.Stop()
		before := skyCells(sc, paint(sc))
		if len(before) == 0 {
			t.Fatal("test premise: the opening sky holds stars")
		}
		var faded bool
		for i := 0; i < 8; i++ {
			tick(sc, 0.5)
			now := skyCells(sc, paint(sc))
			held := 0
			for pos, ch := range now {
				was, ok := before[pos]
				if !ok {
					continue // a breather fading back in at its home
				}
				if was != ch {
					t.Fatalf("the star at (%d,%d) changed glyph %q→%q — fade, not shapeshift", pos[0], pos[1], was, ch)
				}
				held++
			}
			if held == 0 {
				t.Fatal("the steady stars must hold the sky while the breathers fade")
			}
			for pos := range before {
				if _, ok := now[pos]; !ok {
					faded = true
				}
			}
		}
		if !faded {
			t.Fatal("across a full cycle some star must fade out")
		}
	})
	t.Run("happy: assemble pushes the scene's knobs onto the sky", func(t *testing.T) {
		reset()
		sc := New(nil)
		sc.Cfg.MinCycleSeconds, sc.Cfg.MaxCycleSeconds = 2, 3
		sc.Cfg.MinFadeSeconds, sc.Cfg.MaxFadeSeconds = 0.3, 0.7
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		if got := stars.ActiveTwinkle(); got != sc.Cfg.Twinkle() {
			t.Fatalf("the staged sky breathes %+v, want the scene's knobs %+v", got, sc.Cfg.Twinkle())
		}
	})
	t.Run("happy: Use is what New plays on the first Start", func(t *testing.T) {
		reset()
		c := DefaultConfig()
		c.MinCycleSeconds, c.MaxCycleSeconds = 1, 2
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		sc := New(nil)
		if sc.Cfg != c {
			t.Fatalf("New copied %+v, want the active knobs %+v", sc.Cfg, c)
		}
	})
	t.Run("happy: Start after Stop replays with the current knobs", func(t *testing.T) {
		reset()
		sc := New(nil)
		sc.Start()
		_ = paint(sc)
		sc.Cfg.MinCycleSeconds, sc.Cfg.MaxCycleSeconds = 1, 1.5
		sc.Stop()
		sc.Start()
		_ = paint(sc)
		want := sc.Cfg.Twinkle()
		if got := stars.ActiveTwinkle(); got != want {
			t.Fatalf("the replay must rebuild from the current knobs, sky %+v want %+v", got, want)
		}
	})
	t.Run("unhappy: broken knobs still stage a show — the sky keeps its last good breath", func(t *testing.T) {
		reset()
		held := stars.ActiveTwinkle()
		sc := New(nil)
		sc.Cfg.MinCycleSeconds, sc.Cfg.MaxCycleSeconds = 9, 2
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		if inkCells(opening, ie.BlueInk) == 0 || len(starCells(opening)) == 0 {
			t.Fatal("broken knobs must not black out the stage")
		}
		if got := stars.ActiveTwinkle(); got != held {
			t.Fatalf("broken knobs moved the sky to %+v", got)
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
		before := starCells(paint(held))
		held.Update(0)
		held.Update(-3)
		after := starCells(paint(held))
		if len(before) != len(after) {
			t.Fatal("dt<=0 must hold the frame")
		}
	})
}
