package armed

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	StockDelay        = 0.5
	StockCross        = 8.0
	StockShots        = 3
	StockRate         = 2.0
	StepSeconds       = 0.050
	StepPoint         = 0.050
	StepRate          = 0.25
	DefaultConfigPath = "components/armed/config.json"
)

type Config struct {
	Delay      float64        `json:"delay"`
	Cross      float64        `json:"cross"`
	Start      float64        `json:"start"`
	End        float64        `json:"end"`
	LeftShots  int            `json:"leftShots"`
	LeftRate   float64        `json:"leftRate"`
	LeftAim    sprite.Heading `json:"leftAim"`
	RightShots int            `json:"rightShots"`
	RightRate  float64        `json:"rightRate"`
	RightAim   sprite.Heading `json:"rightAim"`
}

type fileJSON struct {
	Delay      *float64        `json:"delay"`
	Cross      *float64        `json:"cross"`
	Start      *float64        `json:"start"`
	End        *float64        `json:"end"`
	LeftShots  *int            `json:"leftShots"`
	LeftRate   *float64        `json:"leftRate"`
	LeftAim    *sprite.Heading `json:"leftAim"`
	RightShots *int            `json:"rightShots"`
	RightRate  *float64        `json:"rightRate"`
	RightAim   *sprite.Heading `json:"rightAim"`
}

type Knob int

const (
	KnobDelay Knob = iota
	KnobCross
	KnobStart
	KnobEnd
	KnobLeftShots
	KnobLeftRate
	KnobLeftAim
	KnobRightShots
	KnobRightRate
	KnobRightAim
	KnobCount
)

func KnobLabel(k Knob) string {
	switch k {
	case KnobDelay:
		return "eagle delay"
	case KnobCross:
		return "eagle cross"
	case KnobStart:
		return "eagle start"
	case KnobEnd:
		return "eagle end"
	case KnobLeftShots:
		return "left shots"
	case KnobLeftRate:
		return "left rate"
	case KnobLeftAim:
		return "left aim"
	case KnobRightShots:
		return "right shots"
	case KnobRightRate:
		return "right rate"
	case KnobRightAim:
		return "right aim"
	default:
		return ""
	}
}

func KnobUnit(k Knob) string {
	switch k {
	case KnobDelay, KnobCross:
		return "s"
	default:
		return ""
	}
}

func headingIdx(h sprite.Heading) int {
	for i, hh := range sprite.Headings {
		if h == hh {
			return i
		}
	}
	return -1
}

func (c Config) aimAt(k Knob) sprite.Heading {
	switch k {
	case KnobLeftAim:
		return c.LeftAim
	case KnobRightAim:
		return c.RightAim
	default:
		return ""
	}
}

func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobDelay:
		return c.Delay
	case KnobCross:
		return c.Cross
	case KnobStart:
		return c.Start
	case KnobEnd:
		return c.End
	case KnobLeftShots:
		return float64(c.LeftShots)
	case KnobRightShots:
		return float64(c.RightShots)
	case KnobLeftRate:
		return c.LeftRate
	case KnobRightRate:
		return c.RightRate
	case KnobLeftAim, KnobRightAim:
		return float64(headingIdx(c.aimAt(k)))
	default:
		return 0
	}
}

func (c Config) Display(k Knob) string {
	switch k {
	case KnobDelay, KnobCross, KnobStart, KnobEnd:
		return fmt.Sprintf("%7.3f%s", c.Value(k), KnobUnit(k))
	case KnobLeftShots:
		return fmt.Sprintf("%7d", c.LeftShots)
	case KnobRightShots:
		return fmt.Sprintf("%7d", c.RightShots)
	case KnobLeftRate:
		return fmt.Sprintf("%7.2f/s", c.LeftRate)
	case KnobRightRate:
		return fmt.Sprintf("%7.2f/s", c.RightRate)
	case KnobLeftAim, KnobRightAim:
		return fmt.Sprintf("%7s", string(c.aimAt(k)))
	default:
		return ""
	}
}

