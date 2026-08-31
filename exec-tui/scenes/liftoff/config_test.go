package liftoff

// Tests written FIRST: Config is the nine live knobs of the liftoff
// scene — how long the climb takes (rise), when the craft leaves the
// pad (lift at), the four ignition offsets from t=0 (¼, ½, ¾, full),
// and the pad dust window (start, run, loss). Time knobs walk 50ms at
// a time, dust loss 0.005/ms. Save/Load round-trip the JSON next to
// the scene; Use is what New plays on the first curtain; a file
// missing keys keeps the stock values for them.

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs are the walkthrough played backwards", func(t *testing.T) {
		c := DefaultConfig()
		if c.RiseSeconds != RiseSeconds {
			t.Fatalf("rise %v, want %v", c.RiseSeconds, RiseSeconds)
		}
		if c.LiftAt != LiftAt {
			t.Fatalf("lift at %v, want %v", c.LiftAt, LiftAt)
		}
		if c.Fire25 != Fire25 || c.Fire50 != Fire50 || c.Fire75 != Fire75 || c.FireFull != FireFull {
			t.Fatalf("ignition offsets %+v, want ¼=%v ½=%v ¾=%v full=%v", c, Fire25, Fire50, Fire75, FireFull)
		}
		if c.DustStart != DustStart || c.DustRun != DustRun || c.DustLoss != DustLoss {
			t.Fatalf("dust %+v, want start=%v run=%v loss=%v", c, DustStart, DustRun, DustLoss)
		}
		if StepSeconds != 0.050 {
			t.Fatalf("step %v, want 50ms", StepSeconds)
		}
		if StepLoss != 0.005 {
			t.Fatalf("loss step %v, want 0.005/ms", StepLoss)
		}
		if KnobCount != 9 {
			t.Fatalf("KnobCount %d, want 9 (rise, lift at, four ignition stages, three dust knobs)", KnobCount)
		}
		if c.WhiteOnly {
			t.Fatal("stock liftoff is the full hull — WhiteOnly is a MAIN knob, default false")
		}
	})
	t.Run("happy: the stock ignition is an ordered ramp that ends at liftoff", func(t *testing.T) {
		c := DefaultConfig()
		if !(c.Fire25 < c.Fire50 && c.Fire50 < c.Fire75 && c.Fire75 < c.FireFull) {
			t.Fatalf("the stock ignition must step up in order: %v %v %v %v", c.Fire25, c.Fire50, c.Fire75, c.FireFull)
		}
		if c.FireFull > c.LiftAt {
			t.Fatalf("stock full power at %v must not come after liftoff at %v", c.FireFull, c.LiftAt)
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
	t.Run("happy: Nudge walks the selected knob by 50ms and stays on the grid", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobRise, 1)
		if got := c.RiseSeconds; math.Abs(got-(RiseSeconds+StepSeconds)) > 1e-9 {
			t.Fatalf("rise after +50ms is %v, want %v", got, RiseSeconds+StepSeconds)
		}
		c.Nudge(KnobLiftAt, -1)
		if got := c.LiftAt; math.Abs(got-(LiftAt-StepSeconds)) > 1e-9 {
			t.Fatalf("lift at after -50ms is %v, want %v", got, LiftAt-StepSeconds)
		}
		c.Nudge(KnobFire25, 1)
		if got := c.Fire25; math.Abs(got-(Fire25+StepSeconds)) > 1e-9 {
			t.Fatalf("fire ¼ after +50ms is %v, want %v", got, Fire25+StepSeconds)
		}
		c.Nudge(KnobFireFull, -1)
		if got := c.FireFull; math.Abs(got-(FireFull-StepSeconds)) > 1e-9 {
			t.Fatalf("fire full after -50ms is %v, want %v", got, FireFull-StepSeconds)
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
		for i := 0; i < 22; i++ {
			c.Nudge(KnobRise, 1)
		}
		want := RiseSeconds + 23*StepSeconds
		if math.Abs(c.RiseSeconds-want) > 1e-12 {
			t.Fatalf("rise after 23 steps is %v, want %v on the 50ms grid", c.RiseSeconds, want)
		}
	})
	t.Run("unhappy: Nudge will not walk a knob below its floor, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.RiseSeconds = StepSeconds
		c.Nudge(KnobRise, -1)
		if c.RiseSeconds != StepSeconds {
			t.Fatalf("rise %v, want the 50ms floor", c.RiseSeconds)
		}
		c.LiftAt = 0
		c.Nudge(KnobLiftAt, -1)
		if c.LiftAt != 0 {
			t.Fatalf("lift at %v, want 0 — liftoff may open the scene but never precede it", c.LiftAt)
		}
		c.Fire25 = 0
		c.Nudge(KnobFire25, -1)
		if c.Fire25 != 0 {
			t.Fatalf("fire ¼ %v, want 0 — ignition may kick at t=0", c.Fire25)
		}
		c.DustStart = 0
		c.Nudge(KnobDustStart, -1)
		if c.DustStart != 0 {
			t.Fatalf("dust start %v, want 0", c.DustStart)
		}
		c.DustRun = 0
		c.Nudge(KnobDustRun, -1)
		if c.DustRun != 0 {
			t.Fatalf("dust run %v, want 0 — a dustless pad is allowed", c.DustRun)
		}
		c.DustLoss = 0
		c.Nudge(KnobDustLoss, -1)
		if c.DustLoss != 0 {
			t.Fatalf("dust loss %v, want 0 — a silent drain is allowed", c.DustLoss)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips the nine knobs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "liftoff.json")
		c := DefaultConfig()
		c.RiseSeconds = 4.25
		c.LiftAt = 1.0
		c.Fire25 = 0.2
		c.Fire50 = 0.45
		c.Fire75 = 0.7
		c.FireFull = 0.95
		c.DustStart = 0.5
		c.DustRun = 1.5
		c.DustLoss = 0.075
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		for k := Knob(0); k < KnobCount; k++ {
			if math.Abs(got.Value(k)-c.Value(k)) > 1e-9 {
				t.Fatalf("round-trip %s: %v, want %v", KnobLabel(k), got.Value(k), c.Value(k))
			}
		}
	})
	t.Run("happy: a file missing keys keeps the stock values for them", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "old.json")
		if err := os.WriteFile(path, []byte(`{"riseSeconds":4.0,"liftAt":2.0}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.RiseSeconds != 4.0 || got.LiftAt != 2.0 {
			t.Fatalf("present keys must load, got %+v", got)
		}
		if got.Fire25 != Fire25 || got.FireFull != FireFull {
			t.Fatalf("missing fire keys loaded %+v, want the stock ignition", got)
		}
		if got.DustStart != DustStart || got.DustRun != DustRun || got.DustLoss != DustLoss {
			t.Fatalf("missing dust keys loaded %+v, want the stock dust", got)
		}
		if got.WhiteOnly {
			t.Fatal("a file that does not name whiteOnly must keep stock false")
		}
	})
	t.Run("happy: WhiteOnly loads from JSON and a missing key stays stock false", func(t *testing.T) {
		on := filepath.Join(t.TempDir(), "white.json")
		if err := os.WriteFile(on, []byte(`{"whiteOnly":true,"dustRun":0}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(on)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if !got.WhiteOnly {
			t.Fatal("whiteOnly true must load")
		}
		if got.DustRun != 0 {
			t.Fatalf("dustRun %v, want 0 — a dustless pad is the operator's", got.DustRun)
		}
		if got.RiseSeconds != RiseSeconds {
			t.Fatalf("unnamed rise loaded %v, want stock", got.RiseSeconds)
		}
	})
	t.Run("unhappy: WhiteOnly false is stock, and a dustless white-only file still Validates", func(t *testing.T) {
		off := filepath.Join(t.TempDir(), "full.json")
		if err := os.WriteFile(off, []byte(`{"whiteOnly":false}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(off)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.WhiteOnly {
			t.Fatal("whiteOnly false must stay the full hull")
		}
		ok := DefaultConfig()
		ok.WhiteOnly = true
		ok.DustRun = 0
		if err := ok.Validate(); err != nil {
			t.Fatalf("white-only with no dust must be playable: %v", err)
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
		live.RiseSeconds = 4.25
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
		if err := os.WriteFile(out, []byte(`{"riseSeconds":0}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("a rise below 50ms must error")
		}
		negLift := DefaultConfig()
		negLift.LiftAt = -0.1
		if err := negLift.Save(filepath.Join(t.TempDir(), "w.json")); err == nil {
			t.Fatal("Save must refuse a negative lift-at")
		}
		negFire := DefaultConfig()
		negFire.Fire50 = -0.1
		if err := negFire.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a negative ignition offset")
		}
		negLoss := DefaultConfig()
		negLoss.DustLoss = -0.1
		if err := negLoss.Save(filepath.Join(t.TempDir(), "z.json")); err == nil {
			t.Fatal("Save must refuse a negative particle loss")
		}
		before := Active()
		badUse := DefaultConfig()
		badUse.RiseSeconds = 0
		if err := Use(badUse); err == nil {
			t.Fatal("Use must reject a rise below 50ms")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
	})
}
