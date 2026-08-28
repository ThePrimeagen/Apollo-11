package shootingstar

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/startrail"
)

// PathKind is the tuner's flight. Fall is the scene: right-to-left,
// high on the right to low on the left. Circle and square stay as
// optional closed loops so the tail is still readable.
type PathKind string

const (
	PathFall   PathKind = "fall"
	PathCircle PathKind = "circle"
	PathSquare PathKind = "square"
)

// Config is the live knobs of the shooting-star scene.
type Config struct {
	Path       PathKind `json:"path"`
	Size       int      `json:"size"`
	RandomSize bool     `json:"randomSize"`
	Speed      float64  `json:"speed"`
	Count      int      `json:"count"`
	Period     float64  `json:"period"`
	MinLife    float64  `json:"minLife"`
	MaxLife    float64  `json:"maxLife"`
	Nozzle     float64  `json:"nozzle"`
	Peak       float64  `json:"peak"`
	Taper      float64  `json:"taper"`
}

type Knob int

const (
	KnobPath Knob = iota
	KnobSize
	KnobRandomSize
	KnobSpeed
	KnobSpawn
	KnobPeriod
	KnobMinLife
	KnobMaxLife
	KnobNozzle
	KnobPeak
	KnobTaper
	KnobCount
)

const (
	StepSpeed  = 2.0
	StepPeriod = 0.005
	StepLife   = 0.05
	StepNozzle = 0.2
	StepPeak   = 1.0
	StepTaper  = 0.05

	DefaultConfigPath = "scenes/shootingstar/config.json"
)

var (
	errPath  = errors.New("shootingstar: path must be fall, circle, or square")
	errSpeed = errors.New("shootingstar: speed must be a finite number")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

func KnobLabel(k Knob) string {
	switch k {
	case KnobPath:
		return "path"
	case KnobSize:
		return "size"
	case KnobRandomSize:
		return "random size"
	case KnobSpeed:
		return "speed"
	case KnobSpawn:
		return "count"
	case KnobPeriod:
		return "period"
	case KnobMinLife:
		return "min life"
	case KnobMaxLife:
		return "max life"
	case KnobNozzle:
		return "nozzle"
	case KnobPeak:
		return "peak"
	case KnobTaper:
		return "taper"
	default:
		return ""
	}
}

func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobPath:
		switch c.Path {
		case PathCircle:
			return 1
		case PathSquare:
			return 2
		default:
			return 0
		}
	case KnobSize:
		return float64(c.Size)
	case KnobRandomSize:
		if c.RandomSize {
			return 1
		}
		return 0
	case KnobSpeed:
		return c.Speed
	case KnobSpawn:
		return float64(c.Count)
	case KnobPeriod:
		return c.Period
	case KnobMinLife:
		return c.MinLife
	case KnobMaxLife:
		return c.MaxLife
	case KnobNozzle:
		return c.Nozzle
	case KnobPeak:
		return c.Peak
	case KnobTaper:
		return c.Taper
	default:
		return 0
	}
}

func DefaultConfig() Config {
	t := startrail.DefaultConfig()
	return Config{
		Path:       PathFall,
		Size:       2,
		RandomSize: false,
		Speed:      28,
		Count:      t.Count,
		Period:     t.Period,
		MinLife:    t.MinLife,
		MaxLife:    t.MaxLife,
		Nozzle:     t.Nozzle,
		Peak:       t.Peak,
		Taper:      t.Taper,
	}
}

func (c Config) Trail() startrail.Config {
	return startrail.Config{
		Count:   c.Count,
		Period:  c.Period,
		MinLife: c.MinLife,
		MaxLife: c.MaxLife,
		Nozzle:  c.Nozzle,
		Peak:    c.Peak,
		Taper:   c.Taper,
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
	if err := startrail.Use(cfg.Trail()); err != nil {
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
	if c.Path != PathFall && c.Path != PathCircle && c.Path != PathSquare {
		return errPath
	}
	if math.IsNaN(c.Speed) || math.IsInf(c.Speed, 0) {
		return errSpeed
	}
	return nil
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("shootingstar: %s: %w", path, err)
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
	raw := []byte(fmt.Sprintf("{\n"+
		"  \"path\": %q,\n"+
		"  \"size\": %d,\n"+
		"  \"randomSize\": %t,\n"+
		"  \"speed\": %.1f,\n"+
		"  \"count\": %d,\n"+
		"  \"period\": %.3f,\n"+
		"  \"minLife\": %.2f,\n"+
		"  \"maxLife\": %.2f,\n"+
		"  \"nozzle\": %.2f,\n"+
		"  \"peak\": %.1f,\n"+
		"  \"taper\": %.2f\n"+
		"}\n",
		c.Path, c.Size, c.RandomSize, c.Speed, c.Count, c.Period, c.MinLife, c.MaxLife, c.Nozzle, c.Peak, c.Taper))
	return os.WriteFile(path, raw, 0o644)
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

func snap(v, step float64) float64 {
	return math.Round(v/step) * step
}

func (c *Config) Nudge(k Knob, dir int) {
	// Never clamp. Size, speed, and every other numeric knob keep
	// whatever the step lands on, including negatives.
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	switch k {
	case KnobPath:
		switch c.Path {
		case PathFall:
			if dir > 0 {
				c.Path = PathCircle
			} else {
				c.Path = PathSquare
			}
		case PathCircle:
			if dir > 0 {
				c.Path = PathSquare
			} else {
				c.Path = PathFall
			}
		default:
			if dir > 0 {
				c.Path = PathFall
			} else {
				c.Path = PathCircle
			}
		}
	case KnobSize:
		c.Size += dir
	case KnobRandomSize:
		c.RandomSize = dir > 0
	case KnobSpeed:
		c.Speed = snap(c.Speed+StepSpeed*float64(dir), StepSpeed)
	case KnobSpawn:
		c.Count += dir
	case KnobPeriod:
		c.Period = snap(c.Period+StepPeriod*float64(dir), StepPeriod)
	case KnobMinLife:
		c.MinLife = snap(c.MinLife+StepLife*float64(dir), StepLife)
	case KnobMaxLife:
		c.MaxLife = snap(c.MaxLife+StepLife*float64(dir), StepLife)
	case KnobNozzle:
		c.Nozzle = snap(c.Nozzle+StepNozzle*float64(dir), StepNozzle)
	case KnobPeak:
		c.Peak = snap(c.Peak+StepPeak*float64(dir), StepPeak)
	case KnobTaper:
		c.Taper = snap(c.Taper+StepTaper*float64(dir), StepTaper)
	}
}
