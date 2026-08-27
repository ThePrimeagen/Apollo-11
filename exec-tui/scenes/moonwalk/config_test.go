package moonwalk

// Tests written FIRST. The moonwalk scene is tunable: how fast the
// stride animates, how fast he covers ground, how long each jump arc
// takes, how long the slide and the flag hoist run, how tall the
// flagpole stands, and how far and how fast the camera pans to the
// rover at the end. The knobs live in a JSON config next to the scene
// so the tuner TUI can save what looks right.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	t.Run("happy: the default knobs are sane and playable", func(t *testing.T) {
		c := DefaultConfig()
		if c.StrideFPS <= 0 || c.RunSpeed <= 0 || c.JumpSeconds <= 0 ||
			c.SlideSeconds <= 0 || c.FlagSeconds <= 0 || c.PanSeconds <= 0 || c.ExitSpeed <= 0 {
			t.Fatalf("default timings must be positive: %+v", c)
		}
		if c.TopSeconds < 0 {
			t.Fatalf("the top hold cannot be negative: %+v", c)
		}
		if c.PoleRows < MinPoleRows || c.PoleRows > MaxPoleRows {
			t.Fatalf("default pole %d out of [%d, %d]", c.PoleRows, MinPoleRows, MaxPoleRows)
		}
		if c.BoxStart < MinBoxStart || c.BoxStart > MaxBoxStart {
			t.Fatalf("default box start %d out of [%d, %d]", c.BoxStart, MinBoxStart, MaxBoxStart)
		}
		if c.PanCols <= 0 {
			t.Fatalf("the ending pan must reveal something: %+v", c)
		}
	})
	t.Run("happy: the defaults stage the requested show", func(t *testing.T) {
		c := DefaultConfig()
		if c.RunSpeed < 18 {
			t.Fatalf("the ground sprint was asked to quicken: %v", c.RunSpeed)
		}
		if c.PoleRows < 20 {
			t.Fatalf("the pole was asked to grow: %d", c.PoleRows)
		}
		if c.TopSeconds <= 0 {
			t.Fatalf("he holds the top for a beat by default: %v", c.TopSeconds)
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
				t.Fatalf("knob %s did not move up from %v", k, before)
			}
			c.Nudge(k, -1)
			if c.Value(k) != before {
				t.Fatalf("knob %s did not come back to %v, got %v", k, before, c.Value(k))
			}
		}
	})
	t.Run("happy: every knob has a name for the tuner row", func(t *testing.T) {
		seen := map[string]bool{}
		for k := Knob(0); k < KnobCount; k++ {
			name := k.String()
			if name == "" || name == "?" {
				t.Fatalf("knob %d has no name", k)
			}
			if seen[name] {
				t.Fatalf("knob name %q repeats", name)
			}
			seen[name] = true
		}
	})
	t.Run("unhappy: knobs clamp at their floors instead of going degenerate", func(t *testing.T) {
		c := DefaultConfig()
		for k := Knob(0); k < KnobCount; k++ {
			for i := 0; i < 500; i++ {
				c.Nudge(k, -1)
			}
		}
		if c.StrideFPS <= 0 || c.RunSpeed <= 0 || c.JumpSeconds <= 0 ||
			c.SlideSeconds <= 0 || c.FlagSeconds <= 0 || c.PanSeconds <= 0 || c.ExitSpeed <= 0 {
			t.Fatalf("nudging down forever must clamp, got %+v", c)
		}
		if c.TopSeconds < 0 {
			t.Fatalf("the top hold clamps at zero, got %v", c.TopSeconds)
		}
		if c.PoleRows < MinPoleRows {
			t.Fatalf("pole clamps at %d, got %d", MinPoleRows, c.PoleRows)
		}
		if c.BoxStart < MinBoxStart {
			t.Fatalf("box start clamps at %d, got %d", MinBoxStart, c.BoxStart)
		}
		if c.PanCols < 0 {
			t.Fatalf("pan cols cannot go negative, got %d", c.PanCols)
		}
	})
	t.Run("happy: the pan amount has no artificial ceiling", func(t *testing.T) {
		c := DefaultConfig()
		want := c.PanCols + 500
		for i := 0; i < 500; i++ {
			c.Nudge(KnobPanCols, 1)
		}
		if c.PanCols != want {
			t.Fatalf("pan cols hit a rail at %d — it must climb to whatever the operator wants (%d)", c.PanCols, want)
		}
	})
	t.Run("unhappy: an out-of-range knob is a no-op, never a panic", func(t *testing.T) {
		c := DefaultConfig()
		before := c
		c.Nudge(Knob(99), 1)
		if c != before {
			t.Fatalf("nudging a ghost knob changed the config: %+v", c)
		}
	})
}

func TestConfigFile(t *testing.T) {
	t.Run("happy: save and load round-trip every knob", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		c := DefaultConfig()
		c.Nudge(KnobStrideFPS, 3)
		c.Nudge(KnobPoleRows, 2)
		c.Nudge(KnobBoxStart, 2)
		c.Nudge(KnobTopSeconds, 1)
		c.Nudge(KnobExitSpeed, -2)
		c.Nudge(KnobPanCols, -4)
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		back, err := LoadOrDefault(path)
		if err != nil {
			t.Fatalf("LoadOrDefault: %v", err)
		}
		if back != c {
			t.Fatalf("round trip changed the config:\nsaved  %+v\nloaded %+v", c, back)
		}
	})
	t.Run("happy: a missing file quietly plays the defaults", func(t *testing.T) {
		c, err := LoadOrDefault(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatalf("a missing file is not an error: %v", err)
		}
		if c != DefaultConfig() {
			t.Fatalf("missing file must yield defaults, got %+v", c)
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
	t.Run("unhappy: saving into a missing directory errors", func(t *testing.T) {
		c := DefaultConfig()
		if err := c.Save(filepath.Join(t.TempDir(), "no", "dir", "config.json")); err == nil {
			t.Fatal("writing into a missing directory must error")
		}
	})
}
