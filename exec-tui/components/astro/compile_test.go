package astro

// Tests written FIRST. A frame is authored as a 16×16 pixel grid of
// palette letters ('.' is empty sky). The terminal can't draw square
// pixels, so CompileGrid folds every two pixel rows into one cell row
// of half-blocks: ▀ carries a top pixel, ▄ a bottom pixel, █ both in
// one color, and a ▀ with a background carries two different colors.
// The compiler must refuse grids it cannot represent instead of
// guessing.

import (
	"strings"
	"testing"
)

// colorOf resolves a palette letter to its fg index straight from the
// package palette, so the tests never hard-code xterm numbers.
func colorOf(t *testing.T, id string) int {
	t.Helper()
	for _, p := range Palette {
		if p.ID == id {
			return p.FG
		}
	}
	t.Fatalf("palette has no entry %q", id)
	return -1
}

func TestCompileGrid(t *testing.T) {
	t.Run("happy: the four half-block cases land on the right glyphs and colors", func(t *testing.T) {
		suit := colorOf(t, "W")
		dark := colorOf(t, "D")
		sp, err := CompileGrid([]string{
			"W.WW",
			".WWD",
		})
		if err != nil {
			t.Fatalf("CompileGrid: %v", err)
		}
		if sp.Width != 4 || sp.Height != 1 {
			t.Fatalf("compiled to %dx%d, want 4x1 — two pixel rows are one cell row", sp.Width, sp.Height)
		}
		checks := []struct {
			col      int
			ch       rune
			fg, bg   int
			whatItIs string
		}{
			{0, '▀', suit, -1, "top-only pixel"},
			{1, '▄', suit, -1, "bottom-only pixel"},
			{2, '█', suit, -1, "both pixels, one color"},
			{3, '▀', suit, dark, "two colors: top fg, bottom bg"},
		}
		for _, c := range checks {
			cell := sp.At(0, c.col)
			if cell.Ch != c.ch || cell.FG != c.fg || cell.BG != c.bg {
				t.Fatalf("col %d (%s): got %q fg %d bg %d, want %q fg %d bg %d",
					c.col, c.whatItIs, string(cell.Ch), cell.FG, cell.BG, string(c.ch), c.fg, c.bg)
			}
		}
	})
	t.Run("happy: empty sky compiles to transparent cells", func(t *testing.T) {
		sp, err := CompileGrid([]string{"..", ".."})
		if err != nil {
			t.Fatalf("CompileGrid: %v", err)
		}
		cell := sp.At(0, 0)
		if !cell.Transparent() {
			t.Fatalf("empty pixels must stay transparent, got %q fg %d bg %d", string(cell.Ch), cell.FG, cell.BG)
		}
	})
	t.Run("unhappy: an unknown letter is an error naming the letter", func(t *testing.T) {
		_, err := CompileGrid([]string{"W?", "WW"})
		if err == nil {
			t.Fatal("an unknown pixel letter must be an error")
		}
		if !strings.Contains(err.Error(), "?") {
			t.Fatalf("the error must name the letter, got %v", err)
		}
	})
	t.Run("unhappy: odd row counts and ragged rows are errors", func(t *testing.T) {
		if _, err := CompileGrid([]string{"WW"}); err == nil {
			t.Fatal("one pixel row cannot make half-block cells — must error")
		}
		if _, err := CompileGrid([]string{"WW", "W"}); err == nil {
			t.Fatal("ragged rows must error")
		}
		if _, err := CompileGrid(nil); err == nil {
			t.Fatal("an empty grid must error")
		}
	})
}

func TestPixelGrids(t *testing.T) {
	t.Run("happy: every pose has a 16x16 grid of palette letters", func(t *testing.T) {
		for _, pose := range Poses {
			grid, ok := grids[pose]
			if !ok {
				t.Fatalf("pose %q has no pixel grid", pose)
			}
			if len(grid) != PxH {
				t.Fatalf("pose %q has %d pixel rows, want %d", pose, len(grid), PxH)
			}
			for r, row := range grid {
				if len([]rune(row)) != PxW {
					t.Fatalf("pose %q row %d has %d pixels, want %d", pose, r, len([]rune(row)), PxW)
				}
			}
			if _, err := CompileGrid(grid); err != nil {
				t.Fatalf("pose %q does not compile: %v", pose, err)
			}
		}
	})
	t.Run("happy: every pose reads differently — no copy-pasted frames", func(t *testing.T) {
		for i, a := range Poses {
			for _, b := range Poses[i+1:] {
				ga := strings.Join(grids[a], "\n")
				gb := strings.Join(grids[b], "\n")
				if ga == gb {
					t.Fatalf("poses %q and %q are identical grids", a, b)
				}
			}
		}
	})
	t.Run("unhappy: no grid smuggles a letter the palette cannot resolve", func(t *testing.T) {
		known := map[rune]bool{'.': true}
		for _, p := range Palette {
			for _, r := range p.ID {
				known[r] = true
			}
		}
		for _, pose := range Poses {
			for r, row := range grids[pose] {
				for c, px := range row {
					if !known[px] {
						t.Fatalf("pose %q pixel (%d,%d) uses unknown letter %q", pose, r, c, string(px))
					}
				}
			}
		}
	})
}
