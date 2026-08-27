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

func TestNozzle(t *testing.T) {
	t.Run("happy: a thick nozzle spawns across the slit, not a point", func(t *testing.T) {
		cfg := testCfg()
		cfg.Spread = 0
		cfg.Direction = Vec2{X: 1, Y: 0}
		cfg.Origin = Vec2{X: 4, Y: 10}
		cfg.Nozzle = 4
		cfg.Count = 80
		e := New(9, cfg)
		e.Update(0.01)
		var minY, maxY float64
		minY = 1e9
		for i, p := range e.Particles {
			if math.Abs(p.Vel.Y) > 1e-9 {
				t.Fatalf("nozzle is thickness, not spread: particle %d has VY=%v", i, p.Vel.Y)
			}
			if p.Pos.Y < minY {
				minY = p.Pos.Y
			}
			if p.Pos.Y > maxY {
				maxY = p.Pos.Y
			}
		}
		if maxY-minY < 2 {
			t.Fatalf("nozzle 4 should span Y, span=%.2f", maxY-minY)
		}
	})
	t.Run("unhappy: a negative nozzle is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.Nozzle = -1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrNozzle) {
			t.Fatalf("got %v, want ErrNozzle", err)
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

func TestSetConfig(t *testing.T) {
	t.Run("happy: get the current config, change count and life, set it back", func(t *testing.T) {
		e := New(1, testCfg())
		cfg := e.Config()
		if cfg.Count != 20 {
			t.Fatalf("starting count %d, want 20", cfg.Count)
		}
		cfg.Count = 3
		cfg.MinLife, cfg.MaxLife = 0.4, 0.4
		if err := e.SetConfig(cfg); err != nil {
			t.Fatalf("SetConfig: %v", err)
		}
		got := e.Config()
		if got.Count != 3 || got.MaxLife != 0.4 || got.MinLife != 0.4 {
			t.Fatalf("engine kept %+v, want count=3 life=0.4", got)
		}
		e.Update(0.01)
		if len(e.Particles) != 3 {
			t.Fatalf("next emit spawned %d, want the new count 3", len(e.Particles))
		}
		for i, p := range e.Particles {
			if math.Abs(p.Life-0.4) > 1e-9 {
				t.Fatalf("particle %d life %f, want the new max life 0.4", i, p.Life)
			}
		}
	})
	t.Run("happy: shrinking MaxDistance kills particles that have already flown past it", func(t *testing.T) {
		cfg := testCfg()
		cfg.Spread = 0
		cfg.MinSpeed, cfg.MaxSpeed = 10, 10
		cfg.MinLife, cfg.MaxLife = 10, 10
		cfg.Count = 4
		cfg.Period = 1
		e := New(2, cfg)
		e.Update(0.01)
		if len(e.Particles) != 4 {
			t.Fatalf("emit %d, want 4", len(e.Particles))
		}
		e.Update(0.5) // ~5 units from origin at speed 10
		live := e.Config()
		live.MaxDistance = 1
		live.Period = 0
		if err := e.SetConfig(live); err != nil {
			t.Fatalf("SetConfig: %v", err)
		}
		e.Update(0.01)
		if len(e.Particles) != 0 {
			t.Fatalf("particles past MaxDistance=1 must die, still %d", len(e.Particles))
		}
	})
	t.Run("unhappy: a bad config is rejected and the running engine is left alone", func(t *testing.T) {
		e := New(1, testCfg())
		e.Update(0.01)
		before := e.Config()
		n := len(e.Particles)
		bad := before
		bad.Count = -4
		if err := e.SetConfig(bad); !errors.Is(err, ErrCount) {
			t.Fatalf("got %v, want ErrCount", err)
		}
		if e.Config() != before {
			t.Fatalf("a rejected set must not clobber the running config: %+v", e.Config())
		}
		e.Update(0.1)
		if len(e.Particles) <= n {
			t.Fatal("the engine must keep emitting at the old count after a rejected set")
		}
	})
	t.Run("unhappy: SetConfig on a nil engine names itself and does not panic", func(t *testing.T) {
		var ghost *Engine
		if err := ghost.SetConfig(testCfg()); err == nil {
			t.Fatal("a nil engine must reject SetConfig")
		}
		if ghost.Config() != (Config{}) {
			t.Fatal("a nil engine has no config")
		}
	})
}

func TestBurst(t *testing.T) {
	t.Run("happy: Burst is the one-shot trigger — Period 0 never auto-emits, the squeeze does", func(t *testing.T) {
		cfg := testCfg()
		cfg.Period = 0
		e := New(1, cfg)
		e.Update(1)
		if len(e.Particles) != 0 {
			t.Fatalf("Period=0 must hold fire, got %d", len(e.Particles))
		}
		e.Burst()
		if len(e.Particles) != cfg.Count {
			t.Fatalf("one squeeze emits %d, want the count %d", len(e.Particles), cfg.Count)
		}
		for i, p := range e.Particles {
			if p.Pos != cfg.Origin {
				t.Fatalf("particle %d burst at %+v, want the origin %+v", i, p.Pos, cfg.Origin)
			}
			if p.Age != 0 {
				t.Fatalf("particle %d burst with age %f, want 0", i, p.Age)
			}
		}
		e.Burst()
		if len(e.Particles) != 2*cfg.Count {
			t.Fatalf("a second squeeze stacks to %d, want %d", len(e.Particles), 2*cfg.Count)
		}
	})
	t.Run("happy: burst particles fly, age, and die like any other", func(t *testing.T) {
		cfg := testCfg()
		cfg.Period = 0
		cfg.MinLife, cfg.MaxLife = 0.1, 0.2
		e := New(2, cfg)
		e.Burst()
		e.Update(0.05)
		if len(e.Particles) == 0 {
			t.Fatal("mid-life the batch must still fly")
		}
		if avgX(e.Particles) >= cfg.Origin.X {
			t.Fatalf("a leftward burst must drift left, avgX=%.2f origin=%.2f", avgX(e.Particles), cfg.Origin.X)
		}
		e.Update(2)
		if len(e.Particles) != 0 {
			t.Fatalf("after 2s the whole batch must be dead, still %d", len(e.Particles))
		}
	})
	t.Run("unhappy: a zero count bursts nothing", func(t *testing.T) {
		cfg := testCfg()
		cfg.Period = 0
		cfg.Count = 0
		e := New(3, cfg)
		e.Burst()
		if len(e.Particles) != 0 {
			t.Fatalf("Count=0 must burst 0, got %d", len(e.Particles))
		}
	})
	t.Run("unhappy: a nil engine skips the cue without a panic", func(t *testing.T) {
		var ghost *Engine
		ghost.Burst()
	})
}

func TestLiftDrag(t *testing.T) {
	t.Run("happy: lift bends a level flight upward like hot gas rising", func(t *testing.T) {
		cfg := testCfg()
		cfg.Period = 0
		cfg.Spread = 0
		cfg.Direction = Vec2{X: 1, Y: 0}
		cfg.Origin = Vec2{X: 5, Y: 15}
		cfg.MinSpeed, cfg.MaxSpeed = 10, 10
		cfg.MinLife, cfg.MaxLife = 10, 10
		cfg.Lift = 20
		e := New(1, cfg)
		e.Burst()
		e.Update(0.5)
		for i, p := range e.Particles {
			wantVY := -cfg.Lift * 0.5
			if math.Abs(p.Vel.Y-wantVY) > 1e-9 {
				t.Fatalf("particle %d climbs at %v after 0.5s of lift 20, want %v", i, p.Vel.Y, wantVY)
			}
			if p.Pos.Y >= cfg.Origin.Y {
				t.Fatalf("particle %d must have risen above the origin, y=%v", i, p.Pos.Y)
			}
			if p.Vel.X <= 0 {
				t.Fatalf("lift is vertical only — particle %d lost its forward speed %v", i, p.Vel.X)
			}
		}
	})
	t.Run("happy: drag decays speed exponentially, the same at any frame rate", func(t *testing.T) {
		cfg := testCfg()
		cfg.Period = 0
		cfg.Spread = 0
		cfg.MinSpeed, cfg.MaxSpeed = 20, 20
		cfg.MinLife, cfg.MaxLife = 10, 10
		cfg.Drag = 3
		coarse := New(5, cfg)
		fine := New(5, cfg)
		coarse.Burst()
		fine.Burst()
		coarse.Update(0.2)
		for i := 0; i < 20; i++ {
			fine.Update(0.01)
		}
		want := 20 * math.Exp(-3*0.2)
		if got := coarse.Particles[0].Vel.Len(); math.Abs(got-want) > 1e-9 {
			t.Fatalf("one 0.2s step decayed speed to %v, want %v", got, want)
		}
		if got := fine.Particles[0].Vel.Len(); math.Abs(got-want) > 1e-9 {
			t.Fatalf("twenty 0.01s steps decayed speed to %v, want the same %v", got, want)
		}
	})
	t.Run("happy: zero lift and drag leave the straight flight exactly as it was", func(t *testing.T) {
		plain := testCfg()
		plain.Period = 0
		zeroed := plain
		zeroed.Lift, zeroed.Drag = 0, 0
		a := New(9, plain)
		b := New(9, zeroed)
		a.Burst()
		b.Burst()
		a.Update(0.3)
		b.Update(0.3)
		if len(a.Particles) != len(b.Particles) {
			t.Fatalf("populations diverged %d vs %d", len(a.Particles), len(b.Particles))
		}
		for i := range a.Particles {
			if a.Particles[i] != b.Particles[i] {
				t.Fatalf("particle %d diverged: %+v vs %+v", i, a.Particles[i], b.Particles[i])
			}
		}
	})
	t.Run("happy: lift and drag ride along under the swirl too", func(t *testing.T) {
		cfg := swirlCfg()
		cfg.Period = 0
		cfg.Drag = 2
		e := New(3, cfg)
		e.Burst()
		v0 := e.Particles[0].Vel.Len()
		e.Update(0.25)
		want := v0 * math.Exp(-2*0.25)
		if got := e.Particles[0].Vel.Len(); math.Abs(got-want) > 1e-6 {
			t.Fatalf("swirling speck kept speed %v, want it dragged to %v", got, want)
		}
	})
	t.Run("unhappy: a negative lift is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.Lift = -1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrLift) {
			t.Fatalf("got %v, want ErrLift", err)
		}
	})
	t.Run("unhappy: a negative drag is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.Drag = -0.5
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrDrag) {
			t.Fatalf("got %v, want ErrDrag", err)
		}
	})
}

