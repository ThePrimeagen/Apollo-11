package astro

// Tests written FIRST. The generator is the authoring pipeline: it
// compiles the pixel grids into the editable atlas JSON the editor
// opens, and it can dump magnified PNGs of every pose so the art can
// be reviewed outside a terminal. Both writers fail loudly instead of
// leaving half-written art behind.

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestWriteAtlasFile(t *testing.T) {
	t.Run("happy: the written JSON reloads with every pose intact", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "astronaut.json")
		if err := WriteAtlasFile(path); err != nil {
			t.Fatalf("WriteAtlasFile: %v", err)
		}
		a, err := sprite.LoadFile(path)
		if err != nil {
			t.Fatalf("the written file does not load: %v", err)
		}
		built, err := BuildAtlas()
		if err != nil {
			t.Fatalf("BuildAtlas: %v", err)
		}
		for _, pose := range Poses {
			got, ok := a.Frame(Size, pose)
			if !ok {
				t.Fatalf("written atlas lost pose %q", pose)
			}
			want, _ := built.Frame(Size, pose)
			if fingerprint(got) != fingerprint(want) {
				t.Fatalf("pose %q on disk differs from the built atlas", pose)
			}
		}
	})
	t.Run("unhappy: an unwritable path is an error", func(t *testing.T) {
		if err := WriteAtlasFile(filepath.Join(t.TempDir(), "no", "such", "dir", "a.json")); err == nil {
			t.Fatal("writing into a missing directory must error")
		}
	})
}

func TestWritePNGs(t *testing.T) {
	t.Run("happy: one magnified PNG per pose plus the run strip and sheet", func(t *testing.T) {
		dir := t.TempDir()
		const scale = 20
		if err := WritePNGs(dir, scale); err != nil {
			t.Fatalf("WritePNGs: %v", err)
		}
		for _, pose := range Poses {
			p := filepath.Join(dir, "astronaut-"+string(pose)+".png")
			f, err := os.Open(p)
			if err != nil {
				t.Fatalf("pose %q has no PNG: %v", pose, err)
			}
			img, err := png.Decode(f)
			f.Close()
			if err != nil {
				t.Fatalf("pose %q PNG does not decode: %v", pose, err)
			}
			b := img.Bounds()
			if b.Dx() != PxW*scale || b.Dy() != PxH*scale {
				t.Fatalf("pose %q PNG is %dx%d, want %dx%d", pose, b.Dx(), b.Dy(), PxW*scale, PxH*scale)
			}
		}
		for _, name := range []string{"astronaut-run-strip.png", "astronaut-sheet.png"} {
			f, err := os.Open(filepath.Join(dir, name))
			if err != nil {
				t.Fatalf("%s missing: %v", name, err)
			}
			if _, err := png.Decode(f); err != nil {
				t.Fatalf("%s does not decode: %v", name, err)
			}
			f.Close()
		}
	})
	t.Run("unhappy: an unwritable directory is an error", func(t *testing.T) {
		if err := WritePNGs(filepath.Join(t.TempDir(), "missing", "dir"), 8); err == nil {
			t.Fatal("a missing directory must error")
		}
		if err := WritePNGs(t.TempDir(), 0); err == nil {
			t.Fatal("a zero scale draws nothing — must error")
		}
	})
}
