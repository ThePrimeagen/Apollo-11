package dust

// Tests written FIRST. The painter turns live particles into cells by
// concentration: a heavy cell is a half shade in light gray, a strong
// cell a quarter shade in mid gray, and the thin fringe is braille in
// deep gray — the exact dots computed from where each speck sits
// inside its cell, two dot columns wide and four dot rows tall, so one
// value covers every possible braille swirl.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

// rig is an engine whose particles are placed by hand.
func rig(ps ...particle.Particle) *particle.Engine {
	e := particle.New(1, particle.Config{
		Width: 100, Height: 100,
		Origin:    particle.Vec2{X: 50, Y: 50},
		Direction: particle.Vec2{X: 1, Y: 0},
	})
	e.Particles = ps
	return e
}

func at(x, y float64) particle.Particle {
	return particle.Particle{Pos: particle.Vec2{X: x, Y: y}, Life: 1}
}

func TestBraille(t *testing.T) {
	t.Run("happy: each corner of a cell earns its own braille dot", func(t *testing.T) {
		c := DefaultPuff()
		for _, tc := range []struct {
			x, y float64
			want rune
		}{
			{3.1, 4.1, '⠁'}, // top-left dot
			{3.6, 4.6, '⠐'}, // second row, right column
			{3.2, 5.2, '⠄'}, // third row, left column
			{3.9, 5.9, '⢀'}, // bottom-right dot
		} {
			sp := paint(c, 10, 5, rig(at(tc.x, tc.y)))
			got := sp.At(2, 3)
			if got.Ch != tc.want {
				t.Fatalf("speck at (%v, %v) painted %q, want %q", tc.x, tc.y, got.Ch, tc.want)
			}
			if got.FG != c.BrailleFG || got.BG != -1 {
				t.Fatalf("fringe dust must wear the deep gray %d over nothing, got fg=%d bg=%d", c.BrailleFG, got.FG, got.BG)
			}
		}
	})
	t.Run("happy: two specks in one cell merge into one braille glyph", func(t *testing.T) {
		sp := paint(DefaultPuff(), 10, 5, rig(at(3.1, 4.1), at(3.9, 5.9)))
		if got := sp.At(2, 3).Ch; got != '⢁' {
			t.Fatalf("merged cell painted %q, want %q", got, '⢁')
		}
	})
	t.Run("happy: specks merge across engines too", func(t *testing.T) {
		sp := paint(DefaultPuff(), 10, 5, rig(at(3.1, 4.1)), rig(at(3.9, 5.9)))
		if got := sp.At(2, 3).Ch; got != '⢁' {
			t.Fatalf("two engines on one cell painted %q, want %q", got, '⢁')
		}
	})
	t.Run("unhappy: dead specks, empty engines, and nil engines paint nothing", func(t *testing.T) {
		dead := at(3.1, 4.1)
		dead.Life = 0
		sp := paint(DefaultPuff(), 10, 5, rig(dead), rig(), nil)
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				if !sp.At(r, c).Transparent() {
					t.Fatalf("cell (%d,%d) painted %q for a dead speck", r, c, sp.At(r, c).Ch)
				}
			}
		}
	})
	t.Run("unhappy: specks outside the sprite are clipped, never a panic", func(t *testing.T) {
		sp := paint(DefaultPuff(), 4, 2, rig(at(50, 50), at(-1, -1)))
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				if !sp.At(r, c).Transparent() {
					t.Fatalf("cell (%d,%d) painted from an off-stage speck", r, c)
				}
			}
		}
	})
}

func TestPaintLadder(t *testing.T) {
	stack := func(n int) *particle.Engine {
		ps := make([]particle.Particle, n)
		for i := range ps {
			ps[i] = at(3.2, 4.3)
		}
		return rig(ps...)
	}
	c := DefaultPuff()
	c.QuarterAt = 3
	c.HalfAt = 6
	t.Run("happy: concentration climbs braille, quarter shade, half shade", func(t *testing.T) {
		for _, tc := range []struct {
			n    int
			want rune
			fg   int
		}{
			{1, '⠁', c.BrailleFG},
			{2, '⠁', c.BrailleFG},
			{3, '░', c.QuarterFG},
			{5, '░', c.QuarterFG},
			{6, '▒', c.HalfFG},
			{9, '▒', c.HalfFG},
		} {
			got := paint(c, 10, 5, stack(tc.n)).At(2, 3)
			if got.Ch != tc.want || got.FG != tc.fg {
				t.Fatalf("%d specks painted %q fg=%d, want %q fg=%d", tc.n, got.Ch, got.FG, tc.want, tc.fg)
			}
		}
	})
	t.Run("unhappy: untouched cells stay transparent sky", func(t *testing.T) {
		sp := paint(c, 10, 5, stack(4))
		if !sp.At(0, 0).Transparent() || !sp.At(4, 9).Transparent() {
			t.Fatal("cells without dust must stay transparent")
		}
		if sp.Width != 10 || sp.Height != 5 {
			t.Fatalf("the painter must return the asked stage, got %dx%d", sp.Width, sp.Height)
		}
	})
}
