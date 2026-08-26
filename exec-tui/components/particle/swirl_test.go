package particle

// Tests written FIRST. Swirl is the engine's second mode: the straight
// update flies particles along their velocity and nothing else; the
// swirl update curls them to the side like cartoon wind — every speck
// curves toward the loop side, and every second one sweeps one full
// loop before flying on. The mode is data on the Config, the straight
// zero value stays the default, and SideSwirl is the one switch that
// turns the swirl on, looping upward or downward.

import (
	"errors"
	"math"
	"testing"
)

func swirlCfg() Config {
	cfg := Config{
		Width:     200,
		Height:    100,
		Origin:    Vec2{X: 100, Y: 50},
		Direction: Vec2{X: 1, Y: 0},
		Count:     20,
		Period:    1,
		MinLife:   8,
		MaxLife:   8,
		MinSpeed:  2,
		MaxSpeed:  2,
		Spread:    0,
	}
	return cfg.SideSwirl(true)
}

// curlWindow is when the plan turns: [start, end) in particle age.
func curlWindow(c Curl) (start, end float64) {
	if c.Rate == 0 {
		return 0, 0
	}
	return c.Delay, c.Delay + c.Turn/math.Abs(c.Rate)
}

func TestSideSwirl(t *testing.T) {
	t.Run("happy: the zero mode is straight and SideSwirl flips a copy into swirl", func(t *testing.T) {
		cfg := testCfg()
		if cfg.Mode != ModeStraight {
			t.Fatalf("default mode %v, want ModeStraight", cfg.Mode)
		}
		up := cfg.SideSwirl(true)
		if up.Mode != ModeSwirl || !up.SwirlUp {
			t.Fatalf("SideSwirl(true) = mode %v up %v, want swirl looping up", up.Mode, up.SwirlUp)
		}
		down := cfg.SideSwirl(false)
		if down.Mode != ModeSwirl || down.SwirlUp {
			t.Fatalf("SideSwirl(false) = mode %v up %v, want swirl looping down", down.Mode, down.SwirlUp)
		}
		if cfg.Mode != ModeStraight {
			t.Fatal("SideSwirl must return a copy, not flip the receiver")
		}
		if err := New(1, up).Validate(); err != nil {
			t.Fatalf("a swirl config must validate: %v", err)
		}
	})
	t.Run("unhappy: an unknown mode is rejected", func(t *testing.T) {
		cfg := testCfg()
		cfg.Mode = Mode(99)
		if err := New(1, cfg).Validate(); !errors.Is(err, ErrMode) {
			t.Fatalf("got %v, want ErrMode", err)
		}
	})
}

func TestSwirlEmit(t *testing.T) {
	t.Run("happy: half the batch sweeps a full loop, half curves gently, all inside their life", func(t *testing.T) {
		e := New(3, swirlCfg())
		e.Update(0.01)
		if len(e.Particles) != 20 {
			t.Fatalf("emit %d, want 20", len(e.Particles))
		}
		loops, arcs := 0, 0
		for i, p := range e.Particles {
			if p.Age != 0 {
				t.Fatalf("particle %d spawned with age %f, want 0", i, p.Age)
			}
			c := p.Curl
			switch {
			case math.Abs(c.Turn-2*math.Pi) < 1e-9:
				loops++
			case c.Turn > 0 && c.Turn < math.Pi:
				arcs++
			default:
				t.Fatalf("particle %d curls %f radians, want one loop or a gentle arc", i, c.Turn)
			}
			if c.Rate >= 0 {
				t.Fatalf("particle %d spins %f; an up loop on a rightward flight must spin negative", i, c.Rate)
			}
			if c.Delay < 0 {
				t.Fatalf("particle %d has a negative curl delay %f", i, c.Delay)
			}
			if _, end := curlWindow(c); end > p.Life+1e-9 {
				t.Fatalf("particle %d curls until %f but dies at %f", i, end, p.Life)
			}
		}
		if loops != 10 || arcs != 10 {
			t.Fatalf("dealt %d loops and %d arcs, want half and half", loops, arcs)
		}
	})
	t.Run("unhappy: straight mode deals no curls", func(t *testing.T) {
		e := New(3, testCfg())
		e.Update(0.01)
		if len(e.Particles) == 0 {
			t.Fatal("need particles")
		}
		for i, p := range e.Particles {
			if p.Curl != (Curl{}) {
				t.Fatalf("straight particle %d carries a curl plan %+v", i, p.Curl)
			}
		}
	})
}

// run steps the engine n times by dt without re-emitting.
func run(e *Engine, n int, dt float64) {
	e.Cfg.Period = 0
	for i := 0; i < n; i++ {
		e.Update(dt)
	}
}

