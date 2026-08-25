package adjuststars

// The tuner edits the stars component's own config: the JSON lives at
// components/stars/config.json, not inside this tool's folder. Tests
// written before the move.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigHome(t *testing.T) {
	t.Run("happy: the default path points into the stars component", func(t *testing.T) {
		if DefaultConfigPath != filepath.Join("components", "stars", "config.json") {
			t.Fatalf("DefaultConfigPath = %q, want components/stars/config.json", DefaultConfigPath)
		}
	})
	t.Run("happy: the shipped config opens from the module root", func(t *testing.T) {
		m, err := Open(filepath.Join("..", "..", DefaultConfigPath), 0)
		if err != nil {
			t.Fatalf("the component's shipped config must open: %v", err)
		}
		if m.Path == "" {
			t.Fatal("the opened tuner must remember its path")
		}
	})
	t.Run("unhappy: the config no longer hides next to the tuner", func(t *testing.T) {
		if _, err := os.Stat("stars.json"); err == nil {
			t.Fatal("stars.json must not live in cmd/adjuststars anymore — it belongs to components/stars")
		}
	})
}
