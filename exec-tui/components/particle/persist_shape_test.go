package particle

// Tests written FIRST. Persist trails are a triangle, not a ribbon:
// Peak concentrates spawn on the origin (a steep power, not a mild
// normal — Peak<=1 is the old uniform slit) and Taper cuts max life
// by how far a speck sits from that spine, so the outsides die first.

import (
	"errors"
	"math"
	"testing"
)

func TestPersistPeak(t *testing.T) {
	t.Run("happy: a steep peak piles specks on the origin and still allows a thin fringe", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Nozzle = 10
		cfg.Peak = 8
		cfg.Count = 400
		cfg.MinLife, cfg.MaxLife = 10, 10
		e := New(11, cfg)
		e.Burst()
		if len(e.Particles) != 400 {
			t.Fatalf("burst %d, want 400", len(e.Particles))
		}
		inner, outer, farthest := 0, 0, 0.0
		originY := cfg.Origin.Y
		for i, p := range e.Particles {
			if math.Abs(p.Pos.X-cfg.Origin.X) > 1e-9 {
				t.Fatalf("a rightward nozzle is vertical: particle %d left X=%v", i, p.Pos.X)
			}
			d := math.Abs(p.Pos.Y - originY)
			if d < 0.5 {
				inner++
			}
			if d > 3 {
				outer++
			}
			if d > farthest {
				farthest = d
			}
		}
		if inner < 250 {
			t.Fatalf("peak 8 must park most specks on the spine, inner=%d", inner)
		}
		if inner < outer*6 {
			t.Fatalf("the spine must dwarf the fringe: inner=%d outer=%d", inner, outer)
		}
		if farthest < 1 {
			t.Fatalf("peak still has to allow a spread, farthest=%.2f", farthest)
		}
		if farthest > cfg.Nozzle/2+1e-9 {
			t.Fatalf("no speck may leave the nozzle, farthest=%.2f nozzle/2=%.2f", farthest, cfg.Nozzle/2)
		}
	})
	t.Run("happy: Peak<=1 keeps the old uniform slit", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Nozzle = 10
		cfg.Peak = 1
		cfg.Count = 400
		e := New(13, cfg)
		e.Burst()
		inner, outer := 0, 0
		for _, p := range e.Particles {
			d := math.Abs(p.Pos.Y - cfg.Origin.Y)
			if d < 0.5 {
				inner++
			}
			if d > 3 {
				outer++
			}
		}
		if inner >= outer {
			t.Fatalf("a uniform slit is wider on the outside: inner=%d outer=%d", inner, outer)
		}
	})
	t.Run("unhappy: a negative peak is rejected and Peak does not throw flying exhaust", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Peak = -1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrPeak) {
			t.Fatalf("got %v, want ErrPeak", err)
		}
		fly := testCfg()
		fly.Nozzle = 8
		fly.Peak = 12
		fly.Spread = 0
		fly.Direction = Vec2{X: 1, Y: 0}
		fly.Origin = Vec2{X: 4, Y: 10}
		fly.Count = 200
		e := New(9, fly)
		e.Update(0.01)
		var minY, maxY float64
		minY = 1e9
		for _, p := range e.Particles {
			if p.Pos.Y < minY {
				minY = p.Pos.Y
			}
			if p.Pos.Y > maxY {
				maxY = p.Pos.Y
			}
		}
		if maxY-minY < 4 {
			t.Fatalf("straight-mode Peak must not pinch the gun nozzle, span=%.2f", maxY-minY)
		}
	})
}

func TestPersistTaper(t *testing.T) {
	t.Run("happy: taper=1 gives the spine the long life and the fringe MinLife", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Nozzle = 10
		cfg.Peak = 1
		cfg.Taper = 1
		cfg.Count = 300
		cfg.MinLife, cfg.MaxLife = 0.2, 2.0
		e := New(17, cfg)
		e.Burst()
		if len(e.Particles) != 300 {
			t.Fatalf("burst %d, want 300", len(e.Particles))
		}
		half := cfg.Nozzle / 2
		var spineLife, fringeLife float64
		spine, fringe := 0, 0
		for i, p := range e.Particles {
			d := math.Abs(p.Pos.Y - cfg.Origin.Y)
			tFrac := d / half
			if tFrac > 1 {
				tFrac = 1
			}
			maxHere := cfg.MaxLife - tFrac*(cfg.MaxLife-cfg.MinLife)
			if p.Life > maxHere+1e-9 {
				t.Fatalf("particle %d life %.3f exceeds the taper ceiling %.3f at |off|=%.2f", i, p.Life, maxHere, d)
			}
			if p.Life < cfg.MinLife-1e-9 {
				t.Fatalf("particle %d life %.3f fell under MinLife", i, p.Life)
			}
			if d < 0.6 {
				spine++
				spineLife += p.Life
			}
			if d > 3.5 {
				fringe++
				fringeLife += p.Life
			}
		}
		if spine == 0 || fringe == 0 {
			t.Fatalf("need both spine and fringe samples, spine=%d fringe=%d", spine, fringe)
		}
		if spineLife/float64(spine) <= fringeLife/float64(fringe)+0.4 {
			t.Fatalf("spine should outlive the fringe: spine avg %.2f fringe avg %.2f", spineLife/float64(spine), fringeLife/float64(fringe))
		}
	})
	t.Run("happy: taper=0 ignores offset so the fringe can still roll MaxLife", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Nozzle = 10
		cfg.Peak = 1
		cfg.Taper = 0
		cfg.Count = 300
		cfg.MinLife, cfg.MaxLife = 0.2, 2.0
		e := New(19, cfg)
		e.Burst()
		longFringe := 0
		for _, p := range e.Particles {
			d := math.Abs(p.Pos.Y - cfg.Origin.Y)
			if d > 3.5 && p.Life > 1.5 {
				longFringe++
			}
		}
		if longFringe == 0 {
			t.Fatal("with taper off, an outside speck must still be allowed a long life")
		}
	})
	t.Run("unhappy: taper outside 0..1 is rejected and a zero nozzle cannot shrink life", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Taper = -0.1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrTaper) {
			t.Fatalf("got %v, want ErrTaper", err)
		}
		cfg.Taper = 1.1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrTaper) {
			t.Fatalf("taper 1.1 got %v, want ErrTaper", err)
		}
		cfg = persistCfg()
		cfg.Nozzle = 0
		cfg.Taper = 1
		cfg.Count = 40
		cfg.MinLife, cfg.MaxLife = 0.2, 2.0
		e := New(3, cfg)
		e.Burst()
		for i, p := range e.Particles {
			if p.Pos != cfg.Origin {
				t.Fatalf("nozzle 0 must park on the origin, particle %d at %+v", i, p.Pos)
			}
			if p.Life < 0.2-1e-9 || p.Life > 2+1e-9 {
				t.Fatalf("nozzle 0 has no offset to taper: particle %d life %.3f", i, p.Life)
			}
		}
	})
}
