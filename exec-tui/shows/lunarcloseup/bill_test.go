package lunarcloseup

// Tests written FIRST: 02. Walkthrough is a composable five-scene
// bill. Scene one, "pause": the still sky alone — a blank stage
// the audience can sit on for as long as it likes; only the cut moves
// the show along. Scene two, "Lunar Lander Close-Up": the zoomed-in
// Apollo craft slides in from the right the moment the curtain rises
// — no baked-in wait — hull only, cold engine, the sky surging from
// rest to a 1.25 peak then settling to cruise so the hull holds
// center. Scene three, "fire": the parked craft lights the booster
// and the stars slow by 60% over five seconds. Scene four, "fall":
// the north-facing lander, fire down, drops from the top of the
// stage to the bottom. Scene five, "landing": a huge moon horizon
// (five rows high in the middle, one row at the edges) and the
// north-facing lander coming down onto it. After the last scene
// there is nothing left.
//
// One stars.Continuity seeds every scene's sky, so a cut never jumps
// or skips a single star: each new starfield opens on the exact frame
// the last one left on screen.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 27
)

var sceneNames = []string{"pause", "Lunar Lander Close-Up", "fire", "fall", "landing"}

func render(sc screenplay.Scene) string {
	return paint(sc).Render()
}

func paint(sc screenplay.Scene) *screenplay.Screen {
	scr := screenplay.NewScreen(stageW, stageH)
	sc.Render(scr)
	return scr
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

func tick(sc screenplay.Scene, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		sc.Update(dt)
	}
}

// openShow composes one bill into a screenplay and stages the opening
// scene on a test-sized screen.
func openShow() (*screenplay.Screenplay, *screenplay.Screen) {
	p := screenplay.Compose(Bill())
	p.Start()
	scr := screenplay.NewScreen(stageW, stageH)
	p.Render(scr)
	return p, scr
}

// run plays the show frame by frame, the way the runner does.
func run(p *screenplay.Screenplay, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		p.Update(dt)
	}
}

// cut stops the old scene and stages the new one, like one space press.
func cut(p *screenplay.Screenplay, scr *screenplay.Screen) {
	p.Next()
	p.Render(scr)
}

// frame is the rendered screen this instant.
func frame(p *screenplay.Screenplay, scr *screenplay.Screen) string {
	p.Render(scr)
	return scr.Render()
}

// starCells maps every star-glyph cell of the screen to its rune.
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

func hasStar(v string) bool {
	for _, g := range stars.Glyphs {
		if strings.ContainsRune(v, g) {
			return true
		}
	}
	return false
}

func hasFire(v string) bool {
	return strings.ContainsAny(v, "⠁⠒⠶")
}

// hotBraille reports a braille fire cell wearing the booster's hot
// inks — the landing dust wears only the gray ramp, so this is the
// plume and nothing else.
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

// offHullDust scans the screen beyond the hull columns — where no
// fire or hull rune can reach (the north plume lives in a 12-column
// box under the bell) — for dust-ladder glyphs: braille or shades.
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

func hasHull(v string) bool {
	return strings.ContainsRune(v, '▌') || strings.ContainsRune(v, '▟')
}