func TestSwirlUpdate(t *testing.T) {
	t.Run("happy: a looper flies straight, sweeps exactly one loop, and flies on", func(t *testing.T) {
		e := New(5, swirlCfg())
		e.Update(0.01)
		idx := -1
		for i, p := range e.Particles {
			if math.Abs(p.Curl.Turn-2*math.Pi) < 1e-9 {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Fatal("no looper in the batch")
		}
		v0 := e.Particles[idx].Vel
		start, end := curlWindow(e.Particles[idx].Curl)
		const dt = 0.005
		wentUp, wentBack := false, false
		e.Cfg.Period = 0
		for e.Particles[idx].Age < start-2*dt {
			e.Update(dt)
		}
		if v := e.Particles[idx].Vel; math.Abs(v.X-v0.X) > 1e-9 || math.Abs(v.Y-v0.Y) > 1e-9 {
			t.Fatalf("the heading must hold until the curl starts, vel %+v want %+v", v, v0)
		}
		for e.Particles[idx].Age < end+2*dt {
			e.Update(dt)
			if v := e.Particles[idx].Vel; v.Y < -1e-9 {
				wentUp = true
			}
			if v := e.Particles[idx].Vel; v.X < -1e-9 {
				wentBack = true
			}
		}
		if !wentUp {
			t.Fatal("an upward loop must point the flight up at some moment")
		}
		if !wentBack {
			t.Fatal("a full loop must point the flight backwards at some moment")
		}
		v := e.Particles[idx].Vel
		if math.Abs(v.X-v0.X) > 1e-6 || math.Abs(v.Y-v0.Y) > 1e-6 {
			t.Fatalf("after one full loop the heading must return, vel %+v want %+v", v, v0)
		}
		x := e.Particles[idx].Pos.X
		run(e, 40, dt)
		if e.Particles[idx].Pos.X <= x {
			t.Fatal("after the loop the speck must keep flying out")
		}
	})
	t.Run("happy: an arc curves up by exactly its dealt turn", func(t *testing.T) {
		e := New(5, swirlCfg())
		e.Update(0.01)
		idx := -1
		for i, p := range e.Particles {
			if p.Curl.Turn > 0 && p.Curl.Turn < math.Pi {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Fatal("no arc in the batch")
		}
		p0 := e.Particles[idx]
		_, end := curlWindow(p0.Curl)
		steps := int(end/0.005) + 40
		run(e, steps, 0.005)
		p := e.Particles[idx]
		if p.Pos.Y >= p0.Pos.Y {
			t.Fatalf("an up arc must rise, y %f from %f", p.Pos.Y, p0.Pos.Y)
		}
		got := math.Atan2(p.Vel.Y, p.Vel.X) - math.Atan2(p0.Vel.Y, p0.Vel.X)
		want := math.Copysign(p0.Curl.Turn, p0.Curl.Rate)
		if math.Abs(got-want) > 1e-6 {
			t.Fatalf("arc swept %f radians, want the dealt %f", got, want)
		}
	})
	t.Run("unhappy: dt <= 0 moves, ages, and turns nothing", func(t *testing.T) {
		e := New(5, swirlCfg())
		e.Update(0.01)
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

func TestSwirlMirroring(t *testing.T) {
	t.Run("happy: for an upward loop, rightward and leftward flights spin opposite ways and both rise", func(t *testing.T) {
		right := New(7, swirlCfg())
		leftCfg := swirlCfg()
		leftCfg.Direction = Vec2{X: -1, Y: 0}
		left := New(7, leftCfg)
		right.Update(0.01)
		left.Update(0.01)
		for i, p := range right.Particles {
			if p.Curl.Rate >= 0 {
				t.Fatalf("rightward particle %d spins %f, want negative for an up loop", i, p.Curl.Rate)
			}
		}
		for i, p := range left.Particles {
			if p.Curl.Rate <= 0 {
				t.Fatalf("leftward particle %d spins %f, want positive for an up loop", i, p.Curl.Rate)
			}
		}
		run(right, 400, 0.005)
		run(left, 400, 0.005)
		for _, e := range []*Engine{right, left} {
			rose := false
			for _, p := range e.Particles {
				if p.Curl.Turn < math.Pi && p.Pos.Y < e.Cfg.Origin.Y-0.05 {
					rose = true
				}
			}
			if !rose {
				t.Fatalf("an up swirl heading %+v must lift its arcs", e.Cfg.Direction)
			}
		}
	})
	t.Run("unhappy: a downward loop flips both spins, and a vertical flight picks one side deterministically", func(t *testing.T) {
		downCfg := swirlCfg()
		downCfg = downCfg.SideSwirl(false)
		down := New(7, downCfg)
		down.Update(0.01)
		for i, p := range down.Particles {
			if p.Curl.Rate <= 0 {
				t.Fatalf("rightward particle %d spins %f, want positive for a down loop", i, p.Curl.Rate)
			}
		}
		vertCfg := swirlCfg()
		vertCfg.Direction = Vec2{X: 0, Y: -1}
		vert := New(7, vertCfg)
		vert.Update(0.01)
		for i, p := range vert.Particles {
			if p.Curl.Rate <= 0 {
				t.Fatalf("vertical particle %d spins %f, want the deterministic positive side", i, p.Curl.Rate)
			}
		}
	})
}

func TestSwirlKills(t *testing.T) {
	t.Run("happy: swirling specks still die when life runs out", func(t *testing.T) {
		cfg := swirlCfg()
		cfg.MinLife, cfg.MaxLife = 0.2, 0.3
		e := New(9, cfg)
		e.Update(0.01)
		if len(e.Particles) == 0 {
			t.Fatal("need particles")
		}
		run(e, 1, 2)
		if len(e.Particles) != 0 {
			t.Fatalf("after 2s every swirling speck should be dead, still %d", len(e.Particles))
		}
	})
	t.Run("unhappy: the box edge and MaxDistance still kill mid-swirl", func(t *testing.T) {
		cfg := swirlCfg()
		cfg.Width, cfg.Height = 8, 8
		cfg.Origin = Vec2{X: 7, Y: 4}
		cfg.MinSpeed, cfg.MaxSpeed = 10, 10
		e := New(9, cfg)
		e.Update(0.01)
		run(e, 1, 0.5)
		if len(e.Particles) != 0 {
			t.Fatalf("x=12 is outside width 8, still %d live mid-swirl", len(e.Particles))
		}
		far := swirlCfg()
		far.MinSpeed, far.MaxSpeed = 20, 20
		far.MaxDistance = 3
		e = New(9, far)
		e.Update(0.01)
		run(e, 1, 0.5)
		if len(e.Particles) != 0 {
			t.Fatalf("10 units past a 3-unit leash, still %d live", len(e.Particles))
		}
	})
}
