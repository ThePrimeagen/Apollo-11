package explorer

// Tests written FIRST: the explorer scene's config is the four
// twinkle knobs — the cycle range and the fade range, each a min and
// a max in seconds — plus the scene's own copy of the shooting-star
// knobs, so the Big E's meteor is editable right here and can drift
// from the shooting-star scene's own tuning. The cycle knobs walk
// 250ms at a time, the fade knobs 50ms; every twinkle knob lives
// between the stars package's twinkle rails, and a pair can never
// cross. The star knobs walk the shooting-star package's own steps
// and never touch the twinkle. Use validates and makes the knobs
// active for New, and pushes the twinkle onto the stars package so
// the sky breathes them live. s writes the JSON next to the scene; a
// missing file is the stock config, a broken one is an error worth
// stopping for. LoadOrInherit is how the runner opens: a config file
// that carries no "star" section inherits the shooting-star scene's
// own tuned knobs instead of stock, so the Big E flies the star as
// edited until it saves a star of its own.

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/shootingstar"
)

func reset() {
	Reset()
	stars.ResetTwinkle()
	shootingstar.Reset()
}

func TestConfigDefaults(t *testing.T) {
	t.Cleanup(reset)
	t.Run("happy: the stock knobs validate and mirror the stock twinkle", func(t *testing.T) {
		d := DefaultConfig()
		if err := d.Validate(); err != nil {
			t.Fatalf("the stock config must validate: %v", err)
		}
		if d.Twinkle() != stars.DefaultTwinkle() {
			t.Fatalf("the stock knobs %+v must speak the stock twinkle %+v", d.Twinkle(), stars.DefaultTwinkle())
		}
		Reset()
		if Active() != DefaultConfig() {
			t.Fatalf("after Reset the active knobs are %+v", Active())
		}
	})
	t.Run("happy: every knob wears a label and reads a value", func(t *testing.T) {
		c := DefaultConfig()
		want := map[Knob]struct {
			label string
			value float64
		}{
			KnobMinCycle:       {"min cycle", c.MinCycleSeconds},
			KnobMaxCycle:       {"max cycle", c.MaxCycleSeconds},
			KnobMinFade:        {"min fade", c.MinFadeSeconds},
			KnobMaxFade:        {"max fade", c.MaxFadeSeconds},
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
		if KnobCount != 14 {
			t.Fatalf("the panel holds %d knobs, want the four twinkle ranges plus the ten star knobs", KnobCount)
		}
		for k, w := range want {
			if KnobLabel(k) != w.label {
				t.Fatalf("knob %d is labeled %q, want %q", k, KnobLabel(k), w.label)
			}
			if c.Value(k) != w.value {
				t.Fatalf("knob %q reads %v, want %v", w.label, c.Value(k), w.value)
			}
		}
		if KnobLabel(KnobCount) != "" || KnobLabel(-1) != "" {
			t.Fatal("a knob off the panel has no label")
		}
	})
	t.Run("happy: the stock star is the shooting-star scene's stock knobs", func(t *testing.T) {
		if DefaultConfig().Star != shootingstar.DefaultConfig() {
			t.Fatalf("the stock star is %+v, want the shooting-star stock %+v", DefaultConfig().Star, shootingstar.DefaultConfig())
		}
	})
	t.Run("unhappy: the tuner-only path knob stays off the panel — the Big E star flies one fixed crossing", func(t *testing.T) {
		for k := Knob(0); k < KnobCount; k++ {
			if KnobLabel(k) == "star path" {
				t.Fatal("the embedded star has no path to pick — the path knob belongs to the shooting-star tuner alone")
			}
		}
	})
}

func TestNudge(t *testing.T) {
	t.Run("happy: cycles walk 250ms, fades walk 50ms, both ways", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobMaxCycle, 1)
		if got, want := c.MaxCycleSeconds, DefaultConfig().MaxCycleSeconds+CycleStepSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("max cycle after +1 is %v, want %v", got, want)
		}
		c.Nudge(KnobMaxCycle, -1)
		if got, want := c.MaxCycleSeconds, DefaultConfig().MaxCycleSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("max cycle after -1 is %v, want %v", got, want)
		}
		c.Nudge(KnobMinFade, 1)
		if got, want := c.MinFadeSeconds, DefaultConfig().MinFadeSeconds+FadeStepSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("min fade after +1 is %v, want %v", got, want)
		}
	})
	t.Run("happy: a nudged config always validates — the knobs cannot leave the rails", func(t *testing.T) {
		c := DefaultConfig()
		for i := 0; i < 400; i++ {
			c.Nudge(KnobMinCycle, -1)
			c.Nudge(KnobMaxCycle, 1)
			c.Nudge(KnobMinFade, -1)
			c.Nudge(KnobMaxFade, 1)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the widest knobs must validate: %v", err)
		}
		if c.MinCycleSeconds != stars.MinTwinkleCycle || c.MaxCycleSeconds != stars.MaxTwinkleCycle {
			t.Fatalf("the cycle knobs must stop at the rails, got [%v, %v]", c.MinCycleSeconds, c.MaxCycleSeconds)
		}
		if c.MinFadeSeconds != stars.MinTwinkleFade || c.MaxFadeSeconds != stars.MaxTwinkleFade {
			t.Fatalf("the fade knobs must stop at the rails, got [%v, %v]", c.MinFadeSeconds, c.MaxFadeSeconds)
		}
	})
	t.Run("unhappy: a min never climbs past its max, and a max never dips under its min", func(t *testing.T) {
		c := DefaultConfig()
		for i := 0; i < 400; i++ {
			c.Nudge(KnobMinCycle, 1)
		}
		if c.MinCycleSeconds > c.MaxCycleSeconds {
			t.Fatalf("min cycle %v crossed max cycle %v", c.MinCycleSeconds, c.MaxCycleSeconds)
		}
		if c.MinCycleSeconds != c.MaxCycleSeconds {
			t.Fatalf("min cycle must clamp at its partner, stopped at %v under %v", c.MinCycleSeconds, c.MaxCycleSeconds)
		}
		for i := 0; i < 400; i++ {
			c.Nudge(KnobMaxFade, -1)
		}
		if c.MaxFadeSeconds < c.MinFadeSeconds {
			t.Fatalf("max fade %v crossed min fade %v", c.MaxFadeSeconds, c.MinFadeSeconds)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("clamped knobs must validate: %v", err)
		}
	})
	t.Run("unhappy: a bad cursor or a zero dir is a no-op", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobCount, 1)
		c.Nudge(-1, 1)
		c.Nudge(KnobMinCycle, 0)
		var ghost *Config
		ghost.Nudge(KnobMinCycle, 1)
		if c != DefaultConfig() {
			t.Fatalf("a refused nudge moved the knobs to %+v", c)
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
		c.Nudge(KnobStarRandomSize, 1)
		if !c.Star.RandomSize {
			t.Fatal("star random size after +1 must be on")
		}
		c.Nudge(KnobStarCount, -1)
		if got, want := c.Star.Count, DefaultConfig().Star.Count-1; got != want {
			t.Fatalf("star count after -1 is %v, want %v", got, want)
		}
		c.Nudge(KnobStarPeriod, 1)
		if got, want := c.Star.Period, DefaultConfig().Star.Period+shootingstar.StepPeriod; math.Abs(got-want) > 1e-9 {
			t.Fatalf("star period after +1 is %v, want %v", got, want)
		}
		c.Nudge(KnobStarTaper, -1)
		if got, want := c.Star.Taper, DefaultConfig().Star.Taper-shootingstar.StepTaper; math.Abs(got-want) > 1e-9 {
			t.Fatalf("star taper after -1 is %v, want %v", got, want)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("nudged star knobs must validate: %v", err)
		}
	})
	t.Run("unhappy: a star knob never moves the twinkle, and a twinkle knob never moves the star", func(t *testing.T) {
		c := DefaultConfig()
		for i := 0; i < 40; i++ {
			c.Nudge(KnobStarSpeed, 1)
			c.Nudge(KnobStarSize, -1)
			c.Nudge(KnobStarNozzle, 1)
		}
		d := DefaultConfig()
		if c.MinCycleSeconds != d.MinCycleSeconds || c.MaxCycleSeconds != d.MaxCycleSeconds ||
			c.MinFadeSeconds != d.MinFadeSeconds || c.MaxFadeSeconds != d.MaxFadeSeconds {
			t.Fatalf("star nudges moved the twinkle: %+v", c)
		}
		c = DefaultConfig()
		for i := 0; i < 40; i++ {
			c.Nudge(KnobMinCycle, 1)
			c.Nudge(KnobMaxFade, -1)
		}
		if c.Star != DefaultConfig().Star {
			t.Fatalf("twinkle nudges moved the star: %+v", c.Star)
		}
	})
}

