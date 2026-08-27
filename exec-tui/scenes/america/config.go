package america

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Config is the live knobs on the scene: how long the flag takes to
// fade in from black, when the eagle enters, how long its crossing
// takes — the eagle's speed — where the flight starts and ends as
// fractions of the full off-right-to-off-left span, and the one
// talon shotgun: how many shells the gun fires across one crossing,
// and which of the eight compass points the barrel aims. The
// standalone runner nudges the time knobs 50ms, the path knobs 0.05
// of the span, the shot count one shell, and the aim one compass
// point at a time; Play rebuilds the scene from whatever they hold.
// s writes this JSON next to the scene.
type Config struct {
	FadeSeconds  float64        `json:"fadeSeconds"`
	EagleDelay   float64        `json:"eagleDelay"`
	CrossSeconds float64        `json:"crossSeconds"`
	EagleStart   float64        `json:"eagleStart"`
	EagleEnd     float64        `json:"eagleEnd"`
	Shots        int            `json:"shots"`
	Aim          sprite.Heading `json:"aim"`
}

// fileJSON is the on-disk shape. Every key is a pointer so a file
// missing one keeps that knob at stock. leftShots / leftAim are the
// old two-gun names: a leftover file still loads the leading talon.
type fileJSON struct {
	FadeSeconds  *float64        `json:"fadeSeconds"`
	EagleDelay   *float64        `json:"eagleDelay"`
	CrossSeconds *float64        `json:"crossSeconds"`
	EagleStart   *float64        `json:"eagleStart"`
	EagleEnd     *float64        `json:"eagleEnd"`
	Shots        *int            `json:"shots"`
	Aim          *sprite.Heading `json:"aim"`
	LeftShots    *int            `json:"leftShots"`
	LeftAim      *sprite.Heading `json:"leftAim"`
}

// Knob is which knob the cursor is on.
type Knob int

const (
	KnobFade Knob = iota
	KnobDelay
	KnobCross
	KnobStart
	KnobEnd
	KnobShots
	KnobAim
	KnobCount
)

// KnobLabel is the panel name of knob k.
func KnobLabel(k Knob) string {
	switch k {
	case KnobFade:
		return "flag fade"
	case KnobDelay:
		return "eagle delay"
	case KnobCross:
		return "eagle cross"
	case KnobStart:
		return "eagle start"
	case KnobEnd:
		return "eagle end"
	case KnobShots:
		return "shots"
	case KnobAim:
		return "aim"
	default:
		return ""
	}
}

// KnobUnit is what the panel prints after knob k's value: the time
// knobs are seconds, everything else speaks for itself.
func KnobUnit(k Knob) string {
	switch k {
	case KnobFade, KnobDelay, KnobCross:
		return "s"
	default:
		return ""
	}
}

// headingIdx is h's slot on the compass, or -1 off it.
func headingIdx(h sprite.Heading) int {
	for i, hh := range sprite.Headings {
		if h == hh {
			return i
		}
	}
	return -1
}

// Value is the selected knob's current setting: seconds, a span
// fraction, a shell count, or the aim's slot on the compass.
func (c Config) Value(k Knob) float64 {
	switch k {
	case KnobFade:
		return c.FadeSeconds
	case KnobDelay:
		return c.EagleDelay
	case KnobCross:
		return c.CrossSeconds
	case KnobStart:
		return c.EagleStart
	case KnobEnd:
		return c.EagleEnd
	case KnobShots:
		return float64(c.Shots)
	case KnobAim:
		return float64(headingIdx(c.Aim))
	default:
		return 0
	}
}

// Display is knob k's panel reading, seven columns wide: seconds for
// the time knobs, a bare fraction for the path knobs, a shell count
// for the shots, a compass point for the aim.
func (c Config) Display(k Knob) string {
	switch k {
	case KnobFade, KnobDelay, KnobCross, KnobStart, KnobEnd:
		return fmt.Sprintf("%7.3f%s", c.Value(k), KnobUnit(k))
	case KnobShots:
		return fmt.Sprintf("%7d", c.Shots)
	case KnobAim:
		return fmt.Sprintf("%7s", string(c.Aim))
	default:
		return ""
	}
}

const (
	// StepSeconds is one tick of a time knob: 50ms.
	StepSeconds = 0.050

	// StepPoint is one tick of a path knob: 0.05 of the flight
	// span — the same grid the time knobs walk.
	StepPoint = 0.050

	// DefaultConfigPath is the scene's own JSON, relative to the
	// module root.
	DefaultConfigPath = "scenes/america/config.json"
)

var (
	errFade  = errors.New("america: flag fade must not be negative")
	errDelay = errors.New("america: eagle delay must not be negative")
	errCross = errors.New("america: eagle cross must be at least 50ms")
	errStart = errors.New("america: eagle start must sit inside the span")
	errEnd   = errors.New("america: eagle end must sit inside the span")
	errPath  = errors.New("america: eagle end must be at least one step past its start")
	errShots = errors.New("america: shell count must not be negative")
	errAim   = errors.New("america: aim must sit on the eight-point compass")

	activeMu sync.Mutex
	active   = DefaultConfig()
)

