package explorer

// Tests written FIRST: the explorer scene's config is the four
// twinkle knobs — the cycle range and the fade range, each a min and
// a max in seconds. The cycle knobs walk 250ms at a time, the fade
// knobs 50ms; every knob lives between the stars package's twinkle
// rails, and a pair can never cross: nudging a min into its max (or a
// max into its min) clamps at the partner. Use validates and makes
// the knobs active for New, and pushes the same numbers onto the
// stars package so the sky breathes them live. s writes the JSON next
// to the scene; a missing file is the stock config, a broken one is
// an error worth stopping for.

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
)

func reset() {
	Reset()
	stars.ResetTwinkle()
}

func TestConfigDefaults(t *testing.T) {
	t.Cleanup(reset)
	t.Run("happy: the stock knobs validate and mirror the stock twinkle", func(t *testing.T) {
		d := DefaultConfig()
		if err := d.Validate(); err != nil {
			t.Fatalf("the stock config must validate: %v", err)
		}
		if d.Twinkle() != stars.DefaultTwinkle() {
			t.Fatalf("the stock knobs %+v must speak the stock twinkle %+v", d.Twinkle(), stars.DefaultTwinkle())
		}
		Reset()
		if Active() != DefaultConfig() {
			t.Fatalf("after Reset the active knobs are %+v", Active())
		}
	})
	t.Run("happy: every knob wears a label and reads a value", func(t *testing.T) {
		c := DefaultConfig()
		want := map[Knob]struct {
			label string
			value float64
		}{
			KnobMinCycle: {"min cycle", c.MinCycleSeconds},
			KnobMaxCycle: {"max cycle", c.MaxCycleSeconds},
			KnobMinFade:  {"min fade", c.MinFadeSeconds},
			KnobMaxFade:  {"max fade", c.MaxFadeSeconds},
		}
		if KnobCount != 4 {
			t.Fatalf("the panel holds %d knobs, want the four twinkle ranges", KnobCount)
		}
		for k, w := range want {
			if KnobLabel(k) != w.label {
				t.Fatalf("knob %d is labeled %q, want %q", k, KnobLabel(k), w.label)
			}
			if c.Value(k) != w.value {
				t.Fatalf("knob %q reads %v, want %v", w.label, c.Value(k), w.value)
			}
		}
		if KnobLabel(KnobCount) != "" || KnobLabel(-1) != "" {
			t.Fatal("a knob off the panel has no label")
		}
	})
}

func TestNudge(t *testing.T) {
	t.Run("happy: cycles walk 250ms, fades walk 50ms, both ways", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobMaxCycle, 1)
		if got, want := c.MaxCycleSeconds, DefaultConfig().MaxCycleSeconds+CycleStepSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("max cycle after +1 is %v, want %v", got, want)
		}
		c.Nudge(KnobMaxCycle, -1)
		if got, want := c.MaxCycleSeconds, DefaultConfig().MaxCycleSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("max cycle after -1 is %v, want %v", got, want)
		}
		c.Nudge(KnobMinFade, 1)
		if got, want := c.MinFadeSeconds, DefaultConfig().MinFadeSeconds+FadeStepSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("min fade after +1 is %v, want %v", got, want)
		}
	})
	t.Run("happy: a nudged config always validates — the knobs cannot leave the rails", func(t *testing.T) {
		c := DefaultConfig()
		for i := 0; i < 400; i++ {
			c.Nudge(KnobMinCycle, -1)
			c.Nudge(KnobMaxCycle, 1)
			c.Nudge(KnobMinFade, -1)
			c.Nudge(KnobMaxFade, 1)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the widest knobs must validate: %v", err)
		}
		if c.MinCycleSeconds != stars.MinTwinkleCycle || c.MaxCycleSeconds != stars.MaxTwinkleCycle {
			t.Fatalf("the cycle knobs must stop at the rails, got [%v, %v]", c.MinCycleSeconds, c.MaxCycleSeconds)
		}
		if c.MinFadeSeconds != stars.MinTwinkleFade || c.MaxFadeSeconds != stars.MaxTwinkleFade {
			t.Fatalf("the fade knobs must stop at the rails, got [%v, %v]", c.MinFadeSeconds, c.MaxFadeSeconds)
		}
	})
	t.Run("unhappy: a min never climbs past its max, and a max never dips under its min", func(t *testing.T) {
		c := DefaultConfig()
		for i := 0; i < 400; i++ {
			c.Nudge(KnobMinCycle, 1)
		}
		if c.MinCycleSeconds > c.MaxCycleSeconds {
			t.Fatalf("min cycle %v crossed max cycle %v", c.MinCycleSeconds, c.MaxCycleSeconds)
		}
		if c.MinCycleSeconds != c.MaxCycleSeconds {
			t.Fatalf("min cycle must clamp at its partner, stopped at %v under %v", c.MinCycleSeconds, c.MaxCycleSeconds)
		}
		for i := 0; i < 400; i++ {
			c.Nudge(KnobMaxFade, -1)
		}
		if c.MaxFadeSeconds < c.MinFadeSeconds {
			t.Fatalf("max fade %v crossed min fade %v", c.MaxFadeSeconds, c.MinFadeSeconds)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("clamped knobs must validate: %v", err)
		}
	})
	t.Run("unhappy: a bad cursor or a zero dir is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobCount, 1)
		c.Nudge(-1, 1)
		c.Nudge(KnobMinCycle, 0)
		var ghost *Config
		ghost.Nudge(KnobMinCycle, 1)
		if c != DefaultConfig() {
			t.Fatalf("a refused nudge moved the knobs to %+v", c)
		}
	})
}

