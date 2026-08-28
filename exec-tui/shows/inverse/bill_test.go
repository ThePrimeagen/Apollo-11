package inverse

// Tests written FIRST: 04. Inverse Walkthrough is a composable
// three-scene bill — the walkthrough played backwards. Scene one,
// "liftoff": the lander parked on the moon horizon ignites (¼, ½, ¾,
// full), kicks the mirrored pad dust, climbs off the top on the
// landing's mirrored ease, and the empty moon holds. Scene two,
// "engines on": the west-facing craft parked at center stage, tail
// fire burning, bobbling on its sine. Scene three, "engines off": the
// very same bobble scene with the engine out — it holds ad infinitum;
// only the cut ends the show. The bobble scene is used twice, the way
// the walkthrough plays it engine off then engine on, just reversed.
//
// One stars.Continuity seeds every scene's sky, so a cut never jumps
// or skips a single star.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 27
)

var sceneNames = []string{"liftoff", "engines on", "engines off"}

func openShow() (*screenplay.Screenplay, *screenplay.Screen) {
	p := screenplay.Compose(Bill())
	p.Start()
	scr := screenplay.NewScreen(stageW, stageH)
	p.Render(scr)
	return p, scr
}

func run(p *screenplay.Screenplay, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		p.Update(dt)
	}
}