func TestUse(t *testing.T) {
	t.Cleanup(reset)
	t.Run("happy: Use makes the knobs active and the sky breathes them", func(t *testing.T) {
		c := DefaultConfig()
		c.MinCycleSeconds, c.MaxCycleSeconds = 1, 3
		c.MinFadeSeconds, c.MaxFadeSeconds = 0.2, 0.6
		if err := Use(c); err != nil {
			t.Fatalf("Use: %v", err)
		}
		if Active() != c {
			t.Fatalf("Active %+v, want %+v", Active(), c)
		}
		if stars.ActiveTwinkle() != c.Twinkle() {
			t.Fatalf("the sky breathes %+v, want the knobs %+v", stars.ActiveTwinkle(), c.Twinkle())
		}
	})
	t.Run("happy: Use carries the star knobs to Active for New", func(t *testing.T) {
		reset()
		c := DefaultConfig()
		c.Star.Speed = 44
		c.Star.Size = 3
		if err := Use(c); err != nil {
			t.Fatalf("Use: %v", err)
		}
		if Active().Star != c.Star {
			t.Fatalf("Active star %+v, want %+v", Active().Star, c.Star)
		}
	})
	t.Run("unhappy: broken knobs are refused and nothing moves", func(t *testing.T) {
		reset()
		before, sky := Active(), stars.ActiveTwinkle()
		brokenStar := DefaultConfig()
		brokenStar.Star.Path = "zigzag"
		nanStar := DefaultConfig()
		nanStar.Star.Speed = math.NaN()
		for _, c := range []Config{
			mkConfig(5, 2, 0.2, 0.6),
			mkConfig(1, 3, 0.6, 0.2),
			mkConfig(0, 3, 0.2, 0.6),
			mkConfig(1, 3, 0.2, 1e9),
			mkConfig(math.NaN(), 3, 0.2, 0.6),
			brokenStar,
			nanStar,
		} {
			if err := Use(c); err == nil {
				t.Fatalf("Use(%+v) must be refused", c)
			}
			if Active() != before || stars.ActiveTwinkle() != sky {
				t.Fatal("a refused Use moved the active knobs or the sky")
			}
		}
	})
}