func TestUse(t *testing.T) {
	t.Cleanup(reset)
	t.Run("happy: Use makes the knobs active and the sky breathes them", func(t *testing.T) {
		c := DefaultConfig()
		c.MinCycleSeconds, c.MaxCycleSeconds = 1, 3
		c.MinFadeSeconds, c.MaxFadeSeconds = 0.2, 0.6
		if err := Use(c); err != nil {
			t.Fatalf("Use: %v", err)
		}
		if Active() != c {
			t.Fatalf("Active %+v, want %+v", Active(), c)
		}
		if stars.ActiveTwinkle() != c.Twinkle() {
			t.Fatalf("the sky breathes %+v, want the knobs %+v", stars.ActiveTwinkle(), c.Twinkle())
		}
	})
	t.Run("unhappy: broken knobs are refused and nothing moves", func(t *testing.T) {
		reset()
		before, sky := Active(), stars.ActiveTwinkle()
		for _, c := range []Config{
			{MinCycleSeconds: 5, MaxCycleSeconds: 2, MinFadeSeconds: 0.2, MaxFadeSeconds: 0.6},
			{MinCycleSeconds: 1, MaxCycleSeconds: 3, MinFadeSeconds: 0.6, MaxFadeSeconds: 0.2},
			{MinCycleSeconds: 0, MaxCycleSeconds: 3, MinFadeSeconds: 0.2, MaxFadeSeconds: 0.6},
			{MinCycleSeconds: 1, MaxCycleSeconds: 3, MinFadeSeconds: 0.2, MaxFadeSeconds: 1e9},
			{MinCycleSeconds: math.NaN(), MaxCycleSeconds: 3, MinFadeSeconds: 0.2, MaxFadeSeconds: 0.6},
		} {
			if err := Use(c); err == nil {
				t.Fatalf("Use(%+v) must be refused", c)
			}
			if Active() != before || stars.ActiveTwinkle() != sky {
				t.Fatal("a refused Use moved the active knobs or the sky")
			}
		}
	})
}

func TestLoadSave(t *testing.T) {
	t.Cleanup(reset)
	t.Run("happy: save and load round-trip the knobs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "explorer.json")
		c := DefaultConfig()
		c.MinCycleSeconds, c.MaxCycleSeconds = 1.25, 8.5
		c.MinFadeSeconds, c.MaxFadeSeconds = 0.15, 2.4
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got != c {
			t.Fatalf("round-trip lost the knobs: %+v vs %+v", got, c)
		}
	})
	t.Run("happy: a missing file is the stock config, not an error", func(t *testing.T) {
		got, err := LoadOrDefault(filepath.Join(t.TempDir(), "nowhere.json"))
		if err != nil {
			t.Fatalf("LoadOrDefault on a missing file: %v", err)
		}
		if got != DefaultConfig() {
			t.Fatalf("a missing file must be the stock knobs, got %+v", got)
		}
	})
	t.Run("happy: the stock config.json ships beside the scene", func(t *testing.T) {
		if DefaultConfigPath != "scenes/explorer/config.json" {
			t.Fatalf("the config lives beside the scene, not at %q", DefaultConfigPath)
		}
		got, err := Load("config.json")
		if err != nil {
			t.Fatalf("the stock config.json must ship with the scene: %v", err)
		}
		if err := got.Validate(); err != nil {
			t.Fatalf("the shipped config must validate: %v", err)
		}
	})
	t.Run("unhappy: broken JSON and broken knobs refuse to load", func(t *testing.T) {
		dir := t.TempDir()
		garbled := filepath.Join(dir, "garbled.json")
		if err := os.WriteFile(garbled, []byte("{ not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(garbled); err == nil {
			t.Fatal("garbled JSON must refuse to load")
		}
		crossed := filepath.Join(dir, "crossed.json")
		if err := os.WriteFile(crossed, []byte(`{"minCycleSeconds": 9, "maxCycleSeconds": 2, "minFadeSeconds": 0.2, "maxFadeSeconds": 0.6}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(crossed); err == nil {
			t.Fatal("crossed ranges must refuse to load")
		}
		if _, err := LoadOrDefault(garbled); err == nil {
			t.Fatal("LoadOrDefault forgives absence, never breakage")
		}
	})
	t.Run("unhappy: saving broken knobs is refused before the disk", func(t *testing.T) {
		c := Config{MinCycleSeconds: 9, MaxCycleSeconds: 2, MinFadeSeconds: 0.2, MaxFadeSeconds: 0.6}
		path := filepath.Join(t.TempDir(), "never.json")
		if err := c.Save(path); err == nil {
			t.Fatal("broken knobs must not save")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("a refused save must leave no file")
		}
	})
}
