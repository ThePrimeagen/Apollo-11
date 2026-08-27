package skies

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
	RiseSeconds   = 3.0
	CrossSeconds  = 4.0
	StartPoint    = 0.0
	EndPoint      = 1.0
	StockShots    = 3
	StockRate     = 2.0
	StockLeftAim  = sprite.W
	StockRightAim = sprite.E
)

type Config struct {
	RiseSeconds  float64        `json:"riseSeconds"`
	EagleDelay   float64        `json:"eagleDelay"`
	CrossSeconds float64        `json:"crossSeconds"`
	EagleStart   float64        `json:"eagleStart"`
	EagleEnd     float64        `json:"eagleEnd"`
	LeftShots    int            `json:"leftShots"`
	LeftRate     float64        `json:"leftRate"`
	LeftAim      sprite.Heading `json:"leftAim"`
	RightShots   int            `json:"rightShots"`
	RightRate    float64        `json:"rightRate"`
	RightAim     sprite.Heading `json:"rightAim"`
}

type fileJSON struct {
	RiseSeconds  *float64        `json:"riseSeconds"`
	EagleDelay   *float64        `json:"eagleDelay"`
	CrossSeconds *float64        `json:"crossSeconds"`
	EagleStart   *float64        `json:"eagleStart"`
	EagleEnd     *float64        `json:"eagleEnd"`
	LeftShots    *int            `json:"leftShots"`
	LeftRate     *float64        `json:"leftRate"`
	LeftAim      *sprite.Heading `json:"leftAim"`
	RightShots   *int            `json:"rightShots"`
	RightRate    *float64        `json:"rightRate"`
	RightAim     *sprite.Heading `json:"rightAim"`
}

type Knob int

const (
	KnobRise Knob = iota
	KnobDelay
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
	case KnobRise:
		return "sky rise"
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
	case KnobRise, KnobDelay, KnobCross:
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
	case KnobRise:
		return c.RiseSeconds
	case KnobDelay:
		return c.EagleDelay
	case KnobCross:
		return c.CrossSeconds
	case KnobStart:
		return c.EagleStart
	case KnobEnd:
		return c.EagleEnd
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
	case KnobRise, KnobDelay, KnobCross, KnobStart, KnobEnd:
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

const (
	StepSeconds       = 0.050
	StepPoint         = 0.050
	StepRate          = 0.25
	DefaultConfigPath = "scenes/skies/config.json"
)

var (
	errRise  = errors.New("skies: sky rise must not be negative")
	errDelay = errors.New("skies: eagle delay must not be negative")
	errCross = errors.New("skies: eagle cross must be at least 50ms")
	errStart = errors.New("skies: eagle start must sit inside the span")
	errEnd   = errors.New("skies: eagle end must sit inside the span")
	errPath  = errors.New("skies: eagle end must be at least one step past its start")
	errShots = errors.New("skies: shell counts must not be negative")
	errRate  = errors.New("skies: rates of fire must not be negative")
	errAim   = errors.New("skies: aims must sit on the eight-point compass")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

func DefaultConfig() Config {
	return Config{
		RiseSeconds:  RiseSeconds,
		EagleDelay:   2.0,
		CrossSeconds: CrossSeconds,
		EagleStart:   StartPoint,
		EagleEnd:     EndPoint,
		LeftShots:    StockShots,
		LeftRate:     StockRate,
		LeftAim:      StockLeftAim,
		RightShots:   StockShots,
		RightRate:    StockRate,
		RightAim:     StockRightAim,
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
	if c.RiseSeconds < 0 || math.IsNaN(c.RiseSeconds) || math.IsInf(c.RiseSeconds, 0) {
		return errRise
	}
	if c.EagleDelay < 0 || math.IsNaN(c.EagleDelay) || math.IsInf(c.EagleDelay, 0) {
		return errDelay
	}
	if c.CrossSeconds < StepSeconds || math.IsNaN(c.CrossSeconds) || math.IsInf(c.CrossSeconds, 0) {
		return errCross
	}
	if c.EagleStart < 0 || c.EagleStart > 1 || math.IsNaN(c.EagleStart) {
		return errStart
	}
	if c.EagleEnd < 0 || c.EagleEnd > 1 || math.IsNaN(c.EagleEnd) {
		return errEnd
	}
	if c.EagleEnd-c.EagleStart < StepPoint-1e-9 {
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
		return Config{}, fmt.Errorf("skies: %s: %w", path, err)
	}
	c := DefaultConfig()
	if f.RiseSeconds != nil {
		c.RiseSeconds = *f.RiseSeconds
	}
	if f.EagleDelay != nil {
		c.EagleDelay = *f.EagleDelay
	}
	if f.CrossSeconds != nil {
		c.CrossSeconds = *f.CrossSeconds
	}
	if f.EagleStart != nil {
		c.EagleStart = *f.EagleStart
	}
	if f.EagleEnd != nil {
		c.EagleEnd = *f.EagleEnd
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
		"  \"riseSeconds\": %.3f,\n"+
		"  \"eagleDelay\": %.3f,\n"+
		"  \"crossSeconds\": %.3f,\n"+
		"  \"eagleStart\": %.3f,\n"+
		"  \"eagleEnd\": %.3f,\n"+
		"  \"leftShots\": %d,\n"+
		"  \"leftRate\": %.2f,\n"+
		"  \"leftAim\": %q,\n"+
		"  \"rightShots\": %d,\n"+
		"  \"rightRate\": %.2f,\n"+
		"  \"rightAim\": %q\n"+
		"}\n",
		c.RiseSeconds, c.EagleDelay, c.CrossSeconds, c.EagleStart, c.EagleEnd,
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
	c.RiseSeconds = snap(c.RiseSeconds)
	c.EagleDelay = snap(c.EagleDelay)
	c.CrossSeconds = snap(c.CrossSeconds)
	c.EagleStart = snap(c.EagleStart)
	c.EagleEnd = snap(c.EagleEnd)
	c.LeftRate = snapRate(c.LeftRate)
	c.RightRate = snapRate(c.RightRate)
	if c.RiseSeconds < 0 {
		c.RiseSeconds = 0
	}
	if c.EagleDelay < 0 {
		c.EagleDelay = 0
	}
	if c.CrossSeconds < StepSeconds {
		c.CrossSeconds = StepSeconds
	}
	if c.EagleStart < 0 {
		c.EagleStart = 0
	}
	if c.EagleEnd > 1 {
		c.EagleEnd = 1
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
	case KnobRise:
		c.RiseSeconds = v
	case KnobDelay:
		c.EagleDelay = v
	case KnobCross:
		c.CrossSeconds = v
	case KnobStart:
		c.EagleStart = v
	case KnobEnd:
		c.EagleEnd = v
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
		if lim := c.EagleEnd - StepPoint; v > lim {
			v = lim
		}
	case KnobEnd:
		if v > 1 {
			v = 1
		}
		if lim := c.EagleStart + StepPoint; v < lim {
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