// DefaultConfig is the scene's stock tune: the fast two-second fade,
// the eagle entering the moment the fade lands, the four-second
// crossing, the flight spanning off one wing and off the other, and
// three shells from the one leading-talon gun — the barrel raking
// ahead of the flight.
func DefaultConfig() Config {
	return Config{
		FadeSeconds:  FadeSeconds,
		EagleDelay:   FadeSeconds,
		CrossSeconds: CrossSeconds,
		EagleStart:   StartPoint,
		EagleEnd:     EndPoint,
		Shots:        StockShots,
		Aim:          StockAim,
	}
}

// Active is the timing New copies onto an America scene: the last
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

// Validate reports whether the knobs are playable. An instant flag
// and an eagle at t=0 are allowed; a crossing needs a duration; the
// flight path stays inside the span and keeps at least one step of
// length, flying leftward.
func (c Config) Validate() error {
	if c.FadeSeconds < 0 || math.IsNaN(c.FadeSeconds) || math.IsInf(c.FadeSeconds, 0) {
		return errFade
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
	if c.Shots < 0 {
		return errShots
	}
	if headingIdx(c.Aim) < 0 {
		return errAim
	}
	return nil
}

// Load reads an America-config JSON file.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var f fileJSON
	if err := json.Unmarshal(raw, &f); err != nil {
		return Config{}, fmt.Errorf("america: %s: %w", path, err)
	}
	c := DefaultConfig()
	if f.FadeSeconds != nil {
		c.FadeSeconds = *f.FadeSeconds
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
	if f.Shots != nil {
		c.Shots = *f.Shots
	} else if f.LeftShots != nil {
		c.Shots = *f.LeftShots
	}
	if f.Aim != nil {
		c.Aim = *f.Aim
	} else if f.LeftAim != nil {
		c.Aim = *f.LeftAim
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c.snapped(), nil
}

// LoadOrDefault is Load, except a missing file is stock timing, not
// an error — the same courtesy the landing config gets.
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

// Save writes the knobs as JSON, snapped to their grids so the file
// stays easy to edit by hand.
func (c Config) Save(path string) error {
	if err := c.Validate(); err != nil {
		return err
	}
	c = c.snapped()
	raw := []byte(fmt.Sprintf("{\n"+
		"  \"fadeSeconds\": %.3f,\n"+
		"  \"eagleDelay\": %.3f,\n"+
		"  \"crossSeconds\": %.3f,\n"+
		"  \"eagleStart\": %.3f,\n"+
		"  \"eagleEnd\": %.3f,\n"+
		"  \"shots\": %d,\n"+
		"  \"aim\": %q\n"+
		"}\n",
		c.FadeSeconds, c.EagleDelay, c.CrossSeconds, c.EagleStart, c.EagleEnd,
		c.Shots, string(c.Aim)))
	return os.WriteFile(path, raw, 0o644)
}

func snap(v float64) float64 {
	steps := 1 / StepSeconds
	return math.Round(v*steps) / steps
}

func (c Config) snapped() Config {
	c.FadeSeconds = snap(c.FadeSeconds)
	c.EagleDelay = snap(c.EagleDelay)
	c.CrossSeconds = snap(c.CrossSeconds)
	c.EagleStart = snap(c.EagleStart)
	c.EagleEnd = snap(c.EagleEnd)
	if c.FadeSeconds < 0 {
		c.FadeSeconds = 0
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
	return c
}

func (c *Config) set(k Knob, v float64) {
	switch k {
	case KnobFade:
		c.FadeSeconds = v
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

// step is the grid one tick of knob k walks: 50ms on the time knobs,
// 0.05 of the span on the path knobs.
func step(k Knob) float64 {
	if k == KnobStart || k == KnobEnd {
		return StepPoint
	}
	return StepSeconds
}

// Nudge walks the selected knob by dir steps of its grid. The fade
// and the delay will not go negative; the crossing will not go below
// one step; the path knobs stay inside the span and never catch each
// other; the shell count stops at zero; the aim walks the compass and
// wraps at the ends. A bad cursor is a no-op.
func (c *Config) Nudge(k Knob, dir int) {
	if c == nil || dir == 0 || k < 0 || k >= KnobCount {
		return
	}
	switch k {
	case KnobShots:
		c.Shots = flooredShells(c.Shots + dir)
		return
	case KnobAim:
		c.Aim = walkedAim(c.Aim, dir)
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

// flooredShells is a nudged shell count held at zero — a silent gun
// is allowed, a negative one is not.
func flooredShells(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// walkedAim is h walked dir compass points clockwise, wrapping at the
// ends. An aim off the compass walks from north.
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
