package gunfire

// Tests written FIRST. BlastConfig is the JSON that tunes the shotgun
// blast: where the muzzle sits and where it aims, the smoke fuse and
// its climb, the flash brightness ladder, and one engine-knob Layer
// each for the flash, the seven pellets, the sparks, and the smoke.
// UseBlast puts a config in effect for every blast reading it, the
// same way the dust puff works, so the tuner and the demo stay on the
// same values.

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

func TestDefaultBlast(t *testing.T) {
	t.Run("happy: the stock blast validates and fires the Doom seven", func(t *testing.T) {
		c := DefaultBlast()
		if err := c.Validate(); err != nil {
			t.Fatalf("the stock blast must validate: %v", err)
		}
		if c.Pellets.Count != 7 {
			t.Fatalf("the shotgun fires 7 pellets, not %d — that is the Doom number", c.Pellets.Count)
		}
	})
	t.Run("happy: the stock blast is the active blast at boot", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		if ActiveBlast() != DefaultBlast() {
			t.Fatalf("active blast %+v, want the stock %+v", ActiveBlast(), DefaultBlast())
		}
	})
	t.Run("unhappy: every layer is a one-shot — no period anywhere", func(t *testing.T) {
		fl, pe, sp, sm := DefaultBlast().Engines(120, 60)
		for name, cfg := range map[string]particle.Config{
			"flash": fl, "pellets": pe, "sparks": sp, "smoke": sm,
		} {
			if cfg.Period != 0 {
				t.Fatalf("%s layer auto-emits every %vs — a gunshot is a trigger, not a clock", name, cfg.Period)
			}
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("happy: extreme but legal settings pass", func(t *testing.T) {
		c := DefaultBlast()
		c.AngleDeg = -80
		c.MuzzleX, c.MuzzleY = 0, 1
		c.SmokeDelay = 0
		c.SmokeRiseDeg = 89
		c.EdgeAt, c.MidAt, c.CoreAt = 1, 2, 3
		if err := c.Validate(); err != nil {
			t.Fatalf("legal extremes must pass: %v", err)
		}
	})
	t.Run("unhappy: a muzzle outside the stage is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.MuzzleX = -0.1
		if err := c.Validate(); !errors.Is(err, ErrMuzzle) {
			t.Fatalf("got %v, want ErrMuzzle", err)
		}
		c = DefaultBlast()
		c.MuzzleY = 1.2
		if err := c.Validate(); !errors.Is(err, ErrMuzzle) {
			t.Fatalf("got %v, want ErrMuzzle", err)
		}
	})
	t.Run("unhappy: an aim past the rails is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.AngleDeg = 81
		if err := c.Validate(); !errors.Is(err, ErrAngle) {
			t.Fatalf("got %v, want ErrAngle", err)
		}
	})
	t.Run("unhappy: a negative smoke fuse is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.SmokeDelay = -0.1
		if err := c.Validate(); !errors.Is(err, ErrDelay) {
			t.Fatalf("got %v, want ErrDelay", err)
		}
	})
	t.Run("unhappy: a smoke rise outside 0..89 is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.SmokeRiseDeg = 90
		if err := c.Validate(); !errors.Is(err, ErrRise) {
			t.Fatalf("got %v, want ErrRise", err)
		}
		c.SmokeRiseDeg = -1
		if err := c.Validate(); !errors.Is(err, ErrRise) {
			t.Fatalf("got %v, want ErrRise", err)
		}
	})
	t.Run("unhappy: a folded flash ladder is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.MidAt = c.EdgeAt
		if err := c.Validate(); !errors.Is(err, ErrLadder) {
			t.Fatalf("mid<=edge: got %v, want ErrLadder", err)
		}
		c = DefaultBlast()
		c.CoreAt = c.MidAt
		if err := c.Validate(); !errors.Is(err, ErrLadder) {
			t.Fatalf("core<=mid: got %v, want ErrLadder", err)
		}
		c = DefaultBlast()
		c.EdgeAt = 0
		if err := c.Validate(); !errors.Is(err, ErrLadder) {
			t.Fatalf("edge<1: got %v, want ErrLadder", err)
		}
	})
	t.Run("unhappy: a broken layer is rejected and named", func(t *testing.T) {
		c := DefaultBlast()
		c.Pellets.Count = -1
		err := c.Validate()
		if !errors.Is(err, particle.ErrCount) {
			t.Fatalf("got %v, want the engine's ErrCount", err)
		}
		if !strings.Contains(err.Error(), "pellets") {
			t.Fatalf("the error must name the pellets layer, got %q", err)
		}
		c = DefaultBlast()
		c.Smoke.MinLife, c.Smoke.MaxLife = 2, 1
		err = c.Validate()
		if !errors.Is(err, particle.ErrLife) {
			t.Fatalf("got %v, want the engine's ErrLife", err)
		}
		if !strings.Contains(err.Error(), "smoke") {
			t.Fatalf("the error must name the smoke layer, got %q", err)
		}
	})
}

