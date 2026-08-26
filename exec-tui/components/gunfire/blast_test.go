package gunfire

// Tests written FIRST. Blast is the one-shot shotgun component: Start
// builds four quiet engines (flash, pellets, sparks, smoke) and holds
// fire; Fire is the trigger — flash, seven pellets, and sparks burst
// at the muzzle now, and the smoke curls out when its short fuse
// burns down; Update flies the shot and re-reads the active blast so
// the tuner retunes it live; Done reports the played-out shot. There
// is no period clock anywhere: no trigger, no particles.

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
	return len(b.Flash.Particles) + len(b.Pellets.Particles) +
		len(b.Sparks.Particles) + len(b.Smoke.Particles)
}

func TestHoldFire(t *testing.T) {
	t.Run("happy: Start builds the engines but the stage stays clear until the trigger", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		if b.Flash == nil || b.Pellets == nil || b.Sparks == nil || b.Smoke == nil {
			t.Fatal("Start must build all four engines")
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
	t.Run("happy: the trigger bursts flash, seven pellets, and sparks at the muzzle now", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		c := ActiveBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		if !b.Fire() {
			t.Fatal("a started blast must take the trigger")
		}
		if got := len(b.Flash.Particles); got != c.Flash.Count {
			t.Fatalf("flash burst %d, want %d", got, c.Flash.Count)
		}
		if got := len(b.Pellets.Particles); got != 7 {
			t.Fatalf("pellet burst %d, want the Doom seven", got)
		}
		if got := len(b.Sparks.Particles); got != c.Sparks.Count {
			t.Fatalf("spark burst %d, want %d", got, c.Sparks.Count)
		}
	})
	t.Run("happy: the smoke waits on its fuse, curls out once, and never re-bursts", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.SmokeDelay = 0.1
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b := NewBlast(7)
		b.Start(80, 24)
		b.Fire()
		if got := len(b.Smoke.Particles); got != 0 {
			t.Fatalf("smoke burst with the trigger (%d specks) — it must wait %vs", got, c.SmokeDelay)
		}
		b.Update(0.05)
		if got := len(b.Smoke.Particles); got != 0 {
			t.Fatalf("the fuse is only half burnt and %d smoke specks are out", got)
		}
		b.Update(0.06)
		if got := len(b.Smoke.Particles); got != c.Smoke.Count {
			t.Fatalf("a burnt fuse must curl out %d smoke specks, got %d", c.Smoke.Count, got)
		}
		before := len(b.Smoke.Particles)
		b.Update(0.01)
		if got := len(b.Smoke.Particles); got > before {
			t.Fatalf("the fuse burns once — smoke grew %d -> %d", before, got)
		}
	})
	t.Run("happy: a zero fuse smokes with the trigger", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.SmokeDelay = 0
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b := NewBlast(7)
		b.Start(80, 24)
		b.Fire()
		if got := len(b.Smoke.Particles); got != c.Smoke.Count {
			t.Fatalf("SmokeDelay=0 must smoke on the trigger, got %d want %d", got, c.Smoke.Count)
		}
	})
	t.Run("happy: a second trigger stacks a second volley", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		c := ActiveBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		b.Fire()
		b.Fire()
		if got := len(b.Pellets.Particles); got != 2*c.Pellets.Count {
			t.Fatalf("two squeezes hold %d pellets, want %d", got, 2*c.Pellets.Count)
		}
	})
	t.Run("unhappy: dt <= 0 burns no fuse and moves nothing", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.SmokeDelay = 0.05
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
		if len(b.Smoke.Particles) != 0 {
			t.Fatal("dt<=0 must not burn the smoke fuse")
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
	t.Run("happy: idle is not done, flying is not done, played-out is done, re-fire rearms", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		if b.Done() {
			t.Fatal("an untriggered blast has not performed — not done")
		}
		b.Fire()
		b.Update(0.05)
		if b.Done() {
			t.Fatal("the shot is mid-air — not done")
		}
		settle(b, 160, 1.0/30) // > 5s: past every life and the smoke fuse
		if !b.Done() {
			t.Fatalf("the shot must play out and report done, %d still live", live(b))
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
	t.Run("happy: UseBlast mid-flight retunes the next volley live", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		c := ActiveBlast()
		c.Sparks.Count = 3
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b.Update(1.0 / 30)
		b.Fire()
		if got := len(b.Sparks.Particles); got != 3 {
			t.Fatalf("the next volley must wear the new spark count, got %d want 3", got)
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
		if got := len(b.Pellets.Particles); got != DefaultBlast().Pellets.Count {
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
			t.Fatal("a just-fired blast must be visible")
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
