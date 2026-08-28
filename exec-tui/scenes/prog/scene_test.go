package prog

// Tests written FIRST: the prog scene is the spacelander dropping
// under a twinkling sky and pausing three times — 1202 on the right,
// then 1202 again, then 1201. That is the first three flight alarms
// (two 1202s in P63, then the 1201 in P64). The last drop carries
// the craft off the bottom. Play rebuilds from the current knobs.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/caption"
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

func hasBanner(scr *screenplay.Screen, text string) bool {
	return caption.Painted(scr, text)
}

func TestProgBill(t *testing.T) {
	t.Run("happy: the bill is the one prog scene", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the prog bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "prog" {
			t.Fatalf("the scene is %q, want prog", b[0].Name)
		}
		if b[0].Scene == nil {
			t.Fatal("the prog has no performer")
		}
	})
	t.Run("unhappy: a second scene is not hiding on the bill", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "prog" {
			t.Fatalf("the show opens on %d %q, want one prog", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the prog there is nothing left")
		}
	})
}

func TestProgScene(t *testing.T) {
	t.Cleanup(Reset)
	t.Cleanup(stars.ResetTwinkle)
	t.Run("happy: the drop pauses 1202, then 1202, then 1201, in that order", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Drop1, sc.Cfg.Hold1 = 0.4, 0.4
		sc.Cfg.Drop2, sc.Cfg.Hold2 = 0.4, 0.4
		sc.Cfg.Drop3, sc.Cfg.Hold3 = 0.4, 0.4
		sc.Cfg.Drop4 = 0.4
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		if hasBanner(paint(sc), "1202") {
			t.Fatal("the first 1202 must wait for the first hold")
		}
		tick(sc, 0.5)
		if !hasBanner(paint(sc), "1202") {
			t.Fatal("the first pause must say 1202")
		}
		if hasBanner(paint(sc), "1201") {
			t.Fatal("1201 must not appear before the two 1202s")
		}
		tick(sc, 0.5)
		if hasBanner(paint(sc), "1202") {
			t.Fatal("between holds the board must clear")
		}
		tick(sc, 0.3)
		if !hasBanner(paint(sc), "1202") {
			t.Fatal("the second pause must say 1202 again")
		}
		tick(sc, 0.5)
		tick(sc, 0.3)
		if !hasBanner(paint(sc), "1201") {
			t.Fatal("the third pause must say 1201 — 1202, 1202, 1201")
		}
		if hasBanner(paint(sc), "LAND") {
			t.Fatal("the space drop never says LAND — that card belongs on the pad")
		}
	})
	t.Run("happy: the sky twinkles while the craft holds", func(t *testing.T) {
		stars.ResetTwinkle()
		sc := New(nil)
		sc.Start()
		defer sc.Stop()
		opening := paint(sc)
		n := 0
		for y := 0; y < stageH; y++ {
			for x := 0; x < stageW; x++ {
				c := opening.Cell(x, y)
				if c == nil {
					continue
				}
				for _, g := range stars.Glyphs {
					if c.Content == string(g) {
						n++
					}
				}
			}
		}
		if n == 0 {
			t.Fatal("the prog drop plays under the stars")
		}
	})
	t.Run("happy: Use is what New plays on the first Start", func(t *testing.T) {
		t.Cleanup(Reset)
		c := DefaultConfig()
		c.Drop1, c.Hold1 = 0.2, 0.4
		c.Drop2, c.Hold2 = 0.2, 0.2
		c.Drop3, c.Hold3 = 0.2, 0.2
		c.Drop4 = 0.2
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
		tick(sc, 0.3)
		if !hasBanner(paint(sc), "1202") {
			t.Fatal("the first play must already use the active hold")
		}
	})
	t.Run("unhappy: a zero hold skips that card, 1201 never leads, and a stopped scene never panics", func(t *testing.T) {
		sc := New(nil)
		sc.Cfg.Drop1, sc.Cfg.Hold1 = 0.3, 0
		sc.Cfg.Drop2, sc.Cfg.Hold2 = 0.3, 0.4
		sc.Cfg.Drop3, sc.Cfg.Hold3 = 0.3, 0.4
		sc.Cfg.Drop4 = 0.3
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		tick(sc, 0.35)
		if hasBanner(paint(sc), "1202") {
			t.Fatal("a zero first hold must skip the first 1202")
		}
		tick(sc, 0.3)
		if !hasBanner(paint(sc), "1202") {
			t.Fatal("the second hold is still a 1202")
		}
		if hasBanner(paint(sc), "1201") {
			t.Fatal("1201 is the third alarm, never the first visible card")
		}
		sc.Stop()
		sc.Start()
		sc.Update(1)
		sc.Stop()
	})
}
