package fire

import (
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderPNG(t *testing.T) {
	t.Run("happy: a lit flame paints a fixed-size bitmap", func(t *testing.T) {
		f := New(9)
		warm(f, 0.5)
		img, err := RenderPNG(f.View(), 12)
		if err != nil {
			t.Fatal(err)
		}
		b := img.Bounds()
		if b.Dx() != ViewCols*12 || b.Dy() != ViewRows*24 {
			t.Fatalf("png %dx%d, want %dx%d", b.Dx(), b.Dy(), ViewCols*12, ViewRows*24)
		}
		if !hasFlame(img) {
			t.Fatal("the flame must actually paint on the void")
		}
	})
	t.Run("unhappy: an empty view is a void rectangle, not an error", func(t *testing.T) {
		img, err := RenderPNG(New(10).View(), 10)
		if err != nil {
			t.Fatal(err)
		}
		if hasFlame(img) {
			t.Fatal("an empty flame must not paint fire pixels")
		}
	})
}

func TestWriteTape(t *testing.T) {
	t.Run("happy: WriteTape writes n same-size frames", func(t *testing.T) {
		dir := t.TempDir()
		paths, err := WriteTape(dir, New(11), 3, 8)
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) != 3 {
			t.Fatalf("paths %d, want 3", len(paths))
		}
		var w, h int
		for i, p := range paths {
			st, err := os.Stat(p)
			if err != nil || st.Size() == 0 {
				t.Fatalf("frame %d missing: %v", i, err)
			}
			img := mustPNG(t, p)
			b := img.Bounds()
			if i == 0 {
				w, h = b.Dx(), b.Dy()
			} else if b.Dx() != w || b.Dy() != h {
				t.Fatalf("frame %d is %dx%d, want fixed %dx%d", i, b.Dx(), b.Dy(), w, h)
			}
		}
	})
	t.Run("unhappy: a zero frame count is an error", func(t *testing.T) {
		if _, err := WriteTape(t.TempDir(), New(12), 0, 8); err == nil {
			t.Fatal("n<=0 must fail")
		}
	})
	t.Run("unhappy: a missing parent directory is an error", func(t *testing.T) {
		f := New(13)
		warm(f, 0.2)
		if err := WritePNG("/no/such/dir/flame.png", f.View(), 8); err == nil {
			t.Fatal("unwritable path must fail")
		}
	})
}

func mustPNG(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func hasFlame(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bb, a := img.At(x, y).RGBA()
			if a > 0 && r > 0x8000 && g > 0x2000 && bb < 0x6000 {
				return true
			}
		}
	}
	return false
}

func TestWritePNG(t *testing.T) {
	t.Run("happy: WritePNG creates a readable file", func(t *testing.T) {
		f := New(14)
		warm(f, 0.4)
		path := filepath.Join(t.TempDir(), "flame.png")
		if err := WritePNG(path, f.View(), 10); err != nil {
			t.Fatal(err)
		}
		st, err := os.Stat(path)
		if err != nil || st.Size() == 0 {
			t.Fatalf("expected a real png, stat=%v err=%v", st, err)
		}
	})
}
