package dust

// Tests written FIRST. The dust component is the landing kick-up: two
// mirrored swirl engines blowing out of a shared floor point, 15° above
// horizontal, with a dead-still gap of columns between the nozzles.
// PuffConfig is the JSON that tunes it — engine knobs, the mirrored
// geometry, and the gray ladder that decides which symbol a cell's
// concentration earns. The config lives with the component.

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

func TestPuffDefaults(t *testing.T) {
	t.Run("happy: the stock puff is a 15° up-looping kick with an 8-column gap", func(t *testing.T) {
		c := DefaultPuff()
		if err := c.Validate(); err != nil {
			t.Fatalf("the stock puff must validate: %v", err)
		}
		if c.AngleDeg != 15 {
			t.Fatalf("stock angle %v, want 15° above horizontal", c.AngleDeg)
		}
		if c.Gap != 8 {
			t.Fatalf("stock gap %v, want 8 still columns", c.Gap)
		}
		if !c.LoopUp {
			t.Fatal("the stock puff must loop upward")
		}
		if c.QuarterAt < 2 || c.HalfAt <= c.QuarterAt {
			t.Fatalf("stock ladder %d/%d must climb: 2 <= quarter < half", c.QuarterAt, c.HalfAt)
		}
		if c.BrailleFG >= c.QuarterFG || c.QuarterFG >= c.HalfFG {
			t.Fatalf("grays %d/%d/%d must brighten with concentration", c.BrailleFG, c.QuarterFG, c.HalfFG)
		}
	})
	t.Run("unhappy: bad geometry, ladders, grays, and engine knobs are named", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			mod  func(*PuffConfig)
			want error
		}{
			{"angle below horizontal", func(c *PuffConfig) { c.AngleDeg = -5 }, ErrAngle},
			{"angle at vertical", func(c *PuffConfig) { c.AngleDeg = 90 }, ErrAngle},
			{"negative gap", func(c *PuffConfig) { c.Gap = -1 }, ErrGap},
			{"quarter too low", func(c *PuffConfig) { c.QuarterAt = 1 }, ErrLadder},
			{"half not past quarter", func(c *PuffConfig) { c.HalfAt = c.QuarterAt }, ErrLadder},
			{"gray off the gray ramp", func(c *PuffConfig) { c.BrailleFG = 196 }, ErrGray},
			{"gray past the ramp", func(c *PuffConfig) { c.HalfFG = 256 }, ErrGray},
			{"negative count", func(c *PuffConfig) { c.Count = -1 }, particle.ErrCount},
			{"reversed speed range", func(c *PuffConfig) { c.MinSpeed, c.MaxSpeed = 9, 2 }, particle.ErrSpeed},
			{"reversed life range", func(c *PuffConfig) { c.MinLife, c.MaxLife = 3, 1 }, particle.ErrLife},
		} {
			c := DefaultPuff()
			tc.mod(&c)
			if err := c.Validate(); !errors.Is(err, tc.want) {
				t.Fatalf("%s: got %v, want %v", tc.name, err, tc.want)
			}
		}
	})
}

