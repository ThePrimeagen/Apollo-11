package fall

// Tests written FIRST: the fall scene is the spacelander dropping
// from the top of the stage to the bottom under a twinkling sky.
// The stars hold their scatter and some fade in and out. Play is
// Start after Stop: a fresh craft from the current drop knob.

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/caption"
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

func mainFallKnobs() Config {
	c := DefaultConfig()
	c.DropSeconds = 2.25
	c.Hold1, c.Hold2, c.Hold3 = 0.8, 0.8, 0.8
	return c
}

func hullGlyphRow(scr *screenplay.Screen) int {
	_, h := scr.Size()
	w, _ := scr.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := scr.Cell(x, y)
			if c != nil && strings.ContainsRune(c.Content, '▟') {
				return y
			}
		}
	}
	return -1
}

func readAlt(scr *screenplay.Screen) (float64, bool) {
	if scr == nil {
		return 0, false
	}
	w, h := scr.Size()
	if h < 1 {
		return 0, false
	}
	var b strings.Builder
	for x := 0; x < w && x < 24; x++ {
		c := scr.Cell(x, 0)
		if c == nil || c.Content == "" {
			b.WriteByte(' ')
			continue
		}
		b.WriteString(c.Content)
	}
	return ParseElevation(b.String())
}

func waitCard(t *testing.T, sc screenplay.Scene, text string, limitSec float64) int {
	t.Helper()
	const dt = 1.0 / 30
	for i := 0; i < int(limitSec/dt)+2; i++ {
		sc.Update(dt)
		scr := paint(sc)
		if caption.Painted(scr, text) {
			row := hullGlyphRow(scr)
			if row < 0 {
				t.Fatalf("%s painted with no hull on stage", text)
			}
			return row
		}
	}
	t.Fatalf("timed out waiting for %s", text)
	return -1
}

func waitHullMove(t *testing.T, sc screenplay.Scene, fromRow int, limitSec float64) {
	t.Helper()
	const dt = 1.0 / 30
	for i := 0; i < int(limitSec/dt)+2; i++ {
		sc.Update(dt)
		row := hullGlyphRow(paint(sc))
		if row >= 0 && row != fromRow {
			return
		}
	}
	t.Fatalf("timed out waiting for the hull to leave row %d after the hold", fromRow)
}

