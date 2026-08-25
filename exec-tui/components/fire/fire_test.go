package fire

// Tests written FIRST. Fire is a thin color layer on particle: 45° trail,
// 100 particles, at most four terminal rows. Occupancy becomes yellow
// (dense core), orange (mid), red (tips). The package does not move the
// particle engine's box; it only paints what Occupancy reports.

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestDefault(t *testing.T) {
	t.Run("happy: the trail is 45 degrees, 100 particles, four rows", func(t *testing.T) {
		f := New(1)
		if err := f.Eng.Validate(); err != nil {
			t.Fatalf("default config: %v", err)
		}
		if f.Eng.Cfg.Count != 5 {
			t.Fatalf("count %d, want 5 per spawn", f.Eng.Cfg.Count)
		}
		d := f.Eng.Cfg.Direction
		if math.Abs(math.Abs(d.X)-math.Abs(d.Y)) > 1e-9 {
			t.Fatalf("direction %+v is not 45°", d)
		}
		if Rows != 4 {
			t.Fatalf("Rows %d, want 4", Rows)
		}
		sp := f.Sprite()
		if sp.Height != 4 || sp.Width != Cols {
			t.Fatalf("sprite %dx%d, want %dx4", sp.Width, sp.Height, Cols)
		}
	})
	t.Run("unhappy: New does not paint until Update has run", func(t *testing.T) {
		f := New(1)
		if !blank(f.Sprite()) {
			t.Fatal("a flame that has never ticked must be empty")
		}
	})
}

func TestUpdate(t *testing.T) {
	t.Run("happy: 10ms of 1ms spawns drops a particle each millisecond", func(t *testing.T) {
		f := New(2)
		f.Update(0.01)
		n := len(f.Eng.Particles)
		if n < 50 || n > 55 {
			t.Fatalf("live %d, want ~50 (5 per ms)", n)
		}
	})
	t.Run("happy: particles travel down-right along the diagonal", func(t *testing.T) {
		f := New(3)
		f.Update(0.01)
		ox, oy := avgPos(f.Eng.Particles)
		f.Update(0.12)
		ax, ay := avgPos(f.Eng.Particles)
		if ax <= ox || ay <= oy {
			t.Fatalf("45° down-right should increase x and y, got (%.2f,%.2f) → (%.2f,%.2f)", ox, oy, ax, ay)
		}
		if math.Abs((ax-ox)-(ay-oy)) > 1.5 {
			t.Fatalf("diagonal drift should stay near 45°, Δx=%.2f Δy=%.2f", ax-ox, ay-oy)
		}
	})
	t.Run("happy: a running trail stays inside four rows and stretches as a trail", func(t *testing.T) {
		f := New(4)
		warm(f, 0.8)
		occ := f.Eng.Occupancy()
		if len(occ) == 0 {
			t.Fatal("expected a live trail")
		}
		minC, maxC := 1<<30, -1
		rows := map[int]bool{}
		for cell := range occ {
			if cell.Row < 0 || cell.Row >= Rows {
				t.Fatalf("occupied row %d is outside 0..%d", cell.Row, Rows-1)
			}
			rows[cell.Row] = true
			if cell.Col < minC {
				minC = cell.Col
			}
			if cell.Col > maxC {
				maxC = cell.Col
			}
		}
		if maxC-minC < 2 {
			t.Fatalf("a trail must span several columns, span=%d", maxC-minC)
		}
		if len(rows) < 2 {
			t.Fatalf("a 45° trail must occupy more than one row, rows=%d", len(rows))
		}
	})
	t.Run("unhappy: dt<=0 does not emit or move", func(t *testing.T) {
		f := New(5)
		f.Update(0)
		f.Update(-1)
		if len(f.Eng.Particles) != 0 {
			t.Fatal("dt<=0 must be a no-op")
		}
	})
}

func TestColor(t *testing.T) {
	t.Run("happy: Color is Style, so the ladder still holds", func(t *testing.T) {
		if Color(250).Ch != '█' {
			t.Fatalf("high heat should be solid, got %+v", Color(250))
		}
		if Color(10).Ch != '⠒' {
			t.Fatalf("H=10 should be two dots, got %+v", Color(10))
		}
	})
	t.Run("happy: a warmed trail has a bright core", func(t *testing.T) {
		f := New(6)
		warm(f, 0.6)
		var yellows int
		sp := f.Sprite()
		for row := 0; row < sp.Height; row++ {
			for col := 0; col < sp.Width; col++ {
				if sp.At(row, col).Ch == '█' {
					yellows++
				}
			}
		}
		if yellows == 0 {
			t.Fatal("expected a yellow core")
		}
	})
	t.Run("unhappy: zero heat is empty", func(t *testing.T) {
		if !Color(0).Transparent() {
			t.Fatalf("count 0 must be empty, got %+v", Color(0))
		}
	})
}

func TestSprite(t *testing.T) {
	t.Run("happy: Render is a fixed 4-row canvas", func(t *testing.T) {
		f := New(7)
		warm(f, 0.4)
		out := f.Render()
		if out == "" {
			t.Fatal("a live flame must render")
		}
		rows := 1
		for _, r := range out {
			if r == '\n' {
				rows++
			}
		}
		if rows != 4 {
			t.Fatalf("render rows %d, want 4", rows)
		}
	})
	t.Run("unhappy: an empty flame renders four blank rows, not a cropped box", func(t *testing.T) {
		sp := New(8).Sprite()
		if sp.Width != Cols || sp.Height != 4 {
			t.Fatalf("empty sprite %dx%d, want fixed %dx4", sp.Width, sp.Height, Cols)
		}
		if !blank(sp) {
			t.Fatal("empty flame must be blank")
		}
	})
}

func warm(f *Flame, seconds float64) {
	const dt = 1.0 / 20
	for t := 0.0; t < seconds; t += dt {
		f.Update(dt)
	}
}

func avgPos(ps []particle.Particle) (x, y float64) {
	if len(ps) == 0 {
		return 0, 0
	}
	for _, p := range ps {
		x += p.Pos.X
		y += p.Pos.Y
	}
	n := float64(len(ps))
	return x / n, y / n
}

func blank(sp sprite.Sprite) bool {
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if !sp.At(r, c).Transparent() {
				return false
			}
		}
	}
	return true
}