var (
	errDelay = errors.New("armed: eagle delay must not be negative")
	errCross = errors.New("armed: eagle cross must be at least 50ms")
	errStart = errors.New("armed: eagle start must sit inside the span")
	errEnd   = errors.New("armed: eagle end must sit inside the span")
	errPath  = errors.New("armed: eagle end must be at least one step past its start")
	errShots = errors.New("armed: shell counts must not be negative")
	errRate  = errors.New("armed: rates of fire must not be negative")
	errAim   = errors.New("armed: aims must sit on the eight-point compass")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

func DefaultConfig() Config {
	return Config{
		Delay:      StockDelay,
		Cross:      StockCross,
		Start:      0,
		End:        1,
		LeftShots:  StockShots,
		LeftRate:   StockRate,
		LeftAim:    sprite.W,
		RightShots: StockShots,
		RightRate:  StockRate,
		RightAim:   sprite.E,
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
	if c.Delay < 0 || math.IsNaN(c.Delay) || math.IsInf(c.Delay, 0) {
		return errDelay
	}
	if c.Cross < StepSeconds || math.IsNaN(c.Cross) || math.IsInf(c.Cross, 0) {
		return errCross
	}
	if c.Start < 0 || c.Start > 1 || math.IsNaN(c.Start) {
		return errStart
	}
	if c.End < 0 || c.End > 1 || math.IsNaN(c.End) {
		return errEnd
	}
	if c.End-c.Start < StepPoint-1e-9 {
		return errPath
	}
	if c.LeftShots < 0 || c.RightShots < 0 {
		return errShots
	}
	if c.LeftRate < 0 || c.RightRate < 0 || math.IsNaN(c.LeftRate) || math.IsNaN(c.RightRate) {
		return errRate
	}
	if headingIdx(c.LeftAim) < 0 || headingIdx(c.RightAim) < 0 {
		return errAim
	}
	return nil
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var f fileJSON
	if err := json.Unmarshal(raw, &f); err != nil {
		return Config{}, fmt.Errorf("armed: %s: %w", path, err)
	}
	c := DefaultConfig()
	if f.Delay != nil {
		c.Delay = *f.Delay
	}
	if f.Cross != nil {
		c.Cross = *f.Cross
	}
	if f.Start != nil {
		c.Start = *f.Start
	}
	if f.End != nil {
		c.End = *f.End
	}
	if f.LeftShots != nil {
		c.LeftShots = *f.LeftShots
	}
	if f.LeftRate != nil {
		c.LeftRate = *f.LeftRate
	}
	if f.LeftAim != nil {
		c.LeftAim = *f.LeftAim
	}
	if f.RightShots != nil {
		c.RightShots = *f.RightShots
	}
	if f.RightRate != nil {
		c.RightRate = *f.RightRate
	}
	if f.RightAim != nil {
		c.RightAim = *f.RightAim
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c.snapped(), nil
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
	c = c.snapped()
	raw := []byte(fmt.Sprintf("{\n"+
		"  \"delay\": %.3f,\n"+
		"  \"cross\": %.3f,\n"+
		"  \"start\": %.3f,\n"+
		"  \"end\": %.3f,\n"+
		"  \"leftShots\": %d,\n"+
		"  \"leftRate\": %.2f,\n"+
		"  \"leftAim\": %q,\n"+
		"  \"rightShots\": %d,\n"+
		"  \"rightRate\": %.2f,\n"+
		"  \"rightAim\": %q\n"+
		"}\n",
		c.Delay, c.Cross, c.Start, c.End,
		c.LeftShots, c.LeftRate, string(c.LeftAim), c.RightShots, c.RightRate, string(c.RightAim)))
	return os.WriteFile(path, raw, 0o644)
}

func snap(v float64) float64 {
	steps := 1 / StepSeconds
	return math.Round(v*steps) / steps
}

func snapRate(v float64) float64 {
	return math.Round(v/StepRate) * StepRate
}

func (c Config) snapped() Config {
	c.Delay = snap(c.Delay)
	c.Cross = snap(c.Cross)
	c.Start = snap(c.Start)
	c.End = snap(c.End)
	c.LeftRate = snapRate(c.LeftRate)
	c.RightRate = snapRate(c.RightRate)
	if c.Delay < 0 {
		c.Delay = 0
	}
	if c.Cross < StepSeconds {
		c.Cross = StepSeconds
	}
	if c.Start < 0 {
		c.Start = 0
	}
	if c.End > 1 {
		c.End = 1
	}
	if c.LeftRate < 0 {
		c.LeftRate = 0
	}
	if c.RightRate < 0 {
		c.RightRate = 0
	}
	return c
}

func (c *Config) set(k Knob, v float64) {
	switch k {
	case KnobDelay:
		c.Delay = v
	case KnobCross:
		c.Cross = v
	case KnobStart:
		c.Start = v
	case KnobEnd:
		c.End = v
	}
}

func step(k Knob) float64 {
	switch k {
	case KnobStart, KnobEnd:
		return StepPoint
	default:
		return StepSeconds
	}
}

func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	switch k {
	case KnobLeftShots:
		c.LeftShots = flooredShells(c.LeftShots + dir)
		return
	case KnobRightShots:
		c.RightShots = flooredShells(c.RightShots + dir)
		return
	case KnobLeftRate:
		c.LeftRate = floorRate(c.LeftRate + StepRate*float64(dir))
		return
	case KnobRightRate:
		c.RightRate = floorRate(c.RightRate + StepRate*float64(dir))
		return
	case KnobLeftAim:
		c.LeftAim = walkedAim(c.LeftAim, dir)
		return
	case KnobRightAim:
		c.RightAim = walkedAim(c.RightAim, dir)
		return
	}
	v := snap(c.Value(k) + step(k)*float64(dir))
	switch k {
	case KnobCross:
		if v < StepSeconds {
			v = StepSeconds
		}
	case KnobStart:
		if v < 0 {
			v = 0
		}
		if lim := c.End - StepPoint; v > lim {
			v = lim
		}
	case KnobEnd:
		if v > 1 {
			v = 1
		}
		if lim := c.Start + StepPoint; v < lim {
			v = lim
		}
	default:
		if v < 0 {
			v = 0
		}
	}
	c.set(k, v)
}

func flooredShells(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

func floorRate(v float64) float64 {
	if v < 0 {
		return 0
	}
	return snapRate(v)
}

func walkedAim(h sprite.Heading, dir int) sprite.Heading {
	n := len(sprite.Headings)
	idx := headingIdx(h)
	if idx < 0 {
		idx = 0
		if dir > 0 {
			dir--
		}
	}
	return sprite.Headings[((idx+dir)%n+n)%n]
}
