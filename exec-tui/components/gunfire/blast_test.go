package gunfire

// Tests written FIRST. Blast is the one-shot muzzle flame on an
// eight-point compass: one shared white-hot core and one flame engine
// per direction, each burning its own tune. Start builds all nine
// quiet; Fire is the trigger — the core and every heading's flame
// burst at the muzzle now, the whole rose at once, and Doom's second
// flash frame follows every heading on a short fuse. Update flies
// every direction's flame and re-reads the active blast so the tuner
// retunes it live; Done reports a blast with nothing burning anywhere.
// No period clock: no trigger, no fire.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// settle runs the blast n frames of dt without any new trigger.
func settle(b *Blast, n int, dt float64) {
	for i := 0; i < n; i++ {
		b.Update(dt)
	}
}

func live(b *Blast) int {
	n := len(b.Core.Particles)
	for _, e := range b.Flames {
		n += len(e.Particles)
	}
	return n
}

func TestHoldFire(t *testing.T) {
	t.Run("happy: Start builds the core and all eight flames, every one holding fire", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		if b.Core == nil {
			t.Fatal("Start must build the core")
		}
		for i, h := range sprite.Headings {
			if b.Flames[i] == nil || b.FlameAt(h) == nil {
				t.Fatalf("Start must build the %s flame", h)
			}
		}
		settle(b, 60, 1.0/30)
		if n := live(b); n != 0 {
			t.Fatalf("two untriggered seconds spawned %d particles — a one-shot must hold fire", n)
		}
	})
	t.Run("unhappy: the trigger needs a stage — Fire before Start is a refused no-op", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		if b.Fire() {
			t.Fatal("Fire before Start must report the refused trigger")
		}
		b.Start(80, 24)
		settle(b, 5, 1.0/30)
		if n := live(b); n != 0 {
			t.Fatalf("a pre-Start trigger must not fire later, found %d particles", n)
		}
	})
}

func TestFire(t *testing.T) {
	t.Run("happy: the trigger bursts the core and every compass heading's flame", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		c := ActiveBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		if !b.Fire() {
			t.Fatal("a started blast must take the trigger")
		}
		if got := len(b.Core.Particles); got != c.Core.Count {
			t.Fatalf("core burst %d, want %d", got, c.Core.Count)
		}
		for _, h := range sprite.Headings {
			if got := len(b.FlameAt(h).Particles); got != c.ShotAt(h).Count {
				t.Fatalf("the %s flame burst %d, want %d — the whole rose fires at once", h, got, c.ShotAt(h).Count)
			}
		}
	})
	t.Run("happy: each direction burns its own shot in the same squeeze", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.PulseFrac = 0 // one frame per squeeze keeps the counting plain
		east := c.ShotAt(sprite.E)
		east.Count = 33
		c.SetShot(sprite.E, east)
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b := NewBlast(7)
		b.Start(80, 24)
		b.Fire()
		if got := len(b.FlameAt(sprite.E).Particles); got != 33 {
			t.Fatalf("the E flame burst %d, want its own tuned 33", got)
		}
		if got := len(b.FlameAt(sprite.N).Particles); got != DefaultBlast().ShotAt(sprite.N).Count {
			t.Fatalf("the N flame must keep its own stock count, got %d", got)
		}
	})
	t.Run("happy: Doom's second frame re-pulses every heading", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.PulseDelay = 0.1
		c.PulseFrac = 0.5
		core := c.Core
		core.MaxDistance = 0
		core.MinLife, core.MaxLife = 5, 5
		c.Core = core
		for _, h := range sprite.Headings {
			shot := c.ShotAt(h)
			shot.Count = 40
			shot.MinLife, shot.MaxLife = 5, 5
			shot.MaxDistance = 0
			c.SetShot(h, shot)
		}
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b := NewBlast(7)
		b.Start(80, 24)
		b.Fire()
		b.Update(0.05)
		if got := len(b.FlameAt(sprite.N).Particles); got != 40 {
			t.Fatalf("the fuse is only half burnt and the N flame moved to %d", got)
		}
		b.Update(0.06)
		for _, h := range sprite.Headings {
			if got := len(b.FlameAt(h).Particles); got != 60 {
				t.Fatalf("the re-pulse must add half the %s flame again: %d, want 60", h, got)
			}
		}
		after := live(b)
		b.Update(0.2)
		if live(b) > after {
			t.Fatalf("the fuse burns once — the flame grew %d -> %d", after, live(b))
		}
	})
	t.Run("happy: a zero delay or a zero fraction means a single pulse", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		for name, mod := range map[string]func(*BlastConfig){
			"zero delay": func(c *BlastConfig) { c.PulseDelay = 0 },
			"zero frac":  func(c *BlastConfig) { c.PulseFrac = 0 },
		} {
			c := DefaultBlast()
			core := c.Core
			core.MinLife, core.MaxLife = 5, 5 // nothing dies during the check
			core.MaxDistance = 0
			c.Core = core
			for _, h := range sprite.Headings {
				shot := c.ShotAt(h)
				shot.MinLife, shot.MaxLife = 5, 5
				shot.MaxDistance = 0
				c.SetShot(h, shot)
			}
			mod(&c)
			if err := UseBlast(c); err != nil {
				t.Fatalf("%s UseBlast: %v", name, err)
			}
			b := NewBlast(7)
			b.Start(80, 24)
			b.Fire()
			first := live(b)
			settle(b, 12, 1.0/30)
			if live(b) != first {
				t.Fatalf("%s must mean one pulse only, %d -> %d", name, first, live(b))
			}
		}
	})
	t.Run("unhappy: dt <= 0 burns no fuse and moves nothing", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.PulseDelay = 0.05
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b := NewBlast(7)
		b.Start(80, 24)
		b.Fire()
		n := live(b)
		b.Update(0)
		b.Update(-1)
		if live(b) != n {
			t.Fatalf("dt<=0 changed the population %d -> %d", n, live(b))
		}
	})
	t.Run("unhappy: a nil blast skips every cue and FlameAt hands back nothing", func(t *testing.T) {
		var ghost *Blast
		ghost.Start(10, 10)
		if ghost.Fire() {
			t.Fatal("a nil blast must refuse the trigger")
		}
		ghost.Update(0.1)
		ghost.Render()
		ghost.Stop()
		if ghost.Done() {
			t.Fatal("a nil blast is never done")
		}
		if ghost.FlameAt(sprite.N) != nil {
			t.Fatal("a nil blast holds no flames")
		}
		b := NewBlast(7)
		b.Start(80, 24)
		if b.FlameAt("NNE") != nil {
			t.Fatal("a heading off the compass holds no flame")
		}
	})
}

