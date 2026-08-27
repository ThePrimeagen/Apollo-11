package america

// Tests written FIRST: America is the portable patriot scene. The
// curtain rises on pure black; the full-screened American flag fades
// in slowly — FadeSeconds of ramping color, no motion — and only once
// the flag is fully in does the very large eagle enter from the right
// and cross the whole stage leftward over CrossSeconds, the flag still
// flying beneath it. After the flyover the flag flies alone. The bill
// is one scene named America; a stopped or unstaged scene never
// panics, and waiting before the first render never burns the fade.

import (
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/flag"
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

// bgIndex is the xterm-256 background at (x, y), or -1.
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

// inkAt reads a cell's indexed foreground and background, -1 for none.
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

// eagleCells collects the cells wearing any of the eagle's signature
// inks — its browns and golds. The flag never wears one, at any point
// of its fade, so on this stage those cells are the eagle.
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

func starCount(scr *screenplay.Screen) int {
	n := 0
	for y := 0; y < stageH; y++ {
		for x := 0; x < stageW; x++ {
			if c := scr.Cell(x, y); c != nil && c.Content == string(flag.StarGlyph) {
				n++
			}
		}
	}
	return n
}

func TestAmericaKnobs(t *testing.T) {
	t.Run("happy: the fade is slow and the crossing is a real flight", func(t *testing.T) {
		if FadeSeconds < 4 {
			t.Fatalf("FadeSeconds = %v — the flag fades in slowly, give it at least 4s", FadeSeconds)
		}
		if CrossSeconds <= 0 {
			t.Fatalf("CrossSeconds = %v — the crossing must be a duration", CrossSeconds)
		}
	})
	t.Run("unhappy: the fade and the crossing are separate beats", func(t *testing.T) {
		if FadeSeconds == CrossSeconds {
			t.Fatal("the fade and the crossing must be independent knobs")
		}
	})
}

func TestAmericaBill(t *testing.T) {
	t.Run("happy: the bill is the one scene named America", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the America bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "America" {
			t.Fatalf("the scene is %q, want America", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the scene has no performer")
		}
	})
	t.Run("unhappy: after America there is nothing left on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "America" {
			t.Fatalf("the show opens on %d %q, want one America", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after America there is nothing left")
		}
	})
}

func TestAmericaScene(t *testing.T) {
	t.Run("happy: the curtain rises on pure black", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		scr := paint(sc)
		for y := 0; y < stageH; y++ {
			for x := 0; x < stageW; x++ {
				if got := bgIndex(scr, x, y); got != flag.Black {
					t.Fatalf("cell (%d,%d) opens on bg %d, want black %d", y, x, got, flag.Black)
				}
			}
		}
	})
	t.Run("happy: the flag fades in slowly and lands on its finished colors", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, FadeSeconds/2)
		mid := bgIndex(paint(sc), stageW-1, 0)
		if mid == flag.Black || mid == flag.RedInk {
			t.Fatalf("mid-fade the top stripe wears %d — between black and red, please", mid)
		}
		tick(sc, FadeSeconds/2+0.5)
		scr := paint(sc)
		if got := bgIndex(scr, stageW-1, 0); got != flag.RedInk {
			t.Fatalf("after the fade the top stripe wears %d, want red %d", got, flag.RedInk)
		}
		if got := bgIndex(scr, 0, 0); got != flag.BlueInk {
			t.Fatalf("after the fade the canton wears %d, want blue %d", got, flag.BlueInk)
		}
		if got := bgIndex(scr, stageW-1, stageH-1); got != flag.RedInk {
			t.Fatalf("after the fade the bottom stripe wears %d, want red %d", got, flag.RedInk)
		}
		if got := starCount(scr); got != 50 {
			t.Fatalf("the canton carries %d stars, want 50", got)
		}
	})
	t.Run("happy: no eagle until the flag is fully in", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, FadeSeconds-0.5)
		if got := eagleCells(paint(sc)); len(got) != 0 {
			t.Fatalf("the eagle entered %d cells early — it waits for the fade", len(got))
		}
	})
	t.Run("happy: then the huge eagle crosses right to left with the flag beneath", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, FadeSeconds+CrossSeconds/4)
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
		if got := bgIndex(scr, 0, 0); got != flag.BlueInk {
			t.Fatalf("the canton must keep flying beneath the eagle, wears %d", got)
		}
		if got := bgIndex(scr, stageW-1, 0); got != flag.RedInk {
			t.Fatalf("the stripes must keep flying beneath the eagle, wears %d", got)
		}
	})
	t.Run("happy: the eagle exits and the flag flies alone", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, FadeSeconds+CrossSeconds+1)
		scr := paint(sc)
		if got := eagleCells(scr); len(got) != 0 {
			t.Fatalf("past the crossing %d eagle cells remain — the sky must be clear", len(got))
		}
		if got := bgIndex(scr, stageW-1, 0); got != flag.RedInk {
			t.Fatalf("the flag must keep flying after the flyover, wears %d", got)
		}
	})
	t.Run("happy: a resize mid-fade keeps the clock — no fall back to black", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, FadeSeconds/2)
		wide := screenplay.NewScreen(100, 30)
		sc.Render(wide)
		if got := bgIndex(wide, 99, 0); got == flag.Black {
			t.Fatal("a mid-fade resize must keep the fade clock, not restart from black")
		}
	})
	t.Run("unhappy: waiting before the first render never burns the fade", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		sc.Update(FadeSeconds + CrossSeconds)
		scr := paint(sc)
		if got := bgIndex(scr, stageW-1, 0); got != flag.Black {
			t.Fatalf("the first frame must still open on black, wears %d — the clock starts at the curtain", got)
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		sc := New()
		sc.Start()
		sc.Update(1)
		sc.Stop()
	})
	t.Run("unhappy: rendering onto a nil screen never panics", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		sc.Render(nil)
	})
}

