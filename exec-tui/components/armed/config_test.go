package armed

// Tests written FIRST: Config is the live knobs on the armed-eagle
// composite — delay, cross, start, end, and each talon's shots,
// rate, and aim. The tuner and any scene that reads Active share
// this file. Time knobs nudge 50ms, path knobs 0.05 of the span,
// shot counts one shell, rates 0.25 /s, aims one compass point.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs match the composite defaults", func(t *testing.T) {
		c := DefaultConfig()
		if c.Delay != StockDelay || c.Cross != StockCross {
			t.Fatalf("flight %v/%v, want delay %v cross %v", c.Delay, c.Cross, StockDelay, StockCross)
		}
		if c.Start != 0 || c.End != 1 {
			t.Fatalf("path %v..%v, want the full span", c.Start, c.End)
		}
		if c.LeftShots != StockShots || c.RightShots != StockShots {
			t.Fatalf("shots %d/%d, want %d each", c.LeftShots, c.RightShots, StockShots)
		}
		if c.LeftRate != StockRate || c.RightRate != StockRate {
			t.Fatalf("rates %v/%v, want %v each", c.LeftRate, c.RightRate, StockRate)
		}
		if c.LeftAim != sprite.W || c.RightAim != sprite.E {
			t.Fatalf("aims %s/%s, want W/E", c.LeftAim, c.RightAim)
		}
		if KnobCount != 11 {
			t.Fatalf("KnobCount %d, want 11", KnobCount)
		}
	})
	t.Run("happy: Display and labels cover every knob", func(t *testing.T) {
		c := DefaultConfig()
		if got := c.Display(KnobDelay); got != fmt.Sprintf("%7.3fs", StockDelay) {
			t.Fatalf("Display(delay) %q", got)
		}
		if got := c.Display(KnobLeftRate); got != fmt.Sprintf("%7.2f/s", StockRate) {
			t.Fatalf("Display(left rate) %q", got)
		}
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
		if c.Display(KnobCount) != "" {
			t.Fatal("an off-panel knob must display nothing")
		}
	})
	t.Run("unhappy: bad times, paths, shots, rates, and aims are named", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			mod  func(*Config)
			err  error
		}{
			{"negative delay", func(c *Config) { c.Delay = -1 }, errDelay},
			{"tiny cross", func(c *Config) { c.Cross = 0 }, errCross},
			{"start off the span", func(c *Config) { c.Start = -0.1 }, errStart},
			{"end off the span", func(c *Config) { c.End = 1.2 }, errEnd},
			{"empty path", func(c *Config) { c.Start, c.End = 0.5, 0.5 }, errPath},
			{"negative shots", func(c *Config) { c.LeftShots = -1 }, errShots},
			{"negative rate", func(c *Config) { c.LeftRate = -0.5 }, errRate},
			{"aim off the compass", func(c *Config) { c.LeftAim = "X" }, errAim},
		} {
			c := DefaultConfig()
			tc.mod(&c)
			if err := c.Validate(); err == nil || err != tc.err {
				t.Fatalf("%s: got %v, want %v", tc.name, err, tc.err)
			}
		}
	})
}

func TestNudge(t *testing.T) {
	t.Run("happy: time, path, shots, rates, and aims each walk their own grid", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobDelay, 1)
		if math.Abs(c.Delay-(StockDelay+StepSeconds)) > 1e-9 {
			t.Fatalf("delay %v after +1", c.Delay)
		}
		c.Nudge(KnobLeftShots, 1)
		if c.LeftShots != StockShots+1 {
			t.Fatalf("left shots %d after +1", c.LeftShots)
		}
		c.Nudge(KnobRightAim, 1)
		if c.RightAim != sprite.SE {
			t.Fatalf("right aim %s after +1 from E, want SE", c.RightAim)
		}
	})
	t.Run("unhappy: times, paths, shots, and rates never go past their rails", func(t *testing.T) {
		c := DefaultConfig()
		c.Delay = 0
		c.Nudge(KnobDelay, -1)
		if c.Delay != 0 {
			t.Fatalf("delay %v, want 0", c.Delay)
		}
		c.LeftShots = 0
		c.Nudge(KnobLeftShots, -1)
		if c.LeftShots != 0 {
			t.Fatalf("shots %d, want 0", c.LeftShots)
		}
	})
}

func TestUseLoadSave(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: Use is what Active and New read", func(t *testing.T) {
		t.Cleanup(Reset)
		cfg := DefaultConfig()
		cfg.Delay = 1.5
		cfg.LeftRate = 4
		if err := Use(cfg); err != nil {
			t.Fatal(err)
		}
		got := Active()
		if got.Delay != 1.5 || got.LeftRate != 4 {
			t.Fatalf("Active %+v, want the used knobs", got)
		}
	})
	t.Run("happy: Save/Load round-trip the JSON", func(t *testing.T) {
		c := DefaultConfig()
		c.Delay = 1.5
		c.LeftRate = 3.25
		path := filepath.Join(t.TempDir(), "armed.json")
		if err := c.Save(path); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got.Delay-1.5) > 1e-9 || math.Abs(got.LeftRate-3.25) > 1e-9 {
			t.Fatalf("loaded %+v, want the saved knobs", got)
		}
	})
	t.Run("unhappy: Use rejects a bad cfg and Active holds", func(t *testing.T) {
		t.Cleanup(Reset)
		before := Active()
		bad := DefaultConfig()
		bad.LeftShots = -1
		if err := Use(bad); err == nil {
			t.Fatal("a negative shot count must be rejected")
		}
		if Active() != before {
			t.Fatal("a rejected Use must not clobber Active")
		}
	})
	t.Run("unhappy: a missing key keeps that knob at stock", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "partial.json")
		if err := os.WriteFile(path, []byte("{\"delay\": 1.5}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got.Delay-1.5) > 1e-9 {
			t.Fatalf("delay %v, want 1.5", got.Delay)
		}
		if got.LeftRate != StockRate {
			t.Fatalf("missing rate became %v, want stock", got.LeftRate)
		}
	})
}