func TestUseActive(t *testing.T) {
	t.Run("happy: UseBlast puts the settings in effect and ResetBlast restores stock", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.AngleDeg = 12
		c.Sparks.Count = 40
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		if ActiveBlast() != c {
			t.Fatalf("active %+v, want the used %+v", ActiveBlast(), c)
		}
		ResetBlast()
		if ActiveBlast() != DefaultBlast() {
			t.Fatal("ResetBlast must restore the stock blast")
		}
	})
	t.Run("unhappy: an invalid blast is rejected and the active one holds", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		bad := DefaultBlast()
		bad.MuzzleX = 2
		if err := UseBlast(bad); !errors.Is(err, ErrMuzzle) {
			t.Fatalf("got %v, want ErrMuzzle", err)
		}
		if ActiveBlast() != DefaultBlast() {
			t.Fatal("a rejected blast must not clobber the active one")
		}
	})
}

func TestEngines(t *testing.T) {
	t.Run("happy: flash, pellets, and sparks share the muzzle and the aim", func(t *testing.T) {
		c := DefaultBlast()
		c.MuzzleX, c.MuzzleY = 0.25, 0.5
		c.AngleDeg = 0
		fl, pe, sp, _ := c.Engines(100, 60)
		muzzle := particle.Vec2{X: 25, Y: 30}
		for name, cfg := range map[string]particle.Config{"flash": fl, "pellets": pe, "sparks": sp} {
			if cfg.Origin != muzzle {
				t.Fatalf("%s spawns at %+v, want the muzzle %+v", name, cfg.Origin, muzzle)
			}
			if math.Abs(cfg.Direction.X-1) > 1e-9 || math.Abs(cfg.Direction.Y) > 1e-9 {
				t.Fatalf("%s aims %+v, want level rightward (1, 0)", name, cfg.Direction)
			}
			if cfg.Width != 100 || cfg.Height != 60 {
				t.Fatalf("%s lives in %vx%v, want the 100x60 stage", name, cfg.Width, cfg.Height)
			}
		}
	})
	t.Run("happy: a tilted aim climbs and the smoke climbs harder", func(t *testing.T) {
		c := DefaultBlast()
		c.AngleDeg = 30
		c.SmokeRiseDeg = 30
		fl, _, _, sm := c.Engines(100, 60)
		s, cos := math.Sincos(30 * math.Pi / 180)
		if math.Abs(fl.Direction.X-cos) > 1e-9 || math.Abs(fl.Direction.Y+s) > 1e-9 {
			t.Fatalf("a 30° aim must head (%v, %v), got %+v", cos, -s, fl.Direction)
		}
		s2, c2 := math.Sincos(60 * math.Pi / 180)
		if math.Abs(sm.Direction.X-c2) > 1e-9 || math.Abs(sm.Direction.Y+s2) > 1e-9 {
			t.Fatalf("smoke must climb aim+rise=60°, got %+v", sm.Direction)
		}
	})
	t.Run("happy: the smoke curls with the cartoon wind, the rest fly straight", func(t *testing.T) {
		fl, pe, sp, sm := DefaultBlast().Engines(100, 60)
		if sm.Mode != particle.ModeSwirl || !sm.SwirlUp {
			t.Fatalf("smoke must swirl upward, got mode %v up %v", sm.Mode, sm.SwirlUp)
		}
		for name, cfg := range map[string]particle.Config{"flash": fl, "pellets": pe, "sparks": sp} {
			if cfg.Mode != particle.ModeStraight {
				t.Fatalf("%s must fly straight, got mode %v", name, cfg.Mode)
			}
		}
	})
	t.Run("happy: each layer carries its own knobs onto its engine", func(t *testing.T) {
		c := DefaultBlast()
		c.Flash.MaxDistance = 9
		c.Pellets.Spread = 0.03
		c.Sparks.Nozzle = 1.5
		c.Smoke.Count = 21
		fl, pe, sp, sm := c.Engines(100, 60)
		if fl.Count != c.Flash.Count || fl.MaxDistance != 9 {
			t.Fatalf("flash engine %+v must carry the flash layer %+v", fl, c.Flash)
		}
		if pe.Count != 7 || pe.Spread != 0.03 {
			t.Fatalf("pellet engine %+v must carry the pellet layer %+v", pe, c.Pellets)
		}
		if sp.Nozzle != 1.5 || sp.MinLife != c.Sparks.MinLife {
			t.Fatalf("spark engine %+v must carry the spark layer %+v", sp, c.Sparks)
		}
		if sm.Count != 21 || sm.MaxLife != c.Smoke.MaxLife {
			t.Fatalf("smoke engine %+v must carry the smoke layer %+v", sm, c.Smoke)
		}
	})
	t.Run("unhappy: a muzzle on the edge clamps inside any stage", func(t *testing.T) {
		c := DefaultBlast()
		c.MuzzleX, c.MuzzleY = 1, 1
		fl, _, _, _ := c.Engines(3, 2)
		cfg := fl
		if cfg.Origin.X > cfg.Width || cfg.Origin.Y > cfg.Height || cfg.Origin.X < 0 || cfg.Origin.Y < 0 {
			t.Fatalf("origin %+v fell off a %vx%v stage", cfg.Origin, cfg.Width, cfg.Height)
		}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("an edge muzzle must still validate: %v", err)
		}
	})
}

