package america

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
)

// Config is the live knobs on the scene: how long the flag takes to
// fade in from black, when the eagle enters, and how long its
// crossing takes — the eagle's speed across the stage. The standalone
// runner nudges them 50ms at a time, the same way the landing scene
// tunes; Play rebuilds the scene from whatever they hold. s writes
// this JSON next to the scene.
type Config struct {
	FadeSeconds  float64 `json:"fadeSeconds"`
	EagleDelay   float64 `json:"eagleDelay"`
	CrossSeconds float64 `json:"crossSeconds"`
}

// fileJSON is the on-disk shape. Every key is a pointer so a file
// missing one keeps that knob at stock.
type fileJSON struct {
	FadeSeconds  *float64 `json:"fadeSeconds"`
	EagleDelay   *float64 `json:"eagleDelay"`
	CrossSeconds *float64 `json:"crossSeconds"`
}

// Knob is which timing the cursor is on.
type Knob int

const (
	KnobFade Knob = iota
	KnobDelay
	KnobCross
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobFade:
		return "flag fade"
	case KnobDelay:
		return "eagle delay"
	case KnobCross:
		return "eagle cross"
	default:
		return ""
	}
}

// Value is the selected knob's current seconds.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobFade:
		return c.FadeSeconds
	case KnobDelay:
		return c.EagleDelay
	case KnobCross:
		return c.CrossSeconds
	default:
		return 0
	}
}

const (
	// StepSeconds is one tick of a knob: 50ms.
	StepSeconds = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/america/config.json"
)

var (
	errFade  = errors.New("america: flag fade must not be negative")
	errDelay = errors.New("america: eagle delay must not be negative")
	errCross = errors.New("america: eagle cross must be at least 50ms")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the scene's stock timing: the fast two-second
// fade, the eagle entering the moment the fade lands, the
// four-second crossing.
func DefaultConfig() Config {
	return Config{
		FadeSeconds:  FadeSeconds,
		EagleDelay:   FadeSeconds,
		CrossSeconds: CrossSeconds,
	}
}

// Active is the timing New copies onto an America scene: the last
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

// Validate reports whether the knobs are playable. An instant flag
// and an eagle at t=0 are allowed; a crossing needs a duration.
func (c Config) Validate() error {
	if c.FadeSeconds < 0 || math.IsNaN(c.FadeSeconds) || math.IsInf(c.FadeSeconds, 0) {
		return errFade
	}
	if c.EagleDelay < 0 || math.IsNaN(c.EagleDelay) || math.IsInf(c.EagleDelay, 0) {
		return errDelay
	}
	if c.CrossSeconds < StepSeconds || math.IsNaN(c.CrossSeconds) || math.IsInf(c.CrossSeconds, 0) {
		return errCross
	}
	return nil
}

// Load reads an America-config JSON file.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var f fileJSON
	if err := json.Unmarshal(raw, &f); err != nil {
		return Config{}, fmt.Errorf("america: %s: %w", path, err)
	}
	c := DefaultConfig()
	if f.FadeSeconds != nil {
		c.FadeSeconds = *f.FadeSeconds
	}
	if f.EagleDelay != nil {
		c.EagleDelay = *f.EagleDelay
	}
	if f.CrossSeconds != nil {
		c.CrossSeconds = *f.CrossSeconds
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c.snapped(), nil
}

// LoadOrDefault is Load, except a missing file is stock timing, not
// an error — the same courtesy the landing config gets.
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
		"  \"fadeSeconds\": %.3f,\n"+
		"  \"eagleDelay\": %.3f,\n"+
		"  \"crossSeconds\": %.3f\n"+
		"}\n",
		c.FadeSeconds, c.EagleDelay, c.CrossSeconds))
	return os.WriteFile(path, raw, 0o644)
}

func snap(v float64) float64 {
	steps := 1 / StepSeconds
	return math.Round(v*steps) / steps
}

func (c Config) snapped() Config {
	c.FadeSeconds = snap(c.FadeSeconds)
	c.EagleDelay = snap(c.EagleDelay)
	c.CrossSeconds = snap(c.CrossSeconds)
	if c.FadeSeconds < 0 {
		c.FadeSeconds = 0
	}
	if c.EagleDelay < 0 {
		c.EagleDelay = 0
	}
	if c.CrossSeconds < StepSeconds {
		c.CrossSeconds = StepSeconds
	}
	return c
}

func (c *Config) set(k Knob, v float64) {
	switch k {
	case KnobFade:
		c.FadeSeconds = v
	case KnobDelay:
		c.EagleDelay = v
	case KnobCross:
		c.CrossSeconds = v
	}
}

// Nudge walks the selected knob by dir 50ms steps. The fade and the
// delay will not go negative; the crossing will not go below one
// step. A bad cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	v := snap(c.Value(k) + StepSeconds*float64(dir))
	if k == KnobCross {
		if v < StepSeconds {
			v = StepSeconds
		}
	} else if v < 0 {
		v = 0
	}
	c.set(k, v)
}
