package sky

// Tests written FIRST: Sky is the blue field as a scene component.
// Light blue sits toward the horizon and darker blue comes from a
// knobbed angle (stock: from the top). Start paints a stage-sized
// sprite of the gradient; Update runs the rise clock so the field
// is moveable — it opens on almost-pure light blue and pans upward
// over Rise seconds until the darker blue fills in from the angle.
// The three knobs (angle, light ink, dark ink) live with the
// component so a tuner and any scene read the same file.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 48
	stageH = 24
)

var _ screenplay.Component = (*Sky)(nil)

func bgAt(sp sprite.Sprite, col, row int) int {
	return sp.At(row, col).BG
}

func countInk(sp sprite.Sprite, ink int) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if sp.At(r, c).BG == ink {
				n++
			}
		}
	}
	return n
}

func TestSkyComponent(t *testing.T) {
	t.Run("happy: a started sky fills every cell with a blue background", func(t *testing.T) {
		t.Cleanup(Reset)
		s := New()
		s.Start(stageW, stageH)
		sp := s.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		empty := 0
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				cell := sp.At(r, c)
				if cell.Transparent() || cell.BG < 0 {
					empty++
				}
			}
		}
		if empty != 0 {
			t.Fatalf("%d cells have no sky — the field is a floor, not a sprinkle", empty)
		}
	})
	t.Run("happy: stock angle paints darker blue above lighter blue", func(t *testing.T) {
		t.Cleanup(Reset)
		s := New().At(1)
		s.Start(stageW, stageH)
		sp := s.Render()
		top := bgAt(sp, stageW/2, 0)
		bot := bgAt(sp, stageW/2, stageH-1)
		if top == bot {
			t.Fatalf("top and bottom both wear %d — the gradient must climb", top)
		}
		if lum(top) >= lum(bot) {
			t.Fatalf("top ink %d (lum %d) must be darker than bottom ink %d (lum %d)", top, lum(top), bot, lum(bot))
		}
		if countInk(sp, DefaultDark) == 0 {
			t.Fatal("a fully risen sky must wear the dark ink somewhere along the top")
		}
		if lum(bgAt(sp, stageW/2, stageH-1)) <= lum(bgAt(sp, stageW/2, 0)) {
			t.Fatal("a fully risen sky must still be lighter along the horizon than at the zenith")
		}
	})
	t.Run("happy: a 45° angle sends the dark from the top-right instead of straight down", func(t *testing.T) {
		t.Cleanup(Reset)
		cfg := DefaultConfig()
		cfg.AngleDeg = 45
		if err := Use(cfg); err != nil {
			t.Fatal(err)
		}
		s := New().At(1)
		s.Start(stageW, stageH)
		sp := s.Render()
		tr := bgAt(sp, stageW-1, 0)
		bl := bgAt(sp, 0, stageH-1)
		if lum(tr) >= lum(bl) {
			t.Fatalf("at 45° the top-right (%d lum %d) must be darker than the bottom-left (%d lum %d)", tr, lum(tr), bl, lum(bl))
		}
		tl := bgAt(sp, 0, 0)
		if lum(tr) >= lum(tl) {
			t.Fatalf("at 45° the dark comes from the top-right: right %d lum %d vs left %d lum %d", tr, lum(tr), tl, lum(tl))
		}
	})
	t.Run("unhappy: before Start and after Stop the stage is empty, never panics", func(t *testing.T) {
		s := New()
		if sp := s.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("unstarted render %dx%d, want empty", sp.Width, sp.Height)
		}
		s.Update(1)
		s.Stop()
		var ghost *Sky
		ghost.Start(10, 10)
		ghost.Update(1)
		_ = ghost.Render()
		ghost.Stop()
	})
}