func TestDone(t *testing.T) {
	t.Run("happy: idle is not done, burning is not done, burnt out everywhere is done, re-fire rearms", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		if b.Done() {
			t.Fatal("an untriggered blast has not performed — not done")
		}
		b.Fire()
		if b.Done() {
			t.Fatal("the whole rose is mid-air — not done")
		}
		settle(b, 160, 1.0/30) // > 5s: past every life and the pulse fuse
		if !b.Done() {
			t.Fatalf("the blast must burn out and report done, %d still live", live(b))
		}
		b.Fire()
		if b.Done() {
			t.Fatal("a fresh trigger must rearm the blast")
		}
	})
	t.Run("unhappy: unstarted and stopped blasts are never done", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		if b.Done() {
			t.Fatal("an unstarted blast is not done")
		}
		b.Start(80, 24)
		b.Fire()
		settle(b, 160, 1.0/30)
		b.Stop()
		if b.Done() {
			t.Fatal("a stopped blast holds nothing — not done")
		}
		b.Start(80, 24)
		if b.Done() {
			t.Fatal("a fresh Start rises idle, the old shot forgotten")
		}
	})
}

func TestRetune(t *testing.T) {
	t.Run("happy: UseBlast mid-burn retunes the next volley live, direction by direction", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		c := ActiveBlast()
		north := c.ShotAt(sprite.N)
		north.Count = 3
		c.SetShot(sprite.N, north)
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b.Update(1.0 / 30)
		b.Fire()
		if got := len(b.FlameAt(sprite.N).Particles); got != 3 {
			t.Fatalf("the next N volley must wear the new count, got %d want 3", got)
		}
	})
	t.Run("unhappy: a rejected retune leaves the blast on the old numbers", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		bad := ActiveBlast()
		bad.MuzzleY = 5
		if err := UseBlast(bad); err == nil {
			t.Fatal("an off-stage muzzle must be rejected")
		}
		b.Update(1.0 / 30)
		b.Fire()
		if got := len(b.FlameAt(sprite.N).Particles); got != DefaultBlast().ShotAt(sprite.N).Count {
			t.Fatalf("a rejected retune must keep the stock volley, got %d", got)
		}
	})
}

func TestBlastRender(t *testing.T) {
	t.Run("happy: a fired blast paints onto a stage-sized sprite", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(40, 20)
		b.Fire()
		sp := b.Render()
		if sp.Width != 40 || sp.Height != 20 {
			t.Fatalf("sprite is %dx%d, want the 40x20 stage", sp.Width, sp.Height)
		}
		painted := false
		for r := 0; r < sp.Height && !painted; r++ {
			for c := 0; c < sp.Width; c++ {
				if !sp.At(r, c).Transparent() {
					painted = true
					break
				}
			}
		}
		if !painted {
			t.Fatal("a just-fired flame must be visible")
		}
	})
	t.Run("unhappy: idle, unstarted, and stopped blasts render clear", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		if sp := b.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted blast rendered a %dx%d sprite", sp.Width, sp.Height)
		}
		b.Start(40, 20)
		sp := b.Render()
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				if !sp.At(r, c).Transparent() {
					t.Fatalf("an idle blast painted cell (%d,%d) %q", r, c, sp.At(r, c).Ch)
				}
			}
		}
		b.Fire()
		b.Stop()
		if sp := b.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped blast rendered a %dx%d sprite", sp.Width, sp.Height)
		}
	})
}
