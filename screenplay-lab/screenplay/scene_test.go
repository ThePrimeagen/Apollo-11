package screenplay

// Tests written FIRST: a scene is a named cast playing over time. Advance
// hands the same dt to every actor in order; Paint draws the cast in cast
// order so later actors sit on top. Time never runs backwards and a
// missing stage or actor never panics the show.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

// probe is a rehearsal actor: it logs every Advance and stamps one glyph.
type probe struct {
	glyph    rune
	row, col int
	advanced []float64
	paints   int
}

func (p *probe) Advance(dt float64) { p.advanced = append(p.advanced, dt) }

func (p *probe) Paint(st *Stage) {
	p.paints++
	st.Put(p.row, p.col, sprite.Cell{Ch: p.glyph, FG: 15, BG: -1})
}

func TestSceneAdvance(t *testing.T) {
	t.Run("happy: every actor gets the same dt, once", func(t *testing.T) {
		a, b := &probe{glyph: 'a'}, &probe{glyph: 'b'}
		s := &Scene{Name: "rehearsal", Cast: []Actor{a, b}}
		s.Advance(0.5)
		for _, p := range []*probe{a, b} {
			if len(p.advanced) != 1 || p.advanced[0] != 0.5 {
				t.Fatalf("%q saw %v, want [0.5]", p.glyph, p.advanced)
			}
		}
	})
	t.Run("unhappy: zero and negative dt hold every clock", func(t *testing.T) {
		a := &probe{glyph: 'a'}
		s := &Scene{Cast: []Actor{a}}
		s.Advance(0)
		s.Advance(-1)
		if len(a.advanced) != 0 {
			t.Fatalf("dt<=0 must not reach the cast, saw %v", a.advanced)
		}
	})
	t.Run("unhappy: nil scenes and nil actors are skipped, not panics", func(t *testing.T) {
		var s *Scene
		s.Advance(0.1)
		s.Paint(NewStage(2, 2))
		holed := &Scene{Cast: []Actor{nil, &probe{glyph: 'a'}}}
		holed.Advance(0.1)
		holed.Paint(NewStage(2, 2))
	})
}

func TestScenePaint(t *testing.T) {
	t.Run("happy: cast order is paint order — later actors on top", func(t *testing.T) {
		under := &probe{glyph: 'u', row: 1, col: 1}
		over := &probe{glyph: 'o', row: 1, col: 1}
		s := &Scene{Cast: []Actor{under, over}}
		st := NewStage(3, 3)
		s.Paint(st)
		if got := st.Board.At(1, 1).Ch; got != 'o' {
			t.Fatalf("top glyph %q, want the later actor's o", got)
		}
		if under.paints != 1 || over.paints != 1 {
			t.Fatalf("paints %d/%d, want 1/1", under.paints, over.paints)
		}
	})
	t.Run("unhappy: an empty cast leaves the stage dark", func(t *testing.T) {
		st := NewStage(3, 3)
		(&Scene{Name: "empty"}).Paint(st)
		if lit(st) != 0 {
			t.Fatalf("empty scene lit %d cells", lit(st))
		}
	})
	t.Run("unhappy: a nil stage never reaches the cast", func(t *testing.T) {
		a := &probe{glyph: 'a'}
		(&Scene{Cast: []Actor{a}}).Paint(nil)
		if a.paints != 0 {
			t.Fatalf("nil stage reached the cast %d times", a.paints)
		}
	})
}
