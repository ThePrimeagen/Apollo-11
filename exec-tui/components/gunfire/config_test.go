package gunfire

// Tests written FIRST. BlastConfig is the JSON that tunes the Doom
// muzzle flame: where the muzzle sits and aims (straight up by
// default, the way the flash leaps off the shotgun in first person),
// the two-frame pulse (Doom's flash is a bright sprite then a dimmer
// one), the core brightness ladder, and one engine-knob Layer each
// for the white-hot core and the red flame — the flame carrying lift
// (hot gas rises) and drag (the eruption dies down). UseBlast puts a
// config in effect for every blast reading it, the same way the dust
// puff works, so the tuner and the demo stay on the same values.

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
	t.Run("happy: the stock flame validates and carries the Doom signature", func(t *testing.T) {
		c := DefaultBlast()
		if err := c.Validate(); err != nil {
			t.Fatalf("the stock flame must validate: %v", err)
		}
		if c.Flame.Lift <= 0 || c.Flame.Drag <= 0 {
			t.Fatalf("the flame must rise and die down — lift %v drag %v", c.Flame.Lift, c.Flame.Drag)
		}
		if c.PulseDelay <= 0 || c.PulseFrac <= 0 {
			t.Fatalf("Doom's flash is two frames — pulse delay %v frac %v", c.PulseDelay, c.PulseFrac)
		}
		if c.AngleDeg != 90 {
			t.Fatalf("the stock flame leaps straight up like the first-person flash, aim %v", c.AngleDeg)
		}
	})
	t.Run("happy: the stock flame is the active blast at boot", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		ResetBlast()
		if ActiveBlast() != DefaultBlast() {
			t.Fatalf("active blast %+v, want the stock %+v", ActiveBlast(), DefaultBlast())
		}
	})
	t.Run("unhappy: every layer is a one-shot — no period anywhere", func(t *testing.T) {
		core, flame := DefaultBlast().Engines(120, 60)
		for name, cfg := range map[string]particle.Config{"core": core, "flame": flame} {
			if cfg.Period != 0 {
				t.Fatalf("%s layer auto-emits every %vs — a gunshot is a trigger, not a clock", name, cfg.Period)
			}
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("happy: extreme but legal settings pass", func(t *testing.T) {
		c := DefaultBlast()
		c.AngleDeg = -90
		c.MuzzleX, c.MuzzleY = 0, 1
		c.PulseDelay = 0
		c.PulseFrac = 1
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
	t.Run("unhappy: an aim past straight up or straight down is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.AngleDeg = 91
		if err := c.Validate(); !errors.Is(err, ErrAngle) {
			t.Fatalf("got %v, want ErrAngle", err)
		}
		c.AngleDeg = -91
		if err := c.Validate(); !errors.Is(err, ErrAngle) {
			t.Fatalf("got %v, want ErrAngle", err)
		}
	})
	t.Run("unhappy: a negative pulse delay is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.PulseDelay = -0.1
		if err := c.Validate(); !errors.Is(err, ErrDelay) {
			t.Fatalf("got %v, want ErrDelay", err)
		}
	})
	t.Run("unhappy: a pulse fraction outside 0..1 is rejected", func(t *testing.T) {
		c := DefaultBlast()
		c.PulseFrac = 1.1
		if err := c.Validate(); !errors.Is(err, ErrPulse) {
			t.Fatalf("got %v, want ErrPulse", err)
		}
		c.PulseFrac = -0.1
		if err := c.Validate(); !errors.Is(err, ErrPulse) {
			t.Fatalf("got %v, want ErrPulse", err)
		}
	})
	t.Run("unhappy: a folded core ladder is rejected", func(t *testing.T) {
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
		c.Flame.Count = -1
		err := c.Validate()
		if !errors.Is(err, particle.ErrCount) {
			t.Fatalf("got %v, want the engine's ErrCount", err)
		}
		if !strings.Contains(err.Error(), "flame") {
			t.Fatalf("the error must name the flame layer, got %q", err)
		}
		c = DefaultBlast()
		c.Core.MinLife, c.Core.MaxLife = 2, 1
		err = c.Validate()
		if !errors.Is(err, particle.ErrLife) {
			t.Fatalf("got %v, want the engine's ErrLife", err)
		}
		if !strings.Contains(err.Error(), "core") {
			t.Fatalf("the error must name the core layer, got %q", err)
		}
		c = DefaultBlast()
		c.Flame.Drag = -2
		if err := c.Validate(); !errors.Is(err, particle.ErrDrag) {
			t.Fatalf("got %v, want the engine's ErrDrag", err)
		}
	})
}

func TestUseActive(t *testing.T) {
	t.Run("happy: UseBlast puts the settings in effect and ResetBlast restores stock", func(t *testing.T) {
		t.Cleanup(ResetBlast)
		c := DefaultBlast()
		c.AngleDeg = 45
		c.Flame.Count = 200
		if err := UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		if ActiveBlast() != c {
			t.Fatalf("active %+v, want the used %+v", ActiveBlast(), c)
		}
		ResetBlast()
		if ActiveBlast() != DefaultBlast() {
			t.Fatal("ResetBlast must restore the stock flame")
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
	t.Run("happy: core and flame share the muzzle and the aim", func(t *testing.T) {
		c := DefaultBlast()
		c.MuzzleX, c.MuzzleY = 0.25, 0.5
		c.AngleDeg = 0
		core, flame := c.Engines(100, 60)
		muzzle := particle.Vec2{X: 25, Y: 30}
		for name, cfg := range map[string]particle.Config{"core": core, "flame": flame} {
			if cfg.Origin != muzzle {
				t.Fatalf("%s spawns at %+v, want the muzzle %+v", name, cfg.Origin, muzzle)
			}
			if math.Abs(cfg.Direction.X-1) > 1e-9 || math.Abs(cfg.Direction.Y) > 1e-9 {
				t.Fatalf("%s aims %+v, want level rightward (1, 0)", name, cfg.Direction)
			}
			if cfg.Width != 100 || cfg.Height != 60 {
				t.Fatalf("%s lives in %vx%v, want the 100x60 stage", name, cfg.Width, cfg.Height)
			}
			if cfg.Mode != particle.ModeStraight {
				t.Fatalf("%s must fly straight, got mode %v", name, cfg.Mode)
			}
		}
	})
	t.Run("happy: the stock aim leaps straight up", func(t *testing.T) {
		_, flame := DefaultBlast().Engines(100, 60)
		if math.Abs(flame.Direction.X) > 1e-9 || math.Abs(flame.Direction.Y+1) > 1e-9 {
			t.Fatalf("a 90° aim must head (0, -1), got %+v", flame.Direction)
		}
	})
	t.Run("happy: each layer carries its own knobs, lift and drag included", func(t *testing.T) {
		c := DefaultBlast()
		c.Core.MaxDistance = 9
		c.Flame.Lift = 77
		c.Flame.Drag = 4.5
		c.Flame.Count = 111
		core, flame := c.Engines(100, 60)
		if core.Count != c.Core.Count || core.MaxDistance != 9 {
			t.Fatalf("core engine %+v must carry the core layer %+v", core, c.Core)
		}
		if flame.Count != 111 || flame.Lift != 77 || flame.Drag != 4.5 {
			t.Fatalf("flame engine %+v must carry the flame layer %+v", flame, c.Flame)
		}
	})
	t.Run("unhappy: a muzzle on the edge clamps inside any stage", func(t *testing.T) {
		c := DefaultBlast()
		c.MuzzleX, c.MuzzleY = 1, 1
		core, _ := c.Engines(3, 2)
		if core.Origin.X > core.Width || core.Origin.Y > core.Height || core.Origin.X < 0 || core.Origin.Y < 0 {
			t.Fatalf("origin %+v fell off a %vx%v stage", core.Origin, core.Width, core.Height)
		}
		if err := core.Validate(); err != nil {
			t.Fatalf("an edge muzzle must still validate: %v", err)
		}
	})
}

func TestLoadSave(t *testing.T) {
	t.Run("happy: a blast survives the round trip and the JSON names both layers", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "blast.json")
		c := DefaultBlast()
		c.AngleDeg = 60
		c.Flame.Lift = 44
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
		for _, key := range []string{"core", "flame", "angleDeg", "muzzleX", "pulseDelay", "pulseFrac"} {
			if _, ok := doc[key]; !ok {
				t.Fatalf("saved JSON is missing %q", key)
			}
		}
		var flameDoc map[string]json.RawMessage
		if err := json.Unmarshal(doc["flame"], &flameDoc); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"lift", "drag"} {
			if _, ok := flameDoc[key]; !ok {
				t.Fatalf("the flame layer JSON is missing %q", key)
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
