package editor

// Tests written FIRST. Every LM size lives as its own JSON atlas in
// an assets folder the editor can list and load. Missing or corrupt
// files must fail loudly; a missing folder must not panic.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func writeMiniShip(t *testing.T, dir, name string, ch rune) string {
	t.Helper()
	a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
	sp := sprite.New(13, 5)
	sp.Set(0, 0, sprite.Cell{Ch: ch, FG: 252, BG: -1})
	a.SetFrame(sprite.Size1, sprite.N, sp)
	path := filepath.Join(dir, name+".json")
	if err := a.WriteFile(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestListShips(t *testing.T) {
	t.Run("happy: lists every json atlas in the assets folder, sorted by name", func(t *testing.T) {
		dir := t.TempDir()
		writeMiniShip(t, dir, "bravo", 'B')
		writeMiniShip(t, dir, "alpha", 'A')
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
			t.Fatal(err)
		}

		ships, err := ListShips(dir)
		if err != nil {
			t.Fatalf("ListShips: %v", err)
		}
		if len(ships) != 2 {
			t.Fatalf("want 2 json ships, got %d", len(ships))
		}
		if ships[0].Name != "alpha" || ships[1].Name != "bravo" {
			t.Fatalf("ships must be sorted by name, got %q then %q", ships[0].Name, ships[1].Name)
		}
		if ships[0].Path != filepath.Join(dir, "alpha.json") {
			t.Fatalf("path %q", ships[0].Path)
		}
	})
	t.Run("unhappy: a missing assets folder is an empty list, not a panic", func(t *testing.T) {
		ships, err := ListShips(filepath.Join(t.TempDir(), "no-such-assets"))
		if err != nil {
			t.Fatalf("missing dir must not error, got %v", err)
		}
		if len(ships) != 0 {
			t.Fatalf("missing dir must yield no ships, got %d", len(ships))
		}
	})
}

func TestLoadShip(t *testing.T) {
	t.Run("happy: LoadShip reads a json atlas the editor can draw", func(t *testing.T) {
		path := writeMiniShip(t, t.TempDir(), "tiny", 'X')
		a, err := LoadShip(path)
		if err != nil {
			t.Fatalf("LoadShip: %v", err)
		}
		sp, ok := a.Frame(sprite.Size1, sprite.N)
		if !ok {
			t.Fatal("loaded ship must carry its size-1 north frame")
		}
		if sp.At(0, 0).Ch != 'X' {
			t.Fatalf("loaded glyph %q, want X", string(sp.At(0, 0).Ch))
		}
	})
	t.Run("unhappy: a corrupt json file is an error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "broken.json")
		if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadShip(path); err == nil {
			t.Fatal("corrupt ship json must fail")
		}
	})
}

func TestOpenSnapsToExistingFrame(t *testing.T) {
	t.Run("happy: opening a size-1-only ship lands on a drawable frame", func(t *testing.T) {
		path := writeMiniShip(t, t.TempDir(), "lm-1", 'N')
		m, err := Open(path)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		sp := m.Current()
		if sp.Width != 13 || sp.Height != 5 {
			t.Fatalf("opened canvas %dx%d, want 13x5", sp.Width, sp.Height)
		}
		if sp.At(0, 0).Ch != 'N' {
			t.Fatalf("opened glyph %q, want N", string(sp.At(0, 0).Ch))
		}
	})
	t.Run("unhappy: a missing frame after open must not panic Current", func(t *testing.T) {
		path := writeMiniShip(t, t.TempDir(), "only-n", 'Q')
		m, err := Open(path)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m.Size = sprite.Size4
		m.Heading = sprite.W
		m.snapToExistingFrame()
		_ = m.Current()
	})
}

func TestShippedShipAssets(t *testing.T) {
	t.Run("happy: assets holds a readable json ship for every LM size", func(t *testing.T) {
		dir := FindAssetsDir()
		ships, err := ListShips(dir)
		if err != nil {
			t.Fatalf("ListShips(%q): %v", dir, err)
		}
		want := map[string]sprite.Size{
			"lm-1": sprite.Size1,
			"lm-2": sprite.Size2,
			"lm-3": sprite.Size3,
			"lm-4": sprite.Size4,
		}
		got := map[string]ShipFile{}
		for _, s := range ships {
			got[s.Name] = s
		}
		for name, sz := range want {
			s, ok := got[name]
			if !ok {
				t.Fatalf("assets missing %s.json (dir %q, have %#v)", name, dir, got)
			}
			a, err := LoadShip(s.Path)
			if err != nil {
				t.Fatalf("load %s: %v", name, err)
			}
			w, h := sz.Dim()
			for _, heading := range sprite.Headings {
				sp, ok := a.Frame(sz, heading)
				if !ok {
					t.Fatalf("%s missing heading %s", name, heading)
				}
				if sp.Width != w || sp.Height != h {
					t.Fatalf("%s %s is %dx%d, want %dx%d", name, heading, sp.Width, sp.Height, w, h)
				}
			}
		}
	})
	t.Run("unhappy: FindAssetsDir does not panic when nothing is nearby", func(t *testing.T) {
		dir := FindAssetsDir()
		if dir == "" {
			t.Fatal("FindAssetsDir must return a path even if the folder is missing")
		}
	})
}
