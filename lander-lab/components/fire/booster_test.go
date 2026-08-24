package fire

import (
	"math"
	"testing"

	"github.com/theprimeagen/apollo-11/lander-lab/particle"
)

func TestBooster(t *testing.T) {
	t.Run("happy: left-to-right, four cells wide, two cells tall, no fan", func(t *testing.T) {
		f := Booster(1)
		if err := f.Eng.Validate(); err != nil {
			t.Fatalf("booster config: %v", err)
		}
		cfg := f.Eng.Cfg
		if cfg.Count != Particles {
			t.Fatalf("count %d, want %d", cfg.Count, Particles)
		}
		if cfg.Spread != 0 {
			t.Fatalf("spread %v, want 0 — particles stack, they do not fan", cfg.Spread)
		}
		if math.Abs(cfg.Width-4) > 0.05 {
			t.Fatalf("width %.2f units, want 4", cfg.Width)
		}
		// two terminal rows = four units of height
		if math.Abs(cfg.Height-4) > 0.05 {
			t.Fatalf("height %.2f units, want 4 (two cells)", cfg.Height)
		}
		d := cfg.Direction
		if math.Abs(d.X-1) > 1e-9 || math.Abs(d.Y) > 1e-9 {
			t.Fatalf("direction %+v, want (1, 0) left-to-right", d)
		}
		sp := f.Sprite()
		if sp.Width != 4 || sp.Height != 2 {
			t.Fatalf("sprite %dx%d, want 4x2", sp.Width, sp.Height)
		}
	})
	t.Run("happy: a running booster fills both rows and travels right", func(t *testing.T) {
		f := Booster(2)
		f.Update(0.01)
		for i, p := range f.Eng.Particles {
			if math.Abs(p.Vel.Y) > 1e-9 {
				t.Fatalf("particle %d fanned (VY=%v); travel must stay left-to-right", i, p.Vel.Y)
			}
		}
		ox, _ := avgPos(f.Eng.Particles)
		f.Update(0.1)
		ax, _ := avgPos(f.Eng.Particles)
		if ax <= ox {
			t.Fatalf("left-to-right should increase x, %.2f → %.2f", ox, ax)
		}
		warm(f, 0.4)
		rows := map[int]bool{}
		for cell := range f.Eng.Occupancy() {
			rows[cell.Row] = true
			if cell.Row < 0 || cell.Row > 1 {
				t.Fatalf("two-row flame left rows 0..1 at row %d", cell.Row)
			}
			if cell.Col < 0 || cell.Col >= 4 {
				t.Fatalf("four-unit flame left the box at col %d", cell.Col)
			}
		}
		if len(rows) != 2 {
			t.Fatalf("booster must occupy both rows, got %d", len(rows))
		}
	})
	t.Run("unhappy: one stray particle on the booster does not paint", func(t *testing.T) {
		f := Booster(3)
		f.Eng.Particles = []particle.Particle{{
			Pos:  particle.Vec2{X: 2.2, Y: 2.0},
			Life: 1,
		}}
		if !blank(f.Sprite()) {
			t.Fatal("occupancy 1 must not light a cell")
		}
	})
}
