package director

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

const (
	// DefaultHoldSeconds is how long a scene the file does not name
	// plays in play mode before the cut.
	DefaultHoldSeconds = 8.0

	// HoldStepSeconds is one tick of a hold knob: half a second.
	// Nudge steps; it does not enforce floors or ceilings — a zero
	// or negative hold is the operator's number and cuts at once.
	HoldStepSeconds = 0.5
)

// SceneHold is one scene's dwell in play mode: the scene's marquee
// name and how many seconds it plays before the editor cuts.
type SceneHold struct {
	Scene   string  `json:"scene"`
	Seconds float64 `json:"seconds"`
}

// Config is the editor's own knob file: a hold per scene, kept in
// bill order so the JSON reads like the show. Scenes the file does
// not carry read the stock hold. Scenes sharing one name share one
// hold.
type Config struct {
	Holds []SceneHold `json:"holds"`
}

// HoldFor is the hold for the named scene: the stored number —
// whatever it is — or the stock hold when the name is unknown.
func (c Config) HoldFor(name string) float64 {
	for _, h := range c.Holds {
		if h.Scene == name {
			return h.Seconds
		}
	}
	return DefaultHoldSeconds
}

// SetHold stores the hold for the named scene verbatim — updating in
// place, or appending in first-touch order. Nil-safe.
func (c *Config) SetHold(name string, seconds float64) {
	if c == nil {
		return
	}
	for i := range c.Holds {
		if c.Holds[i].Scene == name {
			c.Holds[i].Seconds = seconds
			return
		}
	}
	c.Holds = append(c.Holds, SceneHold{Scene: name, Seconds: seconds})
}

// Validate reports whether the holds are playable numbers. Zero and
// negative holds are playable — they cut at once; only NaN and Inf,
// numbers no knob can be turned onto, are rejected.
func (c Config) Validate() error {
	for _, h := range c.Holds {
		if math.IsNaN(h.Seconds) || math.IsInf(h.Seconds, 0) {
			return fmt.Errorf("director: the hold for %q is not a number", h.Scene)
		}
	}
	return nil
}

// Load reads a holds JSON file.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("director: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// LoadOrDefault is Load, except a missing file is just the stock
// holds, not an error — the same courtesy every scene config gets.
func LoadOrDefault(path string) (Config, error) {
	c, err := Load(path)
	if err == nil {
		return c, nil
	}
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	return Config{}, err
}

// Save writes the holds as JSON, one line per scene in stored order,
// so the file stays easy to edit by hand.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("{\n  \"holds\": [\n")
	for i, h := range c.Holds {
		sep := ","
		if i == len(c.Holds)-1 {
			sep = ""
		}
		name, err := json.Marshal(h.Scene)
		if err != nil {
			return fmt.Errorf("director: %s: %w", h.Scene, err)
		}
		fmt.Fprintf(&b, "    { \"scene\": %s, \"seconds\": %.3f }%s\n", name, h.Seconds, sep)
	}
	b.WriteString("  ]\n}\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
