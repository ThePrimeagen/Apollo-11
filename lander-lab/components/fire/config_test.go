package fire

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	t.Run("happy: defaults are the 15% ladder entry heats", func(t *testing.T) {
		c := DefaultHeat()
		want := []int{1, 7, 13, 24, 47, 82, 139, 230}
		if len(c.Thresholds) != len(want) {
			t.Fatalf("thresholds %d, want %d", len(c.Thresholds), len(want))
		}
		for i, w := range want {
			if c.Thresholds[i] != w {
				t.Fatalf("threshold[%d]=%d, want %d", i, c.Thresholds[i], w)
			}
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("default must validate: %v", err)
		}
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("unhappy: wrong count is rejected", func(t *testing.T) {
		c := HeatConfig{Thresholds: []int{1, 2, 3}}
		if err := c.Validate(); !errors.Is(err, ErrThresholdCount) {
			t.Fatalf("got %v, want ErrThresholdCount", err)
		}
	})
	t.Run("unhappy: a value below 0 is rejected", func(t *testing.T) {
		c := DefaultHeat()
		c.Thresholds[0] = -1
		if err := c.Validate(); !errors.Is(err, ErrThresholdRange) {
			t.Fatalf("got %v, want ErrThresholdRange", err)
		}
	})
	t.Run("unhappy: a value above 500 is rejected", func(t *testing.T) {
		c := DefaultHeat()
		c.Thresholds[7] = 501
		if err := c.Validate(); !errors.Is(err, ErrThresholdRange) {
			t.Fatalf("got %v, want ErrThresholdRange", err)
		}
	})
}

func TestConfigLoadSave(t *testing.T) {
	t.Run("happy: save then load is a round trip", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "flame.json")
		want := DefaultHeat()
		want.Thresholds[3] = 30
		if err := want.Save(path); err != nil {
			t.Fatal(err)
		}
		got, err := LoadHeat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got.Thresholds[3] != 30 {
			t.Fatalf("loaded %v, want threshold[3]=30", got.Thresholds)
		}
	})
	t.Run("unhappy: a missing file is an error", func(t *testing.T) {
		if _, err := LoadHeat(filepath.Join(t.TempDir(), "nope.json")); err == nil {
			t.Fatal("missing file must fail")
		}
	})
	t.Run("unhappy: invalid JSON is an error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(path, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadHeat(path); err == nil {
			t.Fatal("invalid JSON must fail")
		}
	})
	t.Run("unhappy: out-of-range JSON is an error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "range.json")
		raw, _ := json.Marshal(HeatConfig{Thresholds: []int{1, 7, 13, 24, 47, 82, 139, 900}})
		if err := os.WriteFile(path, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadHeat(path); !errors.Is(err, ErrThresholdRange) {
			t.Fatalf("got %v, want ErrThresholdRange", err)
		}
	})
}