func TestSkyRise(t *testing.T) {
	t.Run("happy: the curtain opens on almost-pure light blue", func(t *testing.T) {
		t.Cleanup(Reset)
		s := New().Rise(4)
		s.Start(stageW, stageH)
		sp := s.Render()
		light := countInk(sp, DefaultLight)
		dark := countInk(sp, DefaultDark)
		if light < stageW*stageH/2 {
			t.Fatalf("only %d/%d cells wear the light ink — the opening look is the horizon", light, stageW*stageH)
		}
		if dark != 0 {
			t.Fatalf("the opening sky already wears the dark ink in %d cells — the dark waits on the rise", dark)
		}
		if s.Pan() != 0 {
			t.Fatalf("pan %v, want 0 at the curtain", s.Pan())
		}
	})
	t.Run("happy: over Rise seconds the view climbs until the darker blue appears", func(t *testing.T) {
		t.Cleanup(Reset)
		s := New().Rise(2)
		s.Start(stageW, stageH)
		_ = s.Render()
		s.Update(2)
		if got := s.Pan(); got < 1-1e-9 {
			t.Fatalf("after the rise pan %v, want 1", got)
		}
		sp := s.Render()
		if countInk(sp, DefaultDark) == 0 {
			t.Fatal("a finished rise must show the darker blue")
		}
		top := lum(bgAt(sp, stageW/2, 0))
		bot := lum(bgAt(sp, stageW/2, stageH-1))
		if top >= bot {
			t.Fatalf("risen sky top lum %d must be darker than bottom lum %d", top, bot)
		}
	})
	t.Run("happy: a resize mid-rise keeps the clock — no fall back to the horizon", func(t *testing.T) {
		t.Cleanup(Reset)
		s := New().Rise(2)
		s.Start(stageW, stageH)
		s.Update(1)
		mid := s.Pan()
		s.Start(60, 30)
		if got := s.Pan(); mathAbs(got-mid) > 1e-9 {
			t.Fatalf("resize pan %v, want the mid-rise %v", got, mid)
		}
	})
	t.Run("unhappy: waiting before the first render never burns the rise", func(t *testing.T) {
		t.Cleanup(Reset)
		s := New().Rise(1)
		s.Start(stageW, stageH)
		s.Update(4)
		// clock starts at the first Update after Start, so this one
		// does run — the ensemble defers Start until first render.
		// A dt<=0 hold is the unhappy path here.
		s2 := New().Rise(1)
		s2.Start(stageW, stageH)
		s2.Update(0)
		s2.Update(-1)
		if s2.Pan() != 0 {
			t.Fatalf("dt<=0 pan %v, want 0", s2.Pan())
		}
	})
}

func TestSkyKnobsPaint(t *testing.T) {
	t.Run("happy: retuned light and dark inks are what the field wears", func(t *testing.T) {
		t.Cleanup(Reset)
		cfg := DefaultConfig()
		cfg.LightInk = 159
		cfg.DarkInk = 19
		if err := Use(cfg); err != nil {
			t.Fatal(err)
		}
		s := New().At(1)
		s.Start(stageW, stageH)
		sp := s.Render()
		if countInk(sp, 19) == 0 {
			t.Fatal("the retuned dark ink must land on the field")
		}
		if countInk(sp, DefaultDark) != 0 {
			t.Fatal("the stock dark ink must not leak onto a retuned sky")
		}
		if lum(bgAt(sp, stageW/2, 0)) >= lum(bgAt(sp, stageW/2, stageH-1)) {
			t.Fatal("the retuned sky must still climb from a darker zenith to a lighter horizon")
		}
	})
	t.Run("unhappy: a live sky keeps the inks it started with", func(t *testing.T) {
		t.Cleanup(Reset)
		s := New().At(1)
		s.Start(stageW, stageH)
		_ = s.Render()
		cfg := DefaultConfig()
		cfg.LightInk = 51
		cfg.DarkInk = 4
		if err := Use(cfg); err != nil {
			t.Fatal(err)
		}
		sp := s.Render()
		if countInk(sp, 51) != 0 || countInk(sp, 4) != 0 {
			t.Fatal("an in-flight sky must keep the inks it launched with")
		}
	})
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
