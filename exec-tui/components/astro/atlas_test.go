package astro

// Tests written FIRST. The astronaut is an original character built to
// the NES-era envelope: a 16×16 pixel frame (16 cells wide, 8 half-block
// rows tall), a handful of flat colors, and one atlas holding every
// pose — stand, three running frames, a jump, and two pole-slide grips.
// The atlas ships as assets/astronaut.json in the same editable format
// as the lunar module art, so the sprite editor can open it directly.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestBuildAtlas(t *testing.T) {
	t.Run("happy: every pose is drawn at 16x8 cells and validates", func(t *testing.T) {
		a, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		for _, pose := range Poses {
			sp, ok := a.Frame(Size, pose)
			if !ok {
				t.Fatalf("pose %q missing from the atlas", pose)
			}
			if err := sp.Validate(); err != nil {
				t.Fatalf("pose %q: %v", pose, err)
			}
			if sp.Width != Cols || sp.Height != Rows {
				t.Fatalf("pose %q is %dx%d cells, want %dx%d", pose, sp.Width, sp.Height, Cols, Rows)
			}
		}
	})
	t.Run("happy: the palette names the whole outfit and round-trips both masks", func(t *testing.T) {
		a, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		want := map[string]bool{"suit": false, "shade": false, "visor": false, "dark": false, "accent": false}
		seenFG := map[int]string{}
		for _, p := range a.Palette {
			if p.ID == "." {
				continue
			}
			if _, ok := want[p.Name]; ok {
				want[p.Name] = true
			}
			if p.FG != p.BG {
				t.Fatalf("palette %q has fg %d bg %d — a pixel color must resolve identically in both masks", p.Name, p.FG, p.BG)
			}
			if prev, dup := seenFG[p.FG]; dup {
				t.Fatalf("palette %q and %q share color %d — the file format would confuse them", prev, p.Name, p.FG)
			}
			seenFG[p.FG] = p.Name
		}
		for name, seen := range want {
			if !seen {
				t.Fatalf("palette is missing the %q color", name)
			}
		}
	})
	t.Run("happy: every painted cell resolves to a palette color", func(t *testing.T) {
		a, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		colors := map[int]bool{}
		for _, p := range a.Palette {
			if p.FG >= 0 {
				colors[p.FG] = true
			}
		}
		for _, pose := range Poses {
			sp, _ := a.Frame(Size, pose)
			for r := 0; r < sp.Height; r++ {
				for c := 0; c < sp.Width; c++ {
					cell := sp.At(r, c)
					if cell.Transparent() {
						continue
					}
					if !colors[cell.FG] {
						t.Fatalf("pose %q cell (%d,%d) fg %d is not in the palette", pose, r, c, cell.FG)
					}
					if cell.BG >= 0 && !colors[cell.BG] {
						t.Fatalf("pose %q cell (%d,%d) bg %d is not in the palette", pose, r, c, cell.BG)
					}
				}
			}
		}
	})
	t.Run("happy: the pole grips reach up toward the pole side", func(t *testing.T) {
		a, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		dark := colorOf(t, "D")
		for _, pose := range []sprite.Heading{PosePole1, PosePole2} {
			sp, _ := a.Frame(Size, pose)
			found := false
			for r := 0; r <= 2 && !found; r++ {
				for c := GripCol - 1; c < Cols; c++ {
					cell := sp.At(r, c)
					if !cell.Transparent() && (cell.FG == dark || cell.BG == dark) {
						found = true
						break
					}
				}
			}
			if !found {
				t.Fatalf("pose %q has no glove near the grip column — nothing to hold the pole with", pose)
			}
		}
	})
}

func TestLoadAtlas(t *testing.T) {
	t.Run("happy: the shipped assets/astronaut.json loads with every pose", func(t *testing.T) {
		path := FindAtlas()
		if filepath.Base(path) != "astronaut.json" {
			t.Fatalf("FindAtlas = %q, want a path to astronaut.json", path)
		}
		a, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		for _, pose := range Poses {
			sp, ok := a.Frame(Size, pose)
			if !ok {
				t.Fatalf("shipped atlas is missing pose %q", pose)
			}
			if sp.Width != Cols || sp.Height != Rows {
				t.Fatalf("shipped pose %q is %dx%d, want %dx%d", pose, sp.Width, sp.Height, Cols, Rows)
			}
		}
	})
	t.Run("unhappy: a path with no atlas is an error, not a blank astronaut", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := LoadPath(filepath.Join(dir, "astronaut.json")); err == nil {
			t.Fatal("a missing file must be an error")
		}
		bad := filepath.Join(dir, "broken.json")
		if err := os.WriteFile(bad, []byte("{"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadPath(bad); err == nil {
			t.Fatal("a corrupt file must be an error")
		}
	})
}
