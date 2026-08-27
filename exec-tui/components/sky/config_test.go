package sky

// Tests written FIRST: Config is the three live knobs on the blue
// sky — the angle the darker blue comes from (0° is straight down
// from the top, 45° is a diagonal), the light-blue ink, and the
// dark-blue ink. The tuner nudges the angle 5°, the inks one xterm
// index at a time. Save/Load round-trip the JSON next to the
// component; Use is what New paints on the first curtain.

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs are a top-down light-to-dark blue", func(t *testing.T) {
		c := DefaultConfig()
		if err := c.Validate(); err != nil {
			t.Fatalf("the stock sky must validate: %v", err)
		}
		if c.AngleDeg != DefaultAngle {
			t.Fatalf("angle %v, want %v — dark comes from the top", c.AngleDeg, DefaultAngle)
		}
		if DefaultAngle != 0 {
			t.Fatalf("DefaultAngle %v, want 0", DefaultAngle)
		}
		if c.LightInk != DefaultLight || c.DarkInk != DefaultDark {
			t.Fatalf("inks %d/%d, want light %d dark %d", c.LightInk, c.DarkInk, DefaultLight, DefaultDark)
		}
		if lum(c.LightInk) <= lum(c.DarkInk) {
			t.Fatalf("stock light %d must be brighter than dark %d", c.LightInk, c.DarkInk)
		}
		if KnobCount != 3 {
			t.Fatalf("KnobCount %d, want 3 (angle, light, dark)", KnobCount)
		}
		if StepAngle != 5 {
			t.Fatalf("StepAngle %v, want 5°", StepAngle)
		}
	})
	t.Run("happy: Display reads every knob in its own language", func(t *testing.T) {
		c := DefaultConfig()
		if got := c.Display(KnobAngle); got != "  0.000°" {
			t.Fatalf("Display(angle) %q, want %q", got, "  0.000°")
		}
		if got := c.Display(KnobLight); got != "    153" {
			t.Fatalf("Display(light) %q, want the stock light ink", got)
		}
		if got := c.Display(KnobDark); got != "     17" {
			t.Fatalf("Display(dark) %q, want the stock dark ink", got)
		}
		if got := c.Display(KnobCount); got != "" {
			t.Fatalf("an off-panel knob displays %q, want nothing", got)
		}
	})
	t.Run("happy: every knob carries a label and reads its own value", func(t *testing.T) {
		c := DefaultConfig()
		seen := map[string]bool{}
		for k := Knob(0); k < KnobCount; k++ {
			label := KnobLabel(k)
			if label == "" {
				t.Fatalf("knob %d has no label", k)
			}
			if seen[label] {
				t.Fatalf("label %q repeats", label)
			}
			seen[label] = true
		}
		if got := c.Value(KnobAngle); got != c.AngleDeg {
			t.Fatalf("Value(angle) %v, want %v", got, c.AngleDeg)
		}
		if got := c.Value(KnobLight); got != float64(c.LightInk) {
			t.Fatalf("Value(light) %v, want %v", got, c.LightInk)
		}
	})
	t.Run("unhappy: a bad angle or an ink off the cube is named", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			mod  func(*Config)
			want error
		}{
			{"angle below the circle", func(c *Config) { c.AngleDeg = -1 }, ErrAngle},
			{"angle past a full turn", func(c *Config) { c.AngleDeg = 360 }, ErrAngle},
			{"light ink 0", func(c *Config) { c.LightInk = 0 }, ErrInk},
			{"dark ink 256", func(c *Config) { c.DarkInk = 256 }, ErrInk},
			{"NaN angle", func(c *Config) { c.AngleDeg = math.NaN() }, ErrAngle},
		} {
			c := DefaultConfig()
			tc.mod(&c)
			if err := c.Validate(); !errors.Is(err, tc.want) {
				t.Fatalf("%s: got %v, want %v", tc.name, err, tc.want)
			}
		}
	})
}

