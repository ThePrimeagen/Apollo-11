package adjustsky

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sky"
)

func TestTuner(t *testing.T) {
	t.Run("happy: NewTuner seeds every knob from the active sky", func(t *testing.T) {
		t.Cleanup(sky.Reset)
		c := sky.DefaultConfig()
		c.AngleDeg = 45
		c.LightInk = 159
		if err := sky.Use(c); err != nil {
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
		t.Cleanup(sky.Reset)
		sky.Reset()
		tu := NewTuner()
		tu.Nudge(1)
		if tu.Cfg.AngleDeg != sky.StepAngle {
			t.Fatalf("angle %v after a nudge, want %v", tu.Cfg.AngleDeg, sky.StepAngle)
		}
		tu.Move(1)
		before := tu.Cfg.LightInk
		tu.Nudge(1)
		if tu.Cfg.LightInk != before+1 {
			t.Fatalf("light %d after a nudge, want %d", tu.Cfg.LightInk, before+1)
		}
	})
	t.Run("unhappy: a nil tuner never panics", func(t *testing.T) {
		var tu *Tuner
		tu.Move(1)
		tu.Nudge(1)
	})
}

func TestConfigHome(t *testing.T) {
	t.Run("happy: the default path points into the sky component", func(t *testing.T) {
		if DefaultConfigPath != filepath.Join("components", "sky", "config.json") {
			t.Fatalf("DefaultConfigPath = %q, want components/sky/config.json", DefaultConfigPath)
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
		if _, err := os.Stat("sky.json"); err == nil {
			t.Fatal("sky.json must not live in cmd/adjustsky — it belongs to components/sky")
		}
	})
}
