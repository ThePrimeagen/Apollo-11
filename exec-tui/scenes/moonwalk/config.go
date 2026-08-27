// Package moonwalk is the tunable astronaut show: the crate climb,
// the pole-top landing, the flag hoist, and the closing pan to the
// rover, all driven by a knob config the tuner TUI can save.
package moonwalk

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// DefaultConfigPath is where the tuner saves the knobs, relative to
// the module root.
const DefaultConfigPath = "scenes/moonwalk/config.json"

// Pole height rails: short enough to always fit a stage, tall enough
// to tower over the crates.
const (
	MinPoleRows = 10
	MaxPoleRows = 28
)

// Box start rails: how many columns before the pole the first stack
// begins — wide enough for three stacks and a landing.
const (
	MinBoxStart = 38
	MaxBoxStart = 90
)

// Config is every knob the scene exposes.
type Config struct {
	StrideFPS    float64 `json:"stride_fps"`
	RunSpeed     float64 `json:"run_speed"`
	JumpSeconds  float64 `json:"jump_seconds"`
	BoxStart     int     `json:"box_start"`
	TopSeconds   float64 `json:"top_seconds"`
	SlideSeconds float64 `json:"slide_seconds"`
	FlagSeconds  float64 `json:"flag_seconds"`
	PoleRows     int     `json:"pole_rows"`
	ExitSpeed    float64 `json:"exit_speed"`
	PanCols      int     `json:"pan_cols"`
	PanSeconds   float64 `json:"pan_seconds"`
}

// DefaultConfig is the show as staged: a quick sprint, three stacks
// parked a hop from the pole, a beat at the top, a slow hoist that
// outlasts the slide, and a pan wide enough to reveal the whole
// module before he runs over and boards it.
func DefaultConfig() Config {
	return Config{
		StrideFPS:    12,
		RunSpeed:     20,
		JumpSeconds:  0.55,
		BoxStart:     46,
		TopSeconds:   0.6,
		SlideSeconds: 1.8,
		FlagSeconds:  3.0,
		PoleRows:     21,
		ExitSpeed:    18,
		PanCols:      30,
		PanSeconds:   1.4,
	}
}

// Knob names one adjustable property.
type Knob int

const (
	KnobStrideFPS Knob = iota
	KnobRunSpeed
	KnobJumpSeconds
	KnobBoxStart
	KnobTopSeconds
	KnobSlideSeconds
	KnobFlagSeconds
	KnobPoleRows
	KnobExitSpeed
	KnobPanCols
	KnobPanSeconds
	KnobCount
)

func (k Knob) String() string {
	switch k {
	case KnobStrideFPS:
		return "stride fps"
	case KnobRunSpeed:
		return "run speed"
	case KnobJumpSeconds:
		return "jump s"
	case KnobBoxStart:
		return "box start"
	case KnobTopSeconds:
		return "top s"
	case KnobSlideSeconds:
		return "slide s"
	case KnobFlagSeconds:
		return "flag s"
	case KnobPoleRows:
		return "pole rows"
	case KnobExitSpeed:
		return "exit speed"
	case KnobPanCols:
		return "pan cols"
	case KnobPanSeconds:
		return "pan s"
	default:
		return "?"
	}
}

// Value reads one knob as a float for display and tests.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobStrideFPS:
		return c.StrideFPS
	case KnobRunSpeed:
		return c.RunSpeed
	case KnobJumpSeconds:
		return c.JumpSeconds
	case KnobBoxStart:
		return float64(c.BoxStart)
	case KnobTopSeconds:
		return c.TopSeconds
	case KnobSlideSeconds:
		return c.SlideSeconds
	case KnobFlagSeconds:
		return c.FlagSeconds
	case KnobPoleRows:
		return float64(c.PoleRows)
	case KnobExitSpeed:
		return c.ExitSpeed
	case KnobPanCols:
		return float64(c.PanCols)
	case KnobPanSeconds:
		return c.PanSeconds
	default:
		return 0
	}
}

// nudgeFloat steps a float knob and rounds to the cent so a nudge up
// and back down always lands on the exact same value.
func nudgeFloat(v, step float64, dir int, lo, hi float64) float64 {
	v = math.Round((v+step*float64(dir))*100) / 100
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func nudgeInt(v, dir, lo, hi int) int {
	v += dir
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Nudge moves one knob a step in a direction, clamped to its rails.
func (c *Config) Nudge(k Knob, dir int) {
	if dir == 0 {
		return
	}
	switch k {
	case KnobStrideFPS:
		c.StrideFPS = nudgeFloat(c.StrideFPS, 0.5, dir, 1, 30)
	case KnobRunSpeed:
		c.RunSpeed = nudgeFloat(c.RunSpeed, 1, dir, 2, 60)
	case KnobJumpSeconds:
		c.JumpSeconds = nudgeFloat(c.JumpSeconds, 0.05, dir, 0.2, 2)
	case KnobBoxStart:
		c.BoxStart = nudgeInt(c.BoxStart, dir, MinBoxStart, MaxBoxStart)
	case KnobTopSeconds:
		c.TopSeconds = nudgeFloat(c.TopSeconds, 0.05, dir, 0, 4)
	case KnobSlideSeconds:
		c.SlideSeconds = nudgeFloat(c.SlideSeconds, 0.05, dir, 0.3, 6)
	case KnobFlagSeconds:
		c.FlagSeconds = nudgeFloat(c.FlagSeconds, 0.05, dir, 0.2, 8)
	case KnobPoleRows:
		c.PoleRows = nudgeInt(c.PoleRows, dir, MinPoleRows, MaxPoleRows)
	case KnobExitSpeed:
		c.ExitSpeed = nudgeFloat(c.ExitSpeed, 1, dir, 4, 60)
	case KnobPanCols:
		// No ceiling — pan as far as you like; the floor just keeps
		// the camera from walking backwards.
		c.PanCols = nudgeInt(c.PanCols, dir, 0, math.MaxInt)
	case KnobPanSeconds:
		c.PanSeconds = nudgeFloat(c.PanSeconds, 0.05, dir, 0.2, 5)
	}
}

// Save writes the knobs as JSON.
func (c Config) Save(path string) error {
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}

// LoadOrDefault reads knobs from path; a missing file is the default
// show, a corrupt file is a loud error.
func LoadOrDefault(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("moonwalk: %s: %w", path, err)
	}
	return c, nil
}
