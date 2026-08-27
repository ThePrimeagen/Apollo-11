package cloud

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

// The xterm-256 grayscale ramp, near-black to near-white.
const (
	GrayMin = 232
	GrayMax = 255
)

var (
	ErrCount  = errors.New("cloud: count must not be negative")
	ErrPuffs  = errors.New("cloud: puffs must not be negative")
	ErrRadius = errors.New("cloud: radius must not be negative")
	ErrSpread = errors.New("cloud: spread must not be negative")
	ErrField  = errors.New("cloud: field must not be negative")
	ErrLadder = errors.New("cloud: thresholds must climb: 1 <= thinAt < thickAt")
	ErrGray   = errors.New("cloud: grays must sit on the xterm gray ramp 232..255")
)

// Config is the JSON that tunes the cloud generator: how many specks
// each blob parks, how many blobs make one cloud, how wide each pool
// is, how far the blobs scatter from the cloud's centre, how many
// clouds a field plants in the upper sky, and the white/gray ladder
// that decides which symbol a cell's concentration earns.
type Config struct {
	Count  int     `json:"count"`
	Puffs  int     `json:"puffs"`
	Radius float64 `json:"radius"`
	Spread float64 `json:"spread"`
	Field  int     `json:"field"`

	ThinAt  int `json:"thinAt"`
	ThickAt int `json:"thickAt"`
	ThinFG  int `json:"thinFG"`
	MidFG   int `json:"midFG"`
	ThickFG int `json:"thickFG"`
}

type fileJSON struct {
	Count   *int     `json:"count"`
	Puffs   *int     `json:"puffs"`
	Radius  *float64 `json:"radius"`
	Spread  *float64 `json:"spread"`
	Field   *int     `json:"field"`
	ThinAt  *int     `json:"thinAt"`
	ThickAt *int     `json:"thickAt"`
	ThinFG  *int     `json:"thinFG"`
	MidFG   *int     `json:"midFG"`
	ThickFG *int     `json:"thickFG"`
}

// Knob is which knob the cursor is on.
type Knob int

const (
	KnobSpecks Knob = iota
	KnobPuffs
	KnobRadius
	KnobSpread
	KnobThinAt
	KnobThickAt
	KnobThinFG
	KnobMidFG
	KnobThickFG
	KnobField
	KnobCount
)

const (
	StepRadius = 0.5
	StepSpread = 0.5

	DefaultConfigPath = "components/cloud/config.json"
)