func frame(p *screenplay.Screenplay, scr *screenplay.Screen) string {
	p.Render(scr)
	return scr.Render()
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

func TestInverseWalkthroughBill(t *testing.T) {
	t.Cleanup(func() {
		liftoff.Reset()
		bobble.Reset()
	})
	t.Run("happy: the bill is three scenes in playing order", func(t *testing.T) {
		b := Bill()
		if len(b) != 3 {
			t.Fatalf("the inverse walkthrough holds %d scenes, want 3", len(b))
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
	t.Run("happy: the bobble scene is used twice — lit, then dark", func(t *testing.T) {
		b := Bill()
		on, ok := b[1].Scene.(*bobble.Show)
		if !ok {
			t.Fatal("engines on must be the bobble scene")
		}
		if !on.Cfg.Engine {
			t.Fatal("engines on must burn the tail fire")
		}
		off, ok := b[2].Scene.(*bobble.Show)
		if !ok {
			t.Fatal("engines off must be the bobble scene")
		}
		if off.Cfg.Engine {
			t.Fatal("engines off must fly a cold engine")
		}
		if _, ok := b[0].Scene.(*liftoff.Show); !ok {
			t.Fatal("the show opens on the liftoff scene")
		}
	})
	t.Run("happy: the show opens on the pad and the craft climbs away", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		opening := frame(p, scr)
		if !strings.ContainsRune(opening, '▟') {
			t.Fatal("the show opens on the lander parked on the pad")
		}
		p.Render(scr)
		if moonBGRows(scr, stageW/2) != moon.HorizonCenterRows {
			t.Fatal("the liftoff plays on the moon horizon")
		}
		run(p, liftoff.LiftAt+liftoff.RiseSeconds+0.2)
		gone := frame(p, scr)
		if strings.ContainsRune(gone, '▟') {
			t.Fatal("past lift-at plus rise the craft must have cleared the top")
		}
		p.Render(scr)
		if hotBraille(scr) {
			t.Fatal("past the climb the booster fire must have left with the hull")
		}
		if strings.ContainsRune(gone, '▌') {
			t.Fatal("the sideways craft waits for the cut")
		}
	})
	t.Run("unhappy: the climb is not a flame cut — the booster burns the whole way up", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		run(p, liftoff.FireFull+0.3)
		p.Render(scr)
		if !hotBraille(scr) {
			t.Fatal("on the pad, past full power, the booster must still be burning")
		}
		if !strings.ContainsRune(frame(p, scr), '▟') {
			t.Fatal("early in the show the hull is still on the pad — a flame cut is not a liftoff")
		}
	})
	t.Run("happy: the liftoff→engines-on cut keeps every star the horizon was not hiding", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		run(p, liftoff.LiftAt+liftoff.RiseSeconds+0.5)
		p.Render(scr)
		before := starCells(scr)
		if len(before) == 0 {
			t.Fatal("test premise: the held liftoff stage must show stars")
		}
		if !p.Next() {
			t.Fatal("the liftoff must cut to engines on")
		}
		p.Render(scr)
		after := starCells(scr)
		hullL := (stageW-lander.BodyCols)/2 - 2
		hullR := (stageW+lander.BodyCols)/2 + lander.FlameCol + 18
		matched := 0
		for pos, ch := range before {
			if pos[0] >= hullL && pos[0] <= hullR {
				continue
			}
			if g, ok := after[pos]; ok {
				if g != ch {
					t.Fatalf("star at (%d,%d) jumped on the cut: %q -> %q", pos[0], pos[1], ch, g)
				}
				matched++
			}
		}
		if matched == 0 {
			t.Fatal("the engines-on sky must open on the liftoff's frame")
		}
	})
	t.Run("happy: engines on parks the sideways craft with the tail fire burning", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		p.Next()
		p.Render(scr)
		if !strings.Contains(frame(p, scr), "▌") {
			t.Fatal("engines on opens with the west hull already parked")
		}
		run(p, 0.5)
		p.Render(scr)
		if !hotBraille(scr) {
			t.Fatal("engines on must burn the tail fire")
		}
		if moonBGRows(scr, stageW/2) != 0 {
			t.Fatal("the parked craft flies in open space — the moon is behind it now")
		}
	})
	t.Run("happy: engines off parks the same craft cold, bobbling ad infinitum", func(t *testing.T) {
		p, scr := openShow()
		defer p.Stop()
		p.Next()
		p.Next()
		p.Render(scr)
		if !strings.Contains(frame(p, scr), "▌") {
			t.Fatal("engines off opens with the west hull still parked")
		}
		run(p, 8.0)
		p.Render(scr)
		if hotBraille(scr) {
			t.Fatal("engines off never lights the tail fire")
		}
		if !strings.Contains(scr.Render(), "▌") {
			t.Fatal("eight seconds on, the craft still holds the park")
		}
	})
	t.Run("happy: the composed show walks the three scenes and then has nothing left", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		if p.Len() != 3 || p.CurrentName() != "liftoff" {
			t.Fatalf("the show opens on %d %q, want three starting on liftoff", p.Len(), p.CurrentName())
		}
		for i, want := range sceneNames[1:] {
			if !p.Next() || p.CurrentName() != want {
				t.Fatalf("cut %d must land on %q, got %q", i+1, want, p.CurrentName())
			}
		}
		if p.Next() {
			t.Fatal("after engines off there is nothing left — the show ends")
		}
		p.Stop()
	})
	t.Run("happy: the bill plays the active knobs on the first curtain", func(t *testing.T) {
		t.Cleanup(func() {
			liftoff.Reset()
			bobble.Reset()
		})
		lc := liftoff.DefaultConfig()
		lc.LiftAt = 0
		lc.RiseSeconds = 0.2
		if err := liftoff.Use(lc); err != nil {
			t.Fatal(err)
		}
		bc := bobble.DefaultConfig()
		bc.AmplitudeCells = 3
		if err := bobble.Use(bc); err != nil {
			t.Fatal(err)
		}
		b := Bill()
		p := screenplay.Compose(b)
		p.Start()
		defer p.Stop()
		scr := screenplay.NewScreen(stageW, stageH)
		p.Render(scr)
		run(p, 0.3)
		p.Render(scr)
		if strings.Contains(scr.Render(), "▟") {
			t.Fatal("the liftoff must fly the saved knobs — a 0.2s climb is long gone")
		}
		if got := b[1].Scene.(*bobble.Show).Cfg.AmplitudeCells; got != 3 {
			t.Fatalf("the bobble entries must carry the active ride, amplitude %d want 3", got)
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		for _, e := range Bill() {
			e.Scene.Start()
			e.Scene.Update(1)
			e.Scene.Stop()
		}
	})
	t.Run("unhappy: the inverse walkthrough is not the forward one", func(t *testing.T) {
		for _, e := range Bill() {
			switch e.Name {
			case "pause", "Lunar Lander Close-Up", "fire", "fall", "landing":
				t.Fatalf("the inverse bill must not carry walkthrough scene %q", e.Name)
			}
		}
	})
}
