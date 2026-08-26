package landing

// Tests written FIRST: Config is the seven live knobs — land duration,
// dust start offset, dust run, and the four booster-stage offsets from
// t=0 — nudged 50ms at a time. Play rebuilds the landing from the
// current knobs so iteration does not require a restart. Save/Load
// round-trip the JSON; Use is what New and the walkthrough play on the
// first curtain. An older file without fire keys keeps the stock fire.

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
		if c.Fire75 != Fire75 || c.Fire50 != Fire50 || c.Fire25 != Fire25 || c.FireOff != FireOff {
			t.Fatalf("fire offsets %+v, want ¾=%v ½=%v ¼=%v off=%v", c, Fire75, Fire50, Fire25, FireOff)
		}
		if c.DustLoss != DustLoss {
			t.Fatalf("dust loss %v, want %v particles/ms", c.DustLoss, DustLoss)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
		if StepLoss != 0.005 {
			t.Fatalf("loss step %v, want 0.005/ms", StepLoss)
		}
		if KnobCount != 8 {
			t.Fatalf("KnobCount %d, want 8 (land, dust, loss, four fire stages)", KnobCount)
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
		c.Nudge(KnobDustLoss, 1)
		if got := c.DustLoss; math.Abs(got-(DustLoss+StepLoss)) > 1e-9 {
			t.Fatalf("dust loss after +1 step is %v, want %v", got, DustLoss+StepLoss)
		}
		c.Nudge(KnobFire75, 1)
		if got := c.Fire75; math.Abs(got-(Fire75+StepSeconds)) > 1e-9 {
			t.Fatalf("fire ¾ after +50ms is %v, want %v", got, Fire75+StepSeconds)
		}
		c.Nudge(KnobFireOff, -1)
		if got := c.FireOff; math.Abs(got-(FireOff-StepSeconds)) > 1e-9 {
			t.Fatalf("fire off after -50ms is %v, want %v", got, FireOff-StepSeconds)
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
		c.DustLoss = 0
		c.Nudge(KnobDustLoss, -1)
		if c.DustLoss != 0 {
			t.Fatalf("dust loss %v, want 0 — a silent drain is allowed", c.DustLoss)
		}
		c.Fire75 = 0
		c.Nudge(KnobFire75, -1)
		if c.Fire75 != 0 {
			t.Fatalf("fire ¾ %v, want 0 — a stage may kick at t=0", c.Fire75)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips the seven knobs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "landing.json")
		c := DefaultConfig()
		c.LandSeconds = 4.25
		c.DustStart = 2.0
		c.DustRun = 1.5
		c.Fire75 = 1.0
		c.Fire50 = 2.0
		c.Fire25 = 3.0
		c.FireOff = 4.0
		c.DustLoss = 0.075
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if math.Abs(got.LandSeconds-c.LandSeconds) > 1e-9 ||
			math.Abs(got.DustStart-c.DustStart) > 1e-9 ||
			math.Abs(got.DustRun-c.DustRun) > 1e-9 ||
			math.Abs(got.Fire75-c.Fire75) > 1e-9 ||
			math.Abs(got.Fire50-c.Fire50) > 1e-9 ||
			math.Abs(got.Fire25-c.Fire25) > 1e-9 ||
			math.Abs(got.FireOff-c.FireOff) > 1e-9 ||
			math.Abs(got.DustLoss-c.DustLoss) > 1e-9 {
			t.Fatalf("round-trip %+v, want %+v", got, c)
		}
	})
	t.Run("happy: a file without fire keys keeps the stock fire offsets", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "old.json")
		if err := os.WriteFile(path, []byte(`{"landSeconds":4.25,"dustStart":2.0,"dustRun":1.5}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.Fire75 != Fire75 || got.Fire50 != Fire50 || got.Fire25 != Fire25 || got.FireOff != FireOff {
			t.Fatalf("missing fire keys loaded %+v, want stock fire offsets", got)
		}
		if got.DustLoss != DustLoss {
			t.Fatalf("missing dust-loss key loaded %v, want stock %v", got.DustLoss, DustLoss)
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
		negFire := DefaultConfig()
		negFire.Fire75 = -0.1
		if err := negFire.Save(filepath.Join(t.TempDir(), "y.json")); err == nil {
			t.Fatal("Save must refuse a negative fire offset")
		}
		negLoss := DefaultConfig()
		negLoss.DustLoss = -0.1
		if err := negLoss.Save(filepath.Join(t.TempDir(), "z.json")); err == nil {
			t.Fatal("Save must refuse a negative particle loss")
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
