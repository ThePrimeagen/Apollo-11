package screenplay

// Tests written FIRST: an Ensemble is the common scene shape — a cast
// of components playing over time. A component lives the lifecycle the
// screenplay promises: Start(w, h) allocates for the stage, Update(dt)
// runs the clock, Render() hands back a sprite the ensemble composites
// in cast order, and Stop frees what Start built — then Start may come
// again. The ensemble owns the staging: components start on the first
// render (when the stage size is finally known), restart when the
// stage resizes, never see an update before their start, and are
// stopped and dropped at the curtain.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// stagehand is a rehearsal component: it logs its lifecycle and stamps
// one glyph onto its stage-sized sprite.
type stagehand struct {
	glyph    rune
	x, y     int
	w, h     int
	starts   []([2]int)
	stops    int
	updated  []float64
	rendered int
}

func (s *stagehand) Start(w, h int) {
	s.w, s.h = w, h
	s.starts = append(s.starts, [2]int{w, h})
}

func (s *stagehand) Update(dt float64) { s.updated = append(s.updated, dt) }

func (s *stagehand) Render() sprite.Sprite {
	s.rendered++
	stage := sprite.New(s.w, s.h)
	stage.Set(s.y, s.x, sprite.Cell{Ch: s.glyph, FG: 15, BG: -1})
	return stage
}

func (s *stagehand) Stop() { s.stops++ }

func TestComponentContract(t *testing.T) {
	t.Run("happy: the ensemble speaks the component lifecycle", func(t *testing.T) {
		// The compile-time pin: a stagehand is a Component.
		var _ Component = (*stagehand)(nil)
	})
}

func TestEnsembleLifecycle(t *testing.T) {
	t.Run("happy: the cast is assembled at the curtain, not before", func(t *testing.T) {
		assembled := 0
		e := &Ensemble{Assemble: func() []Component {
			assembled++
			return []Component{&stagehand{glyph: 'a'}}
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
	})
	t.Run("happy: components start on the first render, with the stage size", func(t *testing.T) {
		a := &stagehand{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Component { return []Component{a} }}
		e.Start()
		if len(a.starts) != 0 {
			t.Fatal("no component may start before the stage size is known")
		}
		e.Render(NewScreen(7, 4))
		if len(a.starts) != 1 || a.starts[0] != [2]int{7, 4} {
			t.Fatalf("component started with %v, want one start at [7 4]", a.starts)
		}
		if a.rendered != 1 {
			t.Fatalf("component rendered %d times, want 1", a.rendered)
		}
	})
	t.Run("happy: updates flow only after the components have started", func(t *testing.T) {
		a := &stagehand{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Component { return []Component{a} }}
		e.Start()
		e.Update(0.5)
		if len(a.updated) != 0 {
			t.Fatalf("update before the first render reached the cast: %v", a.updated)
		}
		e.Render(NewScreen(3, 3))
		e.Update(0.5)
		if len(a.updated) != 1 || a.updated[0] != 0.5 {
			t.Fatalf("component saw %v, want [0.5]", a.updated)
		}
	})
	t.Run("happy: cast order is render order — later components on top", func(t *testing.T) {
		under := &stagehand{glyph: 'u', x: 1, y: 1}
		over := &stagehand{glyph: 'o', x: 1, y: 1}
		e := &Ensemble{Assemble: func() []Component { return []Component{under, over} }}
		e.Start()
		scr := NewScreen(3, 3)
		e.Render(scr)
		if got := scr.Cell(1, 1); got == nil || got.Content != "o" {
			t.Fatalf("top glyph %+v, want the later component's o", got)
		}
	})
	t.Run("happy: a transparent cell spares the component below", func(t *testing.T) {
		under := &stagehand{glyph: 'u', x: 0, y: 0}
		over := &stagehand{glyph: 'o', x: 2, y: 2} // leaves (0,0) transparent
		e := &Ensemble{Assemble: func() []Component { return []Component{under, over} }}
		e.Start()
		scr := NewScreen(3, 3)
		e.Render(scr)
		if got := scr.Cell(0, 0); got == nil || got.Content != "u" {
			t.Fatalf("glyph under a transparent cell = %+v, want u", got)
		}
		if got := scr.Cell(2, 2); got == nil || got.Content != "o" {
			t.Fatalf("top component's glyph = %+v, want o", got)
		}
	})
	t.Run("happy: a resize stops and restarts the cast at the new size", func(t *testing.T) {
		a := &stagehand{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Component { return []Component{a} }}
		e.Start()
		e.Render(NewScreen(4, 4))
		e.Render(NewScreen(4, 4))
		if len(a.starts) != 1 || a.stops != 0 {
			t.Fatalf("a steady stage restarted the cast: starts %v stops %d", a.starts, a.stops)
		}
		e.Render(NewScreen(9, 5))
		if a.stops != 1 {
			t.Fatalf("a resize must stop the old staging, stops %d", a.stops)
		}
		if len(a.starts) != 2 || a.starts[1] != [2]int{9, 5} {
			t.Fatalf("a resize must restart at the new size, starts %v", a.starts)
		}
	})
	t.Run("happy: stop frees the cast; a fresh start rebuilds it", func(t *testing.T) {
		assembled := 0
		var last *stagehand
		e := &Ensemble{Assemble: func() []Component {
			assembled++
			last = &stagehand{glyph: 'a'}
			return []Component{last}
		}}
		e.Start()
		e.Render(NewScreen(3, 3))
		first := last
		e.Stop()
		if first.stops != 1 {
			t.Fatalf("the curtain must stop a started component, stops %d", first.stops)
		}
		e.Update(1.0)
		if len(first.updated) != 0 {
			t.Fatalf("a stopped scene must not tick its old cast, saw %v", first.updated)
		}
		e.Start()
		e.Render(NewScreen(3, 3))
		if assembled != 2 {
			t.Fatalf("restart must reassemble, assembled %d times", assembled)
		}
		if last == first {
			t.Fatal("restart must build a fresh cast")
		}
	})
	t.Run("unhappy: stopping an unstaged scene stops no component", func(t *testing.T) {
		a := &stagehand{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Component { return []Component{a} }}
		e.Start()
		e.Stop() // no render happened: nothing ever started
		if a.stops != 0 {
			t.Fatalf("an unstarted component was stopped %d times", a.stops)
		}
	})
	t.Run("unhappy: zero and negative dt hold every clock", func(t *testing.T) {
		a := &stagehand{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Component { return []Component{a} }}
		e.Start()
		e.Render(NewScreen(3, 3))
		e.Update(0)
		e.Update(-1)
		if len(a.updated) != 0 {
			t.Fatalf("dt<=0 must not reach the cast, saw %v", a.updated)
		}
	})
	t.Run("unhappy: nil components are skipped, not panics", func(t *testing.T) {
		a := &stagehand{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Component { return []Component{nil, a} }}
		e.Start()
		e.Render(NewScreen(2, 2))
		e.Update(0.1)
		if len(a.updated) != 1 || a.rendered != 1 {
			t.Fatal("real components must still play around a nil one")
		}
		e.Stop()
	})
	t.Run("unhappy: a nil screen never reaches the cast", func(t *testing.T) {
		a := &stagehand{glyph: 'a'}
		e := &Ensemble{Assemble: func() []Component { return []Component{a} }}
		e.Start()
		e.Render(nil)
		if a.rendered != 0 || len(a.starts) != 0 {
			t.Fatalf("nil screen reached the cast: %d renders, %v starts", a.rendered, a.starts)
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