func TestLunarCloseUpBill(t *testing.T) {
	t.Cleanup(landing.Reset)
	t.Run("happy: the bill is five scenes in playing order", func(t *testing.T) {
		b := Bill()
		if len(b) != 5 {
			t.Fatalf("the walkthrough holds %d scenes, want 5", len(b))
		}
		for i, want := range sceneNames {
			if b[i].Name != want {
				t.Fatalf("scene %d is %q, want %q", i+1, b[i].Name, want)
			}
			if b[i].Scene == nil {
				t.Fatalf("scene %q has no performer", want)
			}
		}
	})
	t.Run("happy: the pause is a blank stage under still stars", func(t *testing.T) {
		sc := Bill()[0].Scene
		sc.Start()
		defer sc.Stop()
		before := paint(sc)
		if !hasStar(before.Render()) {
			t.Fatal("the pause plays under the stars")
		}
		tick(sc, 2.0)
		after := paint(sc)
		if before.Render() != after.Render() {
			t.Fatal("the pause sky must hold still — a freeze frame, not a drift")
		}
	})
	t.Run("unhappy: the pause never admits the craft or the fire, however long it sits", func(t *testing.T) {
		sc := Bill()[0].Scene
		sc.Start()
		defer sc.Stop()
		_ = render(sc)
		tick(sc, 10.0)
		v := render(sc)
		if hasHull(v) {
			t.Fatal("the pause holds no craft — the lander waits for the cut")
		}
		if hasFire(v) {
			t.Fatal("the pause holds no fire")
		}
	})
	t.Run("happy: scene two flies the hull in the moment the curtain rises", func(t *testing.T) {
		sc := Bill()[1].Scene
		sc.Start()
		defer sc.Stop()
		if strings.ContainsRune(render(sc), '▌') {
			t.Fatal("at t=0 the craft is still off the right wing")
		}
		tick(sc, 0.5)
		v := render(sc)
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("half a second in the hull must already be sliding on stage — the pause scene owns the wait now")
		}
		if hasFire(v) {
			t.Fatal("the fly-in must run a dark engine — no booster fire yet")
		}
	})
	t.Run("unhappy: scene two's engine stays cold through the park", func(t *testing.T) {
		sc := Bill()[1].Scene
		sc.Start()
		defer sc.Stop()
		_ = render(sc)
		tick(sc, lander.FlyInSeconds+1)
		v := render(sc)
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("after the fly-in the hull must be parked on stage")
		}
		if hasFire(v) {
			t.Fatal("the close-up must park with a cold engine")
		}
	})
	t.Run("happy: the pause→fly-in cut renders the identical star frame", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		run(p, 2.0)
		before := frame(p, scr)
		if !p.Next() {
			t.Fatal("the pause must cut to the fly-in")
		}
		after := frame(p, scr)
		if before != after {
			t.Fatal("the fly-in's first frame must be the pause's last frame — no star may jump at the cut")
		}
	})
	t.Run("unhappy: after that cut the sky keeps flying — continuity is not a freeze", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		run(p, 2.0)
		p.Next()
		after := frame(p, scr)
		run(p, 1.0)
		if frame(p, scr) == after {
			t.Fatal("one second into the fly-in the sky must have moved on")
		}
	})
	t.Run("happy: the fly-in→fire cut at the park is pixel-identical", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		cut(p, scr) // the fly-in, staged at t=0
		run(p, lander.FlyInSeconds)
		before := frame(p, scr)
		if !strings.ContainsRune(before, '▌') {
			t.Fatal("test premise: the hull must be parked before the cut")
		}
		if !p.Next() {
			t.Fatal("the fly-in must cut to the fire")
		}
		after := frame(p, scr)
		if before != after {
			t.Fatal("the fire scene must open on the very frame the fly-in parked — hull and every star in place")
		}
	})
	t.Run("unhappy: the fire scene then lights the booster instead of rewinding the sky", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		cut(p, scr)
		run(p, lander.FlyInSeconds)
		p.Next()
		opening := frame(p, scr)
		run(p, 2.0)
		burning := frame(p, scr)
		if burning == opening {
			t.Fatal("the fire scene's sky must keep crawling — slowed, never frozen")
		}
		if !hasFire(burning) {
			t.Fatal("two seconds in the booster must be lit")
		}
	})
	t.Run("happy: the fire→fall cut parks the sky — the drop twinkles instead of crawling", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		cut(p, scr)
		run(p, lander.FlyInSeconds)
		cut(p, scr) // fire
		run(p, 1.5)
		cut(p, scr) // fall
		opening := starCells(scr)
		if len(opening) == 0 {
			t.Fatal("test premise: the fall must show stars")
		}
		run(p, 0.2)
		p.Render(scr)
		held := 0
		for pos, ch := range starCells(scr) {
			if opening[pos] == ch {
				held++
			}
		}
		if held == 0 {
			t.Fatal("the fall sky must hold its scatter — twinkle, not another drift")
		}
	})
	t.Run("happy: the fall→landing cut keeps every star the horizon leaves visible", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		cut(p, scr)
		run(p, lander.FlyInSeconds)
		cut(p, scr) // fire
		run(p, 1.0)
		cut(p, scr) // fall
		run(p, lander.DropSeconds+0.5)
		p.Render(scr)
		before := starCells(scr)
		if len(before) == 0 {
			t.Fatal("test premise: the emptied fall stage must show stars")
		}
		cut(p, scr) // landing
		after := starCells(scr)
		if len(after) == 0 {
			t.Fatal("test premise: the landing sky must show stars above the horizon")
		}
		for pos, ch := range after {
			if before[pos] != ch {
				t.Fatalf("star at (%d,%d) jumped on the landing cut: %q -> %q", pos[0], pos[1], before[pos], ch)
			}
		}
	})
	t.Run("happy: the landing sky twinkles on the pad — scatter holds, breathers fade", func(t *testing.T) {
		t.Cleanup(stars.ResetTwinkle)
		p, scr := openShow()
		defer p.Stop()
		cut(p, scr)
		run(p, lander.FlyInSeconds)
		cut(p, scr)
		run(p, 1.0)
		cut(p, scr)
		run(p, lander.DropSeconds+0.5)
		cut(p, scr) // landing
		opening := starCells(scr)
		if len(opening) == 0 {
			t.Fatal("test premise: the landing sky must show stars")
		}
		sky := map[[2]int]string{}
		for pos, ch := range opening {
			if pos[1] < stageH/2 {
				sky[pos] = ch
			}
		}
		if len(sky) == 0 {
			t.Fatal("test premise: stars sit above the horizon")
		}
		var faded bool
		for i := 0; i < 8; i++ {
			run(p, 0.5)
			p.Render(scr)
			now := starCells(scr)
			held := 0
			for pos, ch := range now {
				if sky[pos] == ch {
					held++
				}
			}
			if held == 0 {
				t.Fatal("the pad sky must hold its scatter while it twinkles")
			}
			for pos := range sky {
				if _, ok := now[pos]; !ok {
					faded = true
				}
			}
		}
		if !faded {
			t.Fatal("the landing sky must twinkle — some star fades on the pad")
		}
	})
	t.Run("happy: scene three parks the west craft with the booster lit", func(t *testing.T) {
		sc := Bill()[2].Scene
		sc.Start()
		defer sc.Stop()
		_ = render(sc) // stage the cast
		tick(sc, 0.5)
		v := render(sc)
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("the fire scene must open with the west-facing craft already on stage")
		}
		if !hasFire(v) {
			t.Fatal("the fire scene must light the booster")
		}
		if !hasStar(v) {
			t.Fatal("the fire scene still plays under the stars")
		}
	})
	t.Run("happy: stock walkthrough fire stays parked — MAIN's knobs turn the sink on", func(t *testing.T) {
		sc := Bill()[2].Scene
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		open := westHullTop(paint(sc))
		if open < 0 {
			t.Fatal("test premise: the fire scene must show the west hull")
		}
		tick(sc, 4)
		mid := paint(sc)
		got := westHullTop(mid)
		if got < 0 {
			t.Fatal("the parked hull must still be on stage")
		}
		// The parked bobble rides ±1 cell; a sink would have dropped
		// several rows by now. Stay inside the park band.
		if got < open-1 || got > open+1 {
			t.Fatalf("stock fire hull top %d, want the park around %d — no sink without MAIN's knobs", got, open)
		}
		if !hasFire(mid.Render()) {
			t.Fatal("the parked craft must keep the booster lit")
		}
	})
	t.Run("happy: a fire show wearing a sink knob eases down once the booster is on", func(t *testing.T) {
		sc := NewFireShow(nil)
		sc.Cfg.SinkSeconds = 4
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		open := westHullTop(paint(sc))
		if open < 0 {
			t.Fatal("test premise: the lit hull must be on stage")
		}
		tick(sc, 3)
		mid := paint(sc)
		got := westHullTop(mid)
		if got < 0 {
			t.Fatal("mid-sink the west hull must still be on stage")
		}
		if got <= open {
			t.Fatalf("mid-sink hull top %d, want below the opening %d", got, open)
		}
		if !strings.ContainsRune(mid.Render(), '▌') {
			t.Fatal("the sinking craft must stay west-facing")
		}
		if !hasFire(mid.Render()) {
			t.Fatal("the sinking craft must keep the booster lit")
		}
	})
	t.Run("happy: scene three's sky slows 60% over five seconds", func(t *testing.T) {
		if stars.BrakeClock(5, 0.6, 5) != 3.5 {
			t.Fatalf("the fire scene's brake must cut 60 percent of speed over 5s (fly clock %g, want 3.5)", stars.BrakeClock(5, 0.6, 5))
		}
		if stars.BrakeClock(10, 0.6, 5) != 5.5 {
			t.Fatal("past the window the fire scene's sky must crawl at 40% speed")
		}
	})
	t.Run("happy: scene four drops a north-facing lander with fire, top to bottom", func(t *testing.T) {
		sc := Bill()[3].Scene
		sc.Start()
		defer sc.Stop()
		_ = render(sc)
		if strings.ContainsRune(render(sc), '▌') {
			t.Fatal("the falling craft must not wear the west-facing hull")
		}
		if strings.ContainsRune(render(sc), '▟') {
			t.Fatal("at t=0 the falling craft must still be off the top")
		}
		tick(sc, lander.DropSeconds/2)
		mid := render(sc)
		if !strings.ContainsRune(mid, '▟') {
			t.Fatal("mid-fall the north hull must be on stage")
		}
		if strings.ContainsRune(mid, '▌') {
			t.Fatal("the falling craft must stay north-facing")
		}
		if !hasFire(mid) {
			t.Fatal("the falling craft must keep the booster lit")
		}
		tick(sc, lander.DropSeconds/2)
		if strings.ContainsRune(render(sc), '▟') {
			t.Fatal("at the end of the drop the craft must have left the bottom")
		}
	})
	t.Run("happy: scene five is a huge moon horizon the lander comes down onto", func(t *testing.T) {
		sc := Bill()[4].Scene
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
		tick(sc, lander.LandSeconds-0.2)
		fireOnMoon := false
		hasPlume := false
		for i := 0; i < 8; i++ {
			sc.Update(1.0 / 30)
			near := paint(sc)
			if hasFire(near.Render()) {
				hasPlume = true
			}
			for y := 0; y < stageH; y++ {
				for x := 0; x < stageW; x++ {
					c := near.Cell(x, y)
					if c == nil || !strings.ContainsAny(c.Content, "⠁⠒⠶") {
						continue
					}
					if isMoonBG(near, x, y) {
						fireOnMoon = true
					}
				}
			}
		}
		if !hasPlume {
			t.Fatal("the plume must still be lit as the craft comes in")
		}
		if !fireOnMoon {
			t.Fatal("the plume must paint on top of the moon floor as the craft comes in")
		}
		tick(sc, lander.LandThrottleLead/6+0.5)
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
	t.Run("happy: the landing kicks dust at DustStart and still blows as the engines start cutting", func(t *testing.T) {
		sc := Bill()[4].Scene
		sc.Start()
		defer sc.Stop()
		_ = paint(sc) // stage the cast
		tick(sc, landing.DustStart+0.2)
		if l, r := offHullDust(paint(sc)); !l || !r {
			t.Fatalf("when the dust offset arrives the pad must kick dust both ways, left=%v right=%v", l, r)
		}
		tick(sc, 0.3)
		if l, r := offHullDust(paint(sc)); !l || !r {
			t.Fatalf("as the engines cut the cloud must still blow both ways — a taper, not a blink, left=%v right=%v", l, r)
		}
	})
	t.Run("unhappy: no dust before DustStart, a fast drain clears after the engines cut, and the fall never kicks", func(t *testing.T) {
		t.Cleanup(landing.Reset)
		c := landing.DefaultConfig()
		c.DustLoss = 2.0
		if err := landing.Use(c); err != nil {
			t.Fatal(err)
		}
		sc := Bill()[4].Scene
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, landing.DustStart-0.2)
		if l, r := offHullDust(paint(sc)); l || r {
			t.Fatal("no dust may kick before the start offset")
		}
		tick(sc, 1.5)
		if l, r := offHullDust(paint(sc)); l || r {
			t.Fatal("a 2/ms drain must have cleared the pad after the engines cut")
		}
		fall := Bill()[3].Scene
		fall.Start()
		defer fall.Stop()
		_ = paint(fall)
		for i := 0; i < int(lander.DropSeconds*30); i++ {
			fall.Update(1.0 / 30)
			if i%15 != 0 {
				continue
			}
			if l, r := offHullDust(paint(fall)); l || r {
				t.Fatal("a falling craft kicks no dust — the pad does that once, from the first step-down through off")
			}
		}
	})
	t.Run("happy: the walkthrough landing plays the active scene knobs on the first curtain", func(t *testing.T) {
		t.Cleanup(landing.Reset)
		c := landing.DefaultConfig()
		c.LandSeconds = 0.2
		if err := landing.Use(c); err != nil {
			t.Fatal(err)
		}
		sc := Bill()[4].Scene
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.25)
		if !strings.ContainsRune(paint(sc).Render(), '▟') {
			t.Fatal("02. Walkthrough must land at the saved land duration")
		}
	})
	t.Run("unhappy: Reset restores stock timing on the next landing scene", func(t *testing.T) {
		t.Cleanup(landing.Reset)
		c := landing.DefaultConfig()
		c.LandSeconds = 0.2
		if err := landing.Use(c); err != nil {
			t.Fatal(err)
		}
		landing.Reset()
		sc := Bill()[4].Scene.(*landing.Show)
		if sc.Cfg.LandSeconds != landing.LandSeconds {
			t.Fatalf("after Reset the walkthrough landing is %v, want stock %v", sc.Cfg.LandSeconds, landing.LandSeconds)
		}
	})
	t.Run("happy: the composed show walks the five scenes and then has nothing left", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		if p.Len() != 5 || p.CurrentName() != "pause" {
			t.Fatalf("the show opens on %d %q, want five starting on pause", p.Len(), p.CurrentName())
		}
		for i, want := range sceneNames[1:] {
			if !p.Next() || p.CurrentName() != want {
				t.Fatalf("cut %d must land on %q, got %q", i+1, want, p.CurrentName())
			}
		}
		if p.Next() {
			t.Fatal("after landing there is nothing left — the show ends")
		}
		p.Stop()
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		for _, e := range Bill() {
			e.Scene.Start()
			e.Scene.Update(1)
			e.Scene.Stop()
		}
	})
	t.Run("unhappy: the walkthrough is not the four-scene premiere", func(t *testing.T) {
		for _, e := range Bill() {
			switch e.Name {
			case "arrival", "dsky", "descent orbit", "the end":
				t.Fatalf("the walkthrough bill must not carry premiere scene %q", e.Name)
			}
		}
		sc := Bill()[1].Scene
		sc.Start()
		defer sc.Stop()
		sc.Update(lander.FlyInSeconds)
		v := render(sc)
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in the walkthrough")
		}
		if strings.Contains(v, "THE END") || strings.Contains(v, "___") {
			t.Fatal("the end card does not appear in the walkthrough")
		}
	})
	t.Run("unhappy: the horizon is not a round disc in the middle of the sky", func(t *testing.T) {
		sc := Bill()[4].Scene
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
}

// hullLeftCol is the leftmost column carrying the west hull's '▌'
// glyph in a rendered frame, ANSI stripped — parked, that column
// never moves, so it tells a settled craft from a sliding one.
func westHullTop(scr *screenplay.Screen) int {
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			c := scr.Cell(x, y)
			if c != nil && strings.ContainsRune(c.Content, '▌') {
				return y
			}
		}
	}
	return -1
}

