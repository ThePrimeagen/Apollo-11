package particle

// Tests written FIRST. The particle engine is a box, a nozzle, a direction,
// a count and a period. Update is the only clock: it moves, it kills, it
// emits. Occupancy counts live particles per cell. The package does not draw.

import (
	"errors"
	"math"
	"testing"
)

func testCfg() Config {
	return Config{
		Width:     40,
		Height:    20,
		Origin:    Vec2{X: 20, Y: 10},
		Direction: Vec2{X: -1, Y: 0},
		Count:     20,
		Period:    0.1,
		MinLife:   0.2,
		MaxLife:   0.6,
		MinSpeed:  4,
		MaxSpeed:  8,
		Spread:    0.25,
	}
}

func TestValidate(t *testing.T) {
	t.Run("happy: a complete config is accepted", func(t *testing.T) {
		e := New(1, testCfg())
		if err := e.Validate(); err != nil {
			t.Fatalf("valid config: %v", err)
		}
	})
	t.Run("happy: zero spread and matching min/max ranges are accepted", func(t *testing.T) {
		cfg := testCfg()
		cfg.Spread = 0
		cfg.MinSpeed, cfg.MaxSpeed = 5, 5
		cfg.MinLife, cfg.MaxLife = 0.4, 0.4
		if err := New(1, cfg).Validate(); err != nil {
			t.Fatalf("tight ranges must pass: %v", err)
		}
	})
	t.Run("unhappy: a zero direction is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.Direction = Vec2{}
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrDirection) {
			t.Fatalf("got %v, want ErrDirection", err)
		}
	})
	t.Run("unhappy: a reversed speed range is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.MinSpeed, cfg.MaxSpeed = 10, 1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrSpeed) {
			t.Fatalf("got %v, want ErrSpeed", err)
		}
	})
	t.Run("unhappy: a reversed life range is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.MinLife, cfg.MaxLife = 2, 0.1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrLife) {
			t.Fatalf("got %v, want ErrLife", err)
		}
	})
	t.Run("unhappy: a non-positive box is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.Width = 0
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrSize) {
			t.Fatalf("width=0: got %v, want ErrSize", err)
		}
		cfg = testCfg()
		cfg.Height = -3
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrSize) {
			t.Fatalf("height<0: got %v, want ErrSize", err)
		}
	})
	t.Run("unhappy: an origin outside the box is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.Origin = Vec2{X: -0.1, Y: 10}
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrOrigin) {
			t.Fatalf("got %v, want ErrOrigin", err)
		}
	})
	t.Run("unhappy: a negative period, count, or spread is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.Period = -1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrPeriod) {
			t.Fatalf("period: got %v, want ErrPeriod", err)
		}
		cfg = testCfg()
		cfg.Count = -4
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrCount) {
			t.Fatalf("count: got %v, want ErrCount", err)
		}
		cfg = testCfg()
		cfg.Spread = -0.2
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrSpread) {
			t.Fatalf("spread: got %v, want ErrSpread", err)
		}
	})
}

func TestNew(t *testing.T) {
	t.Run("happy: a non-unit direction is stored normalized", func(t *testing.T) {
		cfg := testCfg()
		cfg.Direction = Vec2{X: 0, Y: 4}
		e := New(1, cfg)
		if math.Abs(e.Cfg.Direction.X) > 1e-12 || math.Abs(e.Cfg.Direction.Y-1) > 1e-12 {
			t.Fatalf("direction %+v, want (0, 1)", e.Cfg.Direction)
		}
	})
	t.Run("unhappy: New does not emit; the clock has not started", func(t *testing.T) {
		e := New(1, testCfg())
		if len(e.Particles) != 0 {
			t.Fatalf("New must not emit, have %d particles", len(e.Particles))
		}
	})
}