func TestMaxDistance(t *testing.T) {
	t.Run("happy: particles die when they travel farther than MaxDistance from the origin", func(t *testing.T) {
		cfg := testCfg()
		cfg.Spread = 0
		cfg.MinSpeed, cfg.MaxSpeed = 20, 20
		cfg.MinLife, cfg.MaxLife = 10, 10
		cfg.Count = 5
		cfg.Period = 1
		cfg.MaxDistance = 3
		e := New(7, cfg)
		e.Update(0.01)
		if len(e.Particles) != 5 {
			t.Fatalf("emit %d, want 5", len(e.Particles))
		}
		e.Update(0.5) // 10 units of travel, cap is 3
		for i, p := range e.Particles {
			d := math.Hypot(p.Pos.X-cfg.Origin.X, p.Pos.Y-cfg.Origin.Y)
			if d > cfg.MaxDistance+1e-6 {
				t.Fatalf("particle %d flew to distance %.2f, want <= 3", i, d)
			}
		}
		if len(e.Particles) != 0 {
			t.Fatalf("every particle must have hit MaxDistance, still %d", len(e.Particles))
		}
	})
	t.Run("unhappy: MaxDistance 0 is no extra cap, and a negative distance is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.MaxDistance = 0
		if err := New(1, cfg).Validate(); err != nil {
			t.Fatalf("MaxDistance 0 must mean unlimited, got %v", err)
		}
		cfg.MaxDistance = -1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrDistance) {
			t.Fatalf("got %v, want ErrDistance", err)
		}
	})
}
