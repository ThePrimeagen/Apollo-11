package landing

// Tests written FIRST: Config is the three live knobs — land duration,
// dust start offset, dust run — nudged 50ms at a time. Play rebuilds
// the landing from the current knobs so iteration does not require a
// restart. Save/Load round-trip the JSON; Use is what New and the
// walkthrough play on the first curtain.

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs match the portable landing defaults", func(t *testing.T) {
		c := DefaultConfig()
		if c.LandSeconds != LandSeconds {
			t.Fatalf("land %v, want %v", c.LandSeconds, LandSeconds)
		}
		if c.DustStart != DustStart {
			t.Fatalf("dust start %v, want %v", c.DustStart, DustStart)
		}
		if c.DustRun != DustRun {
			t.Fatalf("dust run %v, want %v", c.DustRun, DustRun)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
	})
	t.Run("happy: Nudge walks the selected knob by 50ms and stays on the grid", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobLand, 1)
		if got := c.LandSeconds; math.Abs(got-(LandSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("land after +50ms is %v, want %v", got, LandSeconds+StepSeconds)
		}
		c.Nudge(KnobDustStart, -1)
		if got := c.DustStart; math.Abs(got-(DustStart-StepSeconds)) > 1e-9 {
			t.Fatalf("dust start after -50ms is %v, want %v", got, DustStart-StepSeconds)
		}
		c.Nudge(KnobDustRun, 1)
		if got := c.DustRun; math.Abs(got-(DustRun+StepSeconds)) > 1e-9 {
			t.Fatalf("dust run after +50ms is %v, want %v", got, DustRun+StepSeconds)
		}
		for i := 0; i < 22; i++ {
			c.Nudge(KnobLand, 1)
		}
		want := LandSeconds + 23*StepSeconds
		if math.Abs(c.LandSeconds-want) > 1e-12 {
			t.Fatalf("land after 23 steps is %v, want %v on the 50ms grid", c.LandSeconds, want)
		}
	})
	t.Run("unhappy: Nudge will not walk a knob below its floor, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.DustStart = 0
		c.Nudge(KnobDustStart, -1)
		if c.DustStart != 0 {
			t.Fatalf("dust start %v, want 0 — it cannot go negative", c.DustStart)
		}
		c.LandSeconds = StepSeconds
		c.Nudge(KnobLand, -1)
		if c.LandSeconds != StepSeconds {
			t.Fatalf("land %v, want the 50ms floor", c.LandSeconds)
		}
		c.DustRun = 0
		c.Nudge(KnobDustRun, -1)
		if c.DustRun != 0 {
			t.Fatalf("dust run %v, want 0 — a silent cloud is allowed", c.DustRun)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips the three knobs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "landing.json")
		c := DefaultConfig()
		c.LandSeconds = 4.25
		c.DustStart = 2.0
		c.DustRun = 1.5
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if math.Abs(got.LandSeconds-c.LandSeconds) > 1e-9 ||
			math.Abs(got.DustStart-c.DustStart) > 1e-9 ||
			math.Abs(got.DustRun-c.DustRun) > 1e-9 {
			t.Fatalf("round-trip %+v, want %+v", got, c)
		}
	})
	t.Run("happy: LoadOrDefault is stock when the file is missing, and Use is that config", func(t *testing.T) {
		t.Cleanup(Reset)
		c, err := LoadOrDefault(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatalf("a missing file must keep stock: %v", err)
		}
		if c != DefaultConfig() {
			t.Fatalf("LoadOrDefault %+v, want stock", c)
		}
		live := DefaultConfig()
		live.LandSeconds = 4.25
		if err := Use(live); err != nil {
			t.Fatal(err)
		}
		if Active() != live {
			t.Fatalf("Active %+v, want %+v", Active(), live)
		}
	})
	t.Run("unhappy: missing, broken, and out-of-range files error, and Save/Use refuse a bad knob", func(t *testing.T) {
		t.Cleanup(Reset)
		if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Fatal("a missing file must error")
		}
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(bad); err == nil {
			t.Fatal("broken JSON must error")
		}
		out := filepath.Join(t.TempDir(), "out.json")
		if err := os.WriteFile(out, []byte(`{"landSeconds":0,"dustStart":0,"dustRun":1}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("a land duration below 50ms must error")
		}
		neg := DefaultConfig()
		neg.DustStart = -0.1
		if err := neg.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a negative dust start")
		}
		before := Active()
		badUse := DefaultConfig()
		badUse.LandSeconds = 0
		if err := Use(badUse); err == nil {
			t.Fatal("Use must reject a land duration below 50ms")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
	})
}
