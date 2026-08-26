package editor

// Open is the editor's front door: point it at one .json atlas, or at
// a whole folder of them and it loads them all. A brand-new .json path
// is seeded with a blank canvas — the editor knows no project's art —
// and an empty folder seeds untitled.json, so there is always
// something on disk to edit and save. Corrupt files and missing
// folders are errors, never silent fallbacks. Tests written before the
// implementation.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func countPainted(sp sprite.Sprite) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if !sp.At(r, c).Transparent() {
				n++
			}
		}
	}
	return n
}

func TestOpen(t *testing.T) {
	t.Run("happy: an existing atlas file loads into the editor", func(t *testing.T) {
		dir := t.TempDir()
		path := writeMiniShip(t, dir, "craft", 'C')
		m, err := Open(path)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if m.Path != path {
			t.Fatalf("the editor must remember its path: %q, want %q", m.Path, path)
		}
		if _, ok := m.Atlas.Frame(sprite.Size1, sprite.N); !ok {
			t.Fatal("the loaded atlas must carry the size-1 north frame")
		}
		if m.AssetsDir != dir {
			t.Fatalf("a file open must adopt its folder as the assets dir, got %q want %q", m.AssetsDir, dir)
		}
	})
	t.Run("happy: a missing .json path seeds a blank canvas and writes it", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "fresh.json")
		m, err := Open(path)
		if err != nil {
			t.Fatalf("Open on a fresh path must seed, not fail: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("the seeded atlas must land on disk: %v", err)
		}
		sp, ok := m.Atlas.Frame(sprite.Size4, sprite.N)
		if !ok {
			t.Fatal("the blank seed must carry an editable size-4 north frame")
		}
		if n := countPainted(sp); n != 0 {
			t.Fatalf("the seed must be a blank canvas, found %d painted cells", n)
		}
		if _, ok := m.Atlas.Frame(sprite.Size1, sprite.N); ok {
			t.Fatal("the seed must not smuggle in any project's art")
		}
	})
	t.Run("unhappy: a corrupt atlas file is an error, not a silent reseed", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "broken.json")
		if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(path); err == nil {
			t.Fatal("a corrupt atlas file must surface an error")
		}
	})
	t.Run("unhappy: an unwritable seed path is an error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "no", "such", "dir", "lm.json")
		if _, err := Open(path); err == nil {
			t.Fatal("seeding into a missing directory must error")
		}
	})
	t.Run("unhappy: a missing path that is not .json is an error, not a scaffold", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "no-such-folder")
		if _, err := Open(path); err == nil {
			t.Fatal("a missing non-json path must surface an error")
		}
		if _, err := os.Stat(path); err == nil {
			t.Fatal("a failed open must not leave anything on disk")
		}
	})
}

func TestOpenDir(t *testing.T) {
	t.Run("happy: a folder opens its first atlas and warms every other one", func(t *testing.T) {
		dir := t.TempDir()
		bravoPath := writeMiniShip(t, dir, "bravo", 'Ψ')
		alphaPath := writeMiniShip(t, dir, "alpha", 'Ω')
		m, err := Open(dir)
		if err != nil {
			t.Fatalf("Open(dir): %v", err)
		}
		if m.Path != alphaPath {
			t.Fatalf("the folder must open its first atlas by name, got %q want %q", m.Path, alphaPath)
		}
		if got := m.Current().At(0, 0).Ch; got != 'Ω' {
			t.Fatalf("the canvas must show the opened atlas, got %q want Ω", string(got))
		}
		if m.AssetsDir != dir {
			t.Fatalf("the folder is the assets dir, got %q want %q", m.AssetsDir, dir)
		}
		if len(m.Files) != 2 {
			t.Fatalf("the editor must list every atlas it loaded, got %d want 2", len(m.Files))
		}
		for _, path := range []string{alphaPath, bravoPath} {
			if m.atlases[path] == nil {
				t.Fatalf("every atlas in the folder must be loaded up front, %q is cold", path)
			}
		}
	})
	t.Run("happy: an empty folder seeds untitled.json so there is something to edit", func(t *testing.T) {
		dir := t.TempDir()
		m, err := Open(dir)
		if err != nil {
			t.Fatalf("Open(empty dir): %v", err)
		}
		seed := filepath.Join(dir, "untitled.json")
		if m.Path != seed {
			t.Fatalf("empty folder must open a seeded untitled.json, got %q", m.Path)
		}
		if _, err := os.Stat(seed); err != nil {
			t.Fatalf("the seed must land on disk: %v", err)
		}
	})
	t.Run("unhappy: a corrupt atlas in the folder fails loudly, naming the file", func(t *testing.T) {
		dir := t.TempDir()
		writeMiniShip(t, dir, "good", 'G')
		if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Open(dir)
		if err == nil {
			t.Fatal("a corrupt atlas in the folder must fail the open")
		}
		if !strings.Contains(err.Error(), "bad.json") {
			t.Fatalf("the error must name the corrupt file, got %v", err)
		}
	})
	t.Run("unhappy: a missing folder is an error, not a scaffold", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "missing")
		if _, err := OpenDir(dir); err == nil {
			t.Fatal("a missing folder must surface an error")
		}
		if _, err := os.Stat(dir); err == nil {
			t.Fatal("a failed open must not create the folder")
		}
	})
}
