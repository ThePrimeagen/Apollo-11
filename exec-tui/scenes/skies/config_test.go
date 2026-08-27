package skies

// Tests written FIRST: Config is the live knobs on the Skies scene —
// how long the blue sky takes to tilt up from the horizon (Rise),
// when the eagle enters, how long its crossing takes (the eagle's
// speed), where the flight starts and ends as fractions of the full
// off-right-to-off-left span, and the talon shotguns: how many shells
// each gun fires and at what rate (shots per second), plus which
// compass point each barrel aims. The time knobs nudge 50ms, the
// path knobs 0.05 of the span, the shot counts one shell, the rates
// 0.25 /s, the aims one compass point with wrap.

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs match the scene defaults", func(t *testing.T) {
		c := DefaultConfig()
		if c.RiseSeconds != RiseSeconds {
			t.Fatalf("rise %v, want %v", c.RiseSeconds, RiseSeconds)
		}
		if c.CrossSeconds != CrossSeconds {
			t.Fatalf("cross %v, want %v", c.CrossSeconds, CrossSeconds)
		}
		if c.EagleStart != StartPoint || c.EagleEnd != EndPoint {
			t.Fatalf("path %v..%v, want the stock %v..%v", c.EagleStart, c.EagleEnd, StartPoint, EndPoint)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
		if StepPoint != 0.050 {
			t.Fatalf("path step %v, want 0.05 of the span", StepPoint)
		}
		if StepRate != 0.25 {
			t.Fatalf("rate step %v, want 0.25 /s", StepRate)
		}
		if c.LeftShots != StockShots || c.RightShots != StockShots {
			t.Fatalf("shots %d/%d, want the stock %d each", c.LeftShots, c.RightShots, StockShots)
		}
		if c.LeftRate != StockRate || c.RightRate != StockRate {
			t.Fatalf("rates %v/%v, want the stock %v each", c.LeftRate, c.RightRate, StockRate)
		}
		if c.LeftAim != StockLeftAim || c.RightAim != StockRightAim {
			t.Fatalf("aims %s/%s, want the stock %s/%s", c.LeftAim, c.RightAim, StockLeftAim, StockRightAim)
		}
		if StockLeftAim != sprite.W || StockRightAim != sprite.E {
			t.Fatalf("stock aims %s/%s, want W/E", StockLeftAim, StockRightAim)
		}
		if KnobCount != 11 {
			t.Fatalf("KnobCount %d, want 11", KnobCount)
		}
	})
	t.Run("happy: Display reads every knob in its own language", func(t *testing.T) {
		c := DefaultConfig()
		if got := c.Display(KnobRise); got != fmt.Sprintf("%7.3fs", RiseSeconds) {
			t.Fatalf("Display(rise) %q", got)
		}
		if got := c.Display(KnobStart); got != "  0.000" {
			t.Fatalf("Display(start) %q, want a fraction, not seconds", got)
		}
		if got := c.Display(KnobLeftShots); got != fmt.Sprintf("%7d", StockShots) {
			t.Fatalf("Display(left shots) %q, want a bare count", got)
		}
		if got := c.Display(KnobLeftRate); got != fmt.Sprintf("%7.2f/s", StockRate) {
			t.Fatalf("Display(left rate) %q, want shots per second", got)
		}
		if got := c.Display(KnobRightAim); got != fmt.Sprintf("%7s", string(StockRightAim)) {
			t.Fatalf("Display(right aim) %q, want the compass point", got)
		}
		if got := c.Display(KnobCount); got != "" {
			t.Fatalf("an off-panel knob displays %q, want nothing", got)
		}
	})
	t.Run("happy: every knob carries a unique label", func(t *testing.T) {
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
	})
	t.Run("unhappy: bad times, paths, shots, rates, and aims are named", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			mod  func(*Config)
			err  error
		}{
			{"negative rise", func(c *Config) { c.RiseSeconds = -1 }, errRise},
			{"tiny cross", func(c *Config) { c.CrossSeconds = 0 }, errCross},
			{"start off the span", func(c *Config) { c.EagleStart = -0.1 }, errStart},
			{"end off the span", func(c *Config) { c.EagleEnd = 1.2 }, errEnd},
			{"empty path", func(c *Config) { c.EagleStart, c.EagleEnd = 0.5, 0.5 }, errPath},
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
		c.Nudge(KnobRise, 1)
		if math.Abs(c.RiseSeconds-(RiseSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("rise %v after +1", c.RiseSeconds)
		}
		c.Nudge(KnobStart, 1)
		if math.Abs(c.EagleStart-(StartPoint+StepPoint)) > 1e-9 {
			t.Fatalf("start %v after +1", c.EagleStart)
		}
		c.Nudge(KnobLeftShots, 1)
		if c.LeftShots != StockShots+1 {
			t.Fatalf("left shots %d after +1", c.LeftShots)
		}
		c.Nudge(KnobRightRate, 1)
		if math.Abs(c.RightRate-(StockRate+StepRate)) > 1e-9 {
			t.Fatalf("right rate %v after +1", c.RightRate)
		}
		c.Nudge(KnobLeftAim, 1)
		if c.LeftAim != sprite.NW {
			t.Fatalf("left aim %s after +1 from W, want NW", c.LeftAim)
		}
	})
	t.Run("unhappy: times, paths, shots, and rates never go past their rails", func(t *testing.T) {
		c := DefaultConfig()
		c.RiseSeconds = 0
		c.Nudge(KnobRise, -1)
		if c.RiseSeconds != 0 {
			t.Fatalf("rise %v, want 0", c.RiseSeconds)
		}
		c.EagleStart = 0
		c.Nudge(KnobStart, -1)
		if c.EagleStart != 0 {
			t.Fatalf("start %v, want 0", c.EagleStart)
		}
		c.LeftShots = 0
		c.Nudge(KnobLeftShots, -1)
		if c.LeftShots != 0 {
			t.Fatalf("shots %d, want 0", c.LeftShots)
		}
		c.RightRate = 0
		c.Nudge(KnobRightRate, -1)
		if c.RightRate != 0 {
			t.Fatalf("rate %v, want 0", c.RightRate)
		}
	})
}

func TestUseLoadSave(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: Use is what Active and New read", func(t *testing.T) {
		t.Cleanup(Reset)
		cfg := DefaultConfig()
		cfg.RiseSeconds = 1.5
		cfg.LeftRate = 4
		if err := Use(cfg); err != nil {
			t.Fatal(err)
		}
		got := Active()
		if got.RiseSeconds != 1.5 || got.LeftRate != 4 {
			t.Fatalf("Active %+v, want the used knobs", got)
		}
	})
	t.Run("happy: Save/Load round-trip the JSON", func(t *testing.T) {
		c := DefaultConfig()
		c.RiseSeconds = 1.5
		c.LeftRate = 3.25
		c.EagleEnd = 0.7
		path := filepath.Join(t.TempDir(), "skies.json")
		if err := c.Save(path); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got.RiseSeconds-1.5) > 1e-9 || math.Abs(got.LeftRate-3.25) > 1e-9 || math.Abs(got.EagleEnd-0.7) > 1e-9 {
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
		if err := os.WriteFile(path, []byte("{\"riseSeconds\": 1.5}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got.RiseSeconds-1.5) > 1e-9 {
			t.Fatalf("rise %v, want 1.5", got.RiseSeconds)
		}
		if got.LeftRate != StockRate {
			t.Fatalf("missing rate became %v, want stock", got.LeftRate)
		}
	})
}
