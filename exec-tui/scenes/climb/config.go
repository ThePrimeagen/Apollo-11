package climb

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
)

// Config is the one live knob on the spacelander climb — how long the
// north-facing craft takes to rise from off the bottom to off the
// top. The standalone runner nudges it 50ms at a time; Play rebuilds
// from whatever it holds. s writes this JSON next to the scene.
type Config struct {
	ClimbSeconds float64 `json:"climbSeconds"`
}

// Knob is which timing the cursor is on.
type Knob int

const (
	KnobClimb Knob = iota
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobClimb:
		return "climb"
	default:
		return ""
	}
}

// Value is the selected knob's current seconds.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobClimb:
		return c.ClimbSeconds
	default:
		return 0
	}
}

const (
	// StepSeconds is one tick of the climb knob: 50ms.
	StepSeconds = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/climb/config.json"
)

var (
	errClimb = errors.New("climb: climb duration must be at least 50ms")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the portable climb's stock timing — the fall
// duration, run the other way.
func DefaultConfig() Config {
	return Config{ClimbSeconds: lander.DropSeconds}
}

// Active is the timing New copies onto a climb scene: the last
// successful Use, or stock after Reset.
func Active() Config {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

// Use makes cfg the timing New plays. A bad cfg is rejected and
// Active is unchanged.
func Use(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	activeMu.Lock()
	active = cfg
	activeMu.Unlock()
	return nil
}

// Reset restores stock timing. Tests call this so a Use cannot leak.
func Reset() {
	activeMu.Lock()
	active = DefaultConfig()
	activeMu.Unlock()
}

// Validate reports whether the knobs are playable.
func (c Config) Validate() error {
	if c.ClimbSeconds < StepSeconds || math.IsNaN(c.ClimbSeconds) || math.IsInf(c.ClimbSeconds, 0) {
		return errClimb
	}
	return nil
}

// Load reads a climb-config JSON file.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("climb: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c.snapped(), nil
}

// LoadOrDefault is Load, except a missing file is stock timing, not
// an error.
func LoadOrDefault(path string) (Config, error) {
	c, err := Load(path)
	if err == nil {
		return c, nil
	}
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	return Config{}, err
}

// Save writes the knobs as JSON, snapped to 50ms so the file stays
// easy to edit by hand.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	c = c.snapped()
	raw := []byte(fmt.Sprintf("{\n  \"climbSeconds\": %.3f\n}\n", c.ClimbSeconds))
	return os.WriteFile(path, raw, 0o644)
}

func snap(v float64) float64 {
	steps := 1 / StepSeconds
	return math.Round(v*steps) / steps
}

func (c Config) snapped() Config {
	c.ClimbSeconds = snap(c.ClimbSeconds)
	if c.ClimbSeconds < StepSeconds {
		c.ClimbSeconds = StepSeconds
	}
	return c
}

// Nudge walks the climb by dir steps of 50ms. It will not go below
// one time step. A bad cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	v := snap(c.ClimbSeconds + StepSeconds*float64(dir))
	if v < StepSeconds {
		v = StepSeconds
	}
	c.ClimbSeconds = v
}
