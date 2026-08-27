package particle

// Tests written FIRST. Pool is the engine's third mode: the straight
// update flies particles along their velocity, the swirl update curls
// them, and the pool update parks them — a Burst (or a period emit)
// scatters Count specks around the origin inside PoolRadius and they
// stay put. Clouds are unique because the scatter is seeded; the same
// seed always draws the same puff, a different seed draws another.
// Pooled is the one switch that turns the pool on. Stationary specks
// still age and die, and the box edge still kills anyone who spawned
// past it.

import (
	"errors"
	"math"
	"testing"
)

func poolCfg() Config {
	return Config{
		Width:      80,
		Height:     40,
		Origin:     Vec2{X: 40, Y: 20},
		Direction:  Vec2{X: 0, Y: -1},
		Count:      40,
		Period:     0,
		MinLife:    10,
		MaxLife:    10,
		MinSpeed:   0,
		MaxSpeed:   0,
		PoolRadius: 8,
	}.Pooled()
}

func TestPooled(t *testing.T) {
	t.Run("happy: the zero mode is straight and Pooled flips a copy into the pool", func(t *testing.T) {
		cfg := testCfg()
		if cfg.Mode != ModeStraight {
			t.Fatalf("default mode %v, want ModeStraight", cfg.Mode)
		}
		pool := cfg.Pooled()
		if pool.Mode != ModePool {
			t.Fatalf("Pooled() = mode %v, want ModePool", pool.Mode)
		}
		if cfg.Mode != ModeStraight {
			t.Fatal("Pooled must return a copy, not flip the receiver")
		}
		pool.PoolRadius = 4
		pool.MinSpeed, pool.MaxSpeed = 0, 0
		if err := New(1, pool).Validate(); err != nil {
			t.Fatalf("a pool config must validate: %v", err)
		}
	})
	t.Run("unhappy: a negative pool radius is rejected", func(t *testing.T) {
		cfg := poolCfg()
		cfg.PoolRadius = -1
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrPoolRadius) {
			t.Fatalf("got %v, want ErrPoolRadius", err)
		}
	})
	t.Run("unhappy: an unknown mode is still rejected once the pool exists", func(t *testing.T) {
		cfg := poolCfg()
		cfg.Mode = Mode(99)
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrMode) {
			t.Fatalf("got %v, want ErrMode", err)
		}
	})
}

func TestPoolBurst(t *testing.T) {
	t.Run("happy: a burst scatters Count specks around the origin, none flying", func(t *testing.T) {
		e := New(3, poolCfg())
		e.Burst()
		if len(e.Particles) != 40 {
			t.Fatalf("burst %d, want 40", len(e.Particles))
		}
		moved := 0
		var sumR float64
		for i, p := range e.Particles {
			if p.Vel != (Vec2{}) {
				t.Fatalf("pool particle %d flies %+v — pool specks are parked", i, p.Vel)
			}
			d := math.Hypot(p.Pos.X-e.Cfg.Origin.X, p.Pos.Y-e.Cfg.Origin.Y)
			sumR += d
			if d > 1e-9 {
				moved++
			}
			if p.Age != 0 {
				t.Fatalf("particle %d burst with age %f, want 0", i, p.Age)
			}
			if p.Curl != (Curl{}) {
				t.Fatalf("pool particle %d carries a curl plan %+v", i, p.Curl)
			}
		}
		if moved < 30 {
			t.Fatalf("only %d specks left the origin — a pool must spread out", moved)
		}
		if avg := sumR / float64(len(e.Particles)); avg < 1 {
			t.Fatalf("mean radius %.2f — the puff must fill its pool, not sit on the nozzle", avg)
		}
	})
	t.Run("happy: the same seed draws the same puff, a different seed draws another", func(t *testing.T) {
		a := New(11, poolCfg())
		b := New(11, poolCfg())
		c := New(12, poolCfg())
		a.Burst()
		b.Burst()
		c.Burst()
		if len(a.Particles) == 0 || len(c.Particles) == 0 {
			t.Fatal("need particles")
		}
		same := true
		for i := range a.Particles {
			if a.Particles[i].Pos != b.Particles[i].Pos {
				same = false
				break
			}
		}
		if !same {
			t.Fatal("the same seed must scatter the same puff")
		}
		diff := 0
		n := len(a.Particles)
		if len(c.Particles) < n {
			n = len(c.Particles)
		}
		for i := 0; i < n; i++ {
			if a.Particles[i].Pos != c.Particles[i].Pos {
				diff++
			}
		}
		if diff < 10 {
			t.Fatalf("a different seed only moved %d specks — unique clouds need a unique scatter", diff)
		}
	})
	t.Run("unhappy: a zero pool radius parks every speck on the origin", func(t *testing.T) {
		cfg := poolCfg()
		cfg.PoolRadius = 0
		e := New(3, cfg)
		e.Burst()
		if len(e.Particles) != cfg.Count {
			t.Fatalf("burst %d, want %d", len(e.Particles), cfg.Count)
		}
		for i, p := range e.Particles {
			if p.Pos != cfg.Origin {
				t.Fatalf("particle %d at %+v, want the origin — a zero pool has no spread", i, p.Pos)
			}
		}
	})
}

