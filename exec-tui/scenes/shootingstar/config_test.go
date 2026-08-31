package shootingstar

// Tests written FIRST: Config is the live knobs of the shooting-star
// scene — preview path (circle or square, so the tail is readable),
// star size (or random), flight speed, and the persist-trail knobs
// (count, period, min/max life, nozzle, peak, taper). Peak steepens
// the slit onto the spine; taper cuts max life by |offset|. Knobs
// are never clamped: size, speed, and the rest may be negative or
// past any old rail. The standalone runner walks them live; s writes
// this JSON next to the scene. The scene itself always falls
// right-to-left (high right to low left). Fall is the stock tuner
// path; circle and square stay as optional tail-reading loops.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/startrail"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs are a right-to-left fall over a size-2 star", func(t *testing.T) {
		c := DefaultConfig()
		if c.Path != PathFall {
			t.Fatalf("stock path %q, want fall — the scene and the tuner open right-to-left", c.Path)
		}
		if c.Size != 2 {
			t.Fatalf("stock size %d, want 2", c.Size)
		}
		if c.RandomSize {
			t.Fatal("stock size is set, not random")
		}
		if c.Speed <= 0 {
			t.Fatal("stock must fly")
		}
		if c.Count <= 0 || c.Period <= 0 || c.MinLife <= 0 || c.MaxLife < c.MinLife {
			t.Fatalf("stock trail %+v", c)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("stock must validate: %v", err)
		}
		if c.Delay != 0 {
			t.Fatalf("stock delay %v, want 0 — fly at once", c.Delay)
		}
		if c.StartY != 0 {
			t.Fatalf("stock startY %v, want 0 — the current path start", c.StartY)
		}
		if KnobCount != 13 {
			t.Fatalf("KnobCount %d, want 13 (path, size, random size, speed, count, period, min life, max life, nozzle, peak, taper, delay, start y)", KnobCount)
		}
		if DefaultConfigPath != "scenes/shootingstar/config.json" {
			t.Fatalf("DefaultConfigPath %q, want scenes/shootingstar/config.json", DefaultConfigPath)
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
	t.Run("happy: Nudge walks path fall/circle/square, size, random size, and the trail", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobPath, 1)
		if c.Path != PathCircle {
			t.Fatalf("path after +1 is %q, want circle", c.Path)
		}
		c.Nudge(KnobPath, 1)
		if c.Path != PathSquare {
			t.Fatalf("path after +1 again is %q, want square", c.Path)
		}
		c.Nudge(KnobPath, 1)
		if c.Path != PathFall {
			t.Fatalf("path wraps back to fall, got %q", c.Path)
		}
		c.Nudge(KnobPath, -1)
		if c.Path != PathSquare {
			t.Fatalf("path after -1 is %q, want square", c.Path)
		}
		c.Nudge(KnobSize, 1)
		if c.Size != 3 {
			t.Fatalf("size after +1 is %d, want 3", c.Size)
		}
		c.Nudge(KnobRandomSize, 1)
		if !c.RandomSize {
			t.Fatal("l must switch random size on")
		}
		c.Nudge(KnobRandomSize, -1)
		if c.RandomSize {
			t.Fatal("h must switch random size off")
		}
		before := c.Speed
		c.Nudge(KnobSpeed, 1)
		if c.Speed <= before {
			t.Fatal("speed must climb")
		}
		c.Nudge(KnobSpawn, 1)
		if c.Count != DefaultConfig().Count+1 {
			t.Fatalf("count after +1 is %d", c.Count)
		}
		c.Nudge(KnobPeak, 1)
		if c.Peak <= DefaultConfig().Peak {
			t.Fatal("peak must climb")
		}
		c.Nudge(KnobTaper, -1)
		if c.Taper >= DefaultConfig().Taper {
			t.Fatal("taper must drop")
		}
		if got := c.Value(KnobPath); got != 2 {
			t.Fatalf("square path reads %v, want 2", got)
		}
		c.Nudge(KnobDelay, 1)
		if got, want := c.Delay, DefaultConfig().Delay+StepDelay; mathAbs(got-want) > 1e-9 {
			t.Fatalf("delay after +1 is %v, want %v", got, want)
		}
		c.Nudge(KnobStartY, -1)
		if got, want := c.StartY, DefaultConfig().StartY-StepStartY; mathAbs(got-want) > 1e-9 {
			t.Fatalf("startY after -1 is %v, want %v", got, want)
		}
		if c.Value(KnobDelay) != c.Delay {
			t.Fatalf("Value(delay) %v, want %v", c.Value(KnobDelay), c.Delay)
		}
		if c.Value(KnobStartY) != c.StartY {
			t.Fatalf("Value(start y) %v, want %v", c.Value(KnobStartY), c.StartY)
		}
	})
	t.Run("unhappy: Nudge never clamps — size past 5, speed past 80 and through zero, count through zero, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.Size = 5
		c.Nudge(KnobSize, 1)
		if c.Size != 6 {
			t.Fatalf("size %d, want 6 — no ceiling", c.Size)
		}
		c.Size = 1
		c.Nudge(KnobSize, -1)
		if c.Size != 0 {
			t.Fatalf("size %d, want 0 — no floor", c.Size)
		}
		c.Nudge(KnobSize, -1)
		if c.Size != -1 {
			t.Fatalf("size %d, want -1", c.Size)
		}
		c.Speed = 80
		c.Nudge(KnobSpeed, 1)
		if c.Speed <= 80 {
			t.Fatalf("speed %v, want past 80 — no ceiling", c.Speed)
		}
		c.Speed = 0
		c.Nudge(KnobSpeed, -1)
		if c.Speed >= 0 {
			t.Fatalf("speed %v, want negative — no floor", c.Speed)
		}
		c.Count = 1
		c.Nudge(KnobSpawn, -1)
		if c.Count != 0 {
			t.Fatalf("count %d, want 0", c.Count)
		}
		c.Nudge(KnobSpawn, -1)
		if c.Count != -1 {
			t.Fatalf("count %d, want -1", c.Count)
		}
		c.Nozzle = 0
		c.Nudge(KnobNozzle, -1)
		if c.Nozzle >= 0 {
			t.Fatalf("nozzle %v, want negative", c.Nozzle)
		}
		c.Peak = 1
		c.Nudge(KnobPeak, -1)
		if c.Peak >= 1 {
			t.Fatalf("peak %v, want below 1", c.Peak)
		}
		c.Taper = 0
		c.Nudge(KnobTaper, -1)
		if c.Taper >= 0 {
			t.Fatalf("taper %v, want negative", c.Taper)
		}
		c.Taper = 1
		c.Nudge(KnobTaper, 1)
		if c.Taper <= 1 {
			t.Fatalf("taper %v, want past 1", c.Taper)
		}
		c.Delay = 0
		c.Nudge(KnobDelay, -1)
		if c.Delay >= 0 {
			t.Fatalf("delay %v, want negative — no floor", c.Delay)
		}
		c.StartY = 0
		c.Nudge(KnobStartY, -1)
		if c.StartY >= 0 {
			t.Fatalf("startY %v, want negative — no floor", c.StartY)
		}
		before := c
		c.Nudge(-1, 1)
		c.Nudge(99, 1)
		if c != before {
			t.Fatalf("a bad cursor must not change the knobs, got %+v", c)
		}
	})
	t.Run("happy: Save then Load round-trips every knob", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "shoot.json")
		c := DefaultConfig()
		c.Path = PathSquare
		c.Size = 4
		c.RandomSize = true
		c.Speed = 30
		c.Count = 6
		c.Period = 0.02
		c.MinLife, c.MaxLife = 0.3, 0.8
		c.Nozzle = 3
		c.Peak = 4
		c.Taper = 0.5
		c.Delay = 1.5
		c.StartY = 0.04
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
		if err := os.WriteFile(path, []byte(`{"size":3}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.Size != 3 {
			t.Fatalf("present keys must load, got %+v", got)
		}
		stock := DefaultConfig()
		if got.Path != stock.Path || got.Speed != stock.Speed || got.Count != stock.Count {
			t.Fatalf("missing keys loaded %+v, want stock path/speed/count", got)
		}
	})
	t.Run("happy: LoadOrDefault is stock when the file is missing, and Use is what Active hands out", func(t *testing.T) {
		t.Cleanup(Reset)
		t.Cleanup(startrail.Reset)
		c, err := LoadOrDefault(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatalf("a missing file must keep stock: %v", err)
		}
		if c != DefaultConfig() {
			t.Fatalf("LoadOrDefault %+v, want stock", c)
		}
		live := DefaultConfig()
		live.Path = PathSquare
		live.Size = 3
		if err := Use(live); err != nil {
			t.Fatal(err)
		}
		if Active() != live {
			t.Fatalf("Active %+v, want %+v", Active(), live)
		}
		trail := startrail.Active()
		if trail.Count != live.Count || trail.Period != live.Period {
			t.Fatalf("Use must push the trail knobs, startrail Active %+v", trail)
		}
		if trail.Peak != live.Peak || trail.Taper != live.Taper {
			t.Fatalf("Use must push peak/taper, startrail Active %+v", trail)
		}
	})
	t.Run("unhappy: missing, broken, and unknown-path files error; size and speed are never refused", func(t *testing.T) {
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
		if err := os.WriteFile(out, []byte(`{"path":"diagonal"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("an unknown path must error")
		}
		neg := DefaultConfig()
		neg.Speed = -1
		negPath := filepath.Join(t.TempDir(), "neg.json")
		if err := neg.Save(negPath); err != nil {
			t.Fatalf("Save must keep a negative speed: %v", err)
		}
		gotNeg, err := Load(negPath)
		if err != nil {
			t.Fatalf("Load must keep a negative speed: %v", err)
		}
		if gotNeg.Speed != -1 {
			t.Fatalf("loaded speed %v, want -1", gotNeg.Speed)
		}
		before := Active()
		wide := DefaultConfig()
		wide.Size = 9
		wide.Speed = -4
		if err := Use(wide); err != nil {
			t.Fatalf("Use must keep size 9 and speed -4: %v", err)
		}
		if Active().Size != 9 || Active().Speed != -4 {
			t.Fatalf("Active after an unclamped Use is %+v, want size 9 speed -4", Active())
		}
		_ = Use(before)
	})
}