func hullLeftCol(t *testing.T, v string) int {
	t.Helper()
	best := -1
	for _, line := range strings.Split(ansi.Strip(v), "\n") {
		for c, ch := range []rune(line) {
			if ch == '▌' && (best < 0 || c < best) {
				best = c
			}
		}
	}
	if best < 0 {
		t.Fatal("no hull on stage")
	}
	return best
}

// Tests written FIRST: the close-up entry grows its editable face —
// two knobs: the fly-in, which paces both the sliding sky and the
// hull's westbound glide, and the rush, the peak fly speed the sky
// surges to from rest before it settles back to cruise. The fire
// entry grows two: how far the stars brake and how long the brake
// takes. Stock knobs fly the stock show; the numbers are the
// operator's, verbatim — a nudge below zero stands — and no two
// bills share knobs.
func TestCloseupShow(t *testing.T) {
	t.Run("happy: the close-up entry is the tunable show at the stock fly-in and rush", func(t *testing.T) {
		sc, ok := Bill()[1].Scene.(*CloseupShow)
		if !ok {
			t.Fatalf("the close-up entry is %T, want the close-up show", Bill()[1].Scene)
		}
		if sc.Cfg != DefaultCloseupConfig() {
			t.Fatalf("a fresh show carries %+v, want stock", sc.Cfg)
		}
		if DefaultCloseupConfig().FlyInSeconds != lander.FlyInSeconds {
			t.Fatalf("stock fly-in is %v, want the lander const %v", DefaultCloseupConfig().FlyInSeconds, lander.FlyInSeconds)
		}
		if DefaultCloseupConfig().RushPeak != 1.25 {
			t.Fatalf("stock rush is %v, want 1.25", DefaultCloseupConfig().RushPeak)
		}
	})
	t.Run("happy: the knob face is fly-in then rush", func(t *testing.T) {
		c := DefaultCloseupConfig()
		if c.KnobCount() != 2 || c.KnobLabel(0) != "fly-in" || c.KnobLabel(1) != "rush" {
			t.Fatalf("the close-up carries %d knobs labeled %q/%q, want fly-in then rush", c.KnobCount(), c.KnobLabel(0), c.KnobLabel(1))
		}
		if c.Value(0) != c.FlyInSeconds {
			t.Fatalf("the fly-in knob reads %v, want the config", c.Value(0))
		}
		if c.Value(1) != c.RushPeak {
			t.Fatalf("the rush knob reads %v, want the config", c.Value(1))
		}
		c.Nudge(0, -2)
		if want := lander.FlyInSeconds - 0.5; c.FlyInSeconds != want {
			t.Fatalf("two fly-in steps down read %v, want %v", c.FlyInSeconds, want)
		}
		c.Nudge(1, 1)
		if want := 1.25 + 0.05; c.RushPeak != want {
			t.Fatalf("one rush step reads %v, want %v", c.RushPeak, want)
		}
	})
	t.Run("happy: the close-up sky surges from rest to 1.25 then cruises", func(t *testing.T) {
		if stars.SurgeClock(lander.FlyInSeconds, 1.25, lander.FlyInSeconds) != 3.5 {
			t.Fatalf("the close-up surge must burn 3.5s of fly over a 4s fly-in (fly clock %g)", stars.SurgeClock(lander.FlyInSeconds, 1.25, lander.FlyInSeconds))
		}
		if stars.SurgeClock(lander.FlyInSeconds+2, 1.25, lander.FlyInSeconds) != 5.5 {
			t.Fatal("past the fly-in the close-up sky must cruise at standard speed")
		}
	})
	t.Run("happy: the knob reaches the stage — a one-second fly-in parks in one second", func(t *testing.T) {
		sc := Bill()[1].Scene.(*CloseupShow)
		sc.Cfg.FlyInSeconds = 1
		sc.Start()
		defer sc.Stop()
		_ = render(sc) // stage the cast
		tick(sc, 1.3)
		a := hullLeftCol(t, render(sc))
		tick(sc, 0.4)
		if b := hullLeftCol(t, render(sc)); a != b {
			t.Fatalf("1.3s into a 1s fly-in the hull must be parked, col read %d then %d", a, b)
		}
		stock := Bill()[1].Scene.(*CloseupShow)
		stock.Start()
		defer stock.Stop()
		_ = render(stock)
		tick(stock, 1.3)
		c := hullLeftCol(t, render(stock))
		tick(stock, 0.4)
		if d := hullLeftCol(t, render(stock)); c == d {
			t.Fatal("test premise: the stock four-second slide must still be moving at 1.3s")
		}
	})
	t.Run("unhappy: a nudge below zero stands and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultCloseupConfig()
		c.Nudge(0, -100)
		if want := lander.FlyInSeconds - 25.0; c.FlyInSeconds != want {
			t.Fatalf("a hundred fly-in steps down reads %v, want %v — never clamped", c.FlyInSeconds, want)
		}
		c.Nudge(1, 8000)
		if want := 1.25 + 400.0; c.RushPeak != want {
			t.Fatalf("eight thousand rush steps read %v, want %v — never clamped", c.RushPeak, want)
		}
		before := c
		c.Nudge(5, 1)
		if c != before {
			t.Fatal("a bad cursor must not move any knob")
		}
	})
}

