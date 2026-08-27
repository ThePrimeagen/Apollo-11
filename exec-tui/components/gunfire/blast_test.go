package gunfire

// Tests written FIRST. Blast is the one-shot muzzle flame: Start
// builds two quiet engines (the white-hot core and the red flame) and
// holds fire; Fire is the trigger — both burst at the muzzle now, and
// Doom's second flash frame follows on a short fuse: one dimmer
// re-pulse, a fraction of the first. Update flies the flame and
// re-reads the active blast so the tuner retunes it live; Done
// reports the burnt-out flame. There is no period clock anywhere: no
// trigger, no fire.

import (
	"testing"
)

// settle runs the blast n frames of dt without any new trigger.
func settle(b *Blast, n int, dt float64) {
	for i := 0; i < n; i++ {
		b.Update(dt)
	}
}

func live(b *Blast) int {
	return len(b.Core.Particles) + len(b.Flame.Particles)
}

func TestHoldFire(t *testing.T) {
	t.Run("happy: Start builds the engines but the stage stays clear until the trigger", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		if b.Core == nil || b.Flame == nil {
			t.Fatal("Start must build both engines")
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
	t.Run("happy: the trigger bursts the core and the flame at the muzzle now", func(t *testing.T) {
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
		if got := len(b.Flame.Particles); got != c.Flame.Count {
			t.Fatalf("flame burst %d, want %d", got, c.Flame.Count)
		}
	})
	t.Run("happy: Doom's second frame — a dimmer re-pulse follows on the fuse, once", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.PulseDelay = 0.1
		c.PulseFrac = 0.5
		c.Flame.Count = 40
		c.Core.Count = 20
		c.Flame.MinLife, c.Flame.MaxLife = 5, 5 // nothing dies during the check
		c.Core.MinLife, c.Core.MaxLife = 5, 5
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b := NewBlast(7)
		b.Start(80, 24)
		b.Fire()
		first := live(b)
		b.Update(0.05)
		if live(b) != first {
			t.Fatalf("the fuse is only half burnt and the count moved %d -> %d", first, live(b))
		}
		b.Update(0.06)
		if got := len(b.Flame.Particles); got != 60 {
			t.Fatalf("the re-pulse must add half the flame again: %d, want 60", got)
		}
		if got := len(b.Core.Particles); got != 30 {
			t.Fatalf("the re-pulse must add half the core again: %d, want 30", got)
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
			c.Flame.MinLife, c.Flame.MaxLife = 5, 5
			c.Core.MinLife, c.Core.MaxLife = 5, 5
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
	t.Run("happy: a second trigger stacks a second flame", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		c := ActiveBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		b.Fire()
		b.Fire()
		if got := len(b.Flame.Particles); got != 2*c.Flame.Count {
			t.Fatalf("two squeezes hold %d flame specks, want %d", got, 2*c.Flame.Count)
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
	t.Run("unhappy: a nil blast skips every cue", func(t *testing.T) {
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
	})
}

func TestDone(t *testing.T) {
	t.Run("happy: idle is not done, burning is not done, burnt out is done, re-fire rearms", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		if b.Done() {
			t.Fatal("an untriggered flame has not performed — not done")
		}
		b.Fire()
		b.Update(0.05)
		if b.Done() {
			t.Fatal("the flame is mid-air — not done")
		}
		settle(b, 160, 1.0/30) // > 5s: past every life and the pulse fuse
		if !b.Done() {
			t.Fatalf("the flame must burn out and report done, %d still live", live(b))
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
			t.Fatal("a fresh Start rises idle, the old flame forgotten")
		}
	})
}

func TestRetune(t *testing.T) {
	t.Run("happy: UseBlast mid-burn retunes the next flame live", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		c := ActiveBlast()
		c.Flame.Count = 3
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b.Update(1.0 / 30)
		b.Fire()
		if got := len(b.Flame.Particles); got != 3 {
			t.Fatalf("the next flame must wear the new count, got %d want 3", got)
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
		if got := len(b.Flame.Particles); got != DefaultBlast().Flame.Count {
			t.Fatalf("a rejected retune must keep the stock flame, got %d", got)
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
