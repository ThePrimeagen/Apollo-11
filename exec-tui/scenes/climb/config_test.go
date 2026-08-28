package climb

// Tests written FIRST: Config is the one live knob on the spacelander
// climb — how long the north-facing craft takes to rise from off the
// bottom to off the top. Nudged 50ms at a time. Play rebuilds from
// the current knobs. Save/Load round-trip the JSON; Use is what New
// plays on the first curtain. A missing file is stock; a broken one
// is an error.

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock climb matches the lander's fall duration, run the other way", func(t *testing.T) {
		c := DefaultConfig()
		if c.ClimbSeconds != lander.DropSeconds {
			t.Fatalf("climb %v, want the spacelander's %v", c.ClimbSeconds, lander.DropSeconds)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
		if KnobCount != 1 {
			t.Fatalf("KnobCount %d, want 1 (climb)", KnobCount)
		}
		if KnobLabel(KnobClimb) != "climb" {
			t.Fatalf("knob label %q, want climb", KnobLabel(KnobClimb))
		}
		if c.Value(KnobClimb) != c.ClimbSeconds {
			t.Fatalf("Value(climb) %v, want %v", c.Value(KnobClimb), c.ClimbSeconds)
		}
	})
	t.Run("happy: Nudge walks the climb 50ms and stays on the grid", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobClimb, 1)
		if math.Abs(c.ClimbSeconds-(lander.DropSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("climb after +50ms is %v, want %v", c.ClimbSeconds, lander.DropSeconds+StepSeconds)
		}
		c.Nudge(KnobClimb, -1)
		if math.Abs(c.ClimbSeconds-lander.DropSeconds) > 1e-9 {
			t.Fatalf("climb after -50ms is %v, want stock %v", c.ClimbSeconds, lander.DropSeconds)
		}
	})
	t.Run("unhappy: Nudge will not walk below 50ms, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.ClimbSeconds = StepSeconds
		c.Nudge(KnobClimb, -1)
		if c.ClimbSeconds != StepSeconds {
			t.Fatalf("climb %v, want the 50ms floor", c.ClimbSeconds)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		c.Nudge(KnobClimb, 0)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips the climb", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "climb.json")
		c := DefaultConfig()
		c.ClimbSeconds = 4.25
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if math.Abs(got.ClimbSeconds-c.ClimbSeconds) > 1e-9 {
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
		live.ClimbSeconds = 3.5
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
		if err := os.WriteFile(out, []byte(`{"climbSeconds":0}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("a climb duration below 50ms must error")
		}
		neg := DefaultConfig()
		neg.ClimbSeconds = 0
		if err := neg.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a climb below 50ms")
		}
		before := Active()
		if err := Use(neg); err == nil {
			t.Fatal("Use must reject a climb below 50ms")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
	})
}
