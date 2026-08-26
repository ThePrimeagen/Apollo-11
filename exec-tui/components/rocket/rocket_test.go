package rocket

// Tests written FIRST: the rocket view is the size-4 (26×10) LM from the
// atlas standing on the live booster plume aimed straight down. The art's
// baked-in tilde plume is stripped — the particle fire IS the plume — and
// the flame burns in its own window below the engine bell, never over the
// hull.

import (
	"image"
	_ "image/png"
	"math"
	"os"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestNew(t *testing.T) {
	t.Run("happy: size-4 body with a validated flame aimed straight down", func(t *testing.T) {
		r := New(1)
		if r.Body.Width != 26 || r.Body.Height != 10 {
			t.Fatalf("body %dx%d, want the 26x10 size-4 frame", r.Body.Width, r.Body.Height)
		}
		if err := r.Flame.Eng.Validate(); err != nil {
			t.Fatalf("rocket flame config: %v", err)
		}
		d := r.Flame.Eng.Cfg.Direction
		if math.Abs(d.X) > 1e-9 || math.Abs(d.Y-1) > 1e-9 {
			t.Fatalf("direction %+v, want (0, 1) — fire on the bottom", d)
		}
	})
	t.Run("happy: every hull cell of the atlas frame survives", func(t *testing.T) {
		r := New(2)
		want := lander.DefaultAtlas().MustFrame(sprite.Size4, sprite.N)
		for row := 0; row < want.Height; row++ {
			for col := 0; col < want.Width; col++ {
				w := want.At(row, col)
				got := r.Body.At(row, col)
				if w.Ch == '~' || w.Ch == '≈' {
					continue // the baked-in plume, checked separately
				}
				if got != w {
					t.Fatalf("hull cell (%d,%d) changed: %+v -> %+v", row, col, w, got)
				}
			}
		}
	})
	t.Run("happy: the baked-in tilde plume is stripped from body and view", func(t *testing.T) {
		r := New(3)
		v := r.View()
		for row := 0; row < v.Height; row++ {
			for col := 0; col < v.Width; col++ {
				if ch := v.At(row, col).Ch; ch == '~' || ch == '≈' {
					t.Fatalf("static plume glyph %q left at (%d,%d); the live fire is the plume", ch, row, col)
				}
			}
		}
	})
	t.Run("unhappy: a rocket that has never ticked shows no fire", func(t *testing.T) {
		r := New(4)
		v := r.View()
		fw := r.Flame.Sprite().Width
		for row := FlameRow; row < Rows; row++ {
			for col := FlameCol; col < FlameCol+fw; col++ {
				if !v.At(row, col).Transparent() {
					t.Fatalf("flame window lit at (%d,%d) before any Update", row, col)
				}
			}
		}
	})
}

func TestView(t *testing.T) {
	t.Run("happy: a warmed rocket burns below the bell with a bright core", func(t *testing.T) {
		r := New(5)
		warm(r, 1.0)
		v := r.View()
		if v.Width != Cols || v.Height != Rows {
			t.Fatalf("view %dx%d, want fixed %dx%d", v.Width, v.Height, Cols, Rows)
		}
		lit, bright, nozzle := 0, 0, 0
		for row := FlameRow; row < Rows; row++ {
			for col := 0; col < Cols; col++ {
				ch := v.At(row, col).Ch
				if !flameGlyph(ch) {
					continue
				}
				lit++
				if ch == '█' || ch == '▓' {
					bright++
				}
				if row <= FlameRow+1 && col >= 11 && col <= 15 {
					nozzle++
				}
			}
		}
		if lit < 8 {
			t.Fatalf("a warmed booster must fill the window, lit=%d", lit)
		}
		if bright == 0 {
			t.Fatal("the core at the nozzle must burn bright")
		}
		if nozzle == 0 {
			t.Fatal("fire must pour from directly under the engine bell")
		}
	})
	t.Run("happy: the fire never overwrites the hull", func(t *testing.T) {
		r := New(6)
		warm(r, 1.0)
		v := r.View()
		for row := 0; row < r.Body.Height; row++ {
			for col := 0; col < r.Body.Width; col++ {
				b := r.Body.At(row, col)
				if b.Transparent() {
					continue
				}
				if got := v.At(row, col); got != b {
					t.Fatalf("hull cell (%d,%d) burned: %+v -> %+v", row, col, b, got)
				}
			}
		}
		bell := []rune("▜██████▛")
		for i, want := range bell {
			if got := v.At(7, 9+i).Ch; got != want {
				t.Fatalf("bell glyph %d is %q, want %q", i, got, want)
			}
		}
		for _, col := range []int{0, 3, 22, 25} {
			if got := v.At(8, col).Ch; got != '▁' {
				t.Fatalf("footpad at col %d is %q, want ▁", col, got)
			}
		}
	})
	t.Run("unhappy: a nil rocket views and renders a blank fixed canvas", func(t *testing.T) {
		var r *Rocket
		r.Update(1) // must not panic
		v := r.View()
		if v.Width != Cols || v.Height != Rows {
			t.Fatalf("nil view %dx%d, want %dx%d", v.Width, v.Height, Cols, Rows)
		}
		for row := 0; row < v.Height; row++ {
			for col := 0; col < v.Width; col++ {
				if !v.At(row, col).Transparent() {
					t.Fatalf("nil rocket lit (%d,%d)", row, col)
				}
			}
		}
		out := r.Render()
		rows := 1
		for _, ch := range out {
			if ch == '\n' {
				rows++
			}
		}
		if rows != Rows {
			t.Fatalf("nil render rows %d, want %d", rows, Rows)
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Run("unhappy: dt<=0 emits and moves nothing", func(t *testing.T) {
		r := New(7)
		r.Update(0)
		r.Update(-1)
		if n := len(r.Flame.Eng.Particles); n != 0 {
			t.Fatalf("dt<=0 must be a no-op, got %d particles", n)
		}
	})
}

func TestWriteTape(t *testing.T) {
	t.Run("happy: WriteTape writes n same-size frames of the rocket", func(t *testing.T) {
		dir := t.TempDir()
		paths, err := WriteTape(dir, New(8), 3, 8)
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) != 3 {
			t.Fatalf("paths %d, want 3", len(paths))
		}
		for i, p := range paths {
			st, err := os.Stat(p)
			if err != nil || st.Size() == 0 {
				t.Fatalf("frame %d missing: %v", i, err)
			}
			b := mustPNG(t, p).Bounds()
			if b.Dx() != Cols*8 || b.Dy() != Rows*16 {
				t.Fatalf("frame %d is %dx%d, want %dx%d", i, b.Dx(), b.Dy(), Cols*8, Rows*16)
			}
		}
	})
	t.Run("unhappy: a zero frame count is an error", func(t *testing.T) {
		if _, err := WriteTape(t.TempDir(), New(9), 0, 8); err == nil {
			t.Fatal("n<=0 must fail")
		}
	})
	t.Run("unhappy: a nil rocket is an error", func(t *testing.T) {
		if _, err := WriteTape(t.TempDir(), nil, 3, 8); err == nil {
			t.Fatal("nil rocket must fail")
		}
	})
}

func warm(r *Rocket, seconds float64) {
	const dt = 1.0 / 20
	for t := 0.0; t < seconds; t += dt {
		r.Update(dt)
	}
}

// flameGlyph is the heat ladder's glyph set. Below the hull rows the only
// lander glyph is the ▁ footpad, which is not in the set.
func flameGlyph(ch rune) bool {
	switch ch {
	case '⠁', '⠒', '⠶', '░', '▒', '▄', '▓', '█':
		return true
	}
	return false
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
