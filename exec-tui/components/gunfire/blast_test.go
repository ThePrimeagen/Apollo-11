package gunfire

// Tests written FIRST. Blast is the one-shot muzzle flame on an
// eight-point compass: one shared white-hot core and one flame engine
// per direction, each burning its own tune. Start builds all nine
// quiet; Fire is the trigger — the core and every heading's flame
// burst at the muzzle now, the whole rose at once, and Doom's second
// flash frame follows every heading on a short fuse. Update flies
// every direction's flame and re-reads the active blast so the tuner
// retunes it live; Done reports a blast with nothing burning anywhere.
// No period clock: no trigger, no fire. A blast can be pinned to its
// own tune with Use — then Start, the triggers, Update and Render all
// read that tune instead of the package active, so one gun's shot
// never retunes another's.

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

func TestFireAt(t *testing.T) {
	t.Run("happy: FireAt bursts the core and only the aimed heading's shot from the active config", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.PulseFrac = 0
		east := c.ShotAt(sprite.E)
		east.Count = 33
		c.SetShot(sprite.E, east)
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		b := NewBlast(7)
		b.Start(80, 24)
		if !b.FireAt(sprite.E) {
			t.Fatal("FireAt E after Start must pull the trigger")
		}
		if got := len(b.Core.Particles); got != c.Core.Count {
			t.Fatalf("core burst %d, want %d", got, c.Core.Count)
		}
		if got := len(b.FlameAt(sprite.E).Particles); got != 33 {
			t.Fatalf("the E flame burst %d, want its own tuned 33", got)
		}
		for _, h := range sprite.Headings {
			if h == sprite.E {
				continue
			}
			if n := len(b.FlameAt(h).Particles); n != 0 {
				t.Fatalf("FireAt E must leave %s quiet, found %d particles", h, n)
			}
		}
	})
	t.Run("unhappy: FireAt off the compass is refused and nothing bursts", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		b.Start(80, 24)
		if b.FireAt(sprite.Heading("sideways")) {
			t.Fatal("FireAt off the compass must be refused")
		}
		if n := live(b); n != 0 {
			t.Fatalf("a refused FireAt must not throw particles, found %d", n)
		}
	})
	t.Run("unhappy: FireAt before Start is a refused no-op", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		b := NewBlast(7)
		if b.FireAt(sprite.N) {
			t.Fatal("FireAt before Start must report the refused trigger")
		}
		b.Start(80, 24)
		settle(b, 5, 1.0/30)
		if n := live(b); n != 0 {
			t.Fatalf("a pre-Start FireAt must not fire later, found %d particles", n)
		}
	})
}

func TestFindConfig(t *testing.T) {
	t.Run("happy: FindConfig locates the shipped components/gunfire/config.json", func(t *testing.T) {
		path := FindConfig()
		if path == "" {
			t.Fatal("FindConfig must locate the shipped config")
		}
		c, err := LoadBlast(path)
		if err != nil {
			t.Fatalf("the path FindConfig returned must load: %v", err)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the shipped config must validate: %v", err)
		}
	})
	t.Run("unhappy: FindConfig does not invent a blank config for a missing file", func(t *testing.T) {
		if _, err := LoadBlast(t.TempDir() + "/no-such-gunfire.json"); err == nil {
			t.Fatal("a missing blast config must error — never a blank shot")
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

// TestUse: a blast can be pinned to its own tune, so one gun's shot
// never retunes another's — the package-wide active blast belongs to
// the tuner, not to every gun that fires.
func TestUse(t *testing.T) {
	t.Run("happy: a pinned blast burns its own tune while unpinned blasts follow the package active", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		own := DefaultBlast()
		own.PulseFrac = 0
		shot := own.ShotAt(sprite.E)
		shot.Count = 9
		own.SetShot(sprite.E, shot)

		pinned := NewBlast(7)
		if err := pinned.Use(own); err != nil {
			t.Fatalf("Use: %v", err)
		}
		pinned.Start(80, 24)
		loose := NewBlast(8)
		loose.Start(80, 24)

		if ActiveBlast() != DefaultBlast() {
			t.Fatal("pinning one blast must not touch the package-wide active blast")
		}
		if got := pinned.Config().ShotAt(sprite.E).Count; got != 9 {
			t.Fatalf("the pinned blast's config counts %d on E, want its own 9", got)
		}
		if got := loose.Config().ShotAt(sprite.E).Count; got != DefaultBlast().ShotAt(sprite.E).Count {
			t.Fatalf("an unpinned blast must report the package active, got %d", got)
		}
		pinned.FireAt(sprite.E)
		if got := len(pinned.FlameAt(sprite.E).Particles); got != 9 {
			t.Fatalf("the pinned blast burst %d on E, want its own 9", got)
		}
		loose.FireAt(sprite.E)
		if got := len(loose.FlameAt(sprite.E).Particles); got != DefaultBlast().ShotAt(sprite.E).Count {
			t.Fatalf("the unpinned blast burst %d on E, want the package active's %d", got, DefaultBlast().ShotAt(sprite.E).Count)
		}
		if got := len(pinned.FlameAt(sprite.E).Particles); got != 9 {
			t.Fatalf("the unpinned trigger dragged the pinned blast to %d particles on E", got)
		}
	})
	t.Run("unhappy: a bad tune is rejected and kept out; a nil blast refuses quietly", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		own := DefaultBlast()
		own.PulseFrac = 0
		shot := own.ShotAt(sprite.W)
		shot.Count = 5
		own.SetShot(sprite.W, shot)
		b := NewBlast(7)
		if err := b.Use(own); err != nil {
			t.Fatalf("Use: %v", err)
		}
		b.Start(80, 24)
		bad := own
		bad.Heading = "sideways"
		if err := b.Use(bad); err == nil {
			t.Fatal("a tune off the compass must be rejected")
		}
		b.FireAt(sprite.W)
		if got := len(b.FlameAt(sprite.W).Particles); got != 5 {
			t.Fatalf("after the refused tune the blast burst %d on W, want the kept 5", got)
		}
		var nb *Blast
		if err := nb.Use(own); err == nil {
			t.Fatal("a nil blast must refuse a tune")
		}
		if nb.Config() != ActiveBlast() {
			t.Fatal("a nil blast's config is the package active — never a panic")
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
