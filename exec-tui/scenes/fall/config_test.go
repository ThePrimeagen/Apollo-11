package fall

// Tests written FIRST: Config is the one live knob on the spacelander
// fall — how long the north-facing craft takes to drop from off the
// top to off the bottom. Nudged 50ms at a time. Play rebuilds from
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
	t.Run("happy: the stock drop matches the lander's fall duration", func(t *testing.T) {
		c := DefaultConfig()
		if c.DropSeconds != lander.DropSeconds {
			t.Fatalf("drop %v, want the spacelander's %v", c.DropSeconds, lander.DropSeconds)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
		if KnobCount != 4 {
			t.Fatalf("KnobCount %d, want 4 (drop and three MAIN alarm holds)", KnobCount)
		}
		if KnobLabel(KnobDrop) != "drop" {
			t.Fatalf("knob label %q, want drop", KnobLabel(KnobDrop))
		}
		if KnobLabel(KnobHold1) != "hold 1" || KnobLabel(KnobHold2) != "hold 2" || KnobLabel(KnobHold3) != "hold 3" {
			t.Fatalf("hold labels %q/%q/%q, want hold 1/2/3", KnobLabel(KnobHold1), KnobLabel(KnobHold2), KnobLabel(KnobHold3))
		}
		if c.Value(KnobDrop) != c.DropSeconds {
			t.Fatalf("Value(drop) %v, want %v", c.Value(KnobDrop), c.DropSeconds)
		}
		if c.Hold1 != 0 || c.Hold2 != 0 || c.Hold3 != 0 {
			t.Fatalf("stock holds %+v, want zero so walkthrough stays a plain fall", c)
		}
		if c.Armed() {
			t.Fatal("stock knobs must not arm the MAIN alarm overlay")
		}
	})
	t.Run("happy: Nudge walks the drop 50ms and stays on the grid", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobDrop, 1)
		if math.Abs(c.DropSeconds-(lander.DropSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("drop after +50ms is %v, want %v", c.DropSeconds, lander.DropSeconds+StepSeconds)
		}
		c.Nudge(KnobDrop, -1)
		if math.Abs(c.DropSeconds-lander.DropSeconds) > 1e-9 {
			t.Fatalf("drop after -50ms is %v, want stock %v", c.DropSeconds, lander.DropSeconds)
		}
	})
	t.Run("happy: Nudge walks a hold 50ms and will go negative — no floor", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobHold1, 2)
		if math.Abs(c.Hold1-2*StepSeconds) > 1e-9 {
			t.Fatalf("hold 1 after +100ms is %v, want %v", c.Hold1, 2*StepSeconds)
		}
		c.Hold2 = 0
		c.Nudge(KnobHold2, -1)
		if math.Abs(c.Hold2+StepSeconds) > 1e-9 {
			t.Fatalf("hold 2 after −50ms is %v, want %v — Nudge does not clamp", c.Hold2, -StepSeconds)
		}
		if !c.Armed() {
			t.Fatal("a positive hold must arm the MAIN overlay")
		}
	})
	t.Run("unhappy: Nudge will not walk below 50ms, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.DropSeconds = StepSeconds
		c.Nudge(KnobDrop, -1)
		if c.DropSeconds != StepSeconds {
			t.Fatalf("drop %v, want the 50ms floor", c.DropSeconds)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		c.Nudge(KnobDrop, 0)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips the drop", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "fall.json")
		c := DefaultConfig()
		c.DropSeconds = 4.25
		c.Hold1, c.Hold2, c.Hold3 = 0.8, 0.75, 0.7
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if math.Abs(got.DropSeconds-c.DropSeconds) > 1e-9 {
			t.Fatalf("round-trip drop %v, want %v", got.DropSeconds, c.DropSeconds)
		}
		if math.Abs(got.Hold1-c.Hold1) > 1e-9 || math.Abs(got.Hold2-c.Hold2) > 1e-9 || math.Abs(got.Hold3-c.Hold3) > 1e-9 {
			t.Fatalf("round-trip holds %+v, want %+v", got, c)
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
		live.DropSeconds = 3.5
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
		if err := os.WriteFile(out, []byte(`{"dropSeconds":0}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("a drop duration below 50ms must error")
		}
		neg := DefaultConfig()
		neg.DropSeconds = 0
		if err := neg.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a drop below 50ms")
		}
		before := Active()
		if err := Use(neg); err == nil {
			t.Fatal("Use must reject a drop below 50ms")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
	})
}
