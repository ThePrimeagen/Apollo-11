package explorer

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
)

// Config is the live knobs on the explorer scene: the four twinkle
// ranges the sky breathes — how long one full fade cycle may take,
// min and max, and how long each fade ramp may take, min and max, all
// in seconds. The standalone runner walks the cycle knobs 250ms at a
// time and the fade knobs 50ms; every knob lives between the stars
// package's twinkle rails, and a pair can never cross. s writes this
// JSON next to the scene.
type Config struct {
	MinCycleSeconds float64 `json:"minCycleSeconds"`
	MaxCycleSeconds float64 `json:"maxCycleSeconds"`
	MinFadeSeconds  float64 `json:"minFadeSeconds"`
	MaxFadeSeconds  float64 `json:"maxFadeSeconds"`
}

// Knob is which knob the cursor is on.
type Knob int

const (
	KnobMinCycle Knob = iota
	KnobMaxCycle
	KnobMinFade
	KnobMaxFade
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobMinCycle:
		return "min cycle"
	case KnobMaxCycle:
		return "max cycle"
	case KnobMinFade:
		return "min fade"
	case KnobMaxFade:
		return "max fade"
	default:
		return ""
	}
}

// Value reads one knob in seconds, for display and tests.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobMinCycle:
		return c.MinCycleSeconds
	case KnobMaxCycle:
		return c.MaxCycleSeconds
	case KnobMinFade:
		return c.MinFadeSeconds
	case KnobMaxFade:
		return c.MaxFadeSeconds
	default:
		return 0
	}
}

const (
	// CycleStepSeconds is one tick of a cycle knob: 250ms.
	CycleStepSeconds = 0.250
	// FadeStepSeconds is one tick of a fade knob: 50ms.
	FadeStepSeconds = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/explorer/config.json"
)

var (
	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the stock breathing — the same ranges the stars
// package wakes up with.
func DefaultConfig() Config {
	d := stars.DefaultTwinkle()
	return Config{
		MinCycleSeconds: d.MinCycleSeconds,
		MaxCycleSeconds: d.MaxCycleSeconds,
		MinFadeSeconds:  d.MinFadeSeconds,
		MaxFadeSeconds:  d.MaxFadeSeconds,
	}
}

// Twinkle is the knobs as the stars package speaks them.
func (c Config) Twinkle() stars.TwinkleConfig {
	return stars.TwinkleConfig{
		MinCycleSeconds: c.MinCycleSeconds,
		MaxCycleSeconds: c.MaxCycleSeconds,
		MinFadeSeconds:  c.MinFadeSeconds,
		MaxFadeSeconds:  c.MaxFadeSeconds,
	}
}

// Active is the knobs New copies onto an explorer scene: the last
// successful Use, or stock after Reset.
func Active() Config {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

// Use makes cfg the knobs New plays and pushes the same numbers onto
// the stars package, so the sky breathes them live. A bad cfg is
// rejected and nothing moves.
func Use(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := stars.UseTwinkle(cfg.Twinkle()); err != nil {
		return err
	}
	activeMu.Lock()
	active = cfg
	activeMu.Unlock()
	return nil
}

// Reset restores the stock knobs. Tests call this so a Use cannot
// leak; the sky's own active twinkle is the stars package's to reset.
func Reset() {
	activeMu.Lock()
	active = DefaultConfig()
	activeMu.Unlock()
}

// Validate reports whether the knobs are playable — the stars
// package's twinkle rails are the law here too.
func (c Config) Validate() error {
	return c.Twinkle().Validate()
}

// Load reads an explorer-config JSON file. Keys the file does not
// carry keep their stock values.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("explorer: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// LoadOrDefault is Load, except a missing file is the stock knobs,
// not an error — the same courtesy every scene config gets.
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

// Save writes the knobs as JSON, two decimals so the file stays easy
// to edit by hand.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	raw := []byte(fmt.Sprintf("{\n"+
		"  \"minCycleSeconds\": %.2f,\n"+
		"  \"maxCycleSeconds\": %.2f,\n"+
		"  \"minFadeSeconds\": %.2f,\n"+
		"  \"maxFadeSeconds\": %.2f\n"+
		"}\n",
		c.MinCycleSeconds, c.MaxCycleSeconds, c.MinFadeSeconds, c.MaxFadeSeconds))
	return os.WriteFile(path, raw, 0o644)
}

// snapTo rounds v to the knob's step so a walked knob stays a clean
// multiple.
func snapTo(v, step float64) float64 {
	return math.Round(v/step) * step
}

// clampTo pins v inside [lo, hi].
func clampTo(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Nudge walks the selected knob by dir steps — cycles 250ms at a
// time, fades 50ms. Every knob stops at the stars package's twinkle
// rails, and a pair never crosses: a min climbing into its max (or a
// max dipping into its min) clamps at the partner. A bad cursor is a
// no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	switch k {
	case KnobMinCycle:
		v := snapTo(c.MinCycleSeconds+CycleStepSeconds*float64(dir), CycleStepSeconds)
		c.MinCycleSeconds = clampTo(v, stars.MinTwinkleCycle, math.Min(stars.MaxTwinkleCycle, c.MaxCycleSeconds))
	case KnobMaxCycle:
		v := snapTo(c.MaxCycleSeconds+CycleStepSeconds*float64(dir), CycleStepSeconds)
		c.MaxCycleSeconds = clampTo(v, math.Max(stars.MinTwinkleCycle, c.MinCycleSeconds), stars.MaxTwinkleCycle)
	case KnobMinFade:
		v := snapTo(c.MinFadeSeconds+FadeStepSeconds*float64(dir), FadeStepSeconds)
		c.MinFadeSeconds = clampTo(v, stars.MinTwinkleFade, math.Min(stars.MaxTwinkleFade, c.MaxFadeSeconds))
	case KnobMaxFade:
		v := snapTo(c.MaxFadeSeconds+FadeStepSeconds*float64(dir), FadeStepSeconds)
		c.MaxFadeSeconds = clampTo(v, math.Max(stars.MinTwinkleFade, c.MinFadeSeconds), stars.MaxTwinkleFade)
	}
}
