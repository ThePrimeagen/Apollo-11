package fall

// Tests written FIRST: the fall scene is the spacelander dropping
// from the top of the stage to the bottom under a twinkling sky.
// The stars hold their scatter and some fade in and out. Play is
// Start after Stop: a fresh craft from the current drop knob.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

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

func TestFallBill(t *testing.T) {
	t.Run("happy: the bill is the one fall scene", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the fall bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "fall" {
			t.Fatalf("the scene is %q, want fall", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the fall has no performer")
		}
	})
	t.Run("unhappy: a second scene is not hiding on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "fall" {
			t.Fatalf("the show opens on %d %q, want one fall", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the fall there is nothing left")
		}
	})
}

func TestFallScene(t *testing.T) {
	t.Cleanup(Reset)
	t.Cleanup(stars.ResetTwinkle)
	t.Run("happy: the curtain rises on a north hull off the top under twinkling stars", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		if strings.ContainsRune(opening.Render(), '▟') {
			t.Fatal("at t=0 the lander must still be off the top")
		}
		if len(starCells(opening)) == 0 {
			t.Fatal("the fall plays under the stars")
		}
		tick(sc, lander.DropSeconds/2)
		mid := paint(sc)
		if !strings.ContainsRune(mid.Render(), '▟') {
			t.Fatal("mid-drop the north hull must be on stage")
		}
		if strings.ContainsRune(mid.Render(), '▌') {
			t.Fatal("the falling craft must stay north-facing")
		}
	})
	t.Run("happy: the sky twinkles — stars fade while the scatter holds", func(t *testing.T) {
		stars.ResetTwinkle()
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		before := starCells(paint(sc))
		if len(before) == 0 {
			t.Fatal("test premise: the opening sky holds stars")
		}
		var faded bool
		for i := 0; i < 8; i++ {
			tick(sc, 0.5)
			now := starCells(paint(sc))
			held := 0
			for pos, ch := range now {
				was, ok := before[pos]
				if !ok {
					continue
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
	t.Run("happy: Use is what New plays on the first Start", func(t *testing.T) {
		t.Cleanup(Reset)
		c := DefaultConfig()
		c.DropSeconds = 0.4
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		sc := New(nil)
		if sc.Cfg != c {
			t.Fatalf("New copied %+v, want the active knobs %+v", sc.Cfg, c)
		}
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.25)
		if !strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("a 0.4s drop must already have the hull on stage")
		}
	})
	t.Run("happy: Start after Stop replays from the top with the current knobs", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		_ = paint(sc)
		tick(sc, lander.DropSeconds/2)
		if !strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("test premise: mid-drop the hull must be on stage")
		}
		sc.Stop()
		sc.Start()
		if strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("play must rewind the craft off the top")
		}
		sc.Stop()
	})
	t.Run("unhappy: there is no moon floor, no 1202 card, and a stopped scene never panics", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, lander.DropSeconds/2)
		scr := paint(sc)
		for y := 0; y < stageH; y++ {
			for x := 0; x < stageW; x++ {
				c := scr.Cell(x, y)
				if c == nil || c.Style.Bg == nil {
					continue
				}
				ic, ok := c.Style.Bg.(ansi.IndexedColor)
				if !ok {
					continue
				}
				n := int(ic)
				if n == 251 || n == 247 || n == 243 || n == 240 || n == 249 {
					t.Fatal("the fall is in space — no moon floor")
				}
			}
		}
		if strings.Contains(scr.Render(), "1202") {
			t.Fatal("the plain fall does not flash program alarms — that is the prog scene")
		}
		sc.Stop()
		sc.Start()
		sc.Update(1)
		sc.Stop()
	})
}