func TestUpdateEmit(t *testing.T) {
	t.Run("happy: the first Update emits Count particles at the origin", func(t *testing.T) {
		e := New(1, testCfg())
		e.Update(0.01)
		if len(e.Particles) != 20 {
			t.Fatalf("first emit %d, want 20", len(e.Particles))
		}
		for i, p := range e.Particles {
			if p.Pos != e.Cfg.Origin {
				t.Fatalf("particle %d spawned at %+v, want origin %+v", i, p.Pos, e.Cfg.Origin)
			}
		}
	})
	t.Run("happy: a later period emits another batch", func(t *testing.T) {
		e := New(1, testCfg())
		e.Update(0.01)
		e.Update(0.1)
		if len(e.Particles) != 40 {
			t.Fatalf("after two periods %d, want 40", len(e.Particles))
		}
	})
	t.Run("happy: one large dt catches up missed periods", func(t *testing.T) {
		cfg := testCfg()
		cfg.Period = 0.1
		cfg.MinLife, cfg.MaxLife = 10, 10
		e := New(1, cfg)
		e.Update(0.35)
		if len(e.Particles) != 80 {
			t.Fatalf("dt=0.35 / period=0.1 should emit 4 batches, got %d", len(e.Particles))
		}
	})
	t.Run("unhappy: a zero count emits nothing", func(t *testing.T) {
		cfg := testCfg()
		cfg.Count = 0
		e := New(1, cfg)
		e.Update(1)
		if len(e.Particles) != 0 {
			t.Fatalf("Count=0 must emit 0, got %d", len(e.Particles))
		}
	})
	t.Run("unhappy: a zero period never auto-emits", func(t *testing.T) {
		cfg := testCfg()
		cfg.Period = 0
		e := New(1, cfg)
		e.Update(1)
		if len(e.Particles) != 0 {
			t.Fatalf("Period=0 must not emit, got %d", len(e.Particles))
		}
	})
	t.Run("unhappy: a zero dt does not emit, move, or kill", func(t *testing.T) {
		e := New(1, testCfg())
		e.Update(0.01)
		n := len(e.Particles)
		x := avgX(e.Particles)
		e.Update(0)
		e.Update(-1)
		if len(e.Particles) != n || avgX(e.Particles) != x {
			t.Fatal("dt<=0 must be a no-op")
		}
	})
}

