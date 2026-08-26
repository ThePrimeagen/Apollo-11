package editor

// Tests written FIRST. The editor is generalized: any folder of *.json
// sprite atlases is an assets folder it can list, load, and open — no
// project's file names are baked in. Missing or corrupt files must
// fail loudly; a missing folder must not panic. This repo's own lunar
// assets all live together in one assets/ folder, lm.json included.

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

func TestListAssets(t *testing.T) {
	t.Run("happy: lists every json atlas in the folder, sorted by name", func(t *testing.T) {
		dir := t.TempDir()
		writeMiniShip(t, dir, "bravo", 'B')
		writeMiniShip(t, dir, "alpha", 'A')
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(dir, "nested"), 0o755); err != nil {
			t.Fatal(err)
		}

		assets, err := ListAssets(dir)
		if err != nil {
			t.Fatalf("ListAssets: %v", err)
		}
		if len(assets) != 2 {
			t.Fatalf("want 2 json assets, got %d", len(assets))
		}
		if assets[0].Name != "alpha" || assets[1].Name != "bravo" {
			t.Fatalf("assets must be sorted by name, got %q then %q", assets[0].Name, assets[1].Name)
		}
		if assets[0].Path != filepath.Join(dir, "alpha.json") {
			t.Fatalf("path %q", assets[0].Path)
		}
	})
	t.Run("unhappy: a missing assets folder is an empty list, not a panic", func(t *testing.T) {
		assets, err := ListAssets(filepath.Join(t.TempDir(), "no-such-assets"))
		if err != nil {
			t.Fatalf("missing dir must not error, got %v", err)
		}
		if len(assets) != 0 {
			t.Fatalf("missing dir must yield no assets, got %d", len(assets))
		}
	})
}

func TestLoadAsset(t *testing.T) {
	t.Run("happy: LoadAsset reads a json atlas the editor can draw", func(t *testing.T) {
		path := writeMiniShip(t, t.TempDir(), "tiny", 'X')
		a, err := LoadAsset(path)
		if err != nil {
			t.Fatalf("LoadAsset: %v", err)
		}
		sp, ok := a.Frame(sprite.Size1, sprite.N)
		if !ok {
			t.Fatal("loaded asset must carry its size-1 north frame")
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
		if _, err := LoadAsset(path); err == nil {
			t.Fatal("corrupt asset json must fail")
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

func TestShippedAssets(t *testing.T) {
	t.Run("happy: every lunar atlas lives together in the assets folder", func(t *testing.T) {
		dir := FindAssetsDir()
		assets, err := ListAssets(dir)
		if err != nil {
			t.Fatalf("ListAssets(%q): %v", dir, err)
		}
		got := map[string]Asset{}
		for _, a := range assets {
			got[a.Name] = a
		}
		// lm.json is a lunar asset too: it moved out of
		// components/lander into the one shared folder.
		want := map[string]sprite.Size{
			"lm":   sprite.Size4,
			"lm-1": sprite.Size1,
			"lm-2": sprite.Size2,
			"lm-3": sprite.Size3,
			"lm-4": sprite.Size4,
		}
		for name, sz := range want {
			asset, ok := got[name]
			if !ok {
				t.Fatalf("assets missing %s.json (dir %q, have %#v)", name, dir, got)
			}
			a, err := LoadAsset(asset.Path)
			if err != nil {
				t.Fatalf("load %s: %v", name, err)
			}
			sp, ok := a.Frame(sz, sprite.N)
			if !ok {
				t.Fatalf("%s missing size %d heading N", name, sz)
			}
			w, h := sz.Dim()
			if sp.Width != w || sp.Height != h {
				t.Fatalf("%s size %d is %dx%d, want %dx%d", name, sz, sp.Width, sp.Height, w, h)
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
