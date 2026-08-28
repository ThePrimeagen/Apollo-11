package bobble

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
)

// Config is the live knobs on the bobble: whether the engine burns,
// how long one full up-and-down of the parked ride takes, and how
// many cells it rides from center. The standalone runner flips the
// engine with h/l, walks the period 50ms at a time and the amplitude
// one cell; Play rebuilds the scene from whatever they hold. s writes
// this JSON next to the scene. 03. Inverse Walkthrough plays the same
// Active ride, lit on one entry and dark on the next.
type Config struct {
	Engine         bool    `json:"engine"`
	PeriodSeconds  float64 `json:"periodSeconds"`
	AmplitudeCells int     `json:"amplitudeCells"`
}

// Knob is which knob the cursor is on.
type Knob int

const (
	KnobEngine Knob = iota
	KnobPeriod
	KnobAmplitude
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobEngine:
		return "engine"
	case KnobPeriod:
		return "period"
	case KnobAmplitude:
		return "amplitude"
	default:
		return ""
	}
}

// Value reads one knob as a float for display and tests: the engine
// reads 1 lit and 0 dark.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobEngine:
		if c.Engine {
			return 1
		}
		return 0
	case KnobPeriod:
		return c.PeriodSeconds
	case KnobAmplitude:
		return float64(c.AmplitudeCells)
	default:
		return 0
	}
}

const (
	// StepSeconds is one tick of the period knob: 50ms.
	StepSeconds = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/bobble/config.json"
)

var (
	errPeriod    = errors.New("bobble: the ride period must be at least 50ms")
	errAmplitude = errors.New("bobble: the ride amplitude must not be negative")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the premiere's stock parked ride, engine lit.
func DefaultConfig() Config {
	return Config{
		Engine:         true,
		PeriodSeconds:  lander.BobPeriodSeconds,
		AmplitudeCells: lander.BobAmplitudeCells,
	}
}

// Active is the ride New copies onto a bobble scene: the last
// successful Use, or stock after Reset.
func Active() Config {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

// Use makes cfg the ride New plays. A bad cfg is rejected and Active
// is unchanged.
func Use(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	activeMu.Lock()
	active = cfg
	activeMu.Unlock()
	return nil
}

// Reset restores the stock ride. Tests call this so a Use cannot leak.
func Reset() {
	activeMu.Lock()
	active = DefaultConfig()
	activeMu.Unlock()
}

// Validate reports whether the knobs are playable.
func (c Config) Validate() error {
	if c.PeriodSeconds < StepSeconds || math.IsNaN(c.PeriodSeconds) || math.IsInf(c.PeriodSeconds, 0) {
		return errPeriod
	}
	if c.AmplitudeCells < 0 {
		return errAmplitude
	}
	return nil
}

// Load reads a bobble-config JSON file. Keys the file does not carry
// keep their stock values.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("bobble: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c.snapped(), nil
}

// LoadOrDefault is Load, except a missing file is the stock ride, not
// an error — the same courtesy every scene config gets.
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
	raw := []byte(fmt.Sprintf("{\n"+
		"  \"engine\": %t,\n"+
		"  \"periodSeconds\": %.3f,\n"+
		"  \"amplitudeCells\": %d\n"+
		"}\n",
		c.Engine, c.PeriodSeconds, c.AmplitudeCells))
	return os.WriteFile(path, raw, 0o644)
}

func snap(v float64) float64 {
	steps := 1 / StepSeconds
	return math.Round(v*steps) / steps
}

func (c Config) snapped() Config {
	c.PeriodSeconds = math.Max(snap(c.PeriodSeconds), StepSeconds)
	if c.AmplitudeCells < 0 {
		c.AmplitudeCells = 0
	}
	return c
}

// Nudge walks the selected knob by dir steps. The engine flips on
// with l and off with h, the skies way; the period moves 50ms and
// will not go below one step; the amplitude moves one cell and will
// not go negative. A bad cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	switch k {
	case KnobEngine:
		c.Engine = dir > 0
	case KnobPeriod:
		v := snap(c.PeriodSeconds + StepSeconds*float64(dir))
		if v < StepSeconds {
			v = StepSeconds
		}
		c.PeriodSeconds = v
	case KnobAmplitude:
		v := c.AmplitudeCells + dir
		if v < 0 {
			v = 0
		}
		c.AmplitudeCells = v
	}
}