var (
	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the stock cloud: several overlapping pool blobs
// of white-gray specks, fluffy enough to read as a unique puff.
func DefaultConfig() Config {
	return Config{
		Count:   18,
		Puffs:   4,
		Radius:  6,
		Spread:  10,
		Field:   6,
		ThinAt:  2,
		ThickAt: 5,
		ThinFG:  250,
		MidFG:   252,
		ThickFG: 255,
	}
}

func Active() Config {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

func Use(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	activeMu.Lock()
	active = cfg
	activeMu.Unlock()
	return nil
}

func Reset() {
	activeMu.Lock()
	active = DefaultConfig()
	activeMu.Unlock()
}

func (c Config) Validate() error {
	if c.Count < 0 {
		return ErrCount
	}
	if c.Puffs < 0 {
		return ErrPuffs
	}
	if c.Radius < 0 {
		return ErrRadius
	}
	if c.Spread < 0 {
		return ErrSpread
	}
	if c.Field < 0 {
		return ErrField
	}
	if c.ThinAt < 1 || c.ThickAt <= c.ThinAt {
		return ErrLadder
	}
	for _, g := range []int{c.ThinFG, c.MidFG, c.ThickFG} {
		if g < GrayMin || g > GrayMax {
			return ErrGray
		}
	}
	return nil
}

// Engine is a parked pool world from these knobs, centred on origin
// inside a w×h unit box. Period 0: Burst once and stay.
func (c Config) Engine(w, h float64, origin particle.Vec2) particle.Config {
	return particle.Config{
		Width:      w,
		Height:     h,
		Origin:     origin,
		Direction:  particle.Vec2{X: 0, Y: -1},
		Count:      c.Count,
		Period:     0,
		MinLife:    1e6,
		MaxLife:    1e6,
		MinSpeed:   0,
		MaxSpeed:   0,
		PoolRadius: c.Radius,
	}.Pooled()
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var f fileJSON
	if err := json.Unmarshal(raw, &f); err != nil {
		return Config{}, fmt.Errorf("cloud: %s: %w", path, err)
	}
	c := DefaultConfig()
	if f.Count != nil {
		c.Count = *f.Count
	}
	if f.Puffs != nil {
		c.Puffs = *f.Puffs
	}
	if f.Radius != nil {
		c.Radius = *f.Radius
	}
	if f.Spread != nil {
		c.Spread = *f.Spread
	}
	if f.Field != nil {
		c.Field = *f.Field
	}
	if f.ThinAt != nil {
		c.ThinAt = *f.ThinAt
	}
	if f.ThickAt != nil {
		c.ThickAt = *f.ThickAt
	}
	if f.ThinFG != nil {
		c.ThinFG = *f.ThinFG
	}
	if f.MidFG != nil {
		c.MidFG = *f.MidFG
	}
	if f.ThickFG != nil {
		c.ThickFG = *f.ThickFG
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

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

func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func KnobLabel(k Knob) string {
	switch k {
	case KnobSpecks:
		return "count"
	case KnobPuffs:
		return "puffs"
	case KnobRadius:
		return "radius"
	case KnobSpread:
		return "spread"
	case KnobThinAt:
		return "thin at"
	case KnobThickAt:
		return "thick at"
	case KnobThinFG:
		return "thin gray"
	case KnobMidFG:
		return "mid gray"
	case KnobThickFG:
		return "thick gray"
	case KnobField:
		return "field"
	default:
		return ""
	}
}

func (c Config) Display(k Knob) string {
	switch k {
	case KnobSpecks:
		return fmt.Sprintf("%7d", c.Count)
	case KnobPuffs:
		return fmt.Sprintf("%7d", c.Puffs)
	case KnobRadius:
		return fmt.Sprintf("%7.1f", c.Radius)
	case KnobSpread:
		return fmt.Sprintf("%7.1f", c.Spread)
	case KnobThinAt:
		return fmt.Sprintf("%7d", c.ThinAt)
	case KnobThickAt:
		return fmt.Sprintf("%7d", c.ThickAt)
	case KnobThinFG:
		return fmt.Sprintf("%7d", c.ThinFG)
	case KnobMidFG:
		return fmt.Sprintf("%7d", c.MidFG)
	case KnobThickFG:
		return fmt.Sprintf("%7d", c.ThickFG)
	case KnobField:
		return fmt.Sprintf("%7d", c.Field)
	default:
		return ""
	}
}

func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	switch k {
	case KnobSpecks:
		c.Count = floor0(c.Count + dir)
	case KnobPuffs:
		c.Puffs = floor0(c.Puffs + dir)
	case KnobRadius:
		c.Radius = floor0f(c.Radius + StepRadius*float64(dir))
	case KnobSpread:
		c.Spread = floor0f(c.Spread + StepSpread*float64(dir))
	case KnobThinAt:
		c.ThinAt += dir
		if c.ThinAt < 1 {
			c.ThinAt = 1
		}
		if c.ThinAt >= c.ThickAt {
			c.ThinAt = c.ThickAt - 1
		}
	case KnobThickAt:
		c.ThickAt += dir
		if c.ThickAt <= c.ThinAt {
			c.ThickAt = c.ThinAt + 1
		}
	case KnobThinFG:
		c.ThinFG = railGray(c.ThinFG + dir)
	case KnobMidFG:
		c.MidFG = railGray(c.MidFG + dir)
	case KnobThickFG:
		c.ThickFG = railGray(c.ThickFG + dir)
	case KnobField:
		c.Field = floor0(c.Field + dir)
	}
}

func floor0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

func floor0f(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

func railGray(n int) int {
	if n < GrayMin {
		return GrayMin
	}
	if n > GrayMax {
		return GrayMax
	}
	return n
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
