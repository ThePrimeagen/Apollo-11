package landing

// Tests written FIRST: the landing scene is a portable one-scene bill.
// Seven live knobs retune it 50ms at a time: LandSeconds (top-to-bottom
// fall), DustStart (offset from t=0), DustRun (how long the pad cloud
// blows), and the four fire stage offsets from t=0. Play is Start after
// Stop: a fresh craft from the current knobs. StageSeconds stays a
// code constant for the stock fire defaults.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/bigstar"
	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
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

func throttleLead() float64 { return 3 * StageSeconds }

func TestLandingKnobs(t *testing.T) {
	t.Run("happy: the eight knobs are the landing's timing", func(t *testing.T) {
		if LandSeconds <= 0 {
			t.Fatal("LandSeconds is how fast the lander lands — it must be a duration")
		}
		if DustStart < 0 {
			t.Fatal("DustStart is the offset from t=0 — it must not be negative")
		}
		if DustRun <= 0 {
			t.Fatal("DustRun is how long the pad dust blows — it must be a duration")
		}
		if DustLoss <= 0 {
			t.Fatal("DustLoss is how fast specks leave as the engines cut — it must be a rate")
		}
		if Fire75 < 0 || Fire50 < 0 || Fire25 < 0 || FireOff < 0 {
			t.Fatal("fire stage offsets are from t=0 — they must not be negative")
		}
		if StepSeconds != 0.050 {
			t.Fatalf("live knobs step %v, want 50ms", StepSeconds)
		}
	})
	t.Run("unhappy: the knobs are not the walkthrough's old 3s lead or a two-kick fade", func(t *testing.T) {
		if 3*StageSeconds >= 3 {
			t.Fatal("the booster must step down faster than three one-second stages")
		}
		if LandSeconds == DustStart {
			t.Fatal("land duration and dust start must be independent knobs")
		}
		if DustRun == LandSeconds {
			t.Fatal("dust run and land duration must be independent knobs")
		}
		if Fire75 == FireOff {
			t.Fatal("¾ and engine-off must be independent offsets")
		}
		if DustLoss == DustRun {
			t.Fatal("particle loss and dust run are different units — they must not share a value by accident")
		}
	})
}

func TestLandingBill(t *testing.T) {
	t.Run("happy: the bill is the one landing scene", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the landing bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "landing" {
			t.Fatalf("the scene is %q, want landing", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the landing has no performer")
		}
	})
	t.Run("unhappy: a second scene is not hiding on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "landing" {
			t.Fatalf("the show opens on %d %q, want one landing", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after landing there is nothing left")
		}
	})
}

