package screenplay

// Tests written FIRST: the screen carries the components' pixels onto
// the lip gloss canvas. Sprites speak xterm-256 integers with -1 for
// "no color"; the screen's cells speak lip gloss styles. PutCell maps
// one cell, Blit lays a whole sprite down without letting its
// transparent cells erase the layer below, and every edge clips.

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// litCount lives in screen_test.go; contentAt reads one cell's glyph.
func contentAt(scr *Screen, x, y int) string {
	c := scr.Cell(x, y)
	if c == nil {
		return ""
	}
	return c.Content
}

func stamp() sprite.Sprite {
	sp := sprite.New(2, 2)
	sp.Set(0, 0, sprite.Cell{Ch: 'A', FG: 10, BG: -1})
	sp.Set(1, 1, sprite.Cell{Ch: 'B', FG: 20, BG: 30})
	return sp
}

func TestPutCell(t *testing.T) {
	t.Run("happy: an xterm cell lands as a lip gloss style", func(t *testing.T) {
		scr := NewScreen(6, 3)
		scr.PutCell(2, 1, '#', 178, 94)
		c := scr.Cell(2, 1)
		if c == nil || c.Content != "#" {
			t.Fatalf("cell = %+v, want #", c)
		}
		if c.Style.Fg != ansi.IndexedColor(178) || c.Style.Bg != ansi.IndexedColor(94) {
			t.Fatalf("style %+v, want indexed 178 on 94", c.Style)
		}
	})
	t.Run("happy: -1 means no color on that channel", func(t *testing.T) {
		scr := NewScreen(6, 3)
		scr.PutCell(0, 0, 'x', 252, -1)
		if c := scr.Cell(0, 0); c.Style.Fg != ansi.IndexedColor(252) || c.Style.Bg != nil {
			t.Fatalf("style %+v, want fg only", c.Style)
		}
		scr.PutCell(1, 0, 'y', -1, -1)
		if c := scr.Cell(1, 0); c.Style != (uv.Style{}) {
			t.Fatalf("style %+v, want the zero style", c.Style)
		}
	})
	t.Run("unhappy: out of bounds and nil screens are ignored", func(t *testing.T) {
		scr := NewScreen(4, 2)
		scr.PutCell(-1, 0, '#', 10, -1)
		scr.PutCell(4, 0, '#', 10, -1)
		scr.PutCell(0, 99, '#', 10, -1)
		if litCount(scr) != 0 {
			t.Fatalf("OOB puts must vanish, %d lit", litCount(scr))
		}
		var ghost *Screen
		ghost.PutCell(0, 0, '#', 10, -1)
	})
}

func TestScreenBlit(t *testing.T) {
	t.Run("happy: opaque cells land styled, transparent cells spare the layer below", func(t *testing.T) {
		scr := NewScreen(6, 3)
		scr.PutCell(2, 1, '*', 100, -1) // under the stamp's transparent (0,1)
		scr.Blit(1, 1, stamp())
		if got := contentAt(scr, 1, 1); got != "A" {
			t.Fatalf("stamp corner = %q, want A", got)
		}
		if c := scr.Cell(2, 2); c.Content != "B" || c.Style.Fg != ansi.IndexedColor(20) || c.Style.Bg != ansi.IndexedColor(30) {
			t.Fatalf("stamp corner = %+v, want a styled B", c)
		}
		if got := contentAt(scr, 2, 1); got != "*" {
			t.Fatalf("transparent stamp cell erased the layer below: %q", got)
		}
	})
	t.Run("unhappy: blits clip at every edge instead of wrapping", func(t *testing.T) {
		scr := NewScreen(4, 3)
		scr.Blit(-1, -1, stamp()) // only the stamp's (1,1) survives at (0,0)
		if got := contentAt(scr, 0, 0); got != "B" {
			t.Fatalf("neg-offset blit put %q at origin, want B", got)
		}
		if litCount(scr) != 1 {
			t.Fatalf("neg-offset blit lit %d cells, want 1", litCount(scr))
		}
		scr2 := NewScreen(4, 3)
		scr2.Blit(3, 2, stamp()) // only the stamp's (0,0) fits
		if got := contentAt(scr2, 3, 2); got != "A" {
			t.Fatalf("edge blit put %q, want A", got)
		}
		if litCount(scr2) != 1 {
			t.Fatalf("edge blit lit %d cells, want 1", litCount(scr2))
		}
		scr3 := NewScreen(4, 3)
		scr3.Blit(99, 99, stamp())
		if litCount(scr3) != 0 {
			t.Fatalf("fully offscreen blit lit %d cells", litCount(scr3))
		}
	})
	t.Run("unhappy: a nil screen takes no sprite and no panic", func(t *testing.T) {
		var ghost *Screen
		ghost.Blit(0, 0, stamp())
	})
	t.Run("unhappy: an empty sprite blits nothing", func(t *testing.T) {
		scr := NewScreen(4, 3)
		scr.Blit(0, 0, sprite.Sprite{})
		if litCount(scr) != 0 {
			t.Fatalf("empty sprite lit %d cells", litCount(scr))
		}
	})
}