func TestFireShow(t *testing.T) {
	t.Run("happy: the fire entry is the tunable show at the stock brake, sink off", func(t *testing.T) {
		sc, ok := Bill()[2].Scene.(*FireShow)
		if !ok {
			t.Fatalf("the fire entry is %T, want the fire show", Bill()[2].Scene)
		}
		if sc.Cfg != DefaultFireConfig() {
			t.Fatalf("a fresh show carries %+v, want stock", sc.Cfg)
		}
		want := FireConfig{SlowBy: 0.6, SlowOverSeconds: 5}
		if DefaultFireConfig() != want {
			t.Fatalf("stock fire is %+v, want parked %+v — MAIN's knobs turn the sink on", DefaultFireConfig(), want)
		}
	})
	t.Run("happy: the knob face reads slow by, slow over, then fall", func(t *testing.T) {
		c := DefaultFireConfig()
		if c.KnobCount() != 3 {
			t.Fatalf("the fire show carries %d knobs, want 3", c.KnobCount())
		}
		if c.KnobLabel(0) != "slow by" || c.KnobLabel(1) != "slow over" || c.KnobLabel(2) != "fall" {
			t.Fatalf("labels %q/%q/%q, want slow by/slow over/fall", c.KnobLabel(0), c.KnobLabel(1), c.KnobLabel(2))
		}
		c.Nudge(0, 1)
		if want := 0.6 + 0.05; c.SlowBy != want {
			t.Fatalf("one brake step reads %v, want %v", c.SlowBy, want)
		}
		c.Nudge(1, -4)
		if want := 5 - 1.0; c.SlowOverSeconds != want {
			t.Fatalf("four window steps down read %v, want %v", c.SlowOverSeconds, want)
		}
		c.Nudge(2, 2)
		if want := 0.5; c.SinkSeconds != want {
			t.Fatalf("two sink steps from stock read %v, want %v", c.SinkSeconds, want)
		}
	})
	t.Run("happy: the stock fire show parks the lit craft and holds the park", func(t *testing.T) {
		sc := Bill()[2].Scene.(*FireShow)
		sc.Start()
		defer sc.Stop()
		_ = render(sc)
		tick(sc, 0.5)
		v := render(sc)
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("the fire scene opens on the parked hull")
		}
		if !hasFire(v) {
			t.Fatal("the fire scene burns the booster")
		}
		open := westHullTop(paint(sc))
		tick(sc, 4)
		if got := westHullTop(paint(sc)); got < open-1 || got > open+1 {
			t.Fatalf("stock fire hull top %d, want the park around %d", got, open)
		}
	})
	t.Run("unhappy: a zero sink holds the park, a negative sink does not panic, and a stopped show never panics", func(t *testing.T) {
		zero := NewFireShow(nil)
		zero.Cfg.SinkSeconds = 0
		zero.Start()
		_ = render(zero)
		tick(zero, 2)
		open := westHullTop(paint(zero))
		if open < 0 {
			t.Fatal("a zero sink must keep the hull on stage — not snap it off the bottom")
		}
		tick(zero, 3)
		if got := westHullTop(paint(zero)); got < open-1 || got > open+1 {
			t.Fatalf("zero sink hull top %d, want the park around %d", got, open)
		}
		zero.Stop()

		odd := NewFireShow(nil)
		odd.Cfg.SinkSeconds = -3
		odd.Start()
		odd.Update(1)
		_ = render(odd)
		odd.Stop()

		ghost := NewFireShow(nil)
		ghost.Cfg.SinkSeconds = 400
		ghost.Start()
		ghost.Update(1)
		ghost.Stop()
	})
	t.Run("unhappy: nudges are verbatim — past zero and past one they stand", func(t *testing.T) {
		c := DefaultFireConfig()
		c.Nudge(0, 20)
		if want := 0.6 + 1.0; c.SlowBy != want {
			t.Fatalf("twenty brake steps read %v, want %v — never clamped", c.SlowBy, want)
		}
		c.Nudge(1, -100)
		if want := 5 - 25.0; c.SlowOverSeconds != want {
			t.Fatalf("a hundred window steps down read %v, want %v — never clamped", c.SlowOverSeconds, want)
		}
		c.Nudge(2, -80)
		if want := -20.0; c.SinkSeconds != want {
			t.Fatalf("eighty sink steps read %v, want %v — never clamped", c.SinkSeconds, want)
		}
		before := c
		c.Nudge(7, -1)
		if c != before {
			t.Fatal("a bad cursor must not move any knob")
		}
	})
	t.Run("unhappy: no two bills share knobs", func(t *testing.T) {
		one := Bill()[2].Scene.(*FireShow)
		two := Bill()[2].Scene.(*FireShow)
		one.Cfg.Nudge(0, 3)
		if two.Cfg != DefaultFireConfig() {
			t.Fatal("nudging one bill's fire must not touch another's")
		}
	})
}