func TestPoolUpdate(t *testing.T) {
	t.Run("happy: parked specks stay put while they age", func(t *testing.T) {
		e := New(5, poolCfg())
		e.Burst()
		before := append([]Particle(nil), e.Particles...)
		e.Update(0.5)
		if len(e.Particles) != len(before) {
			t.Fatalf("mid-life the pool lost specks %d -> %d", len(before), len(e.Particles))
		}
		for i, p := range e.Particles {
			if p.Pos != before[i].Pos {
				t.Fatalf("particle %d drifted %+v -> %+v — pool specks are stationary", i, before[i].Pos, p.Pos)
			}
			if p.Vel != (Vec2{}) {
				t.Fatalf("particle %d picked up velocity %+v", i, p.Vel)
			}
			if math.Abs(p.Age-0.5) > 1e-9 {
				t.Fatalf("particle %d age %f, want 0.5", i, p.Age)
			}
			if math.Abs(p.Life-(before[i].Life-0.5)) > 1e-9 {
				t.Fatalf("particle %d life %f, want %f", i, p.Life, before[i].Life-0.5)
			}
		}
	})
	t.Run("happy: a later period emit parks another unique scatter without moving the first", func(t *testing.T) {
		cfg := poolCfg()
		cfg.Period = 0.2
		e := New(7, cfg)
		e.Update(0.01)
		if len(e.Particles) != cfg.Count {
			t.Fatalf("first emit %d, want %d", len(e.Particles), cfg.Count)
		}
		first := append([]Particle(nil), e.Particles...)
		e.Update(0.2)
		if len(e.Particles) != 2*cfg.Count {
			t.Fatalf("after two periods %d, want %d", len(e.Particles), 2*cfg.Count)
		}
		for i, p := range first {
			if e.Particles[i].Pos != p.Pos {
				t.Fatalf("the first scatter drifted when the second parked")
			}
		}
	})
	t.Run("unhappy: dt <= 0 moves, ages, and emits nothing", func(t *testing.T) {
		e := New(5, poolCfg())
		e.Burst()
		before := append([]Particle(nil), e.Particles...)
		e.Update(0)
		e.Update(-2)
		if len(e.Particles) != len(before) {
			t.Fatalf("dt<=0 changed the population %d -> %d", len(before), len(e.Particles))
		}
		for i, p := range e.Particles {
			if p != before[i] {
				t.Fatalf("dt<=0 changed particle %d: %+v -> %+v", i, before[i], p)
			}
		}
	})
}

func TestPoolKills(t *testing.T) {
	t.Run("happy: parked specks still die when life runs out", func(t *testing.T) {
		cfg := poolCfg()
		cfg.MinLife, cfg.MaxLife = 0.2, 0.3
		e := New(9, cfg)
		e.Burst()
		if len(e.Particles) == 0 {
			t.Fatal("need particles")
		}
		e.Update(2)
		if len(e.Particles) != 0 {
			t.Fatalf("after 2s every parked speck should be dead, still %d", len(e.Particles))
		}
	})
	t.Run("unhappy: a speck that scatters outside the box is never born", func(t *testing.T) {
		cfg := poolCfg()
		cfg.Width, cfg.Height = 10, 10
		cfg.Origin = Vec2{X: 9, Y: 5}
		cfg.PoolRadius = 40
		cfg.Count = 80
		e := New(9, cfg)
		e.Burst()
		for i, p := range e.Particles {
			if p.Pos.X < 0 || p.Pos.X > cfg.Width || p.Pos.Y < 0 || p.Pos.Y > cfg.Height {
				t.Fatalf("particle %d spawned outside the box at %+v", i, p.Pos)
			}
		}
		if len(e.Particles) >= cfg.Count {
			t.Fatalf("a pool hanging off the edge kept every speck (%d) — the box must refuse the ones that land outside", len(e.Particles))
		}
	})
}
