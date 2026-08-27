package america

// Tests written FIRST: Config is the three live knobs — how long the
// flag takes to fade in from black, when the eagle enters, and how
// long its crossing takes (the eagle's speed) — nudged 50ms at a
// time, the same way the landing scene tunes. Play rebuilds the scene
// from the current knobs so iteration does not require a restart.
// Save/Load round-trip the JSON next to the scene; Use is what New
// plays on the first curtain; a file missing a key keeps that knob at
// stock.

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs match the scene defaults", func(t *testing.T) {
		c := DefaultConfig()
		if c.FadeSeconds != FadeSeconds {
			t.Fatalf("fade %v, want %v", c.FadeSeconds, FadeSeconds)
		}
		if c.EagleDelay != FadeSeconds {
			t.Fatalf("eagle delay %v, want %v — the eagle enters when the fade lands", c.EagleDelay, FadeSeconds)
		}
		if c.CrossSeconds != CrossSeconds {
			t.Fatalf("cross %v, want %v", c.CrossSeconds, CrossSeconds)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
		if KnobCount != 3 {
			t.Fatalf("KnobCount %d, want 3 (flag fade, eagle delay, eagle cross)", KnobCount)
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
		if got := c.Value(KnobFade); got != c.FadeSeconds {
			t.Fatalf("Value(fade) %v, want %v", got, c.FadeSeconds)
		}
		if got := c.Value(KnobDelay); got != c.EagleDelay {
			t.Fatalf("Value(delay) %v, want %v", got, c.EagleDelay)
		}
		if got := c.Value(KnobCross); got != c.CrossSeconds {
			t.Fatalf("Value(cross) %v, want %v", got, c.CrossSeconds)
		}
		if KnobLabel(KnobCount) != "" || c.Value(KnobCount) != 0 {
			t.Fatal("an off-panel knob has no label and no value")
		}
	})
	t.Run("happy: Nudge walks the selected knob by 50ms and stays on the grid", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobFade, 1)
		if got := c.FadeSeconds; math.Abs(got-(FadeSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("fade after +50ms is %v, want %v", got, FadeSeconds+StepSeconds)
		}
		c.Nudge(KnobDelay, -1)
		if got := c.EagleDelay; math.Abs(got-(FadeSeconds-StepSeconds)) > 1e-9 {
			t.Fatalf("delay after -50ms is %v, want %v", got, FadeSeconds-StepSeconds)
		}
		c.Nudge(KnobCross, 1)
		if got := c.CrossSeconds; math.Abs(got-(CrossSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("cross after +50ms is %v, want %v", got, CrossSeconds+StepSeconds)
		}
		for i := 0; i < 23; i++ {
			c.Nudge(KnobCross, 1)
		}
		want := CrossSeconds + 24*StepSeconds
		if math.Abs(c.CrossSeconds-want) > 1e-12 {
			t.Fatalf("cross after 24 steps is %v, want %v on the 50ms grid", c.CrossSeconds, want)
		}
	})
	t.Run("unhappy: Nudge will not walk a knob below its floor, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.FadeSeconds = 0
		c.Nudge(KnobFade, -1)
		if c.FadeSeconds != 0 {
			t.Fatalf("fade %v, want 0 — an instant flag is allowed", c.FadeSeconds)
		}
		c.EagleDelay = 0
		c.Nudge(KnobDelay, -1)
		if c.EagleDelay != 0 {
			t.Fatalf("delay %v, want 0 — the eagle may enter at t=0", c.EagleDelay)
		}
		c.CrossSeconds = StepSeconds
		c.Nudge(KnobCross, -1)
		if c.CrossSeconds != StepSeconds {
			t.Fatalf("cross %v, want the 50ms floor — the crossing must be a duration", c.CrossSeconds)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips the three knobs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "america.json")
		c := DefaultConfig()
		c.FadeSeconds = 2.5
		c.EagleDelay = 3.0
		c.CrossSeconds = 6.25
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if math.Abs(got.FadeSeconds-c.FadeSeconds) > 1e-9 ||
			math.Abs(got.EagleDelay-c.EagleDelay) > 1e-9 ||
			math.Abs(got.CrossSeconds-c.CrossSeconds) > 1e-9 {
			t.Fatalf("round-trip %+v, want %+v", got, c)
		}
	})
	t.Run("happy: a file missing a key keeps that knob at stock", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "partial.json")
		if err := os.WriteFile(path, []byte(`{"fadeSeconds":2.0}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.FadeSeconds != 2.0 {
			t.Fatalf("fade %v, want the file's 2.0", got.FadeSeconds)
		}
		if got.EagleDelay != FadeSeconds || got.CrossSeconds != CrossSeconds {
			t.Fatalf("missing keys loaded %+v, want stock delay and cross", got)
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
		live.CrossSeconds = 4.0
		if err := Use(live); err != nil {
			t.Fatal(err)
		}
		if Active() != live {
			t.Fatalf("Active %+v, want %+v", Active(), live)
		}
		Reset()
		if Active() != DefaultConfig() {
			t.Fatalf("Reset left %+v, want stock", Active())
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
		if err := os.WriteFile(out, []byte(`{"crossSeconds":0}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("a crossing below 50ms must error")
		}
		negFade := DefaultConfig()
		negFade.FadeSeconds = -0.1
		if err := negFade.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a negative fade")
		}
		negDelay := DefaultConfig()
		negDelay.EagleDelay = -0.1
		if err := negDelay.Save(filepath.Join(t.TempDir(), "y.json")); err == nil {
			t.Fatal("Save must refuse a negative delay")
		}
		before := Active()
		badUse := DefaultConfig()
		badUse.CrossSeconds = 0
		if err := Use(badUse); err == nil {
			t.Fatal("Use must reject a crossing below 50ms")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
	})
}