func TestLandingScene(t *testing.T) {
	t.Cleanup(Reset)
	t.Cleanup(stars.ResetTwinkle)
	t.Run("happy: a huge moon horizon the lander comes down onto", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		if moonBGRows(opening, stageW/2) == 0 {
			t.Fatal("the landing scene must show the moon as a background floor")
		}
		if strings.ContainsRune(opening.Render(), '▟') {
			t.Fatal("at t=0 the lander must still be off the top")
		}
		if moonBGRows(opening, 0) != moon.HorizonEdgeRows {
			t.Fatalf("left edge holds %d moon rows, want %d", moonBGRows(opening, 0), moon.HorizonEdgeRows)
		}
		if moonBGRows(opening, stageW/2) != moon.HorizonCenterRows {
			t.Fatalf("center holds %d moon rows, want %d", moonBGRows(opening, stageW/2), moon.HorizonCenterRows)
		}
		tick(sc, LandSeconds-0.2)
		if !strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("near the pad the north hull must be on stage")
		}
		tick(sc, throttleLead()/6+0.5)
		landed := paint(sc)
		if !strings.ContainsRune(landed.Render(), '▟') {
			t.Fatal("at touchdown the north hull must sit on the surface")
		}
		if strings.ContainsRune(landed.Render(), '▌') {
			t.Fatal("the landing craft must stay north-facing")
		}
		if hotBraille(landed) {
			t.Fatal("at touchdown the booster must cut off — only gray pad dust may remain")
		}
	})
	t.Run("happy: dust starts at DustStart and keeps blowing until the engines cut", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.DustStart = 0.5
		sc.Cfg.DustRun = 5.0
		sc.Cfg.Fire75 = 1.5
		sc.Cfg.Fire50 = 2.0
		sc.Cfg.Fire25 = 2.5
		sc.Cfg.FireOff = 3.0
		sc.Cfg.DustLoss = 0.05
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.7)
		if l, r := offHullDust(paint(sc)); !l || !r {
			t.Fatalf("when the dust offset arrives the pad must kick dust both ways, left=%v right=%v", l, r)
		}
		tick(sc, 0.6)
		if l, r := offHullDust(paint(sc)); !l || !r {
			t.Fatalf("before the first engine cut the cloud must still blow both ways, left=%v right=%v", l, r)
		}
	})
	t.Run("happy: after the engines start cutting the cloud thins, it does not blink out", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.DustStart = 0.2
		sc.Cfg.DustRun = 8.0
		sc.Cfg.Fire75 = 0.5
		sc.Cfg.Fire50 = 1.0
		sc.Cfg.Fire25 = 1.5
		sc.Cfg.FireOff = 2.0
		sc.Cfg.DustLoss = 0.04
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.55)
		if l, r := offHullDust(paint(sc)); !l || !r {
			t.Fatal("test premise: the pad must already be dusty when the engines start cutting")
		}
		tick(sc, 0.2)
		if l, r := offHullDust(paint(sc)); !l || !r {
			t.Fatal("0.2s into the drain the blown fringe must still be in the air — a taper, not a blink")
		}
	})
	t.Run("unhappy: no dust before DustStart, and a high loss clears the pad after the engines cut", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.DustStart = 0.5
		sc.Cfg.DustRun = 8.0
		sc.Cfg.Fire75 = 0.6
		sc.Cfg.Fire50 = 0.7
		sc.Cfg.Fire25 = 0.8
		sc.Cfg.FireOff = 0.9
		sc.Cfg.DustLoss = 2.0
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.3)
		if l, r := offHullDust(paint(sc)); l || r {
			t.Fatal("no dust may kick before the start offset")
		}
		tick(sc, 1.2)
		if l, r := offHullDust(paint(sc)); l || r {
			t.Fatal("a 2/ms drain must have cleared the pad after the engines cut")
		}
	})
	t.Run("happy: Start after Stop replays from the top with the current knobs", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		_ = paint(sc)
		tick(sc, LandSeconds-0.2)
		if !strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("test premise: near the pad the hull must be on stage")
		}
		sc.Stop()
		sc.Start()
		opening := paint(sc)
		if strings.ContainsRune(opening.Render(), '▟') {
			t.Fatal("play must rewind the craft off the top")
		}
		sc.Stop()
	})
	t.Run("happy: Use is what New plays on the first Start, not only after replay", func(t *testing.T) {
		t.Cleanup(Reset)
		c := DefaultConfig()
		c.LandSeconds = 0.2
		if err := Use(c); err != nil {
			t.Fatal(err)
		}
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.25)
		if !strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("the first play must already use the active land duration")
		}
	})
	t.Run("happy: a land-duration nudge is what the next play uses", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.LandSeconds = 0.2
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.25)
		if !strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("a 0.2s land must already be on the pad")
		}
	})
	t.Run("happy: fire offsets are what the next play uses", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Fire75 = 0.2
		sc.Cfg.Fire50 = 0.4
		sc.Cfg.Fire25 = 0.6
		sc.Cfg.FireOff = 0.8
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 1.0)
		if hotBraille(paint(sc)) {
			t.Fatal("past fire-off the booster must already be dark — the pad is still seconds away")
		}
	})
	t.Run("unhappy: fire-off at t=0 never lights, even while the craft is still falling", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.FireOff = 0
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.5)
		if hotBraille(paint(sc)) {
			t.Fatal("fire-off at 0 must keep the booster dark on the way down")
		}
	})
	t.Run("unhappy: changing knobs mid-flight does not teleport the in-flight craft", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, LandSeconds-0.2)
		if !strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("test premise: near the pad the hull must be on stage")
		}
		sc.Cfg.LandSeconds = 0.2
		tick(sc, 0.1)
		if !strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("an in-flight craft must keep the land it launched with")
		}
	})
	t.Run("happy: the landing sky twinkles — stars fade while the scatter holds", func(t *testing.T) {
		stars.ResetTwinkle()
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		before := starCells(paint(sc))
		if len(before) == 0 {
			t.Fatal("the landing sky must show stars above the horizon")
		}
		// Only the sky above the ridge can twinkle — the hull and the
		// moon floor will cover stars on the way down.
		sky := map[[2]int]string{}
		for pos, ch := range before {
			if pos[1] < stageH/2 {
				sky[pos] = ch
			}
		}
		if len(sky) == 0 {
			t.Fatal("test premise: stars sit above the horizon")
		}
		var faded bool
		for i := 0; i < 8; i++ {
			tick(sc, 0.5)
			now := starCells(paint(sc))
			held := 0
			for pos, ch := range now {
				was, ok := sky[pos]
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
			for pos := range sky {
				if _, ok := now[pos]; !ok {
					faded = true
				}
			}
		}
		if !faded {
			t.Fatal("across a full cycle some star must fade out — the pad is not a freeze frame")
		}
	})
	t.Run("happy: one shooting star falls top-left to bottom-right, behind the moon, not behind the lander", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		var seen bool
		var leftX, leftY int
		for i := 0; i < 45; i++ {
			tick(sc, 1.0/30)
			scr := paint(sc)
			x, y, ok := findGlyph(scr, bigstar.CoreGlyph)
			if !ok {
				continue
			}
			if !seen {
				seen = true
				leftX, leftY = x, y
				if x > stageW/3 || y > stageH/3 {
					t.Fatalf("the meteor must enter top-left, at (%d,%d)", x, y)
				}
			} else if x < leftX || y < leftY {
				t.Fatalf("the meteor must travel down-right, (%d,%d) → (%d,%d)", leftX, leftY, x, y)
			}
			if isMoonBG(scr, x, y) {
				t.Fatalf("the meteor core at (%d,%d) sat on the moon floor — it must go behind the horizon", x, y)
			}
			clock := float64(i+1) / 30
			hr, hc := lander.LandPath(stageW, stageH, clock, sc.Cfg.LandSeconds)
			if x >= hc && x < hc+lander.BodyCols && y >= hr && y < hr+lander.BodyRows {
				t.Fatalf("the meteor core at (%d,%d) sits in the hull box — the lander must paint on top", x, y)
			}
		}
		if !seen {
			t.Fatal("the landing must fly one shooting star")
		}
		// Land the craft and make sure a hull cell is never the meteor core.
		tick(sc, LandSeconds)
		scr := paint(sc)
		if x, y, ok := findGlyph(scr, bigstar.CoreGlyph); ok {
			hr, hc := lander.LandPath(stageW, stageH, 45.0/30+LandSeconds, sc.Cfg.LandSeconds)
			if x >= hc && x < hc+lander.BodyCols && y >= hr && y < hr+lander.BodyRows {
				t.Fatal("the meteor must sit behind the lander, not on top of the hull")
			}
		}
	})
	t.Run("happy: right before touchdown the side says 1202, 1202, then LAND", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Code1At, sc.Cfg.Code1Hold = 0.4, 0.4
		sc.Cfg.Code2At, sc.Cfg.Code2Hold = 1.0, 0.4
		sc.Cfg.LandCaptionAt, sc.Cfg.LandCaptionHold = 1.6, 0.8
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.5)
		if !hasBanner(paint(sc), "1202") {
			t.Fatal("the first 1202 must be up before the pad")
		}
		if hasBanner(paint(sc), "LAND") {
			t.Fatal("LAND waits for touchdown")
		}
		tick(sc, 0.7)
		if !hasBanner(paint(sc), "1202") {
			t.Fatal("the second 1202 must follow the first")
		}
		tick(sc, 0.7)
		if !hasBanner(paint(sc), "LAND") {
			t.Fatal("touchdown must say LAND — 1202, 1202, LAND")
		}
		if hasBanner(paint(sc), "1201") {
			t.Fatal("the pad scene does not flash 1201 — that alarm already played on the way down")
		}
	})
	t.Run("unhappy: captions stay dark before their offset, and a looping meteor is refused", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Code1At, sc.Cfg.Code1Hold = 0.8, 0.4
		sc.Cfg.Code2At, sc.Cfg.Code2Hold = 2.0, 0.4
		sc.Cfg.LandCaptionAt, sc.Cfg.LandCaptionHold = 3.0, 1.0
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.3)
		if hasBanner(paint(sc), "1202") || hasBanner(paint(sc), "LAND") {
			t.Fatal("no caption may paint before its offset")
		}
		// One meteor: after a long burn it must not reappear on the left.
		var cores int
		leftHits := 0
		for i := 0; i < 240; i++ {
			tick(sc, 1.0/30)
			x, _, ok := findGlyph(paint(sc), bigstar.CoreGlyph)
			if !ok {
				continue
			}
			cores++
			if cores > 8 && x < stageW/4 {
				leftHits++
			}
		}
		if leftHits > 4 {
			t.Fatal("the landing meteor must not loop back to the left")
		}
	})
	t.Run("unhappy: the horizon is not a round disc in the middle of the sky", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		scr := paint(sc)
		for y := 0; y < stageH/2; y++ {
			for x := 0; x < stageW; x++ {
				if isMoonBG(scr, x, y) {
					t.Fatal("the landing moon must sit on the bottom, not as a disc in the sky")
				}
			}
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		sc := New(nil)
		sc.Start()
		sc.Update(1)
		sc.Stop()
	})
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

func findGlyph(scr *screenplay.Screen, g rune) (x, y int, ok bool) {
	for y = 0; y < stageH; y++ {
		for x = 0; x < stageW; x++ {
			c := scr.Cell(x, y)
			if c != nil && c.Content == string(g) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func hasBanner(scr *screenplay.Screen, text string) bool {
	lines, err := termfont.Lines(3, text)
	if err != nil {
		return false
	}
	body := scr.Render()
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if !strings.Contains(body, trim) {
			return false
		}
	}
	return true
}
