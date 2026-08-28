package landing

// Tests written FIRST: Config is the live knobs — land duration,
// dust start offset, dust run, the four booster-stage offsets from
// t=0, the caption cues, and the scene's own copy of the shooting-
// star knobs so the landing meteor is editable right here, apart from
// every other scene the star appears in. Time knobs nudge 50ms at a
// time; the star knobs walk the shooting-star package's own steps.
// Play rebuilds the landing from the current knobs so iteration does
// not require a restart. Save/Load round-trip the JSON; Use is what
// New and the walkthrough play on the first curtain. An older file
// without fire keys keeps the stock fire, and one without a star
// section keeps the stock small meteor.

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/shootingstar"
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
		if KnobCount != 24 {
			t.Fatalf("KnobCount %d, want 24 (land, dust, loss, four fire stages, two 1202s, LAND, ten star knobs)", KnobCount)
		}
		if Code1At < 0 || Code2At <= Code1At || LandCaptionAt < Code2At {
			t.Fatalf("stock captions must run 1202, 1202, LAND — at %v then %v then %v", Code1At, Code2At, LandCaptionAt)
		}
		if c.Star != shootingstar.MeteorConfig() {
			t.Fatalf("the stock star is %+v, want the stock small meteor %+v", c.Star, shootingstar.MeteorConfig())
		}
	})
	t.Run("happy: every star knob wears a label and reads its value", func(t *testing.T) {
		c := DefaultConfig()
		want := map[Knob]struct {
			label string
			value float64
		}{
			KnobStarSize:       {"star size", float64(c.Star.Size)},
			KnobStarRandomSize: {"star random size", 0},
			KnobStarSpeed:      {"star speed", c.Star.Speed},
			KnobStarCount:      {"star count", float64(c.Star.Count)},
			KnobStarPeriod:     {"star period", c.Star.Period},
			KnobStarMinLife:    {"star min life", c.Star.MinLife},
			KnobStarMaxLife:    {"star max life", c.Star.MaxLife},
			KnobStarNozzle:     {"star nozzle", c.Star.Nozzle},
			KnobStarPeak:       {"star peak", c.Star.Peak},
			KnobStarTaper:      {"star taper", c.Star.Taper},
		}
		for k, w := range want {
			if KnobLabel(k) != w.label {
				t.Fatalf("knob %d is labeled %q, want %q", k, KnobLabel(k), w.label)
			}
			if c.Value(k) != w.value {
				t.Fatalf("knob %q reads %v, want %v", w.label, c.Value(k), w.value)
			}
		}
	})
	t.Run("unhappy: the tuner-only path knob stays off the panel, and star nudges never move the timing", func(t *testing.T) {
		for k := Knob(0); k < KnobCount; k++ {
			if KnobLabel(k) == "star path" {
				t.Fatal("the landing meteor has no path to pick — the path knob belongs to the shooting-star tuner alone")
			}
		}
		c := DefaultConfig()
		for i := 0; i < 40; i++ {
			c.Nudge(KnobStarSpeed, 1)
			c.Nudge(KnobStarSize, -1)
		}
		d := DefaultConfig()
		d.Star = c.Star
		if c != d {
			t.Fatalf("star nudges moved a timing knob: %+v", c)
		}
	})
	t.Run("happy: the star knobs walk the shooting-star steps", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobStarSpeed, 1)
		if got, want := c.Star.Speed, DefaultConfig().Star.Speed+shootingstar.StepSpeed; math.Abs(got-want) > 1e-9 {
			t.Fatalf("star speed after +1 is %v, want %v", got, want)
		}
		c.Nudge(KnobStarSize, 1)
		if got, want := c.Star.Size, DefaultConfig().Star.Size+1; got != want {
			t.Fatalf("star size after +1 is %v, want %v", got, want)
		}
		c.Nudge(KnobStarPeriod, -1)
		if got, want := c.Star.Period, DefaultConfig().Star.Period-shootingstar.StepPeriod; math.Abs(got-want) > 1e-9 {
			t.Fatalf("star period after -1 is %v, want %v", got, want)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("nudged star knobs must validate: %v", err)
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
	t.Run("happy: Save then Load round-trips the knobs, star section included", func(t *testing.T) {
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
		c.Star.Size = 2
		c.Star.Speed = 44
		c.Star.Taper = 0.5
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
		if got.Star != c.Star {
			t.Fatalf("round-trip lost the star: %+v vs %+v", got.Star, c.Star)
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
		if got.Star != shootingstar.MeteorConfig() {
			t.Fatalf("a file without a star section loaded %+v, want the stock small meteor", got.Star)
		}
	})
	t.Run("unhappy: a broken star section refuses to load, save, or play", func(t *testing.T) {
		t.Cleanup(Reset)
		bad := filepath.Join(t.TempDir(), "badstar.json")
		if err := os.WriteFile(bad, []byte(`{"landSeconds":4.25,"dustStart":2.0,"dustRun":1.5,"star":{"path":"zigzag"}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(bad); err == nil {
			t.Fatal("a broken star section must refuse to load")
		}
		c := DefaultConfig()
		c.Star.Path = "zigzag"
		never := filepath.Join(t.TempDir(), "never.json")
		if err := c.Save(never); err == nil {
			t.Fatal("a broken star must not save")
		}
		if _, err := os.Stat(never); !os.IsNotExist(err) {
			t.Fatal("a refused save must leave no file")
		}
		before := Active()
		if err := Use(c); err == nil {
			t.Fatal("Use must reject a broken star")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
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
	t.Run("happy: the caption knobs walk 50ms and round-trip", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobCode1At, 1)
		if math.Abs(c.Code1At-(Code1At+StepSeconds)) > 1e-9 {
			t.Fatalf("1202 a after +50ms is %v, want %v", c.Code1At, Code1At+StepSeconds)
		}
		c.Nudge(KnobLandCaptionAt, -1)
		if math.Abs(c.LandCaptionAt-(LandCaptionAt-StepSeconds)) > 1e-9 {
			t.Fatalf("LAND at after -50ms is %v, want %v", c.LandCaptionAt, LandCaptionAt-StepSeconds)
		}
		path := filepath.Join(t.TempDir(), "codes.json")
		c.Code1At, c.Code1Hold = 1.0, 0.5
		c.Code2At, c.Code2Hold = 2.0, 0.5
		c.LandCaptionAt, c.LandCaptionHold = 3.0, 1.0
		if err := c.Save(path); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got.Code1At-1.0) > 1e-9 || math.Abs(got.Code2At-2.0) > 1e-9 || math.Abs(got.LandCaptionAt-3.0) > 1e-9 {
			t.Fatalf("caption round-trip %+v", got)
		}
	})
	t.Run("happy: a file without caption keys keeps the stock 1202 / 1202 / LAND times", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "oldcaps.json")
		if err := os.WriteFile(path, []byte(`{"landSeconds":4.25,"dustStart":2.0,"dustRun":1.5}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Code1At != Code1At || got.Code2At != Code2At || got.LandCaptionAt != LandCaptionAt {
			t.Fatalf("missing caption keys loaded %+v, want stock 1202/1202/LAND times", got)
		}
	})
	t.Run("unhappy: a negative caption offset is refused, and Nudge will not walk a hold below zero", func(t *testing.T) {
		neg := DefaultConfig()
		neg.Code1At = -0.1
		if err := neg.Save(filepath.Join(t.TempDir(), "n.json")); err == nil {
			t.Fatal("Save must refuse a negative caption offset")
		}
		c := DefaultConfig()
		c.Code1Hold = 0
		c.Nudge(KnobCode1Hold, -1)
		if c.Code1Hold != 0 {
			t.Fatalf("hold a %v, want 0", c.Code1Hold)
		}
		if KnobLabel(KnobCode1At) != "1202 a" || KnobLabel(KnobCode2At) != "1202 b" || KnobLabel(KnobLandCaptionAt) != "LAND at" {
			t.Fatalf("caption labels %q %q %q", KnobLabel(KnobCode1At), KnobLabel(KnobCode2At), KnobLabel(KnobLandCaptionAt))
		}
	})
}
