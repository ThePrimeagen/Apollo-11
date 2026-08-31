package landing

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/shootingstar"
)

// Config is the live knobs on the landing: how long the craft takes
// to come down, when the pad dust starts and how long it blows, how
// fast specks leave as the engines cut, the t=0 offsets of each
// booster step (¾, ½, ¼, off), and Star — the scene's own copy of
// the shooting-star knobs its one meteor flies, editable here apart
// from every other scene the star appears in. The standalone runner
// nudges time knobs 50ms at a time, dust loss 0.005/ms, and the star
// knobs by the shooting-star package's own steps; Play rebuilds the
// scene from whatever they hold. s writes this JSON next to the
// scene. 02. Walkthrough plays the same Active config.
type Config struct {
	LandSeconds     float64             `json:"landSeconds"`
	DustStart       float64             `json:"dustStart"`
	DustRun         float64             `json:"dustRun"`
	Fire75          float64             `json:"fire75"`
	Fire50          float64             `json:"fire50"`
	Fire25          float64             `json:"fire25"`
	FireOff         float64             `json:"fireOff"`
	DustLoss        float64             `json:"dustLoss"`
	Code1At         float64             `json:"code1At"`
	Code1Hold       float64             `json:"code1Hold"`
	Code2At         float64             `json:"code2At"`
	Code2Hold       float64             `json:"code2Hold"`
	LandCaptionAt   float64             `json:"landCaptionAt"`
	LandCaptionHold float64             `json:"landCaptionHold"`
	Star            shootingstar.Config `json:"star"`
}

// fileJSON is the on-disk shape. Fire offsets, caption times, and the
// star section are pointers so an older file that only had land/dust
// keeps the stock fire, the 1202 / 1202 / LAND times, and the stock
// small meteor.
type fileJSON struct {
	LandSeconds     float64              `json:"landSeconds"`
	DustStart       float64              `json:"dustStart"`
	DustRun         float64              `json:"dustRun"`
	Fire75          *float64             `json:"fire75"`
	Fire50          *float64             `json:"fire50"`
	Fire25          *float64             `json:"fire25"`
	FireOff         *float64             `json:"fireOff"`
	DustLoss        *float64             `json:"dustLoss"`
	Code1At         *float64             `json:"code1At"`
	Code1Hold       *float64             `json:"code1Hold"`
	Code2At         *float64             `json:"code2At"`
	Code2Hold       *float64             `json:"code2Hold"`
	LandCaptionAt   *float64             `json:"landCaptionAt"`
	LandCaptionHold *float64             `json:"landCaptionHold"`
	Star            *shootingstar.Config `json:"star"`
}

// Knob is which timing the cursor is on.
type Knob int

const (
	KnobLand Knob = iota
	KnobDustStart
	KnobDustRun
	KnobDustLoss
	KnobFire75
	KnobFire50
	KnobFire25
	KnobFireOff
	KnobCode1At
	KnobCode1Hold
	KnobCode2At
	KnobCode2Hold
	KnobLandCaptionAt
	KnobLandCaptionHold
	KnobStarSize
	KnobStarRandomSize
	KnobStarSpeed
	KnobStarCount
	KnobStarPeriod
	KnobStarMinLife
	KnobStarMaxLife
	KnobStarNozzle
	KnobStarPeak
	KnobStarTaper
	KnobStarDelay
	KnobStarStartY
	KnobCount
)