func TestUpdateMotion(t *testing.T) {
	t.Run("happy: particles travel along the direction", func(t *testing.T) {
		e := New(1, testCfg())
		e.Update(0.01)
		e.Update(0.15)
		if avgX(e.Particles) >= e.Cfg.Origin.X {
			t.Fatalf("leftward emit should move particles left, avgX=%.2f origin=%.2f", avgX(e.Particles), e.Cfg.Origin.X)
		}
	})
	t.Run("happy: every life and speed stays inside the configured range", func(t *testing.T) {
		cfg := testCfg()
		e := New(2, cfg)
		e.Update(0.01)
		if len(e.Particles) == 0 {
			t.Fatal("need particles")
		}
		for _, p := range e.Particles {
			if p.Life < cfg.MinLife-1e-9 || p.Life > cfg.MaxLife+1e-9 {
				t.Fatalf("life %.3f outside [%.2f, %.2f]", p.Life, cfg.MinLife, cfg.MaxLife)
			}
			spd := p.Vel.Len()
			if spd < cfg.MinSpeed-1e-6 || spd > cfg.MaxSpeed+1e-6 {
				t.Fatalf("speed %.3f outside [%.2f, %.2f]", spd, cfg.MinSpeed, cfg.MaxSpeed)
			}
		}
	})
	t.Run("happy: a normal spread puts more particles on the axis than on the fringe", func(t *testing.T) {
		cfg := testCfg()
		cfg.Count = 400
		cfg.Spread = 0.35
		e := New(7, cfg)
		e.Update(0.01)
		var near, far int
		for _, p := range e.Particles {
			angle := math.Atan2(p.Vel.Y, p.Vel.X) - math.Atan2(e.Cfg.Direction.Y, e.Cfg.Direction.X)
			for angle > math.Pi {
				angle -= 2 * math.Pi
			}
			for angle < -math.Pi {
				angle += 2 * math.Pi
			}
			if math.Abs(angle) < cfg.Spread*0.6 {
				near++
			}
			if math.Abs(angle) > cfg.Spread*1.4 {
				far++
			}
		}
		if near <= far {
			t.Fatalf("center of the cone should be denser than the fringe, near=%d far=%d", near, far)
		}
	})
	t.Run("happy: zero spread keeps every heading on the axis", func(t *testing.T) {
		cfg := testCfg()
		cfg.Spread = 0
		cfg.Count = 30
		e := New(3, cfg)
		e.Update(0.01)
		for i, p := range e.Particles {
			cross := p.Vel.X*e.Cfg.Direction.Y - p.Vel.Y*e.Cfg.Direction.X
			if math.Abs(cross) > 1e-9 {
				t.Fatalf("particle %d left the axis, vel=%+v", i, p.Vel)
			}
		}
	})
	t.Run("happy: particles die when life runs out", func(t *testing.T) {
		cfg := testCfg()
		cfg.MinLife, cfg.MaxLife = 0.1, 0.2
		e := New(3, cfg)
		e.Update(0.01)
		if len(e.Particles) == 0 {
			t.Fatal("need particles")
		}
		e.Cfg.Period = 0
		e.Update(2)
		if len(e.Particles) != 0 {
			t.Fatalf("after 2s every particle should be dead, still %d", len(e.Particles))
		}
	})
	t.Run("happy: particles die when they leave the box", func(t *testing.T) {
		cfg := testCfg()
		cfg.Width, cfg.Height = 8, 8
		cfg.Origin = Vec2{X: 7, Y: 4}
		cfg.Direction = Vec2{X: 1, Y: 0}
		cfg.MinSpeed, cfg.MaxSpeed = 10, 10
		cfg.MinLife, cfg.MaxLife = 10, 10
		cfg.Spread = 0
		cfg.Count = 5
		cfg.Period = 1
		e := New(4, cfg)
		e.Update(0.01)
		if len(e.Particles) != 5 {
			t.Fatalf("emit %d, want 5", len(e.Particles))
		}
		e.Update(0.5)
		if len(e.Particles) != 0 {
			t.Fatalf("x=12 is outside width 8, still %d live", len(e.Particles))
		}
	})
}

func TestOccupancy(t *testing.T) {
	t.Run("happy: ten particles stacked on one cell report count 10", func(t *testing.T) {
		e := New(1, testCfg())
		e.Particles = make([]Particle, 10)
		for i := range e.Particles {
			e.Particles[i] = Particle{Pos: Vec2{X: 3.2, Y: 5.1}, Life: 1}
		}
		occ := e.Occupancy()
		if occ[CellOf(3.2, 5.1)] != 10 {
			t.Fatalf("stacked cell count %d, want 10", occ[CellOf(3.2, 5.1)])
		}
	})
	t.Run("happy: CellOf uses square units (cell height is two units)", func(t *testing.T) {
		c := CellOf(3.2, 5.1)
		if c.Col != 3 || c.Row != 2 {
			t.Fatalf("CellOf(3.2, 5.1)=%+v, want {Col:3 Row:2}", c)
		}
	})
	t.Run("unhappy: an empty engine has no occupied cells", func(t *testing.T) {
		e := New(1, testCfg())
		if len(e.Occupancy()) != 0 {
			t.Fatal("empty occupancy must be empty")
		}
	})
	t.Run("unhappy: a dead particle does not occupy a cell", func(t *testing.T) {
		e := New(1, testCfg())
		e.Particles = []Particle{{Pos: Vec2{X: 1, Y: 1}, Life: 0}}
		if len(e.Occupancy()) != 0 {
			t.Fatal("Life<=0 must not count")
		}
	})
}

func avgX(ps []Particle) float64 {
	if len(ps) == 0 {
		return 0
	}
	var sum float64
	for _, p := range ps {
		sum += p.Pos.X
	}
	return sum / float64(len(ps))
}
