package startrail

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

var (
	ErrCount  = errors.New("startrail: count must not be negative")
	ErrPeriod = errors.New("startrail: period must not be negative")
	ErrLife   = errors.New("startrail: min life is greater than max life")
	ErrNozzle = errors.New("startrail: nozzle must not be negative")
	ErrPeak   = errors.New("startrail: peak must not be negative")
	ErrTaper  = errors.New("startrail: taper must be between 0 and 1")
	ErrNeg    = errors.New("startrail: life must not be negative")
)

// Config is the JSON that tunes the persist trail: how many specks
// drop each period, how long they live, how thick the nozzle is, how
// steeply spawn piles on the spine (peak), and how hard max life
// falls off with |offset| (taper).
type Config struct {
	Count   int     `json:"count"`
	Period  float64 `json:"period"`
	MinLife float64 `json:"minLife"`
	MaxLife float64 `json:"maxLife"`
	Nozzle  float64 `json:"nozzle"`
	Peak    float64 `json:"peak"`
	Taper   float64 `json:"taper"`
}

const DefaultConfigPath = "components/startrail/config.json"

var (
	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is a tight, readable comet wake.
func DefaultConfig() Config {
	return Config{
		Count:   3,
		Period:  0.015,
		MinLife: 0.50,
		MaxLife: 0.90,
		Nozzle:  4.0,
		Peak:    6,
		Taper:   0.85,
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
	if c.Period < 0 {
		return ErrPeriod
	}
	if c.MinLife < 0 || c.MaxLife < 0 {
		return ErrNeg
	}
	if c.MinLife > c.MaxLife {
		return ErrLife
	}
	if c.Nozzle < 0 {
		return ErrNozzle
	}
	if c.Peak < 0 {
		return ErrPeak
	}
	if c.Taper < 0 || c.Taper > 1 {
		return ErrTaper
	}
	return nil
}

// ParticleConfig is the persist engine world this trail describes.
func (c Config) ParticleConfig(width, height float64, origin, heading particle.Vec2) particle.Config {
	if heading == (particle.Vec2{}) {
		heading = particle.Vec2{X: 1, Y: 0}
	}
	return particle.Config{
		Width:     width,
		Height:    height,
		Origin:    origin,
		Direction: heading,
		Count:     c.Count,
		Period:    c.Period,
		MinLife:   c.MinLife,
		MaxLife:   c.MaxLife,
		MinSpeed:  0,
		MaxSpeed:  0,
		Nozzle:    c.Nozzle,
		Peak:      c.Peak,
		Taper:     c.Taper,
	}.Persist()
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("startrail: %s: %w", path, err)
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
		"  \"count\": %d,\n"+
		"  \"period\": %.3f,\n"+
		"  \"minLife\": %.2f,\n"+
		"  \"maxLife\": %.2f,\n"+
		"  \"nozzle\": %.2f,\n"+
		"  \"peak\": %.1f,\n"+
		"  \"taper\": %.2f\n"+
		"}\n",
		c.Count, c.Period, c.MinLife, c.MaxLife, c.Nozzle, c.Peak, c.Taper))
	return os.WriteFile(path, raw, 0o644)
}
