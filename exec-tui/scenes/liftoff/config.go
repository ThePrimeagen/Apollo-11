package liftoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
)

// Config is the live knobs on the liftoff: how long the climb takes
// and when it starts, the four ignition offsets from t=0 (¼, ½, ¾,
// full — the landing throttle run backwards), and the pad dust
// window. The standalone runner nudges time knobs 50ms at a time and
// dust loss 0.005/ms; Play rebuilds the scene from whatever they
// hold. s writes this JSON next to the scene. 03. Inverse Walkthrough
// plays the same Active config.
type Config struct {
	RiseSeconds float64 `json:"riseSeconds"`
	LiftAt      float64 `json:"liftAt"`
	Fire25      float64 `json:"fire25"`
	Fire50      float64 `json:"fire50"`
	Fire75      float64 `json:"fire75"`
	FireFull    float64 `json:"fireFull"`
	DustStart   float64 `json:"dustStart"`
	DustRun     float64 `json:"dustRun"`
	DustLoss    float64 `json:"dustLoss"`
}

// Knob is which timing the cursor is on.
type Knob int

const (
	KnobRise Knob = iota
	KnobLiftAt
	KnobFire25
	KnobFire50
	KnobFire75
	KnobFireFull
	KnobDustStart
	KnobDustRun
	KnobDustLoss
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobRise:
		return "rise"
	case KnobLiftAt:
		return "lift at"
	case KnobFire25:
		return "fire 1/4"
	case KnobFire50:
		return "fire 1/2"
	case KnobFire75:
		return "fire 3/4"
	case KnobFireFull:
		return "fire full"
	case KnobDustStart:
		return "dust start"
	case KnobDustRun:
		return "dust run"
	case KnobDustLoss:
		return "dust loss"
	default:
		return ""
	}
}

// Value is the selected knob's current seconds (loss: particles/ms).
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobRise:
		return c.RiseSeconds
	case KnobLiftAt:
		return c.LiftAt
	case KnobFire25:
		return c.Fire25
	case KnobFire50:
		return c.Fire50
	case KnobFire75:
		return c.Fire75
	case KnobFireFull:
		return c.FireFull
	case KnobDustStart:
		return c.DustStart
	case KnobDustRun:
		return c.DustRun
	case KnobDustLoss:
		return c.DustLoss
	default:
		return 0
	}
}

// The stock knobs: the landing's stock timing played backwards. The
// booster steps ¼, ½, ¾ every 0.4s from 0.4s in, reaches full power
// at 1.6s, and the craft leaves the pad that same moment. Dust kicks
// with the first ignition and blows through the early climb. After
// the cut the tail fire holds three seconds, then goes out for good.
const (
	// RiseSeconds is how long the climb from the pad to fully off
	// the top takes. Smaller is a faster liftoff.
	RiseSeconds = 5.0

	// LiftAt is when the craft leaves the pad, from t=0.
	LiftAt = 1.6

	// Fire25/Fire50/Fire75/FireFull are the ignition offsets from
	// t=0: cold, then ¼, ½, ¾, then full power.
	Fire25   = 0.4
	Fire50   = 0.8
	Fire75   = 1.2
	FireFull = 1.6

	// DustStart is when the pad cloud kicks, measured from t=0.
	// Stock is the first ignition.
	DustStart = 0.4

	// DustRun is how long the pad cloud keeps emitting: the
	// ignition ramp plus a two-second linger into the climb.
	DustRun = 3.2

	// DustLoss is how many pad specks leave per millisecond once
	// the run ends.
	DustLoss = 0.05

	// StepSeconds is one tick of a time knob: 50ms.
	StepSeconds = 0.050

	// StepLoss is one tick of the dust-loss knob: 0.005 particles
	// per millisecond.
	StepLoss = 0.005

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/liftoff/config.json"
)