// mkConfig is the stock config with the four twinkle knobs replaced —
// the star section stays valid so a broken twinkle is what refuses.
func mkConfig(minCycle, maxCycle, minFade, maxFade float64) Config {
	c := DefaultConfig()
	c.MinCycleSeconds, c.MaxCycleSeconds = minCycle, maxCycle
	c.MinFadeSeconds, c.MaxFadeSeconds = minFade, maxFade
	return c
}

func TestLoadSave(t *testing.T) {
	t.Cleanup(reset)
	t.Run("happy: save and load round-trip the knobs, star section included", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "explorer.json")
		c := DefaultConfig()
		c.MinCycleSeconds, c.MaxCycleSeconds = 1.25, 8.5
		c.MinFadeSeconds, c.MaxFadeSeconds = 0.15, 2.4
		c.Star.Size = 3
		c.Star.RandomSize = true
		c.Star.Speed = 44
		c.Star.Count = 2
		c.Star.Period = 0.06
		c.Star.Taper = 0.5
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got != c {
			t.Fatalf("round-trip lost the knobs: %+v vs %+v", got, c)
		}
	})
	t.Run("happy: a file without a star section loads the stock star", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "twinkleonly.json")
		if err := os.WriteFile(path, []byte(`{"minCycleSeconds": 1.25, "maxCycleSeconds": 8.5, "minFadeSeconds": 0.15, "maxFadeSeconds": 2.4}`), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.Star != shootingstar.DefaultConfig() {
			t.Fatalf("a file without a star section loaded %+v, want the stock star", got.Star)
		}
		if got.MinCycleSeconds != 1.25 {
			t.Fatalf("the twinkle keys must still load, got %+v", got)
		}
	})
	t.Run("happy: a missing file is the stock config, not an error", func(t *testing.T) {
		got, err := LoadOrDefault(filepath.Join(t.TempDir(), "nowhere.json"))
		if err != nil {
			t.Fatalf("LoadOrDefault on a missing file: %v", err)
		}
		if got != DefaultConfig() {
			t.Fatalf("a missing file must be the stock knobs, got %+v", got)
		}
	})
	t.Run("happy: the stock config.json ships beside the scene", func(t *testing.T) {
		if DefaultConfigPath != "scenes/explorer/config.json" {
			t.Fatalf("the config lives beside the scene, not at %q", DefaultConfigPath)
		}
		got, err := Load("config.json")
		if err != nil {
			t.Fatalf("the stock config.json must ship with the scene: %v", err)
		}
		if err := got.Validate(); err != nil {
			t.Fatalf("the shipped config must validate: %v", err)
		}
	})
	t.Run("unhappy: broken JSON and broken knobs refuse to load", func(t *testing.T) {
		dir := t.TempDir()
		garbled := filepath.Join(dir, "garbled.json")
		if err := os.WriteFile(garbled, []byte("{ not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(garbled); err == nil {
			t.Fatal("garbled JSON must refuse to load")
		}
		crossed := filepath.Join(dir, "crossed.json")
		if err := os.WriteFile(crossed, []byte(`{"minCycleSeconds": 9, "maxCycleSeconds": 2, "minFadeSeconds": 0.2, "maxFadeSeconds": 0.6}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(crossed); err == nil {
			t.Fatal("crossed ranges must refuse to load")
		}
		brokenStar := filepath.Join(dir, "brokenstar.json")
		if err := os.WriteFile(brokenStar, []byte(`{"minCycleSeconds": 2, "maxCycleSeconds": 7, "minFadeSeconds": 0.4, "maxFadeSeconds": 1.6, "star": {"path": "zigzag"}}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(brokenStar); err == nil {
			t.Fatal("a broken star section must refuse to load")
		}
		if _, err := LoadOrDefault(garbled); err == nil {
			t.Fatal("LoadOrDefault forgives absence, never breakage")
		}
	})
	t.Run("unhappy: saving broken knobs is refused before the disk", func(t *testing.T) {
		c := mkConfig(9, 2, 0.2, 0.6)
		path := filepath.Join(t.TempDir(), "never.json")
		if err := c.Save(path); err == nil {
			t.Fatal("broken knobs must not save")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("a refused save must leave no file")
		}
		bad := DefaultConfig()
		bad.Star.Path = "zigzag"
		if err := bad.Save(path); err == nil {
			t.Fatal("a broken star must not save")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("a refused star save must leave no file")
		}
	})
}

func TestLoadOrInherit(t *testing.T) {
	t.Cleanup(reset)
	tuned := shootingstar.DefaultConfig()
	tuned.Size = 3
	tuned.Speed = 44
	t.Run("happy: a missing file inherits the shooting-star knobs it is handed", func(t *testing.T) {
		got, err := LoadOrInherit(filepath.Join(t.TempDir(), "nowhere.json"), tuned)
		if err != nil {
			t.Fatalf("LoadOrInherit on a missing file: %v", err)
		}
		if got.Star != tuned {
			t.Fatalf("the star is %+v, want the inherited %+v", got.Star, tuned)
		}
		want := DefaultConfig()
		want.Star = tuned
		if got != want {
			t.Fatalf("the twinkle must stay stock: %+v", got)
		}
	})
	t.Run("happy: a file without a star section inherits, and its own twinkle still wins", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "twinkleonly.json")
		if err := os.WriteFile(path, []byte(`{"minCycleSeconds": 1.25, "maxCycleSeconds": 8.5, "minFadeSeconds": 0.15, "maxFadeSeconds": 2.4}`), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := LoadOrInherit(path, tuned)
		if err != nil {
			t.Fatalf("LoadOrInherit: %v", err)
		}
		if got.Star != tuned {
			t.Fatalf("the star is %+v, want the inherited %+v", got.Star, tuned)
		}
		if got.MinCycleSeconds != 1.25 || got.MaxFadeSeconds != 2.4 {
			t.Fatalf("the file's twinkle keys must win: %+v", got)
		}
	})
	t.Run("happy: a saved star section beats the inherited knobs — the scene owns its copy", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "pinned.json")
		pinned := DefaultConfig()
		pinned.Star.Size = 5
		pinned.Star.Speed = 12
		if err := pinned.Save(path); err != nil {
			t.Fatal(err)
		}
		got, err := LoadOrInherit(path, tuned)
		if err != nil {
			t.Fatalf("LoadOrInherit: %v", err)
		}
		if got.Star != pinned.Star {
			t.Fatalf("the star is %+v, want the scene's own %+v", got.Star, pinned.Star)
		}
	})
	t.Run("unhappy: LoadOrInherit forgives absence, never breakage", func(t *testing.T) {
		dir := t.TempDir()
		garbled := filepath.Join(dir, "garbled.json")
		if err := os.WriteFile(garbled, []byte("{ not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadOrInherit(garbled, tuned); err == nil {
			t.Fatal("garbled JSON must refuse to load even with a good star to inherit")
		}
	})
	t.Run("unhappy: inheriting a broken star is refused when nothing on disk overrides it", func(t *testing.T) {
		broken := shootingstar.DefaultConfig()
		broken.Path = "zigzag"
		if _, err := LoadOrInherit(filepath.Join(t.TempDir(), "nowhere.json"), broken); err == nil {
			t.Fatal("a broken inherited star must be refused")
		}
	})
}
