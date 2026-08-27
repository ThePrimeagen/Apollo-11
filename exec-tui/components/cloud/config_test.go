package cloud

// Tests written FIRST: Config is the live knobs on the cloud
// generator — how many specks each blob parks, how many blobs make
// one cloud, how wide each pool is, how far the blobs scatter from
// the cloud's centre, and the white/gray ladder that maps
// concentration onto braille, ░, and ▒. The tuner nudges counts by
// one, radii by half a unit, inks one xterm gray at a time.
// Save/Load round-trip the JSON next to the component.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

func TestConfig(t *testing.T) {
	t.Run("happy: the stock knobs grow a fluffy multi-blob cloud", func(t *testing.T) {
		c := DefaultConfig()
		if err := c.Validate(); err != nil {
			t.Fatalf("the stock cloud must validate: %v", err)
		}
		if c.Count < 8 {
			t.Fatalf("count %d — a puff needs a handful of specks", c.Count)
		}
		if c.Puffs < 3 {
			t.Fatalf("puffs %d — a unique cloud is several overlapping blobs", c.Puffs)
		}
		if c.Radius <= 0 {
			t.Fatalf("radius %v — each blob needs a pool to spread in", c.Radius)
		}
		if c.Spread <= 0 {
			t.Fatalf("spread %v — the blobs must sit apart", c.Spread)
		}
		if c.ThinAt < 1 || c.ThickAt <= c.ThinAt {
			t.Fatalf("ladder %d/%d must climb: 1 <= thin < thick", c.ThinAt, c.ThickAt)
		}
		if c.ThinFG >= c.MidFG || c.MidFG >= c.ThickFG {
			t.Fatalf("grays %d/%d/%d must brighten with concentration", c.ThinFG, c.MidFG, c.ThickFG)
		}
		if KnobCount != 10 {
			t.Fatalf("KnobCount %d, want 10", KnobCount)
		}
	})
	t.Run("happy: Engine is a parked pool world from the knobs", func(t *testing.T) {
		c := DefaultConfig()
		cfg := c.Engine(80, 40, particle.Vec2{X: 40, Y: 20})
		if err := cfg.Validate(); err != nil {
			t.Fatalf("the pool engine must validate: %v", err)
		}
		if cfg.Mode != particle.ModePool {
			t.Fatalf("mode %v, want ModePool", cfg.Mode)
		}
		if cfg.PoolRadius != c.Radius {
			t.Fatalf("pool radius %v, want %v", cfg.PoolRadius, c.Radius)
		}
		if cfg.Count != c.Count {
			t.Fatalf("count %d, want %d", cfg.Count, c.Count)
		}
		if cfg.Period != 0 {
			t.Fatalf("period %v, want 0 — clouds Burst once and stay", cfg.Period)
		}
		if cfg.MinSpeed != 0 || cfg.MaxSpeed != 0 {
			t.Fatalf("speed %v..%v, want parked", cfg.MinSpeed, cfg.MaxSpeed)
		}
	})
	t.Run("unhappy: bad counts, radii, ladders, and grays are named", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			mod  func(*Config)
			want error
		}{
			{"negative count", func(c *Config) { c.Count = -1 }, ErrCount},
			{"negative puffs", func(c *Config) { c.Puffs = -1 }, ErrPuffs},
			{"negative radius", func(c *Config) { c.Radius = -1 }, ErrRadius},
			{"negative spread", func(c *Config) { c.Spread = -1 }, ErrSpread},
			{"thin too low", func(c *Config) { c.ThinAt = 0 }, ErrLadder},
			{"thick not past thin", func(c *Config) { c.ThickAt = c.ThinAt }, ErrLadder},
			{"gray off the gray ramp", func(c *Config) { c.ThinFG = 196 }, ErrGray},
			{"gray past the ramp", func(c *Config) { c.ThickFG = 256 }, ErrGray},
		} {
			c := DefaultConfig()
			tc.mod(&c)
			if err := c.Validate(); !errors.Is(err, tc.want) {
				t.Fatalf("%s: got %v, want %v", tc.name, err, tc.want)
			}
		}
	})
}

func TestUseLoadSave(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: Use is what Active and Generate read", func(t *testing.T) {
		t.Cleanup(Reset)
		cfg := DefaultConfig()
		cfg.Puffs = 8
		cfg.Count = 12
		if err := Use(cfg); err != nil {
			t.Fatal(err)
		}
		got := Active()
		if got.Puffs != 8 || got.Count != 12 {
			t.Fatalf("Active %+v, want the used knobs", got)
		}
	})
	t.Run("happy: Save/Load round-trip the JSON", func(t *testing.T) {
		c := DefaultConfig()
		c.Puffs = 7
		c.Radius = 5.5
		path := filepath.Join(t.TempDir(), "cloud.json")
		if err := c.Save(path); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Puffs != 7 || got.Radius != 5.5 {
			t.Fatalf("loaded %+v, want the saved knobs", got)
		}
	})
	t.Run("happy: a missing file is stock, not an error", func(t *testing.T) {
		got, err := LoadOrDefault(filepath.Join(t.TempDir(), "nope.json"))
		if err != nil {
			t.Fatal(err)
		}
		if got != DefaultConfig() {
			t.Fatalf("missing file %+v, want stock", got)
		}
	})
	t.Run("unhappy: Use rejects a bad cfg and Active holds", func(t *testing.T) {
		t.Cleanup(Reset)
		before := Active()
		bad := DefaultConfig()
		bad.Count = -4
		if err := Use(bad); !errors.Is(err, ErrCount) {
			t.Fatalf("got %v, want ErrCount", err)
		}
		if Active() != before {
			t.Fatal("a rejected Use must not clobber Active")
		}
	})
	t.Run("unhappy: a missing key keeps that knob at stock", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "partial.json")
		if err := os.WriteFile(path, []byte("{\"puffs\": 9}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Puffs != 9 {
			t.Fatalf("puffs %d, want 9", got.Puffs)
		}
		if got.Count != DefaultConfig().Count {
			t.Fatalf("missing count became %d, want stock", got.Count)
		}
	})
}

func TestNudge(t *testing.T) {
	t.Run("happy: counts climb and radii walk half a unit", func(t *testing.T) {
		c := DefaultConfig()
		c.Nudge(KnobSpecks, 1)
		if c.Count != DefaultConfig().Count+1 {
			t.Fatalf("count %d after a nudge", c.Count)
		}
		c.Nudge(KnobPuffs, -1)
		if c.Puffs != DefaultConfig().Puffs-1 {
			t.Fatalf("puffs %d after a nudge", c.Puffs)
		}
		want := DefaultConfig().Radius + StepRadius
		c = DefaultConfig()
		c.Nudge(KnobRadius, 1)
		if c.Radius != want {
			t.Fatalf("radius %v, want %v", c.Radius, want)
		}
	})
	t.Run("unhappy: counts and radii never go negative", func(t *testing.T) {
		c := DefaultConfig()
		c.Count = 0
		c.Nudge(KnobSpecks, -1)
		if c.Count != 0 {
			t.Fatalf("count %d, want 0", c.Count)
		}
		c.Radius = 0
		c.Nudge(KnobRadius, -1)
		if c.Radius != 0 {
			t.Fatalf("radius %v, want 0", c.Radius)
		}
		c.Puffs = 0
		c.Nudge(KnobPuffs, -1)
		if c.Puffs != 0 {
			t.Fatalf("puffs %d, want 0", c.Puffs)
		}
	})
}