func TestPuffEngines(t *testing.T) {
	t.Run("happy: two mirrored swirl worlds climb 15° out of a shared floor, a gap apart", func(t *testing.T) {
		c := DefaultPuff()
		left, right := c.Engines(80, 40)
		for side, cfg := range map[string]particle.Config{"left": left, "right": right} {
			if err := cfg.Validate(); err != nil {
				t.Fatalf("%s engine must validate: %v", side, err)
			}
			if cfg.Mode != particle.ModeSwirl || !cfg.SwirlUp {
				t.Fatalf("%s engine must swirl upward, mode %v up %v", side, cfg.Mode, cfg.SwirlUp)
			}
			if cfg.Direction.Y >= 0 {
				t.Fatalf("%s engine must climb (Y is down), direction %+v", side, cfg.Direction)
			}
		}
		rad := 15 * math.Pi / 180
		if math.Abs(left.Direction.X+math.Cos(rad)) > 1e-9 || math.Abs(left.Direction.Y+math.Sin(rad)) > 1e-9 {
			t.Fatalf("left direction %+v, want 15° up of leftward", left.Direction)
		}
		if math.Abs(right.Direction.X-math.Cos(rad)) > 1e-9 || math.Abs(right.Direction.Y+math.Sin(rad)) > 1e-9 {
			t.Fatalf("right direction %+v, want 15° up of rightward", right.Direction)
		}
		if got := right.Origin.X - left.Origin.X; math.Abs(got-c.Gap) > 1e-9 {
			t.Fatalf("nozzles sit %v apart, want the gap %v", got, c.Gap)
		}
		if mid := (left.Origin.X + right.Origin.X) / 2; math.Abs(mid-40) > 1e-9 {
			t.Fatalf("the gap must center on the stage, midpoint %v want 40", mid)
		}
		if left.Origin.Y != right.Origin.Y || left.Origin.Y < 28 || left.Origin.Y > 40 {
			t.Fatalf("nozzles must share a floor near the bottom, y %v and %v", left.Origin.Y, right.Origin.Y)
		}
		if left.Count != c.Count || right.Period != c.Period || left.Nozzle != c.Nozzle {
			t.Fatal("the engine knobs must carry the puff's values")
		}
	})
	t.Run("happy: SideSwirl(false) worlds loop downward when the puff says so", func(t *testing.T) {
		c := DefaultPuff()
		c.LoopUp = false
		left, right := c.Engines(80, 40)
		if left.SwirlUp || right.SwirlUp {
			t.Fatal("LoopUp=false must deal downward loops to both engines")
		}
	})
	t.Run("unhappy: a stage smaller than the gap still yields valid worlds", func(t *testing.T) {
		c := DefaultPuff()
		left, right := c.Engines(4, 4)
		if err := left.Validate(); err != nil {
			t.Fatalf("tiny left engine must clamp its origin inside: %v", err)
		}
		if err := right.Validate(); err != nil {
			t.Fatalf("tiny right engine must clamp its origin inside: %v", err)
		}
		if left.Origin.X < 0 || right.Origin.X > 4 {
			t.Fatalf("origins %v and %v must clamp into the box", left.Origin, right.Origin)
		}
	})
}

func TestPuffFile(t *testing.T) {
	t.Run("happy: Save then LoadPuff round-trips, and UsePuff makes it the active puff", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		path := filepath.Join(t.TempDir(), "dust.json")
		c := DefaultPuff()
		c.Count = 7
		c.Gap = 12
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := LoadPuff(path)
		if err != nil {
			t.Fatalf("LoadPuff: %v", err)
		}
		if got != c {
			t.Fatalf("round-trip %+v, want %+v", got, c)
		}
		if err := UsePuff(got); err != nil {
			t.Fatalf("UsePuff: %v", err)
		}
		if ActivePuff() != got {
			t.Fatalf("active puff %+v, want the used one", ActivePuff())
		}
		ResetPuff()
		if ActivePuff() != DefaultPuff() {
			t.Fatal("ResetPuff must restore the stock puff")
		}
	})
	t.Run("unhappy: missing, broken, and out-of-range files error and never go active", func(t *testing.T) {
		t.Cleanup(ResetPuff)
		if _, err := LoadPuff(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Fatal("a missing file must error")
		}
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadPuff(bad); err == nil {
			t.Fatal("broken JSON must error")
		}
		out := filepath.Join(t.TempDir(), "out.json")
		if err := os.WriteFile(out, []byte(`{"count":-4}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadPuff(out); err == nil {
			t.Fatal("an out-of-range file must error")
		}
		before := ActivePuff()
		invalid := DefaultPuff()
		invalid.Gap = -3
		if err := UsePuff(invalid); err == nil {
			t.Fatal("UsePuff must reject an invalid puff")
		}
		if ActivePuff() != before {
			t.Fatal("a rejected puff must not go active")
		}
		bad2 := DefaultPuff()
		bad2.AngleDeg = 120
		if err := bad2.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse an invalid puff")
		}
	})
}
