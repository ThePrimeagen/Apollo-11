package particle

// Tests written FIRST. Persist is the engine's fourth mode: the straight
// update flies particles along their velocity, the swirl update curls
// them, the pool update scatters a stationary puff, and the persist
// update parks each speck exactly where it was born. A Burst (or a
// period emit) drops Count specks at the origin — nozzle-thick, not a
// disk — and they stay put, age, and die. Persist is the comet trail:
// move the origin and the old specks keep their cells, so a moving
// star leaves a fading wake. Persist is the one switch that turns it
// on. MaxDistance does not apply — trail length is life, not a radius
// around a moving nozzle.

import (
	"errors"
	"math"
	"testing"
)

func persistCfg() Config {
	return Config{
		Width:     80,
		Height:    40,
		Origin:    Vec2{X: 40, Y: 20},
		Direction: Vec2{X: 1, Y: 0},
		Count:     8,
		Period:    0,
		MinLife:   10,
		MaxLife:   10,
		MinSpeed:  0,
		MaxSpeed:  0,
		Spread:    0,
	}.Persist()
}

func TestPersist(t *testing.T) {
	t.Run("happy: the zero mode is straight and Persist flips a copy into persist", func(t *testing.T) {
		cfg := testCfg()
		if cfg.Mode != ModeStraight {
			t.Fatalf("default mode %v, want ModeStraight", cfg.Mode)
		}
		held := cfg.Persist()
		if held.Mode != ModePersist {
			t.Fatalf("Persist() = mode %v, want ModePersist", held.Mode)
		}
		if cfg.Mode != ModeStraight {
			t.Fatal("Persist must return a copy, not flip the receiver")
		}
		if err := New(1, held).Validate(); err != nil {
			t.Fatalf("a persist config must validate: %v", err)
		}
	})
	t.Run("unhappy: an unknown mode is still rejected once persist exists", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Mode = Mode(99)
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrMode) {
			t.Fatalf("got %v, want ErrMode", err)
		}
	})
}

func TestPersistBurst(t *testing.T) {
	t.Run("happy: a burst parks Count specks on the origin, none flying", func(t *testing.T) {
		e := New(3, persistCfg())
		e.Burst()
		if len(e.Particles) != 8 {
			t.Fatalf("burst %d, want 8", len(e.Particles))
		}
		for i, p := range e.Particles {
			if p.Vel != (Vec2{}) {
				t.Fatalf("persist particle %d flies %+v — persist specks are parked", i, p.Vel)
			}
			if p.Pos != e.Cfg.Origin {
				t.Fatalf("particle %d at %+v, want the origin — persist has no pool scatter", i, p.Pos)
			}
			if p.Age != 0 {
				t.Fatalf("particle %d burst with age %f, want 0", i, p.Age)
			}
			if p.Curl != (Curl{}) {
				t.Fatalf("persist particle %d carries a curl plan %+v", i, p.Curl)
			}
		}
	})
	t.Run("happy: a thick nozzle still spreads persist specks across the slit", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Nozzle = 4
		cfg.Count = 40
		e := New(9, cfg)
		e.Burst()
		var minY, maxY float64
		minY = 1e9
		for i, p := range e.Particles {
			if p.Vel != (Vec2{}) {
				t.Fatalf("nozzle is thickness, not flight: particle %d has vel %+v", i, p.Vel)
			}
			if math.Abs(p.Pos.X-cfg.Origin.X) > 1e-9 {
				t.Fatalf("a rightward nozzle is vertical: particle %d left X=%v", i, p.Pos.X)
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
	t.Run("unhappy: a zero count bursts nothing", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Count = 0
		e := New(3, cfg)
		e.Burst()
		if len(e.Particles) != 0 {
			t.Fatalf("Count=0 must burst 0, got %d", len(e.Particles))
		}
	})
}

func TestPersistUpdate(t *testing.T) {
	t.Run("happy: parked specks stay put while they age", func(t *testing.T) {
		e := New(5, persistCfg())
		e.Burst()
		before := append([]Particle(nil), e.Particles...)
		e.Update(0.5)
		if len(e.Particles) != len(before) {
			t.Fatalf("mid-life the trail lost specks %d -> %d", len(before), len(e.Particles))
		}
		for i, p := range e.Particles {
			if p.Pos != before[i].Pos {
				t.Fatalf("particle %d drifted %+v -> %+v — persist specks are stationary", i, before[i].Pos, p.Pos)
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
	t.Run("happy: moving the origin leaves the old specks where they were born", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Period = 0.2
		e := New(7, cfg)
		e.Update(0.01)
		if len(e.Particles) != cfg.Count {
			t.Fatalf("first emit %d, want %d", len(e.Particles), cfg.Count)
		}
		first := append([]Particle(nil), e.Particles...)
		e.Cfg.Origin = Vec2{X: 10, Y: 8}
		e.Update(0.2)
		if len(e.Particles) != 2*cfg.Count {
			t.Fatalf("after two periods %d, want %d", len(e.Particles), 2*cfg.Count)
		}
		for i, p := range first {
			if e.Particles[i].Pos != p.Pos {
				t.Fatalf("the first drop drifted when the origin moved — persist is a trail, not a puff")
			}
		}
		moved := 0
		for _, p := range e.Particles[cfg.Count:] {
			if p.Pos == first[0].Pos {
				continue
			}
			moved++
			if p.Pos != e.Cfg.Origin {
				t.Fatalf("second drop at %+v, want the new origin %+v", p.Pos, e.Cfg.Origin)
			}
		}
		if moved == 0 {
			t.Fatal("the second drop must land on the new origin")
		}
	})
	t.Run("happy: MaxDistance around a moving origin does not kill the wake", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Period = 0.2
		cfg.MaxDistance = 2
		e := New(4, cfg)
		e.Update(0.01)
		e.Cfg.Origin = Vec2{X: 70, Y: 30}
		e.Update(0.2)
		if len(e.Particles) != 2*cfg.Count {
			t.Fatalf("a persist trail must keep the first drop after the origin walks away, still %d want %d", len(e.Particles), 2*cfg.Count)
		}
		for i, p := range e.Particles[:cfg.Count] {
			d := math.Hypot(p.Pos.X-e.Cfg.Origin.X, p.Pos.Y-e.Cfg.Origin.Y)
			if d <= cfg.MaxDistance {
				t.Fatalf("test premise: first drop %d is still inside MaxDistance", i)
			}
		}
	})
	t.Run("unhappy: dt <= 0 moves, ages, and emits nothing", func(t *testing.T) {
		e := New(5, persistCfg())
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

func TestPersistKills(t *testing.T) {
	t.Run("happy: parked specks still die when life runs out", func(t *testing.T) {
		cfg := persistCfg()
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
	t.Run("unhappy: a speck that parks outside the box is never born", func(t *testing.T) {
		cfg := persistCfg()
		cfg.Origin = Vec2{X: 90, Y: 10}
		e := New(2, cfg)
		e.Burst()
		if len(e.Particles) != 0 {
			t.Fatalf("an origin past width 80 must refuse the drop, still %d", len(e.Particles))
		}
	})
}
