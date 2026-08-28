package bobble

// Tests written FIRST: Config is the three live knobs of the bobble
// scene — the engine (on or off), the bobble period in seconds, and
// the bobble amplitude in cells. The period walks 50ms at a time, the
// amplitude one cell, and the engine flips on with l and off with h,
// the skies way. Save/Load round-trip the JSON next to the scene; Use
// is what New plays on the first curtain; a file missing keys keeps
// the stock values for them.

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs are the premiere's parked ride, engine lit", func(t *testing.T) {
		c := DefaultConfig()
		if !c.Engine {
			t.Fatal("the stock bobble burns its tail fire")
		}
		if c.PeriodSeconds != lander.BobPeriodSeconds {
			t.Fatalf("period %v, want the stock %v", c.PeriodSeconds, lander.BobPeriodSeconds)
		}
		if c.AmplitudeCells != lander.BobAmplitudeCells {
			t.Fatalf("amplitude %v, want the stock %v", c.AmplitudeCells, lander.BobAmplitudeCells)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
		if KnobCount != 3 {
			t.Fatalf("KnobCount %d, want 3 (engine, period, amplitude)", KnobCount)
		}
	})
	t.Run("happy: every knob has its own label", func(t *testing.T) {
		seen := map[string]bool{}
		for k := Knob(0); k < KnobCount; k++ {
			l := KnobLabel(k)
			if l == "" {
				t.Fatalf("knob %d has no label", k)
			}
			if seen[l] {
				t.Fatalf("label %q repeats", l)
			}
			seen[l] = true
		}
		if KnobLabel(KnobCount) != "" {
			t.Fatal("a knob off the panel has no label")
		}
	})
	t.Run("happy: Nudge flips the engine and walks the ride", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobEngine, -1)
		if c.Engine {
			t.Fatal("h must switch the engine off")
		}
		c.Nudge(KnobEngine, 1)
		if !c.Engine {
			t.Fatal("l must switch the engine on")
		}
		c.Nudge(KnobPeriod, 1)
		if got := c.PeriodSeconds; math.Abs(got-(lander.BobPeriodSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("period after +50ms is %v, want %v", got, lander.BobPeriodSeconds+StepSeconds)
		}
		c.Nudge(KnobAmplitude, 1)
		if c.AmplitudeCells != lander.BobAmplitudeCells+1 {
			t.Fatalf("amplitude after +1 is %v, want %v", c.AmplitudeCells, lander.BobAmplitudeCells+1)
		}
		if got := c.Value(KnobEngine); got != 1 {
			t.Fatalf("a lit engine reads %v, want 1", got)
		}
	})
	t.Run("unhappy: the period floor is 50ms, the amplitude floor is zero, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.PeriodSeconds = StepSeconds
		c.Nudge(KnobPeriod, -1)
		if c.PeriodSeconds != StepSeconds {
			t.Fatalf("period %v, want the 50ms floor", c.PeriodSeconds)
		}
		c.AmplitudeCells = 0
		c.Nudge(KnobAmplitude, -1)
		if c.AmplitudeCells != 0 {
			t.Fatalf("amplitude %v, want 0 — a level hover is allowed", c.AmplitudeCells)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips the three knobs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bobble.json")
		c := DefaultConfig()
		c.Engine = false
		c.PeriodSeconds = 4.25
		c.AmplitudeCells = 3
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got != c {
			t.Fatalf("round-trip %+v, want %+v", got, c)
		}
	})
	t.Run("happy: a file missing keys keeps the stock values for them", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "old.json")
		if err := os.WriteFile(path, []byte(`{"periodSeconds":4.0}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.PeriodSeconds != 4.0 {
			t.Fatalf("present keys must load, got %+v", got)
		}
		if !got.Engine || got.AmplitudeCells != lander.BobAmplitudeCells {
			t.Fatalf("missing keys loaded %+v, want the stock engine and amplitude", got)
		}
	})
	t.Run("happy: LoadOrDefault is stock when the file is missing, and Use is what Active hands out", func(t *testing.T) {
		t.Cleanup(Reset)
		c, err := LoadOrDefault(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatalf("a missing file must keep stock: %v", err)
		}
		if c != DefaultConfig() {
			t.Fatalf("LoadOrDefault %+v, want stock", c)
		}
		live := DefaultConfig()
		live.Engine = false
		live.PeriodSeconds = 4.0
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
		if err := os.WriteFile(out, []byte(`{"periodSeconds":0}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("a period below 50ms must error")
		}
		negAmp := DefaultConfig()
		negAmp.AmplitudeCells = -1
		if err := negAmp.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a negative amplitude")
		}
		before := Active()
		badUse := DefaultConfig()
		badUse.PeriodSeconds = 0
		if err := Use(badUse); err == nil {
			t.Fatal("Use must reject a period below 50ms")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
	})
}
