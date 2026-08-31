package shootingstar

// Tests written FIRST: DiagonalCrossing is one meteor from the top
// left to the bottom right — the landing's shooting star. NewMeteor
// is that crossing as a performer, once: it does not loop, and after
// it leaves the stage it stays gone. The heading points down-right.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/bigstar"
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/components/startrail"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

var _ screenplay.Component = (*Flyer)(nil)

func TestDiagonalCrossing(t *testing.T) {
	t.Run("happy: the meteor starts high on the left and ends low on the right", func(t *testing.T) {
		c := DiagonalCrossing(boxW, boxH)
		if c.Start.X > boxW*0.12 {
			t.Fatalf("start X=%.2f is not near the left edge", c.Start.X)
		}
		if c.Start.Y > boxH*0.22 {
			t.Fatalf("start Y=%.2f is not near the top", c.Start.Y)
		}
		if c.End.X < boxW*0.88 {
			t.Fatalf("end X=%.2f is not near the right edge", c.End.X)
		}
		if c.End.Y < boxH*0.70 {
			t.Fatalf("end Y=%.2f is not near the bottom", c.End.Y)
		}
		if c.Start.X >= c.End.X {
			t.Fatalf("must run left-to-right, %+v → %+v", c.Start, c.End)
		}
		if c.Start.Y >= c.End.Y {
			t.Fatalf("must fall downward, %+v → %+v", c.Start, c.End)
		}
		p0, h0 := c.At(0)
		if mathAbs(p0.X-c.Start.X) > 1e-9 || mathAbs(p0.Y-c.Start.Y) > 1e-9 {
			t.Fatalf("At(0) %+v, want start %+v", p0, c.Start)
		}
		if h0.X <= 0 {
			t.Fatalf("heading at start %+v must point right", h0)
		}
		p1, _ := c.At(1)
		if mathAbs(p1.X-c.End.X) > 1e-9 || mathAbs(p1.Y-c.End.Y) > 1e-9 {
			t.Fatalf("At(1) %+v, want end %+v", p1, c.End)
		}
	})
	t.Run("unhappy: a zero box does not panic, and DiagonalCrossing is not the right-to-left fall", func(t *testing.T) {
		c := DiagonalCrossing(0, 0)
		p, h := c.At(0.5)
		_ = p
		if h == (particle.Vec2{}) {
			// a collapsed box may park; it must not panic
		}
		c.At(-1)
		c.At(2)
		diag := DiagonalCrossing(boxW, boxH)
		rtl := RandomCrossing(11, boxW, boxH)
		if diag.Start.X >= rtl.Start.X {
			t.Fatalf("diagonal start X=%.2f must be on the left, RTL starts at %.2f", diag.Start.X, rtl.Start.X)
		}
		if diag.End.X <= rtl.End.X {
			t.Fatalf("diagonal end X=%.2f must be on the right, RTL ends at %.2f", diag.End.X, rtl.End.X)
		}
	})
}

