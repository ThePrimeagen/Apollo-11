package transition

// Tests written FIRST: Transition is a background crossfade. Two
// full-stage components paint every frame; each cell's floor walks
// From's background ink toward To's through RGB so a sky can become
// a flag (or any floor become any other) without a hard cut. The
// result is always a floor — every cell carries a background — so
// an eagle, a shotgun, a blast can sit on top. Delay holds From;
// Over is how long the walk takes. A fade of zero or less snaps to
// To the moment the delay elapses. Before Start and after Stop the
// stage is empty; dt <= 0 never moves the clock.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW  = 12
	stageH  = 8
	fromInk = 153
	toInk   = 160
)

var _ screenplay.Component = (*Crossfade)(nil)
var _ screenplay.Component = (*Layers)(nil)

// floor is a solid background slab for the tests.
type floor struct {
	bg     int
	ch     rune
	fg     int
	w, h   int
	staged bool
}

func (f *floor) Start(w, h int) { f.w, f.h = w, h; f.staged = true }
func (f *floor) Update(float64) {}
func (f *floor) Stop()          { f.staged = false }
func (f *floor) Render() sprite.Sprite {
	if !f.staged || f.w < 1 || f.h < 1 {
		return sprite.Sprite{}
	}
	ch := f.ch
	if ch == 0 {
		ch = ' '
	}
	fg := f.fg
	if fg == 0 {
		fg = -1
	}
	sp := sprite.New(f.w, f.h)
	for r := 0; r < f.h; r++ {
		for c := 0; c < f.w; c++ {
			sp.Set(r, c, sprite.Cell{Ch: ch, FG: fg, BG: f.bg})
		}
	}
	return sp
}

func tick(c screenplay.Component, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		c.Update(dt)
	}
}

func TestLerpInk(t *testing.T) {
	t.Run("happy: the walk starts on a, ends on b, and the midpoint is a third ink", func(t *testing.T) {
		if got := LerpInk(fromInk, toInk, 0); got != fromInk {
			t.Fatalf("t=0 got %d, want From %d", got, fromInk)
		}
		if got := LerpInk(fromInk, toInk, 1); got != toInk {
			t.Fatalf("t=1 got %d, want To %d", got, toInk)
		}
		mid := LerpInk(fromInk, toInk, 0.5)
		if mid == fromInk || mid == toInk {
			t.Fatalf("midpoint %d must sit between %d and %d", mid, fromInk, toInk)
		}
	})
	t.Run("unhappy: t outside 0..1 clamps to the nearer end", func(t *testing.T) {
		if got := LerpInk(fromInk, toInk, -0.5); got != fromInk {
			t.Fatalf("t<0 got %d, want From", got)
		}
		if got := LerpInk(fromInk, toInk, 2); got != toInk {
			t.Fatalf("t>1 got %d, want To", got)
		}
	})
}

func TestBlend(t *testing.T) {
	from := sprite.Cell{Ch: ' ', FG: -1, BG: fromInk}
	to := sprite.Cell{Ch: '★', FG: 231, BG: toInk}
	t.Run("happy: t=0 is From, t=1 is To, and the walk keeps a floor", func(t *testing.T) {
		if got := Blend(from, to, 0); got != from {
			t.Fatalf("t=0 %+v, want From %+v", got, from)
		}
		if got := Blend(from, to, 1); got != to {
			t.Fatalf("t=1 %+v, want To %+v", got, to)
		}
		mid := Blend(from, to, 0.5)
		if mid.BG < 0 {
			t.Fatal("a mid-fade cell must keep a background so a later layer can sit on it")
		}
		if mid.BG != LerpInk(fromInk, toInk, 0.5) {
			t.Fatalf("mid BG %d, want the ink lerp %d", mid.BG, LerpInk(fromInk, toInk, 0.5))
		}
	})
	t.Run("unhappy: blending two transparent cells stays transparent", func(t *testing.T) {
		empty := sprite.Cell{Ch: ' ', FG: -1, BG: -1}
		got := Blend(empty, empty, 0.5)
		if !got.Transparent() {
			t.Fatalf("two empty cells blended to %+v, want transparent", got)
		}
	})
}

