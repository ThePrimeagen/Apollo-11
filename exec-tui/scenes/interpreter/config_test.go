package interpreter

// Tests written FIRST: the Interpreter walkthrough's two timings are
// live knobs — how long the spotlight rests on each instruction, and
// how long the camera glides to the next — in a Config the runner
// nudges 50ms at a time and saves as JSON next to the scene. Use
// installs a config as the Active knobs New copies onto the next
// show, exactly the way the Core Set scene tunes. The stop marks are
// derived clock math on the config, so a retimed show still knows
// where its spotlight rests.

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	t.Run("happy: the stock knobs are the scene as staged", func(t *testing.T) {
		c := DefaultConfig()
		if c.HoldSeconds != HoldSeconds || c.GlideSeconds != GlideSeconds {
			t.Fatalf("the defaults must mirror the stock timings: %+v", c)
		}
		if c.HoldSeconds <= 0 {
			t.Fatalf("the spotlight must rest long enough to read by default — hold %v", c.HoldSeconds)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the stock knobs must be playable: %v", err)
		}
	})
	t.Run("happy: the stop marks are the knobs, summed in order", func(t *testing.T) {
		c := DefaultConfig()
		if got := c.StopStart(0); got != 0 {
			t.Fatalf("the first stop begins at the curtain: %v", got)
		}
		for i := 0; i < 5; i++ {
			want := float64(i) * (c.HoldSeconds + c.GlideSeconds)
			if got := c.StopStart(i); math.Abs(got-want) > 1e-9 {
				t.Fatalf("stop %d begins at %v, want %v", i, got, want)
			}
			if got, want := c.GlideStart(i), c.StopStart(i)+c.HoldSeconds; math.Abs(got-want) > 1e-9 {
				t.Fatalf("glide %d begins when hold %d ends: %v, want %v", i, i, got, want)
			}
			if i > 0 && c.StopStart(i) <= c.StopStart(i-1) {
				t.Fatalf("stop %d (%v) must come after stop %d (%v)", i, c.StopStart(i), i-1, c.StopStart(i-1))
			}
		}
	})
	t.Run("unhappy: a broken config is refused, knob by knob", func(t *testing.T) {
		cases := []struct {
			name string
			warp func(*Config)
		}{
			{"negative hold", func(c *Config) { c.HoldSeconds = -1 }},
			{"zero glide", func(c *Config) { c.GlideSeconds = 0 }},
			{"NaN hold", func(c *Config) { c.HoldSeconds = math.NaN() }},
			{"infinite glide", func(c *Config) { c.GlideSeconds = math.Inf(1) }},
			{"NaN glide", func(c *Config) { c.GlideSeconds = math.NaN() }},
		}
		for _, tc := range cases {
			c := DefaultConfig()
			tc.warp(&c)
			if err := c.Validate(); err == nil {
				t.Fatalf("%s must be refused, got a pass: %+v", tc.name, c)
			}
		}
	})
}

func TestKnobs(t *testing.T) {
	t.Run("happy: every knob nudges up and back down to the same value", func(t *testing.T) {
		for k := Knob(0); k < KnobCount; k++ {
			c := DefaultConfig()
			before := c.Value(k)
			c.Nudge(k, 1)
			if c.Value(k) <= before {
				t.Fatalf("knob %s did not move up from %v", KnobLabel(k), before)
			}
			c.Nudge(k, -1)
			if c.Value(k) != before {
				t.Fatalf("knob %s did not come back to %v, got %v", KnobLabel(k), before, c.Value(k))
			}
			if err := c.Validate(); err != nil {
				t.Fatalf("a nudged %s must stay playable: %v", KnobLabel(k), err)
			}
		}
	})
	t.Run("happy: every knob wears its own label and a readable display", func(t *testing.T) {
		c := DefaultConfig()
		seen := map[string]bool{}
		for k := Knob(0); k < KnobCount; k++ {
			label := KnobLabel(k)
			if label == "" {
				t.Fatalf("knob %d has no label", k)
			}
			if seen[label] {
				t.Fatalf("knob label %q repeats", label)
			}
			seen[label] = true
			if c.Display(k) == "" {
				t.Fatalf("knob %s shows nothing on the panel", label)
			}
		}
	})
	t.Run("unhappy: nudging down forever clamps at the floors, still playable", func(t *testing.T) {
		c := DefaultConfig()
		for k := Knob(0); k < KnobCount; k++ {
			for i := 0; i < 500; i++ {
				c.Nudge(k, -1)
			}
		}
		if c.HoldSeconds != 0 {
			t.Fatalf("the hold floors at zero — a spotlight that never rests is a choice, got %v", c.HoldSeconds)
		}
		if c.GlideSeconds != StepSeconds {
			t.Fatalf("the glide floors at one step so the camera keeps a clock, got %v", c.GlideSeconds)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the floored knobs must still play: %v", err)
		}
	})
	t.Run("unhappy: a ghost knob is a no-op, never a panic", func(t *testing.T) {
		c := DefaultConfig()
		before := c
		c.Nudge(Knob(99), 1)
		c.Nudge(Knob(-1), -1)
		if c != before {
			t.Fatalf("nudging a ghost knob changed the config: %+v", c)
		}
		if KnobLabel(Knob(99)) != "" {
			t.Fatal("a ghost knob must have no label")
		}
		if c.Value(Knob(99)) != 0 {
			t.Fatal("a ghost knob must read zero")
		}
	})
}

