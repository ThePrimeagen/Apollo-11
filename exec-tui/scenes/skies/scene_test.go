package skies

// Tests written FIRST: Skies is the blue-sky scene. The curtain
// rises on almost-pure light blue; over RiseSeconds the camera tilts
// up so the darker blue and the generated clouds come into view; then
// the eagle flies in from the right to its end point and the shotgun
// in each talon fires — not along the whole crossing, but after the
// bird is on stage, each gun on its own shot count and rate of fire.
// Every performer is a reusable component: components/sky,
// components/cloud, components/eagle, components/shotgun.

import (
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/cloud"
	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sky"
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

func bgIndex(scr *screenplay.Screen, x, y int) int {
	c := scr.Cell(x, y)
	if c == nil || c.Style.Bg == nil {
		return -1
	}
	ic, ok := c.Style.Bg.(ansi.IndexedColor)
	if !ok {
		return -1
	}
	return int(ic)
}

func inkAt(scr *screenplay.Screen, x, y int) (fg, bg int) {
	fg, bg = -1, -1
	c := scr.Cell(x, y)
	if c == nil {
		return fg, bg
	}
	if ic, ok := c.Style.Fg.(ansi.IndexedColor); ok {
		fg = int(ic)
	}
	if ic, ok := c.Style.Bg.(ansi.IndexedColor); ok {
		bg = int(ic)
	}
	return fg, bg
}

func countBG(scr *screenplay.Screen, ink int) int {
	n := 0
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			if bgIndex(scr, x, y) == ink {
				n++
			}
		}
	}
	return n
}

func eagleCells(scr *screenplay.Screen) [][2]int {
	var out [][2]int
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			fg, bg := inkAt(scr, x, y)
			for _, ink := range eagle.SignatureInks() {
				if fg == ink || bg == ink {
					out = append(out, [2]int{y, x})
					break
				}
			}
		}
	}
	return out
}

func leftmost(cells [][2]int) int {
	l := 1 << 30
	for _, rc := range cells {
		if rc[1] < l {
			l = rc[1]
		}
	}
	return l
}

func gunCells(scr *screenplay.Screen) [][2]int {
	var out [][2]int
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			fg, bg := inkAt(scr, x, y)
			if fg == 178 || bg == 178 {
				out = append(out, [2]int{y, x})
			}
		}
	}
	return out
}

func blastCells(scr *screenplay.Screen) [][2]int {
	var out [][2]int
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			fg, _ := inkAt(scr, x, y)
			if fg == 226 || fg == 208 || fg == 196 {
				out = append(out, [2]int{y, x})
			}
		}
	}
	return out
}

func cloudCells(scr *screenplay.Screen) int {
	n := 0
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			c := scr.Cell(x, y)
			if c == nil || c.Content == "" {
				continue
			}
			r := []rune(c.Content)
			if len(r) == 0 {
				continue
			}
			ch := r[0]
			if (ch >= '⠀' && ch <= '⣿') || ch == '░' || ch == '▒' {
				n++
			}
		}
	}
	return n
}

func TestSkiesKnobs(t *testing.T) {
	t.Run("happy: the stock show tilts the sky, then flies the eagle in", func(t *testing.T) {
		if RiseSeconds <= 0 || RiseSeconds > 4 {
			t.Fatalf("RiseSeconds = %v — the tilt is a short beat, four seconds at most", RiseSeconds)
		}
		if CrossSeconds <= 0 || CrossSeconds > 6 {
			t.Fatalf("CrossSeconds = %v — the crossing stays brisk", CrossSeconds)
		}
		c := DefaultConfig()
		if c.RiseSeconds != RiseSeconds {
			t.Fatalf("rise %v, want %v", c.RiseSeconds, RiseSeconds)
		}
		if c.EagleDelay < c.RiseSeconds/2 {
			t.Fatalf("eagle delay %v — the bird waits for the sky to start climbing", c.EagleDelay)
		}
		if StartPoint != 0 {
			t.Fatalf("StartPoint = %v — the stock eagle enters off the right wing", StartPoint)
		}
		if EndPoint <= StartPoint || EndPoint > 1 {
			t.Fatalf("EndPoint = %v — a point on the span past the start", EndPoint)
		}
	})
	t.Run("unhappy: rise, delay, and crossing stay independent knobs", func(t *testing.T) {
		if RiseSeconds == CrossSeconds && RiseSeconds == DefaultConfig().EagleDelay {
			t.Fatal("the three clocks must be independent knobs")
		}
	})
}

