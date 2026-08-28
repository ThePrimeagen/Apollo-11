package shootingstar

// Tests written FIRST: Config is the live knobs of the shooting-star
// scene — preview path (circle or square, so the tail is readable),
// star size (or random), flight speed, and the persist-trail knobs
// (count, period, min/max life, nozzle, peak, taper). Peak steepens
// the slit onto the spine; taper cuts max life by |offset|. The
// standalone runner walks them live; s writes this JSON next to the
// scene. The scene itself always falls right-to-left (high right to
// low left). Fall is the stock tuner path; circle and square stay as
// optional tail-reading loops.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/bigstar"
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
		if KnobCount != 11 {
			t.Fatalf("KnobCount %d, want 11 (path, size, random size, speed, count, period, min life, max life, nozzle, peak, taper)", KnobCount)
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
	})
	t.Run("unhappy: size stays in 1..MaxSize, count/life/nozzle will not go negative, and a bad cursor is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.Size = bigstar.MaxSize
		c.Nudge(KnobSize, 1)
		if c.Size != bigstar.MaxSize {
			t.Fatalf("size %d, want the MaxSize ceiling", c.Size)
		}
		c.Size = bigstar.MinSize
		c.Nudge(KnobSize, -1)
		if c.Size != bigstar.MinSize {
			t.Fatalf("size %d, want the MinSize floor", c.Size)
		}
		c.Count = 1
		c.Nudge(KnobSpawn, -1)
		if c.Count != 1 {
			t.Fatalf("count %d, want the 1 floor", c.Count)
		}
		c.MinLife = StepLife
		c.MaxLife = StepLife
		c.Nudge(KnobMinLife, -1)
		if c.MinLife != StepLife {
			t.Fatalf("min life %v, want the life floor", c.MinLife)
		}
		c.Nozzle = 0
		c.Nudge(KnobNozzle, -1)
		if c.Nozzle != 0 {
			t.Fatalf("nozzle %v, want the 0 floor", c.Nozzle)
		}
		c.Peak = 1
		c.Nudge(KnobPeak, -1)
		if c.Peak != 1 {
			t.Fatalf("peak %v, want the 1 floor — Peak<=1 is uniform", c.Peak)
		}
		c.Taper = 0
		c.Nudge(KnobTaper, -1)
		if c.Taper != 0 {
			t.Fatalf("taper %v, want the 0 floor", c.Taper)
		}
		c.Taper = 1
		c.Nudge(KnobTaper, 1)
		if c.Taper != 1 {
			t.Fatalf("taper %v, want the 1 ceiling", c.Taper)
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
		if err := os.WriteFile(out, []byte(`{"path":"diagonal"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("an unknown path must error")
		}
		neg := DefaultConfig()
		neg.Speed = -1
		if err := neg.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a negative speed")
		}
		before := Active()
		badUse := DefaultConfig()
		badUse.Size = 0
		if err := Use(badUse); err == nil {
			t.Fatal("Use must reject size 0")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
		wide := DefaultConfig()
		wide.Taper = 2
		if err := Use(wide); err == nil {
			t.Fatal("Use must reject a taper past 1")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected taper is %+v, want %+v", Active(), before)
		}
	})
}
