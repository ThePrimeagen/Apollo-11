package sky

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
)

// Config is the live knobs on the blue sky: the angle the darker
// blue comes from (0° is straight down from the top, 45° is a
// diagonal from the top-right), the light-blue ink, and the
// dark-blue ink. The tuner nudges the angle 5°, the inks one xterm
// index at a time; s writes this JSON next to the component.
type Config struct {
	AngleDeg float64 `json:"angleDeg"`
	LightInk int     `json:"lightInk"`
	DarkInk  int     `json:"darkInk"`
}

type fileJSON struct {
	AngleDeg *float64 `json:"angleDeg"`
	LightInk *int     `json:"lightInk"`
	DarkInk  *int     `json:"darkInk"`
}

// Knob is which knob the cursor is on.
type Knob int

const (
	KnobAngle Knob = iota
	KnobLight
	KnobDark
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobAngle:
		return "angle"
	case KnobLight:
		return "light blue"
	case KnobDark:
		return "dark blue"
	default:
		return ""
	}
}

func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobAngle:
		return c.AngleDeg
	case KnobLight:
		return float64(c.LightInk)
	case KnobDark:
		return float64(c.DarkInk)
	default:
		return 0
	}
}

// Display is knob k's panel reading.
func (c Config) Display(k Knob) string {
	switch k {
	case KnobAngle:
		return fmt.Sprintf("%7.3f°", c.AngleDeg)
	case KnobLight:
		return fmt.Sprintf("%7d", c.LightInk)
	case KnobDark:
		return fmt.Sprintf("%7d", c.DarkInk)
	default:
		return ""
	}
}

const (
	// StepAngle is one tick of the angle knob: 5°.
	StepAngle = 5.0

	// DefaultConfigPath is the component's own JSON, relative to the
	// module root.
	DefaultConfigPath = "components/sky/config.json"
)

var (
	ErrAngle = errors.New("sky: angle must sit on 0..359 degrees")
	ErrInk   = errors.New("sky: inks must sit on the xterm cube 1..255")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the stock sky: dark from the top, pale horizon,
// deep zenith.
func DefaultConfig() Config {
	return Config{
		AngleDeg: DefaultAngle,
		LightInk: DefaultLight,
		DarkInk:  DefaultDark,
	}
}

// Active is the knobs New paints: the last successful Use, or stock
// after Reset.
func Active() Config {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

// Use makes cfg the knobs New paints. A bad cfg is rejected and
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

// Reset restores stock knobs. Tests call this so a Use cannot leak.
func Reset() {
	activeMu.Lock()
	active = DefaultConfig()
	activeMu.Unlock()
}

// Validate reports whether the knobs are playable.
func (c Config) Validate() error {
	if c.AngleDeg < 0 || c.AngleDeg >= 360 || math.IsNaN(c.AngleDeg) || math.IsInf(c.AngleDeg, 0) {
		return ErrAngle
	}
	if c.LightInk < 1 || c.LightInk > 255 || c.DarkInk < 1 || c.DarkInk > 255 {
		return ErrInk
	}
	return nil
}

// Load reads a sky-config JSON file.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var f fileJSON
	if err := json.Unmarshal(raw, &f); err != nil {
		return Config{}, fmt.Errorf("sky: %s: %w", path, err)
	}
	c := DefaultConfig()
	if f.AngleDeg != nil {
		c.AngleDeg = *f.AngleDeg
	}
	if f.LightInk != nil {
		c.LightInk = *f.LightInk
	}
	if f.DarkInk != nil {
		c.DarkInk = *f.DarkInk
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// LoadOrDefault is Load, except a missing file is stock knobs, not
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

// Save writes the knobs as JSON.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	raw := []byte(fmt.Sprintf("{\n  \"angleDeg\": %.3f,\n  \"lightInk\": %d,\n  \"darkInk\": %d\n}\n",
		c.AngleDeg, c.LightInk, c.DarkInk))
	return os.WriteFile(path, raw, 0o644)
}

// Nudge walks the selected knob by dir steps of its grid. The angle
// wraps the circle; the inks rail on 1..255. A bad cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	switch k {
	case KnobAngle:
		v := c.AngleDeg + StepAngle*float64(dir)
		v = math.Mod(v, 360)
		if v < 0 {
			v += 360
		}
		c.AngleDeg = v
	case KnobLight:
		c.LightInk = railInk(c.LightInk + dir)
	case KnobDark:
		c.DarkInk = railInk(c.DarkInk + dir)
	}
}

func railInk(n int) int {
	if n < 1 {
		return 1
	}
	if n > 255 {
		return 255
	}
	return n
}