// star maps a panel knob onto the shooting-star package's own. The
// path knob stays off the panel: the landing meteor flies one fixed
// diagonal, so the tuner-only flight selector would be an inert knob
// here.
func (k Knob) star() (shootingstar.Knob, bool) {
	if k < KnobStarSize || k > KnobStarStartY {
		return 0, false
	}
	return shootingstar.KnobSize + shootingstar.Knob(k-KnobStarSize), true
}

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobLand:
		return "land"
	case KnobDustStart:
		return "dust start"
	case KnobDustRun:
		return "dust run"
	case KnobDustLoss:
		return "dust loss"
	case KnobFire75:
		return "fire 3/4"
	case KnobFire50:
		return "fire 1/2"
	case KnobFire25:
		return "fire 1/4"
	case KnobFireOff:
		return "fire off"
	case KnobCode1At:
		return "1202 a"
	case KnobCode1Hold:
		return "hold a"
	case KnobCode2At:
		return "1202 b"
	case KnobCode2Hold:
		return "hold b"
	case KnobLandCaptionAt:
		return "LAND at"
	case KnobLandCaptionHold:
		return "LAND hold"
	default:
		if sk, ok := k.star(); ok {
			return "star " + shootingstar.KnobLabel(sk)
		}
		return ""
	}
}

// Value is the selected knob's current seconds.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobLand:
		return c.LandSeconds
	case KnobDustStart:
		return c.DustStart
	case KnobDustRun:
		return c.DustRun
	case KnobDustLoss:
		return c.DustLoss
	case KnobFire75:
		return c.Fire75
	case KnobFire50:
		return c.Fire50
	case KnobFire25:
		return c.Fire25
	case KnobFireOff:
		return c.FireOff
	case KnobCode1At:
		return c.Code1At
	case KnobCode1Hold:
		return c.Code1Hold
	case KnobCode2At:
		return c.Code2At
	case KnobCode2Hold:
		return c.Code2Hold
	case KnobLandCaptionAt:
		return c.LandCaptionAt
	case KnobLandCaptionHold:
		return c.LandCaptionHold
	default:
		if sk, ok := k.star(); ok {
			return c.Star.Value(sk)
		}
		return 0
	}
}

const (
	// StepSeconds is one tick of a time knob: 50ms.
	StepSeconds = 0.050

	// StepLoss is one tick of the dust-loss knob: 0.005 particles
	// per millisecond (5 specks a second).
	StepLoss = 0.005

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/landing/config.json"
)

var (
	errLand      = errors.New("landing: land duration must be at least 50ms")
	errDustStart = errors.New("landing: dust start must not be negative")
	errDustRun   = errors.New("landing: dust run must not be negative")
	errFire      = errors.New("landing: fire stage offsets must not be negative")
	errDustLoss  = errors.New("landing: particle loss must not be negative")
	errCaption   = errors.New("landing: caption offsets and holds must not be negative")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the portable landing's stock timing over the
// stock small meteor.
func DefaultConfig() Config {
	return Config{
		LandSeconds:     LandSeconds,
		DustStart:       DustStart,
		DustRun:         DustRun,
		Fire75:          Fire75,
		Fire50:          Fire50,
		Fire25:          Fire25,
		FireOff:         FireOff,
		DustLoss:        DustLoss,
		Code1At:         Code1At,
		Code1Hold:       Code1Hold,
		Code2At:         Code2At,
		Code2Hold:       Code2Hold,
		LandCaptionAt:   LandCaptionAt,
		LandCaptionHold: LandCaptionHold,
		Star:            shootingstar.MeteorConfig(),
	}
}

// Active is the timing New copies onto a landing scene: the last
// successful Use, or stock after Reset.
func Active() Config {
	activeMu.Lock()
	defer activeMu.Unlock()
	return active
}

// Use makes cfg the timing New and 02. Walkthrough play. A bad cfg
// is rejected and Active is unchanged.
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
	if c.LandSeconds < StepSeconds || math.IsNaN(c.LandSeconds) || math.IsInf(c.LandSeconds, 0) {
		return errLand
	}
	if c.DustStart < 0 || math.IsNaN(c.DustStart) || math.IsInf(c.DustStart, 0) {
		return errDustStart
	}
	if c.DustRun < 0 || math.IsNaN(c.DustRun) || math.IsInf(c.DustRun, 0) {
		return errDustRun
	}
	for _, v := range []float64{c.Fire75, c.Fire50, c.Fire25, c.FireOff} {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return errFire
		}
	}
	if c.DustLoss < 0 || math.IsNaN(c.DustLoss) || math.IsInf(c.DustLoss, 0) {
		return errDustLoss
	}
	for _, v := range []float64{c.Code1At, c.Code1Hold, c.Code2At, c.Code2Hold, c.LandCaptionAt, c.LandCaptionHold} {
		if v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return errCaption
		}
	}
	return c.Star.Validate()
}

