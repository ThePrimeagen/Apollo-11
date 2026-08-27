package astro

// Tests written FIRST. The moonwalk scene needs three props drawn in
// the same pixel language as the astronaut and shipped in the same
// editable atlas: an 8-bit supply crate he can stand on (stacked two
// high for the double jump), a small American flag that rides up the
// pole, and the lunar rover the camera pans to at the end. They are
// original art in the house palette — which grows one color: the
// flag's canton blue.

import (
	"strings"
	"testing"
)

func TestPropGrids(t *testing.T) {
	t.Run("happy: every prop has a grid that compiles at its size", func(t *testing.T) {
		want := map[string][2]int{ // pixel W, pixel H
			string(PropBlock): {BlockPxW, BlockPxH},
			string(PropFlag):  {FlagPxW, FlagPxH},
			string(PropRover): {RoverPxW, RoverPxH},
		}
		for _, prop := range Props {
			grid, ok := grids[prop]
			if !ok {
				t.Fatalf("prop %q has no pixel grid", prop)
			}
			dims := want[string(prop)]
			if len(grid) != dims[1] {
				t.Fatalf("prop %q has %d pixel rows, want %d", prop, len(grid), dims[1])
			}
			for r, row := range grid {
				if len([]rune(row)) != dims[0] {
					t.Fatalf("prop %q row %d has %d pixels, want %d", prop, r, len([]rune(row)), dims[0])
				}
			}
			sp, err := CompileGrid(grid)
			if err != nil {
				t.Fatalf("prop %q does not compile: %v", prop, err)
			}
			if sp.Width != dims[0] || sp.Height != dims[1]/2 {
				t.Fatalf("prop %q compiled to %dx%d cells, want %dx%d", prop, sp.Width, sp.Height, dims[0], dims[1]/2)
			}
		}
	})
	t.Run("happy: the atlas carries the props next to the poses", func(t *testing.T) {
		a, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		for _, prop := range Props {
			sp, ok := a.Frame(Size, prop)
			if !ok {
				t.Fatalf("atlas is missing prop %q", prop)
			}
			if err := sp.Validate(); err != nil {
				t.Fatalf("prop %q: %v", prop, err)
			}
		}
	})
	t.Run("happy: the flag flies the canton blue and the stripes red", func(t *testing.T) {
		blue := colorOf(t, "B")
		red := colorOf(t, "R")
		a, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		sp, _ := a.Frame(Size, PropFlag)
		seenBlue, seenRed := false, false
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				cell := sp.At(r, c)
				if cell.FG == blue || cell.BG == blue {
					seenBlue = true
				}
				if cell.FG == red || cell.BG == red {
					seenRed = true
				}
			}
		}
		if !seenBlue || !seenRed {
			t.Fatalf("the flag must carry blue and red (blue %v, red %v)", seenBlue, seenRed)
		}
	})
	t.Run("happy: the palette names the flag blue with fg == bg", func(t *testing.T) {
		for _, p := range Palette {
			if p.Name == "flagblue" {
				if p.FG != p.BG {
					t.Fatalf("flagblue fg %d bg %d — must match for the mask round trip", p.FG, p.BG)
				}
				return
			}
		}
		t.Fatal("palette has no flagblue entry")
	})
	t.Run("unhappy: no prop grid smuggles unknown letters", func(t *testing.T) {
		known := map[rune]bool{'.': true}
		for _, p := range Palette {
			for _, r := range p.ID {
				known[r] = true
			}
		}
		for _, prop := range Props {
			for r, row := range grids[prop] {
				for c, px := range row {
					if !known[px] {
						t.Fatalf("prop %q pixel (%d,%d) uses unknown letter %q", prop, r, c, string(px))
					}
				}
			}
		}
	})
	t.Run("unhappy: props and poses never collide by name", func(t *testing.T) {
		seen := map[string]bool{}
		for _, h := range Poses {
			seen[string(h)] = true
		}
		for _, h := range Props {
			if seen[string(h)] {
				t.Fatalf("frame name %q appears twice", h)
			}
			seen[string(h)] = true
		}
		_ = strings.TrimSpace("")
	})
}
