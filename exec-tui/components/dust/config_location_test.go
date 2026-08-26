package dust

// The dust-off tuner edits this component's own config: the JSON lives
// at components/dust/config.json, next to the code it tunes.

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigHome(t *testing.T) {
	t.Run("happy: the shipped config lives with the component and validates", func(t *testing.T) {
		c, err := LoadPuff("config.json")
		if err != nil {
			t.Fatalf("the component's shipped config must load: %v", err)
		}
		if err := c.Validate(); err != nil {
			t.Fatalf("the shipped config must validate: %v", err)
		}
	})
	t.Run("unhappy: no config hides next to the tuner", func(t *testing.T) {
		if _, err := os.Stat(filepath.Join("..", "..", "cmd", "adjustdust", "config.json")); err == nil {
			t.Fatal("config.json must not live in cmd/adjustdust — it belongs to components/dust")
		}
	})
}