func TestNudge(t *testing.T) {
	t.Run("happy: the angle walks 5° and wraps the circle", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobAngle, 1)
		if math.Abs(c.AngleDeg-5) > 1e-9 {
			t.Fatalf("angle %v, want 5", c.AngleDeg)
		}
		c.Nudge(KnobAngle, -2)
		if math.Abs(c.AngleDeg-355) > 1e-9 {
			t.Fatalf("angle %v, want 355 — the circle wraps", c.AngleDeg)
		}
		c.AngleDeg = 355
		c.Nudge(KnobAngle, 1)
		if math.Abs(c.AngleDeg) > 1e-9 {
			t.Fatalf("angle %v, want 0 after wrapping past 355", c.AngleDeg)
		}
	})
	t.Run("happy: the inks walk one index and rail on the cube", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobLight, 1)
		if c.LightInk != DefaultLight+1 {
			t.Fatalf("light %d, want %d", c.LightInk, DefaultLight+1)
		}
		c.LightInk = 255
		c.Nudge(KnobLight, 1)
		if c.LightInk != 255 {
			t.Fatalf("light %d, want 255 — the cube rails", c.LightInk)
		}
		c.DarkInk = 1
		c.Nudge(KnobDark, -1)
		if c.DarkInk != 1 {
			t.Fatalf("dark %d, want 1 — the cube rails", c.DarkInk)
		}
	})
	t.Run("unhappy: a bad cursor or a zero step is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		before := c
		c.Nudge(KnobCount, 1)
		c.Nudge(KnobAngle, 0)
		var ghost *Config
		ghost.Nudge(KnobAngle, 1)
		if c != before {
			t.Fatalf("a no-op nudge changed %+v", c)
		}
	})
}

func TestUseLoadSave(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: Use is what Active and New read", func(t *testing.T) {
		t.Cleanup(Reset)
		cfg := DefaultConfig()
		cfg.AngleDeg = 45
		cfg.LightInk = 159
		if err := Use(cfg); err != nil {
			t.Fatal(err)
		}
		got := Active()
		if got.AngleDeg != 45 || got.LightInk != 159 {
			t.Fatalf("Active %+v, want the used knobs", got)
		}
	})
	t.Run("happy: Save/Load round-trip the JSON", func(t *testing.T) {
		c := DefaultConfig()
		c.AngleDeg = 45
		c.LightInk = 123
		c.DarkInk = 19
		path := filepath.Join(t.TempDir(), "sky.json")
		if err := c.Save(path); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.AngleDeg != 45 || got.LightInk != 123 || got.DarkInk != 19 {
			t.Fatalf("loaded %+v, want the saved knobs", got)
		}
	})
	t.Run("happy: a missing file is stock, not an error", func(t *testing.T) {
		got, err := LoadOrDefault(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatal(err)
		}
		if got != DefaultConfig() {
			t.Fatalf("missing file %+v, want stock", got)
		}
	})
	t.Run("unhappy: Use rejects a bad cfg and Active holds", func(t *testing.T) {
		t.Cleanup(Reset)
		before := Active()
		bad := DefaultConfig()
		bad.LightInk = 0
		if err := Use(bad); !errors.Is(err, ErrInk) {
			t.Fatalf("got %v, want ErrInk", err)
		}
		if Active() != before {
			t.Fatal("a rejected Use must not clobber Active")
		}
	})
	t.Run("unhappy: a missing key keeps that knob at stock", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "partial.json")
		if err := os.WriteFile(path, []byte("{\"angleDeg\": 45}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.AngleDeg != 45 {
			t.Fatalf("angle %v, want 45", got.AngleDeg)
		}
		if got.LightInk != DefaultLight || got.DarkInk != DefaultDark {
			t.Fatalf("missing inks became %d/%d, want stock", got.LightInk, got.DarkInk)
		}
	})
}
