package stars

// Tests written FIRST: SkyConfig is the sky's file config, the same
// shape of loading the fire's heat ladder uses — a JSON file, a stock
// default, Load/Save with validation, and a package-active setting
// (UseSky/ActiveSky/ResetSky) that scenes read so a tuned file just
// works. Sentinel errors for every way a file can be wrong.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSkyConfig(t *testing.T) {
	t.Run("happy: the stock sky is valid and matches the stock knobs", func(t *testing.T) {
		c := DefaultSky()
		if err := c.Validate(); err != nil {
			t.Fatalf("DefaultSky must validate: %v", err)
		}
		if c.FlyStrategy().Delay != Drift.Delay {
			t.Fatalf("stock delays %v, want drift's %v", c.FlyStrategy().Delay, Drift.Delay)
		}
		if c.DensityLayers() != DefaultDensity {
			t.Fatalf("stock densities %v, want %v", c.DensityLayers(), DefaultDensity)
		}
	})
	t.Run("happy: a sky survives the save/load round trip", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "sky.json")
		want := SkyConfig{Delay: []int{1, 2, 3, 4}, Density: []int{10, 20, 30, 40}}
		if err := want.Save(path); err != nil {
			t.Fatalf("save: %v", err)
		}
		got, err := LoadSky(path)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		if got.FlyStrategy().Delay != want.FlyStrategy().Delay || got.DensityLayers() != want.DensityLayers() {
			t.Fatalf("round trip changed the sky: %+v -> %+v", want, got)
		}
	})
	t.Run("happy: UseSky puts the settings in effect; ResetSky takes them out", func(t *testing.T) {
		t.Cleanup(ResetSky)
		c := SkyConfig{Delay: []int{1, 1, 1, 1}, Density: []int{100, 100, 100, 100}}
		if err := UseSky(c); err != nil {
			t.Fatalf("UseSky: %v", err)
		}
		if ActiveSky().DensityLayers() != [4]int{100, 100, 100, 100} {
			t.Fatalf("active densities %v, want the used sky", ActiveSky().DensityLayers())
		}
		ResetSky()
		if ActiveSky().DensityLayers() != DefaultDensity {
			t.Fatalf("reset densities %v, want stock", ActiveSky().DensityLayers())
		}
	})
	t.Run("happy: the strategy a config describes flies its delays", func(t *testing.T) {
		c := SkyConfig{Delay: []int{2, 3, 4, 5}, Density: []int{5, 5, 5, 5}}
		s := c.FlyStrategy()
		if s.Delay != [4]int{2, 3, 4, 5} {
			t.Fatalf("strategy delays %v", s.Delay)
		}
		if s.Name == "" || s.Name == Still.Name {
			t.Fatalf("a tuned strategy needs its own moving name, got %q", s.Name)
		}
	})
	t.Run("unhappy: wrong layer counts are ErrLayerCount", func(t *testing.T) {
		for _, c := range []SkyConfig{
			{Delay: []int{1, 2, 3}, Density: []int{1, 2, 3, 4}},
			{Delay: []int{1, 2, 3, 4, 5}, Density: []int{1, 2, 3, 4}},
			{Delay: []int{1, 2, 3, 4}, Density: []int{1}},
			{},
		} {
			if err := c.Validate(); !errors.Is(err, ErrLayerCount) {
				t.Fatalf("%+v: got %v, want ErrLayerCount", c, err)
			}
		}
	})
	t.Run("happy: all movements at zero validate and describe a sky that holds still", func(t *testing.T) {
		c := SkyConfig{Delay: []int{0, 0, 0, 0}, Density: []int{10, 20, 30, 40}}
		if err := c.Validate(); err != nil {
			t.Fatalf("zero movement must just work, got %v", err)
		}
		path := filepath.Join(t.TempDir(), "still-sky.json")
		if err := c.Save(path); err != nil {
			t.Fatalf("save: %v", err)
		}
		got, err := LoadSky(path)
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		s := got.FlyStrategy()
		a := Field{Width: 60, Height: 24, Tick: 0, Strategy: s}
		b := Field{Width: 60, Height: 24, Tick: 90, Strategy: s}
		if a.Render() != b.Render() {
			t.Fatal("a zero-movement sky must hold every star")
		}
	})
	t.Run("unhappy: out-of-range knobs are range errors", func(t *testing.T) {
		bad := SkyConfig{Delay: []int{-1, 2, 3, 4}, Density: []int{1, 2, 3, 4}}
		if err := bad.Validate(); !errors.Is(err, ErrDelayRange) {
			t.Fatalf("delay -1: got %v, want ErrDelayRange", err)
		}
		bad = SkyConfig{Delay: []int{1, 2, 3, MaxDelay + 1}, Density: []int{1, 2, 3, 4}}
		if err := bad.Validate(); !errors.Is(err, ErrDelayRange) {
			t.Fatalf("delay over: got %v, want ErrDelayRange", err)
		}
		bad = SkyConfig{Delay: []int{1, 2, 3, 4}, Density: []int{0, 2, 3, 4}}
		if err := bad.Validate(); !errors.Is(err, ErrDensityRange) {
			t.Fatalf("density 0: got %v, want ErrDensityRange", err)
		}
		bad = SkyConfig{Delay: []int{1, 2, 3, 4}, Density: []int{1, 2, 3, MaxDensity + 1}}
		if err := bad.Validate(); !errors.Is(err, ErrDensityRange) {
			t.Fatalf("density over: got %v, want ErrDensityRange", err)
		}
	})
	t.Run("unhappy: UseSky rejects a bad sky and the active sky holds", func(t *testing.T) {
		t.Cleanup(ResetSky)
		before := ActiveSky()
		if err := UseSky(SkyConfig{Delay: []int{9}, Density: []int{9}}); err == nil {
			t.Fatal("an invalid sky must be rejected")
		}
		if ActiveSky().DensityLayers() != before.DensityLayers() {
			t.Fatal("a rejected sky must not touch the active one")
		}
	})
	t.Run("unhappy: loading a missing or broken file is an error", func(t *testing.T) {
		if _, err := LoadSky(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Fatal("a missing file must error")
		}
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSky(bad); err == nil {
			t.Fatal("a broken file must error")
		}
		invalid := filepath.Join(t.TempDir(), "invalid.json")
		if err := os.WriteFile(invalid, []byte(`{"delay":[1,2],"density":[3]}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadSky(invalid); !errors.Is(err, ErrLayerCount) {
			t.Fatal("an out-of-shape file must fail validation")
		}
	})
	t.Run("unhappy: saving somewhere impossible is an error, and bad skies never hit disk", func(t *testing.T) {
		c := DefaultSky()
		if err := c.Save(filepath.Join(t.TempDir(), "no", "such", "dir", "sky.json")); err == nil {
			t.Fatal("saving into a missing directory must error")
		}
		path := filepath.Join(t.TempDir(), "sky.json")
		if err := (SkyConfig{Delay: []int{0}, Density: []int{0}}).Save(path); err == nil {
			t.Fatal("an invalid sky must not save")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("a rejected save must leave no file")
		}
	})
}
