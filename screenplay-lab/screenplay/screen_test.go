package screenplay

// Tests written FIRST: the screen is the shared render target — a lip
// gloss cell canvas, one content+style cell per terminal cell, plus the
// bookkeeping a screenplay needs: size, resize-follows-the-terminal,
// and a resized flag that warns the next render to repaint everything.
// Writes past an edge vanish; nothing here ever panics.

import (
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
)

func ink(fg int) uv.Style { return uv.Style{Fg: ansi.IndexedColor(uint8(fg))} }

// litCount counts cells holding visible content.
func litCount(s *Screen) int {
	n := 0
	w, h := s.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := s.Cell(x, y)
			if c == nil {
				continue
			}
			if c.Content != "" && c.Content != " " {
				n++
			}
		}
	}
	return n
}

func TestNewScreen(t *testing.T) {
	t.Run("happy: a fresh screen is the asked-for size and fully dark", func(t *testing.T) {
		s := NewScreen(10, 4)
		if w, h := s.Size(); w != 10 || h != 4 {
			t.Fatalf("size %dx%d, want 10x4", w, h)
		}
		if litCount(s) != 0 {
			t.Fatalf("a new screen must be blank, %d cells lit", litCount(s))
		}
		if s.Resized() {
			t.Fatal("a new screen has not been resized")
		}
		if s.Canvas() == nil {
			t.Fatal("the lip gloss canvas must be reachable")
		}
	})
	t.Run("happy: a put cell comes back and renders", func(t *testing.T) {
		s := NewScreen(10, 4)
		s.Put(9, 3, '#', ink(10))
		c := s.Cell(9, 3)
		if c == nil || c.Content != "#" {
			t.Fatalf("cell (9,3) = %+v, want #", c)
		}
		if c.Style.Fg != ansi.IndexedColor(10) {
			t.Fatalf("ink %v, want indexed 10", c.Style.Fg)
		}
		v := s.Render()
		if !strings.Contains(v, "#") {
			t.Fatalf("render lost the glyph: %q", v)
		}
		if got := strings.Count(v, "\n"); got != 3 {
			t.Fatalf("a glyph on the last row keeps all 4 rows, got %d newlines", got)
		}
	})
	t.Run("unhappy: zero and negative sizes are safe and render empty", func(t *testing.T) {
		for _, dims := range [][2]int{{0, 0}, {-3, 2}, {5, -1}} {
			s := NewScreen(dims[0], dims[1])
			s.Put(0, 0, '#', ink(10))
			s.Clear()
			if v := s.Render(); v != "" {
				t.Fatalf("screen %v must render empty, got %q", dims, v)
			}
			if w, h := s.Size(); w < 0 || h < 0 {
				t.Fatalf("screen %v reports negative size %dx%d", dims, w, h)
			}
		}
	})
	t.Run("unhappy: a nil screen ignores every call", func(t *testing.T) {
		var s *Screen
		s.Put(0, 0, '#', ink(10))
		s.Clear()
		s.Resize(4, 4)
		if w, h := s.Size(); w != 0 || h != 0 {
			t.Fatalf("nil size %dx%d, want 0x0", w, h)
		}
		if s.Cell(0, 0) != nil || s.Canvas() != nil || s.Resized() || s.Render() != "" {
			t.Fatal("a nil screen must be inert")
		}
	})
}

func TestPut(t *testing.T) {
	t.Run("happy: the later put wins the cell", func(t *testing.T) {
		s := NewScreen(6, 3)
		s.Put(2, 1, 'a', ink(10))
		s.Put(2, 1, 'b', ink(20))
		if c := s.Cell(2, 1); c.Content != "b" || c.Style.Fg != ansi.IndexedColor(20) {
			t.Fatalf("cell = %+v, want the later b", c)
		}
		if litCount(s) != 1 {
			t.Fatalf("%d cells lit, want 1", litCount(s))
		}
	})
	t.Run("unhappy: out-of-bounds puts vanish without a panic", func(t *testing.T) {
		s := NewScreen(6, 3)
		for _, at := range [][2]int{{-1, 0}, {0, -1}, {6, 0}, {0, 3}, {99, 99}} {
			s.Put(at[0], at[1], '#', ink(10))
		}
		if litCount(s) != 0 {
			t.Fatalf("OOB puts must be ignored, %d cells lit", litCount(s))
		}
	})
}

func TestClear(t *testing.T) {
	t.Run("happy: clear blanks every cell", func(t *testing.T) {
		s := NewScreen(6, 3)
		s.Put(0, 0, 'a', ink(10))
		s.Put(5, 2, 'b', ink(20))
		s.Clear()
		if litCount(s) != 0 {
			t.Fatalf("clear left %d cells lit", litCount(s))
		}
	})
}

func TestResize(t *testing.T) {
	t.Run("happy: resize follows the terminal and flags the change", func(t *testing.T) {
		s := NewScreen(6, 3)
		s.Resize(8, 5)
		if w, h := s.Size(); w != 8 || h != 5 {
			t.Fatalf("size %dx%d after resize, want 8x5", w, h)
		}
		if !s.Resized() {
			t.Fatal("a resize must flag the next frame for a full repaint")
		}
		s.Put(7, 4, '#', ink(10))
		if c := s.Cell(7, 4); c == nil || c.Content != "#" {
			t.Fatalf("the grown area must take cells, got %+v", c)
		}
	})
	t.Run("unhappy: a same-size resize does not cry wolf", func(t *testing.T) {
		s := NewScreen(6, 3)
		s.Resize(6, 3)
		if s.Resized() {
			t.Fatal("resizing to the same size is not a change")
		}
	})
	t.Run("unhappy: shrinking to nothing and back stays safe", func(t *testing.T) {
		s := NewScreen(6, 3)
		s.Resize(0, 0)
		s.Put(0, 0, '#', ink(10))
		if v := s.Render(); v != "" {
			t.Fatalf("a zero screen must render empty, got %q", v)
		}
		s.Resize(4, 2)
		s.Put(3, 1, '#', ink(10))
		if c := s.Cell(3, 1); c == nil || c.Content != "#" {
			t.Fatal("a regrown screen must take cells again")
		}
	})
}
