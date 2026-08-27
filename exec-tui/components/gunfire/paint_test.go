package gunfire

// Tests written FIRST. The painter burns each compass direction in
// its own colors. A flame cell's glyph thickens with how many specks
// share it (░ ▒ ▓ █) and its color walks that DIRECTION's five-stop
// ramp by the age of the cell's freshest speck — so the stock shots
// cool through Doom red while a retuned East shot can burn plasma
// blue, side by side. The shared core climbs the config's
// concentration ladder to a white-hot block and outshines any flame
// it touches.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
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

// flamesWith lays engines onto the eight compass slots: pairs of
// (heading, engine), every other slot nil.
func flamesWith(pairs ...any) []*particle.Engine {
	flames := make([]*particle.Engine, len(sprite.Headings))
	for i := 0; i < len(pairs); i += 2 {
		h := pairs[i].(sprite.Heading)
		e := pairs[i+1].(*particle.Engine)
		for j, hh := range sprite.Headings {
			if hh == h {
				flames[j] = e
			}
		}
	}
	return flames
}

func TestCoreLadder(t *testing.T) {
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
			{4, '▓', 226, -1},
			{7, '█', 231, 220},
			{12, '█', 231, 220},
		} {
			got := paint(c, 10, 5, stack(tc.n, 3.2, 4.3), nil).At(2, 3)
			if got.Ch != tc.want || got.FG != tc.fg || got.BG != tc.bg {
				t.Fatalf("%d core specks painted %q fg=%d bg=%d, want %q fg=%d bg=%d",
					tc.n, got.Ch, got.FG, got.BG, tc.want, tc.fg, tc.bg)
			}
		}
	})
	t.Run("unhappy: dead core specks and empty engines paint nothing", func(t *testing.T) {
		dead := at(3.2, 4.3)
		dead.Life = 0
		sp := paint(c, 10, 5, rig(dead), flamesWith(sprite.N, rig()))
		for r := 0; r < sp.Height; r++ {
			for col := 0; col < sp.Width; col++ {
				if !sp.At(r, col).Transparent() {
					t.Fatalf("cell (%d,%d) painted %q for a dead core", r, col, sp.At(r, col).Ch)
				}
			}
		}
	})
}

func TestFlameRamp(t *testing.T) {
	t.Run("happy: a flame cell walks its direction's five stops as it ages", func(t *testing.T) {
		for _, tc := range []struct {
			age, life float64
			fg        int
		}{
			{0.1, 0.9, 226}, // a tenth burnt: stop one
			{3, 7, 208},     // three tenths: stop two
			{1, 1, 196},     // half burnt: stop three
			{7, 3, 160},     // seven tenths: stop four
			{9, 1, 124},     // nearly out: stop five
		} {
			sp := paint(DefaultBlast(), 10, 5, nil, flamesWith(sprite.N, rig(aged(3.2, 4.3, tc.age, tc.life))))
			got := sp.At(2, 3)
			if got.Ch != '░' || got.FG != tc.fg {
				t.Fatalf("a flame speck %v/%v burnt painted %q fg=%d, want '░' fg=%d",
					tc.age, tc.life, got.Ch, got.FG, tc.fg)
			}
		}
	})
	t.Run("happy: every direction burns its own colors — a blue East beside a red North", func(t *testing.T) {
		c := DefaultBlast()
		east := c.ShotAt(sprite.E)
		east.Colors = [5]int{21, 27, 33, 39, 45}
		c.SetShot(sprite.E, east)
		sp := paint(c, 10, 5, nil, flamesWith(
			sprite.N, rig(at(3.2, 4.3)),
			sprite.E, rig(at(6.2, 4.3)),
		))
		if got := sp.At(2, 3); got.FG != 226 {
			t.Fatalf("the N speck wears fg=%d, want its own red-ramp 226", got.FG)
		}
		if got := sp.At(2, 6); got.FG != 21 {
			t.Fatalf("the E speck wears fg=%d, want its own blue-ramp 21", got.FG)
		}
	})
	t.Run("happy: a flame cell thickens with density — shades up to a full block", func(t *testing.T) {
		for _, tc := range []struct {
			n    int
			want rune
		}{
			{1, '░'},
			{2, '▒'},
			{4, '▓'},
			{6, '█'},
			{9, '█'},
		} {
			got := paint(DefaultBlast(), 10, 5, nil, flamesWith(sprite.N, stack(tc.n, 3.2, 4.3))).At(2, 3)
			if got.Ch != tc.want || got.FG != 226 {
				t.Fatalf("%d fresh flame specks painted %q fg=%d, want %q fg=226",
					tc.n, got.Ch, got.FG, tc.want)
			}
		}
	})
	t.Run("happy: the freshest speck colors the cell — the hottest wins", func(t *testing.T) {
		young := aged(3.2, 4.2, 0.1, 0.9)
		old := aged(3.4, 4.4, 9, 1)
		got := paint(DefaultBlast(), 10, 5, nil, flamesWith(sprite.N, rig(young, old))).At(2, 3)
		if got.Ch != '▒' || got.FG != 226 {
			t.Fatalf("a young and an old speck painted %q fg=%d, want '▒' fg=226", got.Ch, got.FG)
		}
	})
	t.Run("unhappy: dead flame specks paint nothing and off-stage specks are clipped", func(t *testing.T) {
		dead := at(3.2, 4.3)
		dead.Life = 0
		sp := paint(DefaultBlast(), 4, 2, nil, flamesWith(sprite.N, rig(dead, at(50, 50), at(-1, -1))))
		for r := 0; r < sp.Height; r++ {
			for col := 0; col < sp.Width; col++ {
				if !sp.At(r, col).Transparent() {
					t.Fatalf("cell (%d,%d) painted %q from a dead or off-stage speck", r, col, sp.At(r, col).Ch)
				}
			}
		}
	})
}

func TestZOrder(t *testing.T) {
	t.Run("happy: the white-hot core outshines any direction's flame sharing its cell", func(t *testing.T) {
		core := rig(at(3.2, 4.2))
		flame := rig(at(3.3, 4.3))
		got := paint(DefaultBlast(), 10, 5, core, flamesWith(sprite.W, flame)).At(2, 3)
		if got.Ch != '·' || got.FG != 214 {
			t.Fatalf("the shared cell wears %q fg=%d, want the core fringe '·' fg=214", got.Ch, got.FG)
		}
	})
	t.Run("unhappy: an empty core and empty directions never mask the flame beneath", func(t *testing.T) {
		flame := rig(at(3.1, 4.1))
		got := paint(DefaultBlast(), 10, 5, rig(), flamesWith(sprite.SE, flame)).At(2, 3)
		if got.Ch != '░' || got.FG != 226 {
			t.Fatalf("an empty core must let the SE flame through, got %q fg=%d", got.Ch, got.FG)
		}
		if sp := paint(DefaultBlast(), 10, 5, nil, nil); !sp.At(2, 3).Transparent() {
			t.Fatal("no engines at all must paint nothing")
		}
	})
}
