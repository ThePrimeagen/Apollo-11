package landing

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
)

// Config is the three live knobs on the landing: how long the craft
// takes to come down, when the pad dust starts, and how long it blows.
// The standalone runner nudges them 50ms at a time; Play rebuilds the
// scene from whatever they hold. s writes this JSON next to the scene.
// 02. Walkthrough plays the same Active config.
type Config struct {
	LandSeconds float64 `json:"landSeconds"`
	DustStart   float64 `json:"dustStart"`
	DustRun     float64 `json:"dustRun"`
}

// Knob is which timing the cursor is on.
type Knob int

const (
	KnobLand Knob = iota
	KnobDustStart
	KnobDustRun
	KnobCount
)

const (
	// StepSeconds is one tick of a live knob: 50ms.
	StepSeconds = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/landing/config.json"
)

var (
	errLand      = errors.New("landing: land duration must be at least 50ms")
	errDustStart = errors.New("landing: dust start must not be negative")
	errDustRun   = errors.New("landing: dust run must not be negative")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the portable landing's stock timing: a 5s fall,
// dust from the first booster step-down, blowing through off and the
// old two-second linger.
func DefaultConfig() Config {
	return Config{
		LandSeconds: LandSeconds,
		DustStart:   DustStart,
		DustRun:     DustRun,
	}
}

// Active is the timing New copies onto a landing scene: the last
// successful Use, or stock after Reset.
func Active() Config {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

// Use makes cfg the timing New and 02. Walkthrough play. A bad cfg
// is rejected and Active is unchanged.
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
	if c.LandSeconds < StepSeconds || math.IsNaN(c.LandSeconds) || math.IsInf(c.LandSeconds, 0) {
		return errLand
	}
	if c.DustStart < 0 || math.IsNaN(c.DustStart) || math.IsInf(c.DustStart, 0) {
		return errDustStart
	}
	if c.DustRun < 0 || math.IsNaN(c.DustRun) || math.IsInf(c.DustRun, 0) {
		return errDustRun
	}
	return nil
}

// Load reads a landing-config JSON file.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("landing: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c.snapped(), nil
}

// LoadOrDefault is Load, except a missing file is stock timing, not
// an error — the same courtesy the dust puff file gets.
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

// Save writes the three knobs as JSON, snapped to 50ms so the file
// stays easy to edit by hand.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	c = c.snapped()
	raw := []byte(fmt.Sprintf("{\n  \"landSeconds\": %.3f,\n  \"dustStart\": %.3f,\n  \"dustRun\": %.3f\n}\n",
		c.LandSeconds, c.DustStart, c.DustRun))
	return os.WriteFile(path, raw, 0o644)
}

func snap(v float64) float64 {
	return math.Round(v/StepSeconds) * StepSeconds
}

func (c Config) snapped() Config {
	c.LandSeconds = snap(c.LandSeconds)
	c.DustStart = snap(c.DustStart)
	c.DustRun = snap(c.DustRun)
	if c.LandSeconds < StepSeconds {
		c.LandSeconds = StepSeconds
	}
	if c.DustStart < 0 {
		c.DustStart = 0
	}
	if c.DustRun < 0 {
		c.DustRun = 0
	}
	return c
}

// Nudge walks the selected knob by dir steps of 50ms. Land will not
// go below one step; dust start and run will not go negative. A bad
// cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 {
		return
	}
	step := StepSeconds * float64(dir)
	switch k {
	case KnobLand:
		c.LandSeconds = snap(c.LandSeconds + step)
		if c.LandSeconds < StepSeconds {
			c.LandSeconds = StepSeconds
		}
	case KnobDustStart:
		c.DustStart = snap(c.DustStart + step)
		if c.DustStart < 0 {
			c.DustStart = 0
		}
	case KnobDustRun:
		c.DustRun = snap(c.DustRun + step)
		if c.DustRun < 0 {
			c.DustRun = 0
		}
	}
}