func TestAmericaScenePlaysConfig(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: New plays the Active config on the first curtain", func(t *testing.T) {
		t.Cleanup(Reset)
		fast := DefaultConfig()
		fast.FadeSeconds = 0.3
		fast.EagleDelay = 0.4
		fast.CrossSeconds = 2.0
		if err := Use(fast); err != nil {
			t.Fatal(err)
		}
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.5)
		if got := bgIndex(paint(sc), stageW-1, 0); got != flag.RedInk {
			t.Fatalf("a 0.3s fade must be at full color by 0.5s, wears %d", got)
		}
		tick(sc, 0.5)
		if len(eagleCells(paint(sc))) == 0 {
			t.Fatal("with a 0.4s delay and a 2s crossing the eagle must be on stage at 1s")
		}
	})
	t.Run("happy: a nudged knob is what the next play uses", func(t *testing.T) {
		sc := New()
		sc.Start()
		_ = paint(sc)
		if got := bgIndex(paint(sc), stageW-1, 0); got != flag.Black {
			t.Fatal("test premise: the stock curtain opens black")
		}
		sc.Stop()
		sc.Cfg.FadeSeconds = 0.2
		sc.Cfg.EagleDelay = 0.2
		sc.Cfg.CrossSeconds = 1.0
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.5)
		scr := paint(sc)
		if got := bgIndex(scr, stageW-1, 0); got != flag.RedInk {
			t.Fatalf("the replay must fade in 0.2s, wears %d at 0.5s", got)
		}
		if len(eagleCells(scr)) == 0 {
			t.Fatal("the replay must fly the eagle on the nudged 0.2s delay")
		}
	})
	t.Run("unhappy: changing knobs mid-flight never retimes the running scene", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 2.0)
		sc.Cfg.FadeSeconds = 0.1
		tick(sc, 0.5)
		if got := bgIndex(paint(sc), stageW-1, 0); got == flag.RedInk {
			t.Fatal("an in-flight fade must keep the timing it launched with")
		}
	})
}

// The eagle detector reads the eagle's own inks, so the flag must
// never wear them — and the flag's glyph language stays its own.
func TestAmericaDetectorPremise(t *testing.T) {
	t.Run("happy: before the eagle enters, the stage is spaces, stars and half blocks", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, FadeSeconds-0.2)
		scr := paint(sc)
		stars := 0
		for y := 0; y < stageH; y++ {
			for x := 0; x < stageW; x++ {
				c := scr.Cell(x, y)
				if c == nil {
					continue
				}
				switch c.Content {
				case "", " ", "▀":
				case string(flag.StarGlyph):
					stars++
				default:
					t.Fatalf("before the eagle, cell (%d,%d) holds %q — the flag is spaces, stars and half blocks", y, x, c.Content)
				}
			}
		}
		if stars != 50 {
			t.Fatalf("the fading flag already carries %d star glyphs, want 50", stars)
		}
	})
	t.Run("unhappy: the flag never wears a signature ink, at any point of the fade", func(t *testing.T) {
		sc := New()
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		at := 0.0
		for target := 0.25; target < FadeSeconds-0.3; target += 0.25 {
			tick(sc, target-at)
			at = target
			if got := eagleCells(paint(sc)); len(got) != 0 {
				t.Fatalf("at %.2fs the fading flag wears eagle ink in %d cells — the detector premise is broken", target, len(got))
			}
		}
	})
}
