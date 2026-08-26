package gunfire

// Tests written FIRST. The painter is the Doom muzzle palette on the
// xterm cube. The flash climbs a concentration ladder from an orange
// fringe dot to a white-hot core block. Pellets are pale tracer heads
// dragging a dim straw trail one unit behind. Sparks cool through the
// fire ramp — yellow, orange, red — as they age. Smoke is computed
// braille in grays that dim with age, thickening to a shade block
// where it piles up. Layers stack smoke → sparks → pellets → flash.

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

func aged(x, y, age, life float64) particle.Particle {
	return particle.Particle{Pos: particle.Vec2{X: x, Y: y}, Age: age, Life: life}
}

func stack(n int, x, y float64) *particle.Engine {
	ps := make([]particle.Particle, n)
	for i := range ps {
		ps[i] = at(x, y)
	}
	return rig(ps...)
}

func TestFlashLadder(t *testing.T) {
	c := DefaultBlast()
	c.EdgeAt, c.MidAt, c.CoreAt = 2, 4, 7
	t.Run("happy: concentration climbs fringe dot, edge star, mid shade, white-hot core", func(t *testing.T) {
		for _, tc := range []struct {
			n      int
			want   rune
			fg, bg int
		}{
			{1, '·', 214, -1},
			{2, '*', 220, -1},
			{3, '*', 220, -1},
			{4, '▓', 226, -1},
			{6, '▓', 226, -1},
			{7, '█', 231, 220},
			{12, '█', 231, 220},
		} {
			got := paint(c, 10, 5, stack(tc.n, 3.2, 4.3), nil, nil, nil).At(2, 3)
			if got.Ch != tc.want || got.FG != tc.fg || got.BG != tc.bg {
				t.Fatalf("%d flash specks painted %q fg=%d bg=%d, want %q fg=%d bg=%d",
					tc.n, got.Ch, got.FG, got.BG, tc.want, tc.fg, tc.bg)
			}
		}
	})
	t.Run("unhappy: dead flash specks and empty engines paint nothing", func(t *testing.T) {
		dead := at(3.2, 4.3)
		dead.Life = 0
		sp := paint(c, 10, 5, rig(dead), rig(), nil, nil)
		for r := 0; r < sp.Height; r++ {
			for col := 0; col < sp.Width; col++ {
				if !sp.At(r, col).Transparent() {
					t.Fatalf("cell (%d,%d) painted %q for a dead flash", r, col, sp.At(r, col).Ch)
				}
			}
		}
	})
}

func TestPellets(t *testing.T) {
	t.Run("happy: a flying pellet paints a pale head dragging a dim trail one unit behind", func(t *testing.T) {
		p := aged(10.5, 8.3, 0.1, 0.5)
		p.Vel = particle.Vec2{X: 60, Y: 0}
		sp := paint(DefaultBlast(), 20, 10, nil, rig(p), nil, nil)
		head := sp.At(4, 10)
		if head.Ch != '•' || head.FG != 230 {
			t.Fatalf("head painted %q fg=%d, want '•' fg=230", head.Ch, head.FG)
		}
		trail := sp.At(4, 9)
		if trail.Ch != '·' || trail.FG != 178 {
			t.Fatalf("trail painted %q fg=%d, want '·' fg=178", trail.Ch, trail.FG)
		}
	})
	t.Run("unhappy: a newborn pellet shows no trail poking out of the muzzle", func(t *testing.T) {
		p := aged(10.5, 8.3, 0, 0.5)
		p.Vel = particle.Vec2{X: 60, Y: 0}
		sp := paint(DefaultBlast(), 20, 10, nil, rig(p), nil, nil)
		if got := sp.At(4, 10).Ch; got != '•' {
			t.Fatalf("the newborn head must still paint, got %q", got)
		}
		if !sp.At(4, 9).Transparent() {
			t.Fatalf("a newborn pellet dragged a trail %q behind the muzzle", sp.At(4, 9).Ch)
		}
	})
}

func TestSparks(t *testing.T) {
	t.Run("happy: sparks cool yellow, orange, then ember red as they age", func(t *testing.T) {
		for _, tc := range []struct {
			age, life float64
			want      rune
			fg        int
		}{
			{0.1, 0.9, '*', 226}, // a tenth in: yellow
			{1, 1, '+', 208},     // half spent: orange
			{4, 1, '·', 160},     // nearly out: ember red
		} {
			sp := paint(DefaultBlast(), 10, 5, nil, nil, rig(aged(3.2, 4.3, tc.age, tc.life)), nil)
			got := sp.At(2, 3)
			if got.Ch != tc.want || got.FG != tc.fg {
				t.Fatalf("a spark %v/%v old painted %q fg=%d, want %q fg=%d",
					tc.age, tc.life, got.Ch, got.FG, tc.want, tc.fg)
			}
		}
	})
	t.Run("unhappy: specks off the stage are clipped, never a panic", func(t *testing.T) {
		sp := paint(DefaultBlast(), 4, 2, nil, nil, rig(at(50, 50), at(-1, -1)), nil)
		for r := 0; r < sp.Height; r++ {
			for col := 0; col < sp.Width; col++ {
				if !sp.At(r, col).Transparent() {
					t.Fatalf("cell (%d,%d) painted from an off-stage spark", r, col)
				}
			}
		}
	})
}

