package fall

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
)

// Config is the live knobs on the spacelander fall — how long the
// north-facing craft takes to drop from off the top to off the
// bottom, and three optional holds (MAIN's 1202 / 1202 / 1201
// pauses). Stock holds are zero: walkthrough stays a plain drop.
// The standalone runner nudges 50ms at a time; Play rebuilds from
// whatever they hold. s writes this JSON next to the scene.
type Config struct {
	DropSeconds float64 `json:"dropSeconds"`
	Hold1       float64 `json:"hold1"`
	Hold2       float64 `json:"hold2"`
	Hold3       float64 `json:"hold3"`
}

// Knob is which timing the cursor is on.
type Knob int

const (
	KnobDrop Knob = iota
	KnobHold1
	KnobHold2
	KnobHold3
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobDrop:
		return "drop"
	case KnobHold1:
		return "hold 1"
	case KnobHold2:
		return "hold 2"
	case KnobHold3:
		return "hold 3"
	default:
		return ""
	}
}

// Value is the selected knob's current seconds.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobDrop:
		return c.DropSeconds
	case KnobHold1:
		return c.Hold1
	case KnobHold2:
		return c.Hold2
	case KnobHold3:
		return c.Hold3
	default:
		return 0
	}
}

// Armed reports whether any hold is a positive pause — MAIN turns
// the alarm overlay on by setting these; stock walkthrough leaves
// them at zero.
func (c Config) Armed() bool {
	return c.Hold1 > 0 || c.Hold2 > 0 || c.Hold3 > 0
}

const (
	// StepSeconds is one tick of the drop knob: 50ms.
	StepSeconds = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/fall/config.json"
)

var (
	errDrop = errors.New("fall: drop duration must be at least 50ms")
	errHold = errors.New("fall: a hold must be a finite number")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the portable fall's stock timing.
func DefaultConfig() Config {
	return Config{DropSeconds: lander.DropSeconds}
}

// Active is the timing New copies onto a fall scene: the last
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

// Validate reports whether the knobs are playable. Holds may be zero
// or negative — those skip a card — but they must be finite.
func (c Config) Validate() error {
	if c.DropSeconds < StepSeconds || math.IsNaN(c.DropSeconds) || math.IsInf(c.DropSeconds, 0) {
		return errDrop
	}
	for _, v := range []float64{c.Hold1, c.Hold2, c.Hold3} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return errHold
		}
	}
	return nil
}

// Load reads a fall-config JSON file.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("fall: %s: %w", path, err)
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
	raw := []byte(fmt.Sprintf("{\n  \"dropSeconds\": %.3f,\n  \"hold1\": %.3f,\n  \"hold2\": %.3f,\n  \"hold3\": %.3f\n}\n",
		c.DropSeconds, c.Hold1, c.Hold2, c.Hold3))
	return os.WriteFile(path, raw, 0o644)
}

func snap(v float64) float64 {
	steps := 1 / StepSeconds
	return math.Round(v*steps) / steps
}

func (c Config) snapped() Config {
	c.DropSeconds = snap(c.DropSeconds)
	if c.DropSeconds < StepSeconds {
		c.DropSeconds = StepSeconds
	}
	c.Hold1 = snap(c.Hold1)
	c.Hold2 = snap(c.Hold2)
	c.Hold3 = snap(c.Hold3)
	return c
}

func (c *Config) set(k Knob, v float64) {
	switch k {
	case KnobDrop:
		c.DropSeconds = v
	case KnobHold1:
		c.Hold1 = v
	case KnobHold2:
		c.Hold2 = v
	case KnobHold3:
		c.Hold3 = v
	}
}

// Nudge walks the selected knob by dir steps of 50ms. The drop will
// not go below one time step (the existing floor). Holds are the
// operator's number — Nudge does not clamp them. A bad cursor is a
// no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	v := snap(c.Value(k) + StepSeconds*float64(dir))
	if k == KnobDrop && v < StepSeconds {
		v = StepSeconds
	}
	c.set(k, v)
}
