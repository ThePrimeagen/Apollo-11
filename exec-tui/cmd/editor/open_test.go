package editor

// Open is the editor's front door: it reads the atlas at path, or — on
// the very first run — seeds the file with the hand-drawn default art
// so there is always something to edit and save. Tests written before
// the implementation.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestOpen(t *testing.T) {
	t.Run("happy: an existing atlas file loads into the editor", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "lm.json")
		if err := lander.DefaultAtlas().WriteFile(path); err != nil {
			t.Fatal(err)
		}
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
	})
	t.Run("happy: a missing file seeds the default art and writes it", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "fresh.json")
		m, err := Open(path)
		if err != nil {
			t.Fatalf("Open on a fresh path must seed, not fail: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("the seeded atlas must land on disk: %v", err)
		}
		if _, ok := m.Atlas.Frame(sprite.Size4, sprite.W); !ok {
			t.Fatal("the seeded atlas must be the full default art")
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
}