func TestConfigFile(t *testing.T) {
	t.Run("happy: save and load round-trip every knob", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		c := DefaultConfig()
		c.Nudge(KnobHold, -3)
		c.Nudge(KnobGlide, 4)
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		back, err := LoadOrDefault(path)
		if err != nil {
			t.Fatalf("LoadOrDefault: %v", err)
		}
		if back != c {
			t.Fatalf("round trip changed the knobs:\nsaved  %+v\nloaded %+v", c, back)
		}
	})
	t.Run("happy: a missing file quietly plays the stock show", func(t *testing.T) {
		c, err := LoadOrDefault(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatalf("a missing file is not an error: %v", err)
		}
		if c != DefaultConfig() {
			t.Fatalf("a missing file must yield the defaults, got %+v", c)
		}
	})
	t.Run("happy: a file missing a knob keeps that knob at stock", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte("{\n  \"glideSeconds\": 1.5\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		c, err := LoadOrDefault(path)
		if err != nil {
			t.Fatalf("a sparse file must load: %v", err)
		}
		if c.GlideSeconds != 1.5 {
			t.Fatalf("the named knob must land: glide %v, want 1.5", c.GlideSeconds)
		}
		if c.HoldSeconds != HoldSeconds {
			t.Fatalf("the unnamed knob must stay stock: %+v", c)
		}
	})
	t.Run("unhappy: a corrupt file is a loud error, not silent defaults", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadOrDefault(path); err == nil {
			t.Fatal("corrupt JSON must error")
		}
	})
	t.Run("unhappy: a file with a broken knob is refused", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, []byte("{\n  \"holdSeconds\": -1\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadOrDefault(path); err == nil {
			t.Fatal("a negative hold must be refused on load")
		}
	})
	t.Run("unhappy: a broken config refuses to save, and a missing directory errors", func(t *testing.T) {
		c := DefaultConfig()
		c.GlideSeconds = 0
		if err := c.Save(filepath.Join(t.TempDir(), "config.json")); err == nil {
			t.Fatal("saving a broken config must error")
		}
		good := DefaultConfig()
		if err := good.Save(filepath.Join(t.TempDir(), "no", "dir", "config.json")); err == nil {
			t.Fatal("writing into a missing directory must error")
		}
	})
}

func TestUseActive(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: Use makes the knobs the next New plays", func(t *testing.T) {
		t.Cleanup(Reset)
		fast := DefaultConfig()
		fast.HoldSeconds = 0.5
		fast.GlideSeconds = 0.4
		if err := Use(fast); err != nil {
			t.Fatalf("Use: %v", err)
		}
		if Active() != fast {
			t.Fatalf("Active must hold the used knobs: %+v", Active())
		}
		if got := New().Cfg; got != fast {
			t.Fatalf("New must copy the Active knobs, got %+v", got)
		}
	})
	t.Run("happy: Reset restores the stock knobs", func(t *testing.T) {
		fast := DefaultConfig()
		fast.HoldSeconds = 0.1
		if err := Use(fast); err != nil {
			t.Fatalf("Use: %v", err)
		}
		Reset()
		if Active() != DefaultConfig() {
			t.Fatalf("Reset must restore stock, got %+v", Active())
		}
	})
	t.Run("unhappy: a broken config is rejected and Active holds", func(t *testing.T) {
		t.Cleanup(Reset)
		before := Active()
		bad := DefaultConfig()
		bad.GlideSeconds = math.NaN()
		if err := Use(bad); err == nil {
			t.Fatal("Use must refuse a broken config")
		}
		if Active() != before {
			t.Fatalf("a refused Use must leave Active alone: %+v", Active())
		}
	})
}
