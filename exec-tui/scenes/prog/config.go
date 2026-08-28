package prog

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
)

// Config is the seven live knobs on the program-alarm drop — four
// fall segments and three holds, 50ms at a time. The codes themselves
// are not knobs: historically 1202, 1202, then 1201. Play rebuilds
// from the current knobs. s writes this JSON next to the scene.
type Config struct {
	Drop1 float64 `json:"drop1"`
	Hold1 float64 `json:"hold1"`
	Drop2 float64 `json:"drop2"`
	Hold2 float64 `json:"hold2"`
	Drop3 float64 `json:"drop3"`
	Hold3 float64 `json:"hold3"`
	Drop4 float64 `json:"drop4"`
}

// Knob is which timing the cursor is on.
type Knob int

const (
	KnobDrop1 Knob = iota
	KnobHold1
	KnobDrop2
	KnobHold2
	KnobDrop3
	KnobHold3
	KnobDrop4
	KnobCount
)

const (
	// Drop1..Drop4 are the stock fall segments: four equal quarters
	// of the spacelander's six-second drop.
	Drop1 = 1.5
	Drop2 = 1.5
	Drop3 = 1.5
	Drop4 = 1.5
	// Hold1..Hold3 are the stock pauses under 1202, 1202, then 1201.
	Hold1 = 0.8
	Hold2 = 0.8
	Hold3 = 0.8

	// StepSeconds is one tick of a time knob: 50ms.
	StepSeconds = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/prog/config.json"
)

var (
	errDrop = errors.New("prog: each drop duration must be at least 50ms")
	errHold = errors.New("prog: a hold must not be negative")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobDrop1:
		return "drop 1"
	case KnobHold1:
		return "hold 1"
	case KnobDrop2:
		return "drop 2"
	case KnobHold2:
		return "hold 2"
	case KnobDrop3:
		return "drop 3"
	case KnobHold3:
		return "hold 3"
	case KnobDrop4:
		return "drop 4"
	default:
		return ""
	}
}

// Value is the selected knob's current seconds.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobDrop1:
		return c.Drop1
	case KnobHold1:
		return c.Hold1
	case KnobDrop2:
		return c.Drop2
	case KnobHold2:
		return c.Hold2
	case KnobDrop3:
		return c.Drop3
	case KnobHold3:
		return c.Hold3
	case KnobDrop4:
		return c.Drop4
	default:
		return 0
	}
}

// Codes is the historical first-three-alarm order: two 1202s in P63,
// then the 1201 in P64. Not a knob.
func Codes() []string {
	return []string{"1202", "1202", "1201"}
}

// Beats is the knobs as the lander's pausing drop.
func (c Config) Beats() []lander.DropBeat {
	return []lander.DropBeat{
		{Drop: c.Drop1, Hold: c.Hold1},
		{Drop: c.Drop2, Hold: c.Hold2},
		{Drop: c.Drop3, Hold: c.Hold3},
		{Drop: c.Drop4, Hold: 0},
	}
}

// DefaultConfig is the portable program-alarm drop's stock timing.
func DefaultConfig() Config {
	return Config{
		Drop1: Drop1, Hold1: Hold1,
		Drop2: Drop2, Hold2: Hold2,
		Drop3: Drop3, Hold3: Hold3,
		Drop4: Drop4,
	}
}

// Active is the timing New copies onto a prog scene: the last
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

func finiteNonNeg(v float64) bool {
	return v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0)
}

// Validate reports whether the knobs are playable.
func (c Config) Validate() error {
	for _, v := range []float64{c.Drop1, c.Drop2, c.Drop3, c.Drop4} {
		if v < StepSeconds || math.IsNaN(v) || math.IsInf(v, 0) {
			return errDrop
		}
	}
	for _, v := range []float64{c.Hold1, c.Hold2, c.Hold3} {
		if !finiteNonNeg(v) {
			return errHold
		}
	}
	return nil
}

// Load reads a prog-config JSON file.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("prog: %s: %w", path, err)
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
	raw := []byte(fmt.Sprintf("{\n"+
		"  \"drop1\": %.3f,\n"+
		"  \"hold1\": %.3f,\n"+
		"  \"drop2\": %.3f,\n"+
		"  \"hold2\": %.3f,\n"+
		"  \"drop3\": %.3f,\n"+
		"  \"hold3\": %.3f,\n"+
		"  \"drop4\": %.3f\n"+
		"}\n",
		c.Drop1, c.Hold1, c.Drop2, c.Hold2, c.Drop3, c.Hold3, c.Drop4))
	return os.WriteFile(path, raw, 0o644)
}

func snap(v float64) float64 {
	steps := 1 / StepSeconds
	return math.Round(v*steps) / steps
}

func (c Config) snapped() Config {
	c.Drop1 = snap(c.Drop1)
	c.Hold1 = snap(c.Hold1)
	c.Drop2 = snap(c.Drop2)
	c.Hold2 = snap(c.Hold2)
	c.Drop3 = snap(c.Drop3)
	c.Hold3 = snap(c.Hold3)
	c.Drop4 = snap(c.Drop4)
	if c.Drop1 < StepSeconds {
		c.Drop1 = StepSeconds
	}
	if c.Drop2 < StepSeconds {
		c.Drop2 = StepSeconds
	}
	if c.Drop3 < StepSeconds {
		c.Drop3 = StepSeconds
	}
	if c.Drop4 < StepSeconds {
		c.Drop4 = StepSeconds
	}
	if c.Hold1 < 0 {
		c.Hold1 = 0
	}
	if c.Hold2 < 0 {
		c.Hold2 = 0
	}
	if c.Hold3 < 0 {
		c.Hold3 = 0
	}
	return c
}

func (c *Config) set(k Knob, v float64) {
	switch k {
	case KnobDrop1:
		c.Drop1 = v
	case KnobHold1:
		c.Hold1 = v
	case KnobDrop2:
		c.Drop2 = v
	case KnobHold2:
		c.Hold2 = v
	case KnobDrop3:
		c.Drop3 = v
	case KnobHold3:
		c.Hold3 = v
	case KnobDrop4:
		c.Drop4 = v
	}
}

func (c Config) isDrop(k Knob) bool {
	return k == KnobDrop1 || k == KnobDrop2 || k == KnobDrop3 || k == KnobDrop4
}

// Nudge walks the selected knob by dir steps of 50ms. A drop will
// not go below one time step; a hold will not go negative. A bad
// cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	v := snap(c.Value(k) + StepSeconds*float64(dir))
	if c.isDrop(k) {
		if v < StepSeconds {
			v = StepSeconds
		}
	} else if v < 0 {
		v = 0
	}
	c.set(k, v)
}
