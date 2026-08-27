package cloud

// Tests written FIRST: Cloud is a stationary puff of pool particles
// as a scene component. The generator (Generate) builds a unique
// layout of overlapping blobs from a seed and the live knobs —
// count, puffs, radius, spread, and the white/gray ladder — so two
// seeds never draw the same cloud. Start parks the specks; Update
// keeps them put (and re-reads the active knobs only on a fresh
// Start); Render paints concentration as braille / ░ / ▒. A Field
// of generated clouds rides the same rise pan the blue sky uses,
// so they wait off the top of a horizon shot and drift into view
// as the camera tilts up.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 64
	stageH = 24
)

var _ screenplay.Component = (*Cloud)(nil)
var _ screenplay.Component = (*Field)(nil)

func cloudGlyph(c sprite.Cell) bool {
	return (c.Ch >= '⠀' && c.Ch <= '⣿') || c.Ch == '░' || c.Ch == '▒'
}

func glyphCount(sp sprite.Sprite) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if cloudGlyph(sp.At(r, c)) {
				n++
			}
		}
	}
	return n
}

func TestCloudComponent(t *testing.T) {
	t.Run("happy: Generate parks a unique puff and the first frame already has cloud", func(t *testing.T) {
		t.Cleanup(Reset)
		cl := Generate(11, Active())
		if cl == nil {
			t.Fatal("Generate must return a cloud")
		}
		cl.Start(stageW, stageH)
		if len(cl.Engines) == 0 {
			t.Fatal("Start must build the pool engines the generator laid out")
		}
		for i, e := range cl.Engines {
			if e == nil {
				t.Fatalf("engine %d is nil", i)
			}
			if e.Cfg.Mode != particle.ModePool {
				t.Fatalf("engine %d mode %v, want ModePool — clouds are parked specks", i, e.Cfg.Mode)
			}
			if len(e.Particles) == 0 {
				t.Fatalf("engine %d has no specks — Generate must Burst the pool", i)
			}
		}
		sp := cl.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", sp.Width, sp.Height, stageW, stageH)
		}
		if glyphCount(sp) == 0 {
			t.Fatal("a started cloud must already be painted")
		}
	})
	t.Run("happy: New is Generate on the active knobs", func(t *testing.T) {
		t.Cleanup(Reset)
		cl := New(7)
		cl.Start(stageW, stageH)
		if glyphCount(cl.Render()) == 0 {
			t.Fatal("New must generate a visible cloud from the active knobs")
		}
	})
	t.Run("happy: parked specks stay put across updates", func(t *testing.T) {
		t.Cleanup(Reset)
		cl := Generate(11, Active())
		cl.Start(stageW, stageH)
		before := map[[2]int]rune{}
		sp := cl.Render()
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				cell := sp.At(r, c)
				if cloudGlyph(cell) {
					before[[2]int{r, c}] = cell.Ch
				}
			}
		}
		cl.Update(0.5)
		sp = cl.Render()
		after := 0
		moved := 0
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				cell := sp.At(r, c)
				if !cloudGlyph(cell) {
					continue
				}
				after++
				if before[[2]int{r, c}] != cell.Ch {
					moved++
				}
			}
		}
		if after != len(before) {
			t.Fatalf("the puff changed size %d -> %d — pool clouds hold still", len(before), after)
		}
		if moved != 0 {
			t.Fatalf("%d cells changed glyph — a stationary pool must not drift", moved)
		}
	})
	t.Run("unhappy: before Start and after Stop the stage is empty, never panics", func(t *testing.T) {
		cl := New(3)
		if sp := cl.Render(); sp.Width != 0 && glyphCount(sp) != 0 {
			t.Fatal("unstarted render must not paint")
		}
		cl.Update(1)
		cl.Stop()
		var ghost *Cloud
		ghost.Start(10, 10)
		ghost.Update(1)
		_ = ghost.Render()
		ghost.Stop()
	})
}

func TestGenerateUnique(t *testing.T) {
	t.Run("happy: two seeds draw two different clouds, one seed is deterministic", func(t *testing.T) {
		t.Cleanup(Reset)
		a := Generate(11, Active())
		b := Generate(11, Active())
		c := Generate(12, Active())
		a.Start(stageW, stageH)
		b.Start(stageW, stageH)
		c.Start(stageW, stageH)
		if !sameLayout(a, b) {
			t.Fatal("the same seed must generate the same puff layout")
		}
		if sameLayout(a, c) {
			t.Fatal("a different seed must generate a different puff — unique clouds")
		}
	})
	t.Run("happy: more puffs cover more of the stage than one blob", func(t *testing.T) {
		t.Cleanup(Reset)
		one := Active()
		one.Puffs = 1
		many := Active()
		many.Puffs = 6
		a := Generate(11, one)
		b := Generate(11, many)
		a.Start(stageW, stageH)
		b.Start(stageW, stageH)
		if glyphCount(b.Render()) <= glyphCount(a.Render()) {
			t.Fatalf("six puffs painted %d cells, one puff painted %d — the generator must spread extra blobs", glyphCount(b.Render()), glyphCount(a.Render()))
		}
	})
	t.Run("unhappy: a silent puff (count 0) generates a cloud that paints nothing", func(t *testing.T) {
		t.Cleanup(Reset)
		cfg := Active()
		cfg.Count = 0
		cl := Generate(11, cfg)
		cl.Start(stageW, stageH)
		if glyphCount(cl.Render()) != 0 {
			t.Fatal("Count=0 must be a clear sky, not a leftover puff")
		}
	})
}

func TestFieldRise(t *testing.T) {
	t.Run("happy: a field opens off the top of a horizon shot and rides into view", func(t *testing.T) {
		t.Cleanup(Reset)
		f := NewField(11).Rise(2)
		f.Start(stageW, stageH)
		if glyphCount(f.Render()) != 0 {
			t.Fatal("at pan 0 the clouds wait in the upper sky, off this horizon shot")
		}
		f.Update(2)
		if f.Pan() < 1-1e-9 {
			t.Fatalf("after the rise pan %v, want 1", f.Pan())
		}
		if glyphCount(f.Render()) == 0 {
			t.Fatal("a finished rise must bring the generated clouds into view")
		}
	})
	t.Run("unhappy: dt <= 0 never pans the field", func(t *testing.T) {
		f := NewField(11).Rise(2)
		f.Start(stageW, stageH)
		f.Update(0)
		f.Update(-1)
		if f.Pan() != 0 {
			t.Fatalf("dt<=0 pan %v, want 0", f.Pan())
		}
	})
}

func sameLayout(a, b *Cloud) bool {
	if a == nil || b == nil || len(a.Engines) != len(b.Engines) {
		return false
	}
	for i, e := range a.Engines {
		oe := b.Engines[i]
		if e == nil || oe == nil || len(e.Particles) != len(oe.Particles) {
			return false
		}
		for j, p := range e.Particles {
			if p.Pos != oe.Particles[j].Pos {
				return false
			}
		}
	}
	return true
}
