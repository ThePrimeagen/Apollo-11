package adjustflame

// The tuner edits the fire component's own config: the JSON lives at
// components/fire/config.json, not inside this tool's folder. Tests
// written before the move.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigHome(t *testing.T) {
	t.Run("happy: the default path points into the fire component", func(t *testing.T) {
		if DefaultConfigPath != filepath.Join("components", "fire", "config.json") {
			t.Fatalf("DefaultConfigPath = %q, want components/fire/config.json", DefaultConfigPath)
		}
	})
	t.Run("happy: the shipped config opens from the module root", func(t *testing.T) {
		m, err := Open(filepath.Join("..", "..", DefaultConfigPath))
		if err != nil {
			t.Fatalf("the component's shipped config must open: %v", err)
		}
		if len(m.Thresholds) == 0 {
			t.Fatal("the opened config must carry thresholds")
		}
	})
	t.Run("unhappy: the config no longer hides next to the tuner", func(t *testing.T) {
		if _, err := os.Stat("flame.json"); err == nil {
			t.Fatal("flame.json must not live in cmd/adjustflame anymore — it belongs to components/fire")
		}
	})
}
