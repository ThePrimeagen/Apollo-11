package screenplay

// Tests written FIRST: the stage is the fixed board one frame is composed
// on. Put writes a cell, Blit lays a sprite down without letting its
// transparent cells erase what is already there, and everything past an
// edge is clipped — never a panic, never a wrap.

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

func lit(st *Stage) int {
	n := 0
	w, h := st.Size()
	for r := 0; r < h; r++ {
		for c := 0; c < w; c++ {
			if !st.Board.At(r, c).Transparent() {
				n++
			}
		}
	}
	return n
}

func mark(ch rune) sprite.Cell { return sprite.Cell{Ch: ch, FG: 10, BG: -1} }

func TestNewStage(t *testing.T) {
	t.Run("happy: a fresh stage is the asked-for size and fully dark", func(t *testing.T) {
		st := NewStage(10, 4)
		w, h := st.Size()
		if w != 10 || h != 4 {
			t.Fatalf("size %dx%d, want 10x4", w, h)
		}
		if lit(st) != 0 {
			t.Fatalf("a new stage must be transparent, %d cells lit", lit(st))
		}
		v := st.Render()
		if got := strings.Count(v, "\n"); got != 3 {
			t.Fatalf("render has %d newlines, want 3 (4 rows)", got)
		}
	})
	t.Run("unhappy: zero and negative sizes are safe and render empty", func(t *testing.T) {
		for _, dims := range [][2]int{{0, 0}, {-3, 2}, {5, -1}} {
			st := NewStage(dims[0], dims[1])
			st.Put(0, 0, mark('#'))
			st.Blit(0, 0, sprite.New(2, 2))
			if v := st.Render(); v != "" {
				t.Fatalf("stage %v must render empty, got %q", dims, v)
			}
		}
	})
	t.Run("unhappy: a nil stage ignores every call", func(t *testing.T) {
		var st *Stage
		st.Put(0, 0, mark('#'))
		st.Blit(0, 0, sprite.New(1, 1))
		if w, h := st.Size(); w != 0 || h != 0 {
			t.Fatalf("nil size %dx%d, want 0x0", w, h)
		}
		if v := st.Render(); v != "" {
			t.Fatalf("nil render %q, want empty", v)
		}
	})
}

func TestPut(t *testing.T) {
	t.Run("happy: put lights exactly one cell", func(t *testing.T) {
		st := NewStage(6, 3)
		st.Put(1, 2, mark('#'))
		if got := st.Board.At(1, 2); got.Ch != '#' || got.FG != 10 {
			t.Fatalf("cell (1,2) = %+v, want the mark", got)
		}
		if lit(st) != 1 {
			t.Fatalf("%d cells lit, want 1", lit(st))
		}
	})
	t.Run("unhappy: out-of-bounds puts vanish without a panic", func(t *testing.T) {
		st := NewStage(6, 3)
		for _, at := range [][2]int{{-1, 0}, {0, -1}, {3, 0}, {0, 6}, {99, 99}} {
			st.Put(at[0], at[1], mark('#'))
		}
		if lit(st) != 0 {
			t.Fatalf("OOB puts must be ignored, %d cells lit", lit(st))
		}
	})
}

func TestBlit(t *testing.T) {
	stamp := func() sprite.Sprite {
		sp := sprite.New(2, 2)
		sp.Set(0, 0, mark('A'))
		sp.Set(1, 1, mark('B'))
		return sp
	}
	t.Run("happy: opaque cells land, transparent cells spare the layer below", func(t *testing.T) {
		st := NewStage(6, 3)
		st.Put(1, 2, mark('*')) // under the stamp's transparent (0,1)
		st.Blit(1, 1, stamp())
		if got := st.Board.At(1, 1).Ch; got != 'A' {
			t.Fatalf("stamp corner = %q, want A", got)
		}
		if got := st.Board.At(2, 2).Ch; got != 'B' {
			t.Fatalf("stamp corner = %q, want B", got)
		}
		if got := st.Board.At(1, 2).Ch; got != '*' {
			t.Fatalf("transparent stamp cell erased the layer below: %q", got)
		}
	})
	t.Run("happy: the later blit wins the overlap", func(t *testing.T) {
		st := NewStage(6, 3)
		st.Blit(0, 0, stamp())
		over := sprite.New(1, 1)
		over.Set(0, 0, mark('Z'))
		st.Blit(0, 0, over)
		if got := st.Board.At(0, 0).Ch; got != 'Z' {
			t.Fatalf("top layer = %q, want Z", got)
		}
	})
	t.Run("unhappy: blits clip at every edge instead of wrapping", func(t *testing.T) {
		st := NewStage(4, 3)
		st.Blit(-1, -1, stamp()) // only the stamp's (1,1) survives at (0,0)
		if got := st.Board.At(0, 0).Ch; got != 'B' {
			t.Fatalf("neg-offset blit put %q at origin, want B", got)
		}
		if lit(st) != 1 {
			t.Fatalf("neg-offset blit lit %d cells, want 1", lit(st))
		}
		st2 := NewStage(4, 3)
		st2.Blit(2, 3, stamp()) // only the stamp's (0,0) fits at (2,3)
		if got := st2.Board.At(2, 3).Ch; got != 'A' {
			t.Fatalf("edge blit put %q, want A", got)
		}
		if lit(st2) != 1 {
			t.Fatalf("edge blit lit %d cells, want 1", lit(st2))
		}
		st3 := NewStage(4, 3)
		st3.Blit(99, 99, stamp())
		if lit(st3) != 0 {
			t.Fatalf("fully offstage blit lit %d cells, want 0", lit(st3))
		}
	})
}
