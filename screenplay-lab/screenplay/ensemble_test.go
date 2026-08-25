package screenplay

// Tests written FIRST: an Ensemble is the common scene shape — a cast
// of sprites playing over time. The cast is assembled when the curtain
// rises (Start), every actor sees the same dt, render order is cast
// order so later actors land on top, and Stop drops the cast. Nothing
// is allocated before Start and nothing panics on the sad paths.

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

// probe is a rehearsal actor: it logs updates and stamps one glyph.
type probe struct {
	glyph    rune
	x, y     int
	updated  []float64
	rendered int
}

func (p *probe) Update(dt float64) { p.updated = append(p.updated, dt) }

func (p *probe) Render(scr *Screen) {
	p.rendered++
	scr.Put(p.x, p.y, p.glyph, uv.Style{})
}

func TestEnsembleLifecycle(t *testing.T) {
	t.Run("happy: the cast is assembled at the curtain, not before", func(t *testing.T) {
		assembled := 0
		var a *probe
		e := &Ensemble{Assemble: func() []Actor {
			assembled++
			a = &probe{glyph: 'a'}
			return []Actor{a}
		}}
		e.Update(0.5)
		e.Render(NewScreen(3, 3))
		if assembled != 0 {
			t.Fatal("nothing may allocate before Start")
		}
		e.Start()
		if assembled != 1 {
			t.Fatalf("Start must assemble once, did %d times", assembled)
		}
		e.Update(0.5)
		if len(a.updated) != 1 || a.updated[0] != 0.5 {
			t.Fatalf("actor saw %v, want [0.5]", a.updated)
		}
	})
	t.Run("happy: cast order is render order — later actors on top", func(t *testing.T) {
		under := &probe{glyph: 'u', x: 1, y: 1}
		over := &probe{glyph: 'o', x: 1, y: 1}
		e := &Ensemble{Assemble: func() []Actor { return []Actor{under, over} }}
		e.Start()
		scr := NewScreen(3, 3)
		e.Render(scr)
		if got := scr.Cell(1, 1); got == nil || got.Content != "o" {
			t.Fatalf("top glyph %+v, want the later actor's o", got)
		}
		if under.rendered != 1 || over.rendered != 1 {
			t.Fatalf("renders %d/%d, want 1/1", under.rendered, over.rendered)
		}
	})
	t.Run("happy: stop drops the cast; a fresh start rebuilds it", func(t *testing.T) {
		assembled := 0
		var a *probe
		e := &Ensemble{Assemble: func() []Actor {
			assembled++
			a = &probe{glyph: 'a'}
			return []Actor{a}
		}}
		e.Start()
		first := a
		e.Stop()
		e.Update(1.0)
		if len(first.updated) != 0 {
			t.Fatalf("a stopped scene must not tick its old cast, saw %v", first.updated)
		}
		e.Start()
		if assembled != 2 {
			t.Fatalf("restart must reassemble, assembled %d times", assembled)
		}
		if a == first {
			t.Fatal("restart must build a fresh cast")
		}
	})
	t.Run("unhappy: zero and negative dt hold every clock", func(t *testing.T) {
		a := &probe{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Actor { return []Actor{a} }}
		e.Start()
		e.Update(0)
		e.Update(-1)
		if len(a.updated) != 0 {
			t.Fatalf("dt<=0 must not reach the cast, saw %v", a.updated)
		}
	})
	t.Run("unhappy: nil actors are skipped, not panics", func(t *testing.T) {
		a := &probe{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Actor { return []Actor{nil, a} }}
		e.Start()
		e.Update(0.1)
		e.Render(NewScreen(2, 2))
		if len(a.updated) != 1 || a.rendered != 1 {
			t.Fatal("real actors must still play around a nil one")
		}
	})
	t.Run("unhappy: a nil screen never reaches the cast", func(t *testing.T) {
		a := &probe{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Actor { return []Actor{a} }}
		e.Start()
		e.Render(nil)
		if a.rendered != 0 {
			t.Fatalf("nil screen reached the cast %d times", a.rendered)
		}
	})
	t.Run("unhappy: no Assemble means an empty, harmless scene", func(t *testing.T) {
		e := &Ensemble{}
		e.Start()
		e.Update(0.5)
		e.Render(NewScreen(2, 2))
		e.Stop()
	})
	t.Run("unhappy: a nil ensemble skips its cue", func(t *testing.T) {
		var e *Ensemble
		e.Start()
		e.Update(0.5)
		e.Render(NewScreen(2, 2))
		e.Stop()
	})
}