// Load reads a landing-config JSON file. A file without a star
// section keeps the stock small meteor; keys inside a star section
// merge over it.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	star := shootingstar.MeteorConfig()
	f := fileJSON{Star: &star}
	if err := json.Unmarshal(raw, &f); err != nil {
		return Config{}, fmt.Errorf("landing: %s: %w", path, err)
	}
	c := Config{
		LandSeconds:     f.LandSeconds,
		DustStart:       f.DustStart,
		DustRun:         f.DustRun,
		Fire75:          Fire75,
		Fire50:          Fire50,
		Fire25:          Fire25,
		FireOff:         FireOff,
		DustLoss:        DustLoss,
		Code1At:         Code1At,
		Code1Hold:       Code1Hold,
		Code2At:         Code2At,
		Code2Hold:       Code2Hold,
		LandCaptionAt:   LandCaptionAt,
		LandCaptionHold: LandCaptionHold,
		Star:            shootingstar.MeteorConfig(),
	}
	if f.Star != nil {
		c.Star = *f.Star
	}
	if f.Fire75 != nil {
		c.Fire75 = *f.Fire75
	}
	if f.Fire50 != nil {
		c.Fire50 = *f.Fire50
	}
	if f.Fire25 != nil {
		c.Fire25 = *f.Fire25
	}
	if f.FireOff != nil {
		c.FireOff = *f.FireOff
	}
	if f.DustLoss != nil {
		c.DustLoss = *f.DustLoss
	}
	if f.Code1At != nil {
		c.Code1At = *f.Code1At
	}
	if f.Code1Hold != nil {
		c.Code1Hold = *f.Code1Hold
	}
	if f.Code2At != nil {
		c.Code2At = *f.Code2At
	}
	if f.Code2Hold != nil {
		c.Code2Hold = *f.Code2Hold
	}
	if f.LandCaptionAt != nil {
		c.LandCaptionAt = *f.LandCaptionAt
	}
	if f.LandCaptionHold != nil {
		c.LandCaptionHold = *f.LandCaptionHold
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c.snapped(), nil
}

