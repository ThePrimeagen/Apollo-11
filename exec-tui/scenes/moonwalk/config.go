// Package moonwalk is the tunable astronaut show: the crate climb,
// the pole-top landing, the flag hoist, and the closing pan to the
// rover, all driven by a knob config the tuner TUI can save.
package moonwalk

import "fmt"

// DefaultConfigPath is where the tuner saves the knobs, relative to
// the module root.
const DefaultConfigPath = "scenes/moonwalk/config.json"

// Pole height rails: short enough to always fit a stage, tall enough
// to tower over the crates.
const (
	MinPoleRows = 10
	MaxPoleRows = 24
)

// Config is every knob the scene exposes.
type Config struct {
	StrideFPS    float64 `json:"stride_fps"`
	RunSpeed     float64 `json:"run_speed"`
	JumpSeconds  float64 `json:"jump_seconds"`
	SlideSeconds float64 `json:"slide_seconds"`
	FlagSeconds  float64 `json:"flag_seconds"`
	PoleRows     int     `json:"pole_rows"`
	PanCols      int     `json:"pan_cols"`
	PanSeconds   float64 `json:"pan_seconds"`
}

// DefaultConfig is the show as originally staged.
func DefaultConfig() Config { return Config{} }

// Knob names one adjustable property.
type Knob int

const (
	KnobStrideFPS Knob = iota
	KnobRunSpeed
	KnobJumpSeconds
	KnobSlideSeconds
	KnobFlagSeconds
	KnobPoleRows
	KnobPanCols
	KnobPanSeconds
	KnobCount
)

func (k Knob) String() string { return "?" }

// Value reads one knob as a float for display and tests.
func (c Config) Value(k Knob) float64 { return 0 }

// Nudge moves one knob a step in a direction, clamped to its rails.
func (c *Config) Nudge(k Knob, dir int) {}

// Save writes the knobs as JSON.
func (c Config) Save(path string) error { return fmt.Errorf("moonwalk: Save not implemented") }

// LoadOrDefault reads knobs from path; a missing file is the default
// show, a corrupt file is a loud error.
func LoadOrDefault(path string) (Config, error) {
	return Config{}, fmt.Errorf("moonwalk: LoadOrDefault not implemented")
}