func TestSkiesBill(t *testing.T) {
	t.Run("happy: the bill is the one scene named Skies", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the Skies bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "Skies" {
			t.Fatalf("the scene is %q, want Skies", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the scene has no performer")
		}
	})
	t.Run("unhappy: after Skies there is nothing left on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "Skies" {
			t.Fatalf("the show opens on %d %q, want one Skies", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after Skies there is nothing left")
		}
	})
}

func TestSkiesScene(t *testing.T) {
	t.Cleanup(sky.Reset)
	t.Cleanup(cloud.Reset)
	t.Cleanup(Reset)
	t.Run("happy: the curtain rises on almost-pure light blue", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		scr := paint(sc)
		light := countBG(scr, sky.DefaultLight)
		dark := countBG(scr, sky.DefaultDark)
		if light < stageW*stageH/2 {
			t.Fatalf("only %d/%d cells wear the light ink — the opening look is the horizon", light, stageW*stageH)
		}
		if dark != 0 {
			t.Fatalf("the opening sky already wears the dark ink in %d cells", dark)
		}
		if cloudCells(scr) != 0 {
			t.Fatal("the clouds wait in the upper sky — the horizon shot is clear")
		}
		if len(eagleCells(scr)) != 0 {
			t.Fatal("no eagle yet — the sky tilts first")
		}
	})
	t.Run("happy: over the rise the darker blue and the clouds come into view", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, RiseSeconds)
		scr := paint(sc)
		if countBG(scr, sky.DefaultDark) == 0 {
			t.Fatal("a finished rise must show the darker blue")
		}
		if cloudCells(scr) == 0 {
			t.Fatal("a finished rise must bring the generated clouds into view")
		}
		top := bgIndex(scr, stageW/2, 0)
		bot := bgIndex(scr, stageW/2, stageH-1)
		if sky.Lum(top) >= sky.Lum(bot) {
			t.Fatalf("risen sky top %d must be darker than bottom %d", top, bot)
		}
	})
	t.Run("happy: then the huge eagle crosses right to left across the blue", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, DefaultConfig().EagleDelay+CrossSeconds/4)
		first := eagleCells(paint(sc))
		if len(first) < 100 {
			t.Fatalf("a quarter into the crossing only %d eagle cells are on stage — the model must be huge", len(first))
		}
		l1 := leftmost(first)
		tick(sc, CrossSeconds/4)
		scr := paint(sc)
		second := eagleCells(scr)
		if len(second) == 0 {
			t.Fatal("halfway in the eagle must still be on stage")
		}
		if l2 := leftmost(second); l2 >= l1 {
			t.Fatalf("the eagle must fly leftward: leftmost went %d -> %d", l1, l2)
		}
		if countBG(scr, sky.DefaultLight) == 0 && countBG(scr, sky.DefaultDark) == 0 {
			t.Fatal("the blue sky must keep flying beneath the eagle")
		}
	})
	t.Run("unhappy: waiting before the first render never burns the rise", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		sc.Update(RiseSeconds + CrossSeconds)
		scr := paint(sc)
		if countBG(scr, sky.DefaultDark) != 0 {
			t.Fatal("the first frame must still open on the horizon — the clock starts at the curtain")
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		sc := New()
		sc.Start()
		sc.Update(1)
		sc.Stop()
	})
}

