package prog

// Tests written FIRST: Config is the seven live knobs on the program-
// alarm drop — four fall segments and three holds, 50ms at a time.
// The codes themselves are not knobs: historically 1202, 1202, then
// 1201. Play rebuilds from the current knobs. Save/Load round-trip
// the JSON; Use is what New plays on the first curtain.

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs are four drops and three holds, and the codes are 1202, 1202, 1201", func(t *testing.T) {
		c := DefaultConfig()
		if c.Drop1 != Drop1 || c.Drop2 != Drop2 || c.Drop3 != Drop3 || c.Drop4 != Drop4 {
			t.Fatalf("drops %+v, want %v %v %v %v", c, Drop1, Drop2, Drop3, Drop4)
		}
		if c.Hold1 != Hold1 || c.Hold2 != Hold2 || c.Hold3 != Hold3 {
			t.Fatalf("holds %+v, want %v %v %v", c, Hold1, Hold2, Hold3)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
		if KnobCount != 7 {
			t.Fatalf("KnobCount %d, want 7 (four drops, three holds)", KnobCount)
		}
		if got := Codes(); len(got) != 3 || got[0] != "1202" || got[1] != "1202" || got[2] != "1201" {
			t.Fatalf("Codes %v, want 1202, 1202, 1201 — the first three flight alarms", got)
		}
		want := map[Knob]string{
			KnobDrop1: "drop 1", KnobHold1: "hold 1",
			KnobDrop2: "drop 2", KnobHold2: "hold 2",
			KnobDrop3: "drop 3", KnobHold3: "hold 3",
			KnobDrop4: "drop 4",
		}
		for k, label := range want {
			if KnobLabel(k) != label {
				t.Fatalf("knob %d is %q, want %q", k, KnobLabel(k), label)
			}
		}
		if KnobLabel(KnobCount) != "" || KnobLabel(-1) != "" {
			t.Fatal("a knob off the panel has no label")
		}
	})
	t.Run("happy: Nudge walks the selected knob 50ms", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobDrop1, 1)
		if math.Abs(c.Drop1-(Drop1+StepSeconds)) > 1e-9 {
			t.Fatalf("drop 1 after +50ms is %v, want %v", c.Drop1, Drop1+StepSeconds)
		}
		c.Nudge(KnobHold2, -1)
		if math.Abs(c.Hold2-(Hold2-StepSeconds)) > 1e-9 {
			t.Fatalf("hold 2 after -50ms is %v, want %v", c.Hold2, Hold2-StepSeconds)
		}
		c.Nudge(KnobDrop4, 1)
		if math.Abs(c.Drop4-(Drop4+StepSeconds)) > 1e-9 {
			t.Fatalf("drop 4 after +50ms is %v, want %v", c.Drop4, Drop4+StepSeconds)
		}
	})
	t.Run("unhappy: a drop will not walk below 50ms, a hold will not go negative, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.Drop1 = StepSeconds
		c.Nudge(KnobDrop1, -1)
		if c.Drop1 != StepSeconds {
			t.Fatalf("drop 1 %v, want the 50ms floor", c.Drop1)
		}
		c.Hold1 = 0
		c.Nudge(KnobHold1, -1)
		if c.Hold1 != 0 {
			t.Fatalf("hold 1 %v, want 0 — a silent pause is allowed", c.Hold1)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		c.Nudge(KnobDrop2, 0)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips the seven knobs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "prog.json")
		c := DefaultConfig()
		c.Drop1, c.Hold1 = 1.0, 0.5
		c.Drop2, c.Hold2 = 1.1, 0.6
		c.Drop3, c.Hold3 = 1.2, 0.7
		c.Drop4 = 1.3
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
		live.Hold3 = 2.0
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
		if err := os.WriteFile(out, []byte(`{"drop1":0,"drop2":1,"drop3":1,"drop4":1,"hold1":0,"hold2":0,"hold3":0}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("a drop duration below 50ms must error")
		}
		neg := DefaultConfig()
		neg.Hold2 = -0.1
		if err := neg.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a negative hold")
		}
		before := Active()
		badUse := DefaultConfig()
		badUse.Drop4 = 0
		if err := Use(badUse); err == nil {
			t.Fatal("Use must reject a drop below 50ms")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
	})
}