var (
	errRise = errors.New("liftoff: rise duration must be at least 50ms")
	errKnob = errors.New("liftoff: every timing knob must be a non-negative number")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the inverse walkthrough's stock timing.
func DefaultConfig() Config {
	return Config{
		RiseSeconds: RiseSeconds,
		LiftAt:      LiftAt,
		Fire25:      Fire25,
		Fire50:      Fire50,
		Fire75:      Fire75,
		FireFull:    FireFull,
		DustStart:   DustStart,
		DustRun:     DustRun,
		DustLoss:    DustLoss,
	}
}

// Active is the timing New copies onto a liftoff scene: the last
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

// Validate reports whether the knobs are playable.
func (c Config) Validate() error {
	if c.RiseSeconds < StepSeconds || math.IsNaN(c.RiseSeconds) || math.IsInf(c.RiseSeconds, 0) {
		return errRise
	}
	for _, v := range []float64{
		c.LiftAt, c.Fire25, c.Fire50, c.Fire75, c.FireFull,
		c.DustStart, c.DustRun, c.DustLoss,
	} {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return errKnob
		}
	}
	return nil
}

// Load reads a liftoff-config JSON file. Keys the file does not carry
// keep their stock values.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := DefaultConfig()
	if err := json.Unmarshal(raw, &c); err != nil {
		return Config{}, fmt.Errorf("liftoff: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c.snapped(), nil
}

// LoadOrDefault is Load, except a missing file is stock timing, not
// an error — the same courtesy every scene config gets.
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
		"  \"riseSeconds\": %.3f,\n"+
		"  \"liftAt\": %.3f,\n"+
		"  \"fire25\": %.3f,\n"+
		"  \"fire50\": %.3f,\n"+
		"  \"fire75\": %.3f,\n"+
		"  \"fireFull\": %.3f,\n"+
		"  \"dustStart\": %.3f,\n"+
		"  \"dustRun\": %.3f,\n"+
		"  \"dustLoss\": %.3f\n"+
		"}\n",
		c.RiseSeconds, c.LiftAt,
		c.Fire25, c.Fire50, c.Fire75, c.FireFull,
		c.DustStart, c.DustRun, c.DustLoss))
	return os.WriteFile(path, raw, 0o644)
}

func snap(v float64) float64 {
	steps := 1 / StepSeconds
	return math.Round(v*steps) / steps
}

func snapLoss(v float64) float64 {
	steps := 1 / StepLoss
	return math.Round(v*steps) / steps
}

func (c Config) snapped() Config {
	c.RiseSeconds = math.Max(snap(c.RiseSeconds), StepSeconds)
	c.LiftAt = math.Max(snap(c.LiftAt), 0)
	c.Fire25 = math.Max(snap(c.Fire25), 0)
	c.Fire50 = math.Max(snap(c.Fire50), 0)
	c.Fire75 = math.Max(snap(c.Fire75), 0)
	c.FireFull = math.Max(snap(c.FireFull), 0)
	c.DustStart = math.Max(snap(c.DustStart), 0)
	c.DustRun = math.Max(snap(c.DustRun), 0)
	c.DustLoss = math.Max(snapLoss(c.DustLoss), 0)
	return c
}

func (c *Config) set(k Knob, v float64) {
	switch k {
	case KnobRise:
		c.RiseSeconds = v
	case KnobLiftAt:
		c.LiftAt = v
	case KnobFire25:
		c.Fire25 = v
	case KnobFire50:
		c.Fire50 = v
	case KnobFire75:
		c.Fire75 = v
	case KnobFireFull:
		c.FireFull = v
	case KnobDustStart:
		c.DustStart = v
	case KnobDustRun:
		c.DustRun = v
	case KnobDustLoss:
		c.DustLoss = v
	}
}

// Nudge walks the selected knob by dir steps. Time knobs move 50ms;
// dust loss moves 0.005/ms. Rise will not go below one time step;
// every other knob will not go negative. A bad cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	if k == KnobDustLoss {
		v := snapLoss(c.DustLoss + StepLoss*float64(dir))
		if v < 0 {
			v = 0
		}
		c.DustLoss = v
		return
	}
	v := snap(c.Value(k) + StepSeconds*float64(dir))
	if k == KnobRise {
		if v < StepSeconds {
			v = StepSeconds
		}
	} else if v < 0 {
		v = 0
	}
	c.set(k, v)
}