func TestFallAlarms(t *testing.T) {
	t.Cleanup(Reset)
	t.Cleanup(stars.ResetTwinkle)
	t.Run("happy: at about 1/3 stage height MAIN knobs freeze the fall and blink 1202 on the right", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg = mainFallKnobs()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		if caption.Painted(paint(sc), "1202") {
			t.Fatal("the first 1202 must wait for the first hold")
		}
		var saw bool
		const dt = 1.0 / 30
		for i := 0; i < 200; i++ {
			sc.Update(dt)
			scr := paint(sc)
			if !caption.Painted(scr, "1202") {
				continue
			}
			saw = true
			row := hullGlyphRow(scr)
			if row < 0 {
				t.Fatal("the first 1202 must catch the hull on stage")
			}
			want := AlarmRows(stageH)
			if row < want[0]-2 || row > want[0]+lander.BodyRows {
				t.Fatalf("first 1202 hull glyph row %d, want around 1/3 stage (alarm row %d)", row, want[0])
			}
			starsHeld := starCells(scr)
			heldRow := row
			var off, on bool
			for j := 0; j < 24; j++ {
				sc.Update(dt)
				now := paint(sc)
				if hullGlyphRow(now) != heldRow {
					t.Fatalf("during the first hold the hull moved %d → %d — the world must freeze", heldRow, hullGlyphRow(now))
				}
				if got := starCells(now); len(starsHeld) > 0 {
					for pos, ch := range starsHeld {
						if nowCh, ok := got[pos]; !ok || nowCh != ch {
							t.Fatal("during the hold the stars must freeze — only the card blinks")
						}
					}
				}
				if caption.Painted(now, "1202") {
					on = true
				} else {
					off = true
				}
			}
			if !on || !off {
				t.Fatal("the 1202 card must blink on the right while the world is frozen")
			}
			if caption.Painted(scr, "1201") {
				t.Fatal("1201 must not lead the two 1202s")
			}
			break
		}
		if !saw {
			t.Fatal("MAIN knobs must paint 1202 about a third of the way down")
		}
	})
	t.Run("happy: two rows later a second 1202, then two rows later 1201", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg = mainFallKnobs()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		first := waitCard(t, sc, "1202", 4)
		waitHullMove(t, sc, first, 2)
		second := waitCard(t, sc, "1202", 4)
		waitHullMove(t, sc, second, 2)
		third := waitCard(t, sc, "1201", 4)
		if second-first < 1 || second-first > 4 {
			t.Fatalf("second 1202 hull row %d, first %d — want about two rows down", second, first)
		}
		if third-second < 1 || third-second > 4 {
			t.Fatalf("1201 hull row %d, second 1202 %d — want about two rows down again", third, second)
		}
	})
	t.Run("happy: top-left elevation matches the flight altitudes and lerps with the hull row", func(t *testing.T) {
		// Canonical first-three-alarm altitudes this repo already uses
		// (cmd/lander script; descent markers). website_spec.md quotes
		// ~29,000 ft for the second P63 1202; lander/descent keep 30,900.
		if Alarm1AltFt != 33500 || Alarm2AltFt != 30900 || Alarm3AltFt != 3000 {
			t.Fatalf("alarm altitudes %v/%v/%v, want 33500/30900/3000", Alarm1AltFt, Alarm2AltFt, Alarm3AltFt)
		}
		sc := New(nil)
		sc.Cfg = mainFallKnobs()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		check := func(want float64, label string) {
			t.Helper()
			scr := paint(sc)
			alt, ok := readAlt(scr)
			if !ok {
				t.Fatalf("%s: top-left elevation is missing", label)
			}
			if math.Abs(alt-want) > 50 {
				t.Fatalf("%s elevation %v, want the flight altitude %v ft", label, alt, want)
			}
		}
		r1 := waitCard(t, sc, "1202", 4)
		check(Alarm1AltFt, "first 1202")
		waitHullMove(t, sc, r1, 2)
		r2 := waitCard(t, sc, "1202", 4)
		check(Alarm2AltFt, "second 1202")
		waitHullMove(t, sc, r2, 2)
		waitCard(t, sc, "1201", 4)
		check(Alarm3AltFt, "1201")

		const dt = 1.0 / 30
		sc2 := New(nil)
		sc2.Cfg = mainFallKnobs()
		sc2.Start()
		defer sc2.Stop()
		_ = paint(sc2)
		var lastAlt float64
		var lastRow int
		var have bool
		for i := 0; i < 200; i++ {
			sc2.Update(dt)
			scr := paint(sc2)
			alt, ok := readAlt(scr)
			row := hullGlyphRow(scr)
			if !ok || row < 0 {
				continue
			}
			if have {
				jump := math.Abs(alt - lastAlt)
				moved := math.Abs(float64(row - lastRow))
				if jump > 20000 && moved < 2 {
					t.Fatalf("elevation jumped %v ft while the hull moved %v rows — must lerp, not step the full 30k", jump, moved)
				}
			}
			lastAlt, lastRow, have = alt, row, true
		}
		if !have {
			t.Fatal("elevation must paint while the hull is on stage")
		}
	})
	t.Run("unhappy: stock walkthrough fall stays a plain drop; zero or negative holds and a stopped scene never panic", func(t *testing.T) {
		stock := New(nil)
		stock.Start()
		_ = paint(stock)
		tick(stock, lander.DropSeconds/2)
		scr := paint(stock)
		if caption.Painted(scr, "1202") || caption.Painted(scr, "1201") {
			t.Fatal("stock walkthrough fall must not flash the MAIN alarm cards")
		}
		if _, ok := readAlt(scr); ok {
			t.Fatal("stock walkthrough fall must not paint the MAIN elevation HUD")
		}
		opening := hullGlyphRow(paint(stock))
		tick(stock, 0.4)
		if hullGlyphRow(paint(stock)) == opening && opening >= 0 {
			t.Fatal("stock walkthrough fall must keep dropping — no MAIN freeze")
		}
		stock.Stop()

		sc := New(nil)
		sc.Cfg.Hold1, sc.Cfg.Hold2, sc.Cfg.Hold3 = 0, -1, 0
		sc.Start()
		_ = paint(sc)
		sc.Update(1)
		if caption.Painted(paint(sc), "1202") {
			t.Fatal("zero and negative holds must skip the cards")
		}
		sc.Stop()
		sc.Start()
		sc.Update(1)
		sc.Stop()
	})
}
