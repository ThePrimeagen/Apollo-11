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

// SceneConfig is one scene's row in MAIN's own config: the scene's
// marquee name, how many seconds it plays in play mode before the
// cut, and — for a scene with knobs — those knobs in the scene's own
// JSON shape.
type SceneConfig struct {
	Scene   string          `json:"scene"`
	Seconds float64         `json:"seconds"`
	Knobs   json.RawMessage `json:"knobs,omitempty"`
}

// Config is MAIN's own set of configs: one row per scene, kept in
// bill order so the JSON reads like the show. It is the show's file,
// not the scenes' — nothing in it ever reaches a scene package's own
// config or its Active. Scenes the file does not carry read the stock
// hold and no knobs; scenes sharing one name share one row.
type Config struct {
	Scenes []SceneConfig `json:"scenes"`
}

// row is the index of the named scene's row, or -1.
func (c Config) row(name string) int {
	for i := range c.Scenes {
		if c.Scenes[i].Scene == name {
			return i
		}
	}
	return -1
}

// HoldFor is the hold for the named scene: the stored number —
// whatever it is — or the stock hold when the name is unknown.
func (c Config) HoldFor(name string) float64 {
	if i := c.row(name); i >= 0 {
		return c.Scenes[i].Seconds
	}
	return DefaultHoldSeconds
}

// KnobsFor is the named scene's knobs in the scene's own JSON shape,
// or nil when the file carries none.
func (c Config) KnobsFor(name string) json.RawMessage {
	if i := c.row(name); i >= 0 {
		return c.Scenes[i].Knobs
	}
	return nil
}

// SetHold stores the hold for the named scene verbatim — updating its
// row in place, or appending one in first-touch order. Nil-safe.
func (c *Config) SetHold(name string, seconds float64) {
	if c == nil {
		return
	}
	if i := c.row(name); i >= 0 {
		c.Scenes[i].Seconds = seconds
		return
	}
	c.Scenes = append(c.Scenes, SceneConfig{Scene: name, Seconds: seconds})
}

// SetKnobs stores the named scene's knobs on its row — a new name
// opens a row at the stock hold. Nil-safe.
func (c *Config) SetKnobs(name string, raw json.RawMessage) {
	if c == nil {
		return
	}
	if i := c.row(name); i >= 0 {
		c.Scenes[i].Knobs = raw
		return
	}
	c.Scenes = append(c.Scenes, SceneConfig{Scene: name, Seconds: DefaultHoldSeconds, Knobs: raw})
}

// Validate reports whether the holds are playable numbers. Zero and
// negative holds are playable — they cut at once; only NaN and Inf,
// numbers no knob can be turned onto, are rejected.
func (c Config) Validate() error {
	for _, s := range c.Scenes {
		if math.IsNaN(s.Seconds) || math.IsInf(s.Seconds, 0) {
			return fmt.Errorf("director: the hold for %q is not a number", s.Scene)
		}
	}
	return nil
}

// Load reads MAIN's config JSON.
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
// show, not an error — the same courtesy every scene config gets.
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

// Save writes the config as JSON, one line per scene in stored order,
// knobs inline, so the file stays easy to read by hand.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("{\n  \"scenes\": [\n")
	for i, s := range c.Scenes {
		sep := ","
		if i == len(c.Scenes)-1 {
			sep = ""
		}
		name, err := json.Marshal(s.Scene)
		if err != nil {
			return fmt.Errorf("director: %s: %w", s.Scene, err)
		}
		fmt.Fprintf(&b, "    { \"scene\": %s, \"seconds\": %.3f", name, s.Seconds)
		if len(s.Knobs) > 0 {
			knobs, err := json.Marshal(s.Knobs)
			if err != nil {
				return fmt.Errorf("director: %s knobs: %w", s.Scene, err)
			}
			fmt.Fprintf(&b, ", \"knobs\": %s", knobs)
		}
		fmt.Fprintf(&b, " }%s\n", sep)
	}
	b.WriteString("  ]\n}\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
