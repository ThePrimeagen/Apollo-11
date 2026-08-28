package interpreter

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
)

// Config is the walkthrough's two timings as live knobs: how long
// the spotlight rests on each instruction, and how long the camera
// glides to the next stop. The standalone runner nudges each knob
// 50ms at a time and s writes this JSON next to the scene; the stop
// marks are derived clock math, so a retimed show still knows where
// its spotlight rests.
type Config struct {
	HoldSeconds  float64 `json:"holdSeconds"`
	GlideSeconds float64 `json:"glideSeconds"`
}

const (
	// StepSeconds is one tick of every knob: 50ms. It is also the
	// floor under the glide — a camera with no duration has no
	// clock to ease along.
	StepSeconds = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/interpreter/config.json"
)

// DefaultConfig is the scene as staged: the stock rest and the stock
// glide.
func DefaultConfig() Config {
	return Config{
		HoldSeconds:  HoldSeconds,
		GlideSeconds: GlideSeconds,
	}
}

// StopStart is when the spotlight reaches instruction i: i full
// hold-plus-glide periods in.
func (c Config) StopStart(i int) float64 {
	return float64(i) * (c.HoldSeconds + c.GlideSeconds)
}

// GlideStart is when the camera leaves instruction i for the next:
// the moment its hold ends.
func (c Config) GlideStart(i int) float64 {
	return c.StopStart(i) + c.HoldSeconds
}

// Knob is which knob the cursor is on.
type Knob int

const (
	KnobHold Knob = iota
	KnobGlide
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobHold:
		return "hold"
	case KnobGlide:
		return "glide"
	default:
		return ""
	}
}

// Value reads one knob's seconds for the panel and the tests.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobHold:
		return c.HoldSeconds
	case KnobGlide:
		return c.GlideSeconds
	default:
		return 0
	}
}

// Display is knob k's panel reading, seconds to the millisecond.
func (c Config) Display(k Knob) string {
	if k < 0 || k >= KnobCount {
		return ""
	}
	return fmt.Sprintf("%7.3fs", c.Value(k))
}

// floor is the lowest a knob may go: the glide keeps one step of
// duration, the hold may reach zero — a spotlight that never rests
// is a choice.
func floor(k Knob) float64 {
	if k == KnobGlide {
		return StepSeconds
	}
	return 0
}

func (c *Config) set(k Knob, v float64) {
	switch k {
	case KnobHold:
		c.HoldSeconds = v
	case KnobGlide:
		c.GlideSeconds = v
	}
}

// Nudge walks the selected knob by dir steps of the 50ms grid,
// rounded to the millisecond so a nudge up and back down always
// lands on the exact same value, clamped at the knob's floor. A bad
// cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	v := math.Round((c.Value(k)+StepSeconds*float64(dir))*1000) / 1000
	if lo := floor(k); v < lo {
		v = lo
	}
	c.set(k, v)
}

var (
	errHold  = errors.New("interpreter: the hold must not be negative")
	errGlide = errors.New("interpreter: the glide must be at least 50ms")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// finite is a knob that is a real duration, not a NaN or an infinity.
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Validate reports whether the knobs are playable. The hold may be
// zero, but the glide keeps at least one step of duration, because
// it divides the camera's clock.
func (c Config) Validate() error {
	if c.HoldSeconds < 0 || !finite(c.HoldSeconds) {
		return errHold
	}
	if c.GlideSeconds < StepSeconds || !finite(c.GlideSeconds) {
		return errGlide
	}
	return nil
}

// Active is the timing New copies onto an Interpreter scene: the
// last successful Use, or stock after Reset.
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

// Load reads an Interpreter config JSON file. Missing keys keep
// their stock values; broken knobs are refused loudly.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("interpreter: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, fmt.Errorf("interpreter: %s: %w", path, err)
	}
	return c, nil
}

// LoadOrDefault is Load, except a missing file is stock timing, not
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

// Save writes the knobs as JSON. A broken config refuses to save.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
