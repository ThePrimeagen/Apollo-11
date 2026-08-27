package adjustcloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/cloud"
)

func TestTuner(t *testing.T) {
	t.Run("happy: NewTuner seeds every knob from the active cloud", func(t *testing.T) {
		t.Cleanup(cloud.Reset)
		c := cloud.DefaultConfig()
		c.Puffs = 8
		c.Count = 12
		if err := cloud.Use(c); err != nil {
			t.Fatalf("Use: %v", err)
		}
		tu := NewTuner()
		if tu.Cfg != c {
			t.Fatalf("tuner seeded %+v, want the active %+v", tu.Cfg, c)
		}
		if tu.Cursor != 0 {
			t.Fatalf("cursor %d, want the first knob", tu.Cursor)
		}
	})
	t.Run("happy: move and nudge walk and turn the knobs", func(t *testing.T) {
		t.Cleanup(cloud.Reset)
		cloud.Reset()
		tu := NewTuner()
		before := tu.Cfg.Count
		tu.Nudge(1)
		if tu.Cfg.Count != before+1 {
			t.Fatalf("count %d after a nudge, want %d", tu.Cfg.Count, before+1)
		}
		tu.Move(1)
		puffs := tu.Cfg.Puffs
		tu.Nudge(-1)
		if tu.Cfg.Puffs != puffs-1 {
			t.Fatalf("puffs %d after -1, want %d", tu.Cfg.Puffs, puffs-1)
		}
	})
	t.Run("unhappy: a nil tuner never panics", func(t *testing.T) {
		var tu *Tuner
		tu.Move(1)
		tu.Nudge(1)
	})
}

func TestConfigHome(t *testing.T) {
	t.Run("happy: the default path points into the cloud component", func(t *testing.T) {
		if DefaultConfigPath != filepath.Join("components", "cloud", "config.json") {
			t.Fatalf("DefaultConfigPath = %q, want components/cloud/config.json", DefaultConfigPath)
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
		if _, err := os.Stat("cloud.json"); err == nil {
			t.Fatal("cloud.json must not live in cmd/adjustcloud — it belongs to components/cloud")
		}
	})
}