func mathAbs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func coreOn(sp sprite.Sprite) (x, y int, ok bool) {
	for y = 0; y < sp.Height; y++ {
		for x = 0; x < sp.Width; x++ {
			if sp.At(y, x).Ch == bigstar.CoreGlyph {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func TestNewMeteor(t *testing.T) {
	t.Cleanup(Reset)
	t.Cleanup(startrail.Reset)
	t.Run("happy: the meteor enters top-left, travels down-right, and then stays gone", func(t *testing.T) {
		m := NewMeteor()
		m.Start(stageW, stageH)
		defer m.Stop()
		m.Update(0.05)
		sp := m.Render()
		x0, y0, ok := coreOn(sp)
		if !ok {
			t.Fatal("the meteor must open on (or entering) stage")
		}
		if x0 > stageW/3 {
			t.Fatalf("a top-left meteor opened at col %d, want the left third", x0)
		}
		if y0 > stageH/3 {
			t.Fatalf("a top-left meteor opened at row %d, want the top third", y0)
		}
		m.Update(0.6)
		x1, y1, ok := coreOn(m.Render())
		if !ok {
			t.Fatal("the meteor must still be on stage a beat later")
		}
		if x1 <= x0 || y1 <= y0 {
			t.Fatalf("must travel down-right, (%d,%d) → (%d,%d)", x0, y0, x1, y1)
		}
		// Burn past one crossing. A looping star would reappear on the left.
		for i := 0; i < 400; i++ {
			m.Update(1.0 / 30)
		}
		if _, _, ok := coreOn(m.Render()); ok {
			t.Fatal("after the crossing the meteor must stay gone — one shooting star, not a loop")
		}
	})
	t.Run("unhappy: NewMeteor is not the looping right-to-left scene, and Stop drops it", func(t *testing.T) {
		sc := New(nil)
		sc.Seed = 11
		sc.Start()
		defer sc.Stop()
		_ = paint(sc)
		if sc.cross.Start.X <= sc.cross.End.X {
			t.Fatal("test premise: the scene still falls right-to-left")
		}
		m := NewMeteor()
		m.Start(stageW, stageH)
		if _, _, ok := coreOn(m.Render()); !ok {
			t.Fatal("test premise: a started meteor paints")
		}
		m.Stop()
		if _, _, ok := coreOn(m.Render()); ok {
			t.Fatal("a stopped meteor must not keep its core on stage")
		}
		var ghost *Flyer
		ghost.Start(4, 2)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}

func TestNewMeteorWith(t *testing.T) {
	t.Cleanup(Reset)
	t.Cleanup(startrail.Reset)
	t.Run("happy: MeteorConfig is the stock small star and NewMeteor flies it", func(t *testing.T) {
		want := DefaultConfig()
		want.Size = 1
		if MeteorConfig() != want {
			t.Fatalf("MeteorConfig is %+v, want the stock knobs one cell small %+v", MeteorConfig(), want)
		}
		m := NewMeteor()
		if m.show == nil || m.show.Cfg != MeteorConfig() {
			t.Fatalf("NewMeteor carries %+v, want MeteorConfig %+v", m.show.Cfg, MeteorConfig())
		}
	})
	t.Run("happy: NewMeteorWith flies the given knobs — a scene tunes its own meteor", func(t *testing.T) {
		fast := MeteorConfig()
		fast.Speed = 400
		fast.Size = 3
		m := NewMeteorWith(fast)
		if m == nil {
			t.Fatal("NewMeteorWith must hand back a flyer")
		}
		m.Start(stageW, stageH)
		defer m.Stop()
		if m.show == nil || m.show.Cfg != fast {
			t.Fatalf("the meteor carries %+v, want the given knobs %+v", m.show.Cfg, fast)
		}
		if m.star == nil || m.star.Size != 3 {
			t.Fatal("the given size must reach the star")
		}
		const dt = 1.0 / 30
		for i := 0; i < 30; i++ {
			m.Update(dt)
		}
		if _, _, ok := coreOn(m.Render()); ok {
			t.Fatal("at speed 400 the one crossing must already be over")
		}
		slow := MeteorConfig()
		slow.Speed = 4
		s := NewMeteorWith(slow)
		s.Start(stageW, stageH)
		defer s.Stop()
		for i := 0; i < 60; i++ {
			s.Update(dt)
		}
		if _, _, ok := coreOn(s.Render()); !ok {
			t.Fatal("at speed 4 the meteor must still be crossing")
		}
	})
	t.Run("happy: NewMeteor leaves a persist trail — the star knobs are the dust", func(t *testing.T) {
		cfg := MeteorConfig()
		cfg.Dust = true
		cfg.Count = 8
		cfg.Period = 0.01
		cfg.MinLife = 0.8
		cfg.MaxLife = 1.2
		cfg.Speed = 8
		m := NewMeteorWith(cfg)
		m.Start(stageW, stageH)
		defer m.Stop()
		if m.trail == nil {
			t.Fatal("the landing meteor must arm a persist trail so count/period/life/nozzle/peak/taper can dust")
		}
		const dt = 1.0 / 30
		for i := 0; i < 20; i++ {
			m.Update(dt)
		}
		sp := m.Render()
		specks := 0
		for y := 0; y < sp.Height; y++ {
			for x := 0; x < sp.Width; x++ {
				ch := sp.At(y, x).Ch
				if ch != 0 && ch != ' ' && ch != bigstar.CoreGlyph {
					specks++
				}
			}
		}
		if specks < 3 {
			t.Fatalf("the meteor must dust a wake, got %d trail cells", specks)
		}
	})
	t.Run("unhappy: a zero-count meteor still flies the core and never panics", func(t *testing.T) {
		cfg := MeteorConfig()
		cfg.Count = 0
		m := NewMeteorWith(cfg)
		m.Start(stageW, stageH)
		defer m.Stop()
		m.Update(0.2)
		if _, _, ok := coreOn(m.Render()); !ok {
			t.Fatal("a zero-count meteor must still show the star")
		}
		specks := 0
		sp := m.Render()
		for y := 0; y < sp.Height; y++ {
			for x := 0; x < sp.Width; x++ {
				ch := sp.At(y, x).Ch
				if ch != 0 && ch != ' ' && ch != bigstar.CoreGlyph {
					specks++
				}
			}
		}
		if specks != 0 {
			t.Fatalf("count 0 must leave no dust, got %d cells", specks)
		}
	})
	t.Run("unhappy: a zero-value config parks the meteor without a panic, and a tuned active never leaks into NewMeteor", func(t *testing.T) {
		z := NewMeteorWith(Config{})
		z.Start(stageW, stageH)
		z.Update(0.5)
		_ = z.Render()
		z.Stop()
		tuned := DefaultConfig()
		tuned.Size = 3
		if err := Use(tuned); err != nil {
			t.Fatal(err)
		}
		m := NewMeteor()
		if m.show.Cfg != MeteorConfig() {
			t.Fatalf("NewMeteor flies %+v, want the stock small meteor no matter what the tuner holds", m.show.Cfg)
		}
	})
}

func TestMeteorDelay(t *testing.T) {
	t.Cleanup(Reset)
	t.Cleanup(startrail.Reset)
	t.Run("happy: a delay keeps the meteor off stage until that time, then it flies", func(t *testing.T) {
		cfg := MeteorConfig()
		cfg.Delay = 0.4
		m := NewMeteorWith(cfg)
		m.Start(stageW, stageH)
		defer m.Stop()
		if _, _, ok := coreOn(m.Render()); ok {
			t.Fatal("the meteor must stay absent before the delay")
		}
		m.Update(0.2)
		if _, _, ok := coreOn(m.Render()); ok {
			t.Fatal("at 0.2s of a 0.4s delay the meteor must still be waiting")
		}
		m.Update(0.25)
		x0, y0, ok := coreOn(m.Render())
		if !ok {
			t.Fatal("past the delay the meteor must fly")
		}
		m.Update(0.3)
		x1, y1, ok := coreOn(m.Render())
		if !ok {
			t.Fatal("once it flies the meteor must stay on the crossing")
		}
		if x1 == x0 && y1 == y0 {
			t.Fatal("once it flies the meteor must move")
		}
	})
	t.Run("unhappy: delay 0 is the current immediate start, and a negative delay does not panic", func(t *testing.T) {
		m := NewMeteor()
		if m.show.Cfg.Delay != 0 {
			t.Fatalf("stock delay %v, want 0 — immediate start", m.show.Cfg.Delay)
		}
		m.Start(stageW, stageH)
		defer m.Stop()
		if _, _, ok := coreOn(m.Render()); !ok {
			t.Fatal("delay 0 must paint at once — the current immediate start")
		}
		neg := MeteorConfig()
		neg.Delay = -2
		z := NewMeteorWith(neg)
		z.Start(stageW, stageH)
		z.Update(0.1)
		if _, _, ok := coreOn(z.Render()); !ok {
			t.Fatal("a negative delay is the operator's number — the meteor must still fly")
		}
		z.Stop()
	})
}

func TestMeteorStartY(t *testing.T) {
	t.Cleanup(Reset)
	t.Cleanup(startrail.Reset)
	t.Run("happy: a higher top-right start paints the meteor higher on the right just after spawn than stock", func(t *testing.T) {
		stock := NewOnceWith(DefaultConfig())
		stock.Start(stageW, stageH)
		defer stock.Stop()
		stock.Update(0.02)
		sx, sy, ok := coreOn(stock.Render())
		if !ok {
			t.Fatal("the stock once-meteor must paint just after spawn")
		}
		if sx < stageW/2 {
			t.Fatalf("stock must open on the right, col %d", sx)
		}
		high := DefaultConfig()
		high.StartY = 0.02
		raised := NewOnceWith(high)
		raised.Start(stageW, stageH)
		defer raised.Stop()
		raised.Update(0.02)
		hx, hy, ok := coreOn(raised.Render())
		if !ok {
			t.Fatal("a raised start must still paint the meteor")
		}
		if hx < stageW/2 {
			t.Fatalf("a raised start must stay on the right, col %d", hx)
		}
		if hy >= sy {
			t.Fatalf("startY 0.02 painted at row %d, stock at %d — want higher on the right (smaller row)", hy, sy)
		}
	})
	t.Run("unhappy: stock origin matches the previous path, and a weird or zero origin does not panic", func(t *testing.T) {
		m := NewMeteor()
		m.Start(stageW, stageH)
		defer m.Stop()
		want := DiagonalCrossing(m.uw, m.uh)
		if m.show.cross.Start != want.Start || m.show.cross.End != want.End {
			t.Fatalf("stock meteor path %+v, want the previous diagonal %+v", m.show.cross, want)
		}
		once := NewOnceWith(DefaultConfig())
		once.Start(stageW, stageH)
		defer once.Stop()
		wantOnce := OnceCrossing(once.uw, once.uh)
		if once.show.cross.Start != wantOnce.Start || once.show.cross.End != wantOnce.End {
			t.Fatalf("stock once path %+v, want the previous once-crossing %+v", once.show.cross, wantOnce)
		}
		zero := DefaultConfig()
		zero.StartY = 0
		z := NewOnceWith(zero)
		z.Start(0, 0)
		z.Update(1)
		_ = z.Render()
		z.Stop()
		weird := DefaultConfig()
		weird.StartY = -8
		w := NewOnceWith(weird)
		w.Start(stageW, stageH)
		w.Update(0.05)
		_ = w.Render()
		w.Stop()
		weird.StartY = 40
		far := NewMeteorWith(weird)
		far.Start(stageW, stageH)
		far.Update(0.05)
		_ = far.Render()
		far.Stop()
	})
}