func TestCrossfade(t *testing.T) {
	newFade := func() *Crossfade {
		return Between(&floor{bg: fromInk}, &floor{bg: toInk, ch: '★', fg: 231}).Over(1)
	}
	t.Run("happy: before the walk the stage is the From floor", func(t *testing.T) {
		c := newFade()
		c.Start(stageW, stageH)
		defer c.Stop()
		sp := c.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		for r := 0; r < stageH; r++ {
			for col := 0; col < stageW; col++ {
				cell := sp.At(r, col)
				if cell.BG != fromInk {
					t.Fatalf("cell %d,%d BG %d, want From %d", col, r, cell.BG, fromInk)
				}
				if cell.Ch == '★' {
					t.Fatal("To's glyph must wait for the walk")
				}
			}
		}
		if c.Frac() != 0 {
			t.Fatalf("Frac %v, want 0", c.Frac())
		}
	})
	t.Run("happy: after Over the stage is the To floor, glyphs and all", func(t *testing.T) {
		c := newFade()
		c.Start(stageW, stageH)
		defer c.Stop()
		_ = c.Render()
		tick(c, 1)
		sp := c.Render()
		stars := 0
		for r := 0; r < stageH; r++ {
			for col := 0; col < stageW; col++ {
				cell := sp.At(r, col)
				if cell.BG != toInk {
					t.Fatalf("cell %d,%d BG %d, want To %d", col, r, cell.BG, toInk)
				}
				if cell.Ch == '★' {
					stars++
				}
			}
		}
		if stars != stageW*stageH {
			t.Fatalf("%d stars, want the To glyph on every cell", stars)
		}
		if c.Frac() != 1 {
			t.Fatalf("Frac %v, want 1", c.Frac())
		}
	})
	t.Run("happy: halfway every cell wears the lerped floor", func(t *testing.T) {
		c := newFade()
		c.Start(stageW, stageH)
		defer c.Stop()
		_ = c.Render()
		tick(c, 0.5)
		want := LerpInk(fromInk, toInk, 0.5)
		sp := c.Render()
		for r := 0; r < stageH; r++ {
			for col := 0; col < stageW; col++ {
				if sp.At(r, col).BG != want {
					t.Fatalf("cell %d,%d BG %d, want lerp %d", col, r, sp.At(r, col).BG, want)
				}
			}
		}
	})
	t.Run("happy: Delay holds From until the delay elapses", func(t *testing.T) {
		c := Between(&floor{bg: fromInk}, &floor{bg: toInk}).Delay(1).Over(1)
		c.Start(stageW, stageH)
		defer c.Stop()
		_ = c.Render()
		tick(c, 0.9)
		if c.Frac() != 0 {
			t.Fatalf("still in the delay Frac %v, want 0", c.Frac())
		}
		if c.Render().At(0, 0).BG != fromInk {
			t.Fatal("the delay must keep painting From")
		}
		tick(c, 0.6)
		if c.Frac() <= 0 || c.Frac() >= 1 {
			t.Fatalf("into the walk Frac %v, want a mid value", c.Frac())
		}
	})
	t.Run("unhappy: a zero Over snaps to To the moment the delay is done", func(t *testing.T) {
		c := Between(&floor{bg: fromInk}, &floor{bg: toInk}).Delay(0.5).Over(0)
		c.Start(stageW, stageH)
		defer c.Stop()
		_ = c.Render()
		tick(c, 0.4)
		if c.Render().At(0, 0).BG != fromInk {
			t.Fatal("inside the delay a snap fade must still show From")
		}
		tick(c, 0.2)
		if c.Render().At(0, 0).BG != toInk {
			t.Fatal("a zero Over must land on To the instant the delay elapses")
		}
	})
	t.Run("unhappy: dt <= 0, a missing layer, and life outside Start never panic", func(t *testing.T) {
		c := Between(&floor{bg: fromInk}, &floor{bg: toInk}).Over(1)
		if sp := c.Render(); sp.Width != 0 {
			t.Fatalf("before Start the stage is %dx%d, want empty", sp.Width, sp.Height)
		}
		c.Update(1)
		c.Start(stageW, stageH)
		c.Update(0)
		c.Update(-1)
		if c.Frac() != 0 {
			t.Fatalf("non-positive dt moved Frac to %v", c.Frac())
		}
		c.Stop()
		if sp := c.Render(); sp.Width != 0 {
			t.Fatalf("after Stop the stage is %dx%d, want empty", sp.Width, sp.Height)
		}
		var none *Crossfade
		none.Start(4, 4)
		none.Update(1)
		_ = none.Render()
		none.Stop()
		Between(nil, nil).Start(4, 4)
	})
}

func TestStack(t *testing.T) {
	t.Run("happy: later layers sit on an earlier floor", func(t *testing.T) {
		base := &floor{bg: fromInk}
		over := &floor{bg: -1, ch: '░', fg: 255}
		s := Stack(base, over)
		s.Start(stageW, stageH)
		defer s.Stop()
		cell := s.Render().At(0, 0)
		if cell.BG != fromInk {
			t.Fatalf("BG %d, want the earlier floor %d", cell.BG, fromInk)
		}
		if cell.Ch != '░' {
			t.Fatalf("glyph %q, want the later layer", cell.Ch)
		}
	})
	t.Run("unhappy: an empty stack and nil layers paint nothing", func(t *testing.T) {
		s := Stack(nil, nil)
		s.Start(stageW, stageH)
		defer s.Stop()
		if cell := s.Render().At(0, 0); !cell.Transparent() {
			t.Fatalf("nil layers painted %+v", cell)
		}
		var none *Layers
		none.Start(4, 4)
		none.Update(1)
		_ = none.Render()
		none.Stop()
	})
}
