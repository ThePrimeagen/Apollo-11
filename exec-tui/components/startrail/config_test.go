package startrail

// Tests written FIRST: Config is the live trail knobs — count, period,
// min/max life, nozzle, peak, taper — the particle half of the comet.
// Peak steepens the persist slit onto the spine; taper cuts max life
// by |offset|. Use/Active is what New reads; Save/Load round-trip the
// JSON next to the component.

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

func TestTrailConfig(t *testing.T) {
	t.Run("happy: the stock knobs are a tight, readable comet wake", func(t *testing.T) {
		c := DefaultConfig()
		if c.Count <= 0 || c.Period <= 0 {
			t.Fatalf("stock must emit: %+v", c)
		}
		if c.MinLife <= 0 || c.MaxLife < c.MinLife {
			t.Fatalf("stock life %+v", c)
		}
		if c.Nozzle < 0 {
			t.Fatalf("stock nozzle %v", c.Nozzle)
		}
		if c.Peak <= 1 {
			t.Fatalf("stock peak %v must be steep (greater than 1)", c.Peak)
		}
		if c.Taper <= 0 || c.Taper > 1 {
			t.Fatalf("stock taper %v must fall in (0,1]", c.Taper)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("stock must validate: %v", err)
		}
		if DefaultConfigPath != "components/startrail/config.json" {
			t.Fatalf("DefaultConfigPath %q, want components/startrail/config.json", DefaultConfigPath)
		}
	})
	t.Run("happy: ParticleConfig is a persist world the trail can arm", func(t *testing.T) {
		c := DefaultConfig()
		origin := particle.Vec2{X: 10, Y: 8}
		heading := particle.Vec2{X: 1, Y: 0}
		got := c.ParticleConfig(80, 40, origin, heading)
		if got.Width != 80 || got.Height != 40 {
			t.Fatalf("ParticleConfig box %+v, want 80x40", got)
		}
		if got.Origin != origin {
			t.Fatalf("origin %+v, want %+v", got.Origin, origin)
		}
		if got.Count != c.Count || got.Period != c.Period {
			t.Fatalf("ParticleConfig dropped the knobs: %+v", got)
		}
		if got.MinLife != c.MinLife || got.MaxLife != c.MaxLife || got.Nozzle != c.Nozzle {
			t.Fatalf("ParticleConfig dropped life/nozzle: %+v", got)
		}
		if got.Peak != c.Peak || got.Taper != c.Taper {
			t.Fatalf("ParticleConfig dropped peak/taper: %+v", got)
		}
		if got.Mode != particle.ModePersist {
			t.Fatalf("mode %v, want ModePersist", got.Mode)
		}
	})
	t.Run("happy: Save then Load round-trips the knobs", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "trail.json")
		c := DefaultConfig()
		c.Count = 5
		c.Period = 0.03
		c.MinLife, c.MaxLife = 0.4, 0.9
		c.Nozzle = 2.5
		c.Peak = 4
		c.Taper = 0.4
		if err := c.Save(path); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.Count != c.Count || math.Abs(got.Period-c.Period) > 1e-9 {
			t.Fatalf("round-trip %+v, want %+v", got, c)
		}
		if math.Abs(got.MinLife-c.MinLife) > 1e-9 || math.Abs(got.MaxLife-c.MaxLife) > 1e-9 || math.Abs(got.Nozzle-c.Nozzle) > 1e-9 {
			t.Fatalf("round-trip life/nozzle %+v, want %+v", got, c)
		}
		if math.Abs(got.Peak-c.Peak) > 1e-9 || math.Abs(got.Taper-c.Taper) > 1e-9 {
			t.Fatalf("round-trip peak/taper %+v, want %+v", got, c)
		}
	})
	t.Run("happy: a file missing keys keeps the stock values for them", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "old.json")
		if err := os.WriteFile(path, []byte(`{"count":7}`+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.Count != 7 {
			t.Fatalf("present keys must load, got %+v", got)
		}
		stock := DefaultConfig()
		if got.Period != stock.Period || got.MinLife != stock.MinLife {
			t.Fatalf("missing keys loaded %+v, want stock period/life", got)
		}
		if got.Peak != stock.Peak || got.Taper != stock.Taper {
			t.Fatalf("missing peak/taper loaded %+v, want stock", got)
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
		live.Count = 2
		live.Period = 0.04
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
		if err := os.WriteFile(out, []byte(`{"count":-1}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(out); err == nil {
			t.Fatal("a negative count must error")
		}
		neg := DefaultConfig()
		neg.Period = -1
		if err := neg.Save(filepath.Join(t.TempDir(), "x.json")); err == nil {
			t.Fatal("Save must refuse a negative period")
		}
		before := Active()
		badUse := DefaultConfig()
		badUse.MinLife, badUse.MaxLife = 2, 0.1
		if err := Use(badUse); err == nil {
			t.Fatal("Use must reject a reversed life range")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected Use is %+v, want %+v", Active(), before)
		}
		negPeak := DefaultConfig()
		negPeak.Peak = -2
		if err := Use(negPeak); err == nil {
			t.Fatal("Use must reject a negative peak")
		}
		wide := DefaultConfig()
		wide.Taper = 1.5
		if err := Use(wide); err == nil {
			t.Fatal("Use must reject a taper past 1")
		}
		if Active() != before {
			t.Fatalf("Active after a rejected peak/taper is %+v, want %+v", Active(), before)
		}
	})
}