func TestSmoke(t *testing.T) {
	t.Run("happy: thin smoke wears computed braille dots that dim with age", func(t *testing.T) {
		for _, tc := range []struct {
			x, y float64
			want rune
		}{
			{3.1, 4.1, '⠁'}, // top-left dot
			{3.6, 4.6, '⠐'}, // second row, right column
			{3.9, 5.9, '⢀'}, // bottom-right dot
		} {
			sp := paint(DefaultBlast(), 10, 5, nil, nil, nil, rig(at(tc.x, tc.y)))
			got := sp.At(2, 3)
			if got.Ch != tc.want || got.FG != 250 {
				t.Fatalf("young smoke at (%v,%v) painted %q fg=%d, want %q fg=250",
					tc.x, tc.y, got.Ch, got.FG, tc.want)
			}
		}
		mid := paint(DefaultBlast(), 10, 5, nil, nil, nil, rig(aged(3.1, 4.1, 1, 1))).At(2, 3)
		if mid.FG != 245 {
			t.Fatalf("half-spent smoke wears fg=%d, want 245", mid.FG)
		}
		old := paint(DefaultBlast(), 10, 5, nil, nil, nil, rig(aged(3.1, 4.1, 8, 1))).At(2, 3)
		if old.FG != 240 {
			t.Fatalf("dying smoke wears fg=%d, want 240", old.FG)
		}
	})
	t.Run("happy: two specks merge dots and three thicken into a shade block", func(t *testing.T) {
		merged := paint(DefaultBlast(), 10, 5, nil, nil, nil, rig(at(3.1, 4.1), at(3.9, 5.9))).At(2, 3)
		if merged.Ch != '⢁' {
			t.Fatalf("two specks merged into %q, want %q", merged.Ch, '⢁')
		}
		thick := paint(DefaultBlast(), 10, 5, nil, nil, nil, stack(3, 3.2, 4.3)).At(2, 3)
		if thick.Ch != '░' || thick.FG != 245 {
			t.Fatalf("three specks painted %q fg=%d, want '░' fg=245", thick.Ch, thick.FG)
		}
	})
	t.Run("unhappy: dead smoke and nil engines paint nothing", func(t *testing.T) {
		dead := at(3.1, 4.1)
		dead.Life = 0
		sp := paint(DefaultBlast(), 10, 5, nil, nil, nil, rig(dead))
		if !sp.At(2, 3).Transparent() {
			t.Fatalf("a dead smoke speck painted %q", sp.At(2, 3).Ch)
		}
		sp = paint(DefaultBlast(), 10, 5, nil, nil, nil, nil)
		for r := 0; r < sp.Height; r++ {
			for col := 0; col < sp.Width; col++ {
				if !sp.At(r, col).Transparent() {
					t.Fatalf("an all-nil paint marked cell (%d,%d)", r, col)
				}
			}
		}
	})
}

func TestZOrder(t *testing.T) {
	t.Run("happy: the flash outshines smoke and sparks sharing its cell", func(t *testing.T) {
		flash := rig(at(3.2, 4.2))
		sparks := rig(at(3.3, 4.3))
		smoke := rig(at(3.4, 4.4))
		got := paint(DefaultBlast(), 10, 5, flash, nil, sparks, smoke).At(2, 3)
		if got.Ch != '·' || got.FG != 214 {
			t.Fatalf("the shared cell wears %q fg=%d, want the flash fringe '·' fg=214", got.Ch, got.FG)
		}
	})
	t.Run("unhappy: an empty layer never masks the layers beneath", func(t *testing.T) {
		smoke := rig(at(3.1, 4.1))
		got := paint(DefaultBlast(), 10, 5, rig(), rig(), rig(), smoke).At(2, 3)
		if got.Ch != '⠁' {
			t.Fatalf("empty upper layers must let the smoke through, got %q", got.Ch)
		}
		if sp := paint(DefaultBlast(), 10, 5, rig(), rig(), rig(), rig()); !sp.At(2, 3).Transparent() {
			t.Fatal("four empty layers must paint nothing")
		}
	})
}
