package fire

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestBooster(t *testing.T) {
	t.Run("happy: 2500 live particles, 5 per 1ms, normal spread, left-to-right", func(t *testing.T) {
		f := Booster(1)
		if err := f.Eng.Validate(); err != nil {
			t.Fatalf("booster config: %v", err)
		}
		cfg := f.Eng.Cfg
		if cfg.Count != 5 {
			t.Fatalf("count %d, want 5 particles per spawn", cfg.Count)
		}
		if math.Abs(cfg.Period-0.001) > 1e-9 {
			t.Fatalf("period %v, want 1ms", cfg.Period)
		}
		if cfg.Spread <= 0 {
			t.Fatal("spread must be a normal around the direction")
		}
		d := cfg.Direction
		if math.Abs(d.X-1) > 1e-9 || math.Abs(d.Y) > 1e-9 {
			t.Fatalf("direction %+v, want (1, 0)", d)
		}
		warm(f, 0.55)
		n := len(f.Eng.Particles)
		if n < 2000 || n > 3250 {
			t.Fatalf("live %d, want ~2500", n)
		}
		var near, far int
		for _, p := range f.Eng.Particles {
			ang := math.Atan2(p.Vel.Y, p.Vel.X)
			if math.Abs(ang) < cfg.Spread*0.7 {
				near++
			}
			if math.Abs(ang) > cfg.Spread*1.5 {
				far++
			}
		}
		if near <= far {
			t.Fatalf("normal should be denser on the axis, near=%d far=%d", near, far)
		}
	})
	t.Run("happy: a warmed booster uses more than one glyph", func(t *testing.T) {
		f := Booster(2)
		warm(f, 0.55)
		seen := map[rune]int{}
		sp := f.Sprite()
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				ch := sp.At(r, c).Ch
				if ch != 0 && ch != ' ' {
					seen[ch]++
				}
			}
		}
		if len(seen) < 3 {
			t.Fatalf("expected a mix of styles, got %v", seen)
		}
	})
	t.Run("unhappy: dt<=0 still emits nothing", func(t *testing.T) {
		f := Booster(3)
		f.Update(0)
		if len(f.Eng.Particles) != 0 {
			t.Fatal("dt<=0 must be a no-op")
		}
	})
	t.Run("unhappy: a lone particle is a dark-red dot, not a hole and not yellow", func(t *testing.T) {
		f := Booster(4)
		f.Eng.Particles = []particle.Particle{{
			Pos:  particle.Vec2{X: 3.2, Y: 3.0},
			Life: 1,
		}}
		sp := f.Sprite()
		var lit sprite.Cell
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				if !sp.At(r, c).Transparent() {
					lit = sp.At(r, c)
				}
			}
		}
		if lit.Ch != '⠁' {
			t.Fatalf("H=1 should be a single braille, got %+v", lit)
		}
		if lit.Ch == '█' {
			t.Fatal("a lone particle must not be solid yellow")
		}
	})
}