func TestSkiesArmedEagle(t *testing.T) {
	t.Cleanup(Reset)
	fast := func(leftShots, rightShots int, leftRate, rightRate float64) Config {
		cfg := DefaultConfig()
		cfg.RiseSeconds = 0.2
		cfg.EagleDelay = 0.2
		cfg.CrossSeconds = 2.0
		cfg.LeftShots = leftShots
		cfg.RightShots = rightShots
		cfg.LeftRate = leftRate
		cfg.RightRate = rightRate
		return cfg
	}
	t.Run("happy: two shotguns ride the talons across the stage", func(t *testing.T) {
		t.Cleanup(Reset)
		if err := Use(fast(1, 1, 4, 4)); err != nil {
			t.Fatal(err)
		}
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.2+0.5)
		first := gunCells(paint(sc))
		if len(first) == 0 {
			t.Fatal("mid-crossing the talon guns must be painted on the bird")
		}
		l1 := leftmost(first)
		tick(sc, 0.5)
		second := gunCells(paint(sc))
		if len(second) == 0 {
			t.Fatal("the guns must stay mounted through the crossing")
		}
		if l2 := leftmost(second); l2 >= l1 {
			t.Fatalf("the guns must ride the bird leftward: leftmost went %d -> %d", l1, l2)
		}
	})
	t.Run("happy: the bird flies in, then the guns fire on their own rate", func(t *testing.T) {
		t.Cleanup(Reset)
		if err := Use(fast(2, 0, 2, 0)); err != nil {
			t.Fatal(err)
		}
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.2+0.2)
		if got := blastCells(paint(sc)); len(got) != 0 {
			t.Fatalf("just after the bird enters the sky holds %d flame cells — the guns wait for their rate", len(got))
		}
		if len(eagleCells(paint(sc))) == 0 {
			t.Fatal("test premise: the bird must already be on stage")
		}
		tick(sc, 0.4)
		if got := blastCells(paint(sc)); len(got) == 0 {
			t.Fatal("past 1/rate of air time the muzzle flame must be in the air")
		}
	})
	t.Run("unhappy: zero shells or a zero rate is a silent flyover — mounted guns, no flame", func(t *testing.T) {
		t.Cleanup(Reset)
		if err := Use(fast(0, 3, 4, 0)); err != nil {
			t.Fatal(err)
		}
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		at := 0.0
		for target := 0.4; target <= 2.4; target += 0.4 {
			tick(sc, target-at)
			at = target
			if got := blastCells(paint(sc)); len(got) != 0 {
				t.Fatalf("at %.1fs a silent gun threw %d flame cells", target, len(got))
			}
		}
	})
	t.Run("unhappy: before the delay the armed bird is fully off stage — guns too", func(t *testing.T) {
		t.Cleanup(Reset)
		cfg := fast(1, 1, 4, 4)
		cfg.EagleDelay = 1.5
		if err := Use(cfg); err != nil {
			t.Fatal(err)
		}
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 1.0)
		if got := gunCells(paint(sc)); len(got) != 0 {
			t.Fatalf("before the delay %d gun cells are on stage — the guns wait with the bird", len(got))
		}
	})
}

func TestSkiesFlightPath(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: an early end point cuts the flight short of the far wing, on time", func(t *testing.T) {
		t.Cleanup(Reset)
		cfg := DefaultConfig()
		cfg.RiseSeconds = 0.2
		cfg.EagleDelay = 0.2
		cfg.CrossSeconds = 2.0
		cfg.EagleEnd = 0.5
		cfg.LeftShots, cfg.RightShots = 0, 0
		if err := Use(cfg); err != nil {
			t.Fatal(err)
		}
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 2.1)
		cells := eagleCells(paint(sc))
		if len(cells) < 100 {
			t.Fatalf("just before an 0.5 end the bird is still mid-stage, found only %d eagle cells", len(cells))
		}
		if l := leftmost(cells); l < 1 {
			t.Fatalf("an 0.5 end must never reach the left wing, leading edge at %d", l)
		}
		tick(sc, 0.4)
		if got := eagleCells(paint(sc)); len(got) != 0 {
			t.Fatalf("past the crossing the flight is over, found %d eagle cells", len(got))
		}
	})
}