func TestLoadSave(t *testing.T) {
	t.Run("happy: a blast survives the round trip and the JSON names all four layers", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "blast.json")
		c := DefaultBlast()
		c.AngleDeg = 8
		c.Flash.Count = 120
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := LoadBlast(path)
		if err != nil {
			t.Fatalf("LoadBlast: %v", err)
		}
		if got != c {
			t.Fatalf("round trip %+v, want %+v", got, c)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("saved JSON must parse: %v", err)
		}
		for _, key := range []string{"flash", "pellets", "sparks", "smoke", "angleDeg", "muzzleX"} {
			if _, ok := doc[key]; !ok {
				t.Fatalf("saved JSON is missing %q", key)
			}
		}
	})
	t.Run("happy: the shipped config loads and validates", func(t *testing.T) {
		c, err := LoadBlast("config.json")
		if err != nil {
			t.Fatalf("the component must ship a loadable config.json: %v", err)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the shipped config must validate: %v", err)
		}
	})
	t.Run("unhappy: missing, broken, and out-of-range files are errors", func(t *testing.T) {
		if _, err := LoadBlast(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Fatal("a missing file must error")
		}
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadBlast(bad); err == nil {
			t.Fatal("broken JSON must error")
		}
		out := filepath.Join(t.TempDir(), "out.json")
		if err := os.WriteFile(out, []byte(`{"muzzleX": 4}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadBlast(out); !errors.Is(err, ErrMuzzle) {
			t.Fatalf("got %v, want ErrMuzzle from an out-of-range file", err)
		}
	})
	t.Run("unhappy: Save refuses an invalid blast", func(t *testing.T) {
		c := DefaultBlast()
		c.AngleDeg = 200
		if err := c.Save(filepath.Join(t.TempDir(), "bad.json")); !errors.Is(err, ErrAngle) {
			t.Fatalf("got %v, want ErrAngle", err)
		}
	})
}
