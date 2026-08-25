package fire

// The fire component owns its own tuning file: config.json sits right
// here, next to the code it configures, instead of hiding inside the
// tuner that edits it. Tests written before the move.

import (
	"os"
	"testing"
)

func TestShippedConfig(t *testing.T) {
	t.Run("happy: config.json ships beside the component and loads", func(t *testing.T) {
		c, err := LoadHeat("config.json")
		if err != nil {
			t.Fatalf("the component's own config.json must load: %v", err)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the shipped config must validate: %v", err)
		}
		if len(c.Thresholds) != RungCount {
			t.Fatalf("shipped config holds %d thresholds, want %d", len(c.Thresholds), RungCount)
		}
	})
	t.Run("unhappy: a missing config file is a load error", func(t *testing.T) {
		if _, err := LoadHeat("no-such-config.json"); err == nil {
			t.Fatal("a missing config file must error")
		}
	})
	t.Run("unhappy: a directory is not a config file", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := LoadHeat(dir); err == nil {
			t.Fatal("loading a directory must error")
		}
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("the failed load must not disturb the path: %v", err)
		}
	})
}
