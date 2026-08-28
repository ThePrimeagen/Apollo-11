package coreset

// Tests written FIRST: every timing of the Core Set lesson becomes a
// live knob — how long the memory unit holds, the drain's stagger and
// burn, how fast the survivor glides to the top, how long it settles
// there before the twelve words begin, the word reveal's beat, the
// anatomy hold, and the priority zoom's fade and glide. The knobs
// live in a Config the runner nudges 50ms at a time and saves as JSON
// next to the scene; Use installs a config as the Active knobs New
// copies onto the next show, exactly the way the America scene tunes.
// The act boundaries stop being package constants and become derived
// clock marks on the config, so a retimed show still knows where its
// acts begin.

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigDefaults(t *testing.T) {
	t.Run("happy: the stock knobs are the scene as staged", func(t *testing.T) {
		c := DefaultConfig()
		if c.UnitSeconds != UnitSeconds || c.FadeBeat != FadeBeat ||
			c.DissolveSeconds != DissolveSeconds || c.MoveSeconds != MoveSeconds ||
			c.WordBeat != WordBeat || c.WordHold != WordHold ||
			c.ZoomFadeSeconds != ZoomFadeSeconds || c.ZoomGlideSeconds != ZoomGlideSeconds {
			t.Fatalf("the defaults must mirror the stock timings: %+v", c)
		}
		if c.SettleSeconds != SettleSeconds {
			t.Fatalf("the stock settle is %v, got %v", SettleSeconds, c.SettleSeconds)
		}
		if c.SettleSeconds <= 0 {
			t.Fatalf("the survivor must rest before the words by default — settle %v", c.SettleSeconds)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the stock knobs must be playable: %v", err)
		}
	})
	t.Run("happy: the act boundaries are the knobs, summed in order", func(t *testing.T) {
		c := DefaultConfig()
		if got := c.FadeStart(); got != c.UnitSeconds {
			t.Fatalf("the drain begins when the unit hold ends: %v, want %v", got, c.UnitSeconds)
		}
		if got, want := c.FadeSeconds(), 14*c.FadeBeat+c.DissolveSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("the drain covers fourteen dissolves: %v, want %v", got, want)
		}
		if got, want := c.MoveStart(), c.FadeStart()+c.FadeSeconds(); math.Abs(got-want) > 1e-9 {
			t.Fatalf("the move follows the drain: %v, want %v", got, want)
		}
		if got, want := c.SettleStart(), c.MoveStart()+c.MoveSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("the settle begins the moment the glide lands: %v, want %v", got, want)
		}
		if got, want := c.WordsStart(), c.SettleStart()+c.SettleSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("the words wait out the settle: %v, want %v", got, want)
		}
		if got, want := c.ZoomStart(), c.WordsStart()+12*c.WordBeat+c.WordHold; math.Abs(got-want) > 1e-9 {
			t.Fatalf("the zoom follows twelve words and the hold: %v, want %v", got, want)
		}
		if got, want := c.ZoomSeconds(), c.ZoomFadeSeconds+c.ZoomGlideSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("the zoom is its fade plus its glide: %v, want %v", got, want)
		}
		if got, want := c.BitsStart(), c.ZoomStart()+c.ZoomSeconds(); math.Abs(got-want) > 1e-9 {
			t.Fatalf("the bits follow the zoom: %v, want %v", got, want)
		}
		marks := []float64{0, c.FadeStart(), c.MoveStart(), c.SettleStart(), c.WordsStart(), c.ZoomStart(), c.BitsStart()}
		for i := 1; i < len(marks); i++ {
			if marks[i] <= marks[i-1] {
				t.Fatalf("act mark %d (%v) must come after mark %d (%v)", i, marks[i], i-1, marks[i-1])
			}
		}
	})
	t.Run("unhappy: a broken config is refused, knob by knob", func(t *testing.T) {
		cases := []struct {
			name string
			warp func(*Config)
		}{
			{"negative unit hold", func(c *Config) { c.UnitSeconds = -1 }},
			{"negative fade beat", func(c *Config) { c.FadeBeat = -0.1 }},
			{"negative dissolve", func(c *Config) { c.DissolveSeconds = -0.1 }},
			{"zero move", func(c *Config) { c.MoveSeconds = 0 }},
			{"negative settle", func(c *Config) { c.SettleSeconds = -0.5 }},
			{"negative word beat", func(c *Config) { c.WordBeat = -0.1 }},
			{"negative word hold", func(c *Config) { c.WordHold = -1 }},
			{"negative zoom fade", func(c *Config) { c.ZoomFadeSeconds = -0.1 }},
			{"zero zoom glide", func(c *Config) { c.ZoomGlideSeconds = 0 }},
			{"NaN word beat", func(c *Config) { c.WordBeat = math.NaN() }},
			{"infinite move", func(c *Config) { c.MoveSeconds = math.Inf(1) }},
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
		if c.MoveSeconds < StepSeconds {
			t.Fatalf("the move floors at one step so the glide keeps a duration, got %v", c.MoveSeconds)
		}
		if c.ZoomGlideSeconds < StepSeconds {
			t.Fatalf("the zoom glide floors at one step, got %v", c.ZoomGlideSeconds)
		}
		if c.UnitSeconds < 0 || c.FadeBeat < 0 || c.DissolveSeconds < 0 ||
			c.SettleSeconds < 0 || c.WordBeat < 0 || c.WordHold < 0 || c.ZoomFadeSeconds < 0 {
			t.Fatalf("no hold may go negative: %+v", c)
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
		c.Nudge(KnobUnit, -3)
		c.Nudge(KnobMove, 4)
		c.Nudge(KnobSettle, 2)
		c.Nudge(KnobWordBeat, -1)
		c.Nudge(KnobZoomGlide, 3)
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
		if err := os.WriteFile(path, []byte("{\n  \"moveSeconds\": 2.5\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		c, err := LoadOrDefault(path)
		if err != nil {
			t.Fatalf("a sparse file must load: %v", err)
		}
		if c.MoveSeconds != 2.5 {
			t.Fatalf("the named knob must land: move %v, want 2.5", c.MoveSeconds)
		}
		if c.SettleSeconds != SettleSeconds || c.WordBeat != WordBeat {
			t.Fatalf("the unnamed knobs must stay stock: %+v", c)
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
		if err := os.WriteFile(path, []byte("{\n  \"moveSeconds\": -1\n}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadOrDefault(path); err == nil {
			t.Fatal("a negative move must be refused on load")
		}
	})
	t.Run("unhappy: a broken config refuses to save", func(t *testing.T) {
		c := DefaultConfig()
		c.MoveSeconds = -1
		if err := c.Save(filepath.Join(t.TempDir(), "config.json")); err == nil {
			t.Fatal("saving a broken config must error")
		}
	})
	t.Run("unhappy: saving into a missing directory errors", func(t *testing.T) {
		c := DefaultConfig()
		if err := c.Save(filepath.Join(t.TempDir(), "no", "dir", "config.json")); err == nil {
			t.Fatal("writing into a missing directory must error")
		}
	})
}

func TestUseActive(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: Use makes the knobs the next New plays", func(t *testing.T) {
		t.Cleanup(Reset)
		fast := DefaultConfig()
		fast.UnitSeconds = 0.5
		fast.MoveSeconds = 0.4
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
		fast.WordBeat = 0.1
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
		bad.MoveSeconds = math.NaN()
		if err := Use(bad); err == nil {
			t.Fatal("Use must refuse a broken config")
		}
		if Active() != before {
			t.Fatalf("a refused Use must leave Active alone: %+v", Active())
		}
	})
}
