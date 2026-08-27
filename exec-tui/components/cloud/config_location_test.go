package cloud

// The cloud tuner edits this component's own config: the JSON lives
// at components/cloud/config.json, next to the code it tunes.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigHome(t *testing.T) {
	t.Run("happy: the shipped config lives with the component and validates", func(t *testing.T) {
		c, err := Load("config.json")
		if err != nil {
			t.Fatalf("the component's shipped config must load: %v", err)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the shipped config must validate: %v", err)
		}
	})
	t.Run("unhappy: no config hides next to the tuner", func(t *testing.T) {
		if _, err := os.Stat(filepath.Join("..", "..", "cmd", "adjustcloud", "config.json")); err == nil {
			t.Fatal("config.json must not live in cmd/adjustcloud — it belongs to components/cloud")
		}
	})
}