// LoadOrDefault is Load, except a missing file is stock timing, not
// an error — the same courtesy the dust puff file gets.
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
		"  \"landSeconds\": %.3f,\n"+
		"  \"dustStart\": %.3f,\n"+
		"  \"dustRun\": %.3f,\n"+
		"  \"fire75\": %.3f,\n"+
		"  \"fire50\": %.3f,\n"+
		"  \"fire25\": %.3f,\n"+
		"  \"fireOff\": %.3f,\n"+
		"  \"dustLoss\": %.3f,\n"+
		"  \"code1At\": %.3f,\n"+
		"  \"code1Hold\": %.3f,\n"+
		"  \"code2At\": %.3f,\n"+
		"  \"code2Hold\": %.3f,\n"+
		"  \"landCaptionAt\": %.3f,\n"+
		"  \"landCaptionHold\": %.3f,\n"+
		"  \"star\": {\n"+
		"    \"path\": %q,\n"+
		"    \"size\": %d,\n"+
		"    \"randomSize\": %t,\n"+
		"    \"speed\": %.1f,\n"+
		"    \"count\": %d,\n"+
		"    \"period\": %.3f,\n"+
		"    \"minLife\": %.2f,\n"+
		"    \"maxLife\": %.2f,\n"+
		"    \"nozzle\": %.2f,\n"+
		"    \"peak\": %.1f,\n"+
		"    \"taper\": %.2f,\n"+
		"    \"delay\": %.3f,\n"+
		"    \"startY\": %.2f\n"+
		"  }\n"+
		"}\n",
		c.LandSeconds, c.DustStart, c.DustRun,
		c.Fire75, c.Fire50, c.Fire25, c.FireOff, c.DustLoss,
		c.Code1At, c.Code1Hold, c.Code2At, c.Code2Hold,
		c.LandCaptionAt, c.LandCaptionHold,
		c.Star.Path, c.Star.Size, c.Star.RandomSize, c.Star.Speed, c.Star.Count,
		c.Star.Period, c.Star.MinLife, c.Star.MaxLife, c.Star.Nozzle, c.Star.Peak, c.Star.Taper,
		c.Star.Delay, c.Star.StartY))
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
	c.LandSeconds = snap(c.LandSeconds)
	c.DustStart = snap(c.DustStart)
	c.DustRun = snap(c.DustRun)
	c.Fire75 = snap(c.Fire75)
	c.Fire50 = snap(c.Fire50)
	c.Fire25 = snap(c.Fire25)
	c.FireOff = snap(c.FireOff)
	c.DustLoss = snapLoss(c.DustLoss)
	c.Code1At = snap(c.Code1At)
	c.Code1Hold = snap(c.Code1Hold)
	c.Code2At = snap(c.Code2At)
	c.Code2Hold = snap(c.Code2Hold)
	c.LandCaptionAt = snap(c.LandCaptionAt)
	c.LandCaptionHold = snap(c.LandCaptionHold)
	if c.LandSeconds < StepSeconds {
		c.LandSeconds = StepSeconds
	}
	if c.DustStart < 0 {
		c.DustStart = 0
	}
	if c.DustRun < 0 {
		c.DustRun = 0
	}
	if c.Fire75 < 0 {
		c.Fire75 = 0
	}
	if c.Fire50 < 0 {
		c.Fire50 = 0
	}
	if c.Fire25 < 0 {
		c.Fire25 = 0
	}
	if c.FireOff < 0 {
		c.FireOff = 0
	}
	if c.DustLoss < 0 {
		c.DustLoss = 0
	}
	if c.Code1At < 0 {
		c.Code1At = 0
	}
	if c.Code1Hold < 0 {
		c.Code1Hold = 0
	}
	if c.Code2At < 0 {
		c.Code2At = 0
	}
	if c.Code2Hold < 0 {
		c.Code2Hold = 0
	}
	if c.LandCaptionAt < 0 {
		c.LandCaptionAt = 0
	}
	if c.LandCaptionHold < 0 {
		c.LandCaptionHold = 0
	}
	return c
}

func (c *Config) set(k Knob, v float64) {
	switch k {
	case KnobLand:
		c.LandSeconds = v
	case KnobDustStart:
		c.DustStart = v
	case KnobDustRun:
		c.DustRun = v
	case KnobDustLoss:
		c.DustLoss = v
	case KnobFire75:
		c.Fire75 = v
	case KnobFire50:
		c.Fire50 = v
	case KnobFire25:
		c.Fire25 = v
	case KnobFireOff:
		c.FireOff = v
	case KnobCode1At:
		c.Code1At = v
	case KnobCode1Hold:
		c.Code1Hold = v
	case KnobCode2At:
		c.Code2At = v
	case KnobCode2Hold:
		c.Code2Hold = v
	case KnobLandCaptionAt:
		c.LandCaptionAt = v
	case KnobLandCaptionHold:
		c.LandCaptionHold = v
	}
}

// Nudge walks the selected knob by dir steps. Time knobs move 50ms;
// dust loss moves 0.005/ms; star knobs walk the shooting-star
// package's own steps. Land will not go below one time step; every
// other time knob will not go negative. A bad cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	if sk, ok := k.star(); ok {
		c.Star.Nudge(sk, dir)
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
	if k == KnobLand {
		if v < StepSeconds {
			v = StepSeconds
		}
	} else if v < 0 {
		v = 0
	}
	c.set(k, v)
}
