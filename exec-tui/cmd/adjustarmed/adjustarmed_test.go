package adjustarmed

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/armed"
)

func TestTuner(t *testing.T) {
	t.Run("happy: NewTuner seeds every knob from the active composite", func(t *testing.T) {
		t.Cleanup(armed.Reset)
		c := armed.DefaultConfig()
		c.Delay = 1.5
		c.LeftRate = 4
		if err := armed.Use(c); err != nil {
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
		t.Cleanup(armed.Reset)
		armed.Reset()
		tu := NewTuner()
		tu.Nudge(1)
		if tu.Cfg.Delay != armed.StockDelay+armed.StepSeconds {
			t.Fatalf("delay %v after a nudge, want %v", tu.Cfg.Delay, armed.StockDelay+armed.StepSeconds)
		}
		tu.Move(1)
		before := tu.Cfg.Cross
		tu.Nudge(1)
		if tu.Cfg.Cross != before+armed.StepSeconds {
			t.Fatalf("cross %v after a nudge, want %v", tu.Cfg.Cross, before+armed.StepSeconds)
		}
	})
	t.Run("unhappy: a nil tuner never panics", func(t *testing.T) {
		var tu *Tuner
		tu.Move(1)
		tu.Nudge(1)
	})
}

func TestConfigHome(t *testing.T) {
	t.Run("happy: the default path points into the armed component", func(t *testing.T) {
		if DefaultConfigPath != filepath.Join("components", "armed", "config.json") {
			t.Fatalf("DefaultConfigPath = %q, want components/armed/config.json", DefaultConfigPath)
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
		if _, err := os.Stat("armed.json"); err == nil {
			t.Fatal("armed.json must not live in cmd/adjustarmed — it belongs to components/armed")
		}
	})
}
