// Package gunfire is the one-shot Doom muzzle flame on an eight-point
// compass: the red flame that comes out when the shotgun goes off,
// tunable direction by direction. One squeeze and the flame leaps
// from the muzzle along every heading at once — a white-hot heart
// wrapped in tongues that cool through that direction's own five-stop
// color ramp as they rise, slow, and die. Doom's flash is two sprite
// frames, so a dimmer second pulse follows the first on a short fuse.
// There is no period clock anywhere: the blast holds fire until Fire,
// burns out, and leaves the stage exactly as it found it.
//
// Every number is data. BlastConfig carries the muzzle, the heading
// the shared core aims (N, NE, E, SE, S, SW, W, NW — the same compass
// the lander atlas speaks), the two-frame pulse, the core brightness
// ladder, the shared white-hot core, and one Shot per direction — the
// full engine-knob Layer plus its color ramp. The knobs live in this
// component's config file and become the active blast via UseBlast, so
// the tuner and the demo stay on the same values. The tuner plays all
// eight headings at once, the way the flame config does.
package gunfire

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

var (
	ErrMuzzle  = errors.New("gunfire: muzzle must sit on the stage (fractions 0..1)")
	ErrHeading = errors.New("gunfire: heading must be one of the eight compass points")
	ErrDelay   = errors.New("gunfire: pulse delay must be non-negative")
	ErrPulse   = errors.New("gunfire: pulse fraction must be 0..1")
	ErrLadder  = errors.New("gunfire: core ladder must climb: 1 <= edgeAt < midAt < coreAt")
	ErrColor   = errors.New("gunfire: color stops must sit on the xterm cube 1..255")
)

// Layer is the engine-knob bundle one part of the blast burns under:
// how many specks a squeeze throws, how long and how fast they fly,
// how wide the fan is, how thick the muzzle is, how far they may get
// (0 is unleashed), how hard they rise, and how fast they die down.
type Layer struct {
	Count       int     `json:"count"`
	MinLife     float64 `json:"minLife"`
	MaxLife     float64 `json:"maxLife"`
	MinSpeed    float64 `json:"minSpeed"`
	MaxSpeed    float64 `json:"maxSpeed"`
	Spread      float64 `json:"spread"`
	Nozzle      float64 `json:"nozzle"`
	MaxDistance float64 `json:"maxDistance"`
	Lift        float64 `json:"lift"`
	Drag        float64 `json:"drag"`
}

// Shot is one compass direction's flame: the engine knobs plus the
// five color stops its tongues cool through — Colors[0] wears the
// freshest specks, Colors[4] the dying embers.
type Shot struct {
	Layer
	Colors [5]int `json:"colors"`
}

// BlastConfig is the JSON that tunes the shot: where the muzzle sits
// (fractions of the stage), the compass heading the shared core aims,
// Doom's two-frame pulse (the dimmer re-burst that follows the first
// on a short fuse), the concentration ladder that decides how bright
// a core cell burns, the shared core, and the eight shots. Fire
// bursts every shot at once.
type BlastConfig struct {
	MuzzleX    float64        `json:"muzzleX"`    // muzzle, as a fraction of stage width
	MuzzleY    float64        `json:"muzzleY"`    // muzzle, as a fraction of stage height
	Heading    sprite.Heading `json:"heading"`    // the compass point the shared core aims
	PulseDelay float64        `json:"pulseDelay"` // seconds between Doom's two flash frames; 0 = one frame
	PulseFrac  float64        `json:"pulseFrac"`  // second frame size, as a fraction of the first; 0 = one frame
	EdgeAt     int            `json:"edgeAt"`     // core concentration that earns the star
	MidAt      int            `json:"midAt"`      // … the yellow shade block
	CoreAt     int            `json:"coreAt"`     // … the white-hot core block

	Core Layer `json:"core"` // the shared white-hot pop at the muzzle

	N  Shot `json:"n"`
	NE Shot `json:"ne"`
	E  Shot `json:"e"`
	SE Shot `json:"se"`
	S  Shot `json:"s"`
	SW Shot `json:"sw"`
	W  Shot `json:"w"`
	NW Shot `json:"nw"`
}

// DoomRamp is the stock cooling ramp every direction ships with:
// bright yellow at birth, through orange and red, down to a maroon
// ember.
var DoomRamp = [5]int{226, 208, 196, 160, 124}

// DefaultBlast is the stock Doom shotgun flame: the muzzle at center
// screen so every heading has room to leap, with every direction
// carrying the same tuned flame and the Doom red ramp, ready to be
// pulled apart shot by shot. One squeeze fires the whole rose.
func DefaultBlast() BlastConfig {
	shot := Shot{
		Layer:  Layer{Count: 140, MinLife: 0.14, MaxLife: 0.62, MinSpeed: 16, MaxSpeed: 36, Spread: 0.24, Nozzle: 2.4, Lift: 46, Drag: 3},
		Colors: DoomRamp,
	}
	return BlastConfig{
		MuzzleX:    0.5,
		MuzzleY:    0.5,
		Heading:    sprite.N,
		PulseDelay: 0.11,
		PulseFrac:  0.6,
		EdgeAt:     2,
		MidAt:      4,
		CoreAt:     7,
		Core:       Layer{Count: 80, MinLife: 0.04, MaxLife: 0.12, MinSpeed: 8, MaxSpeed: 20, Spread: 0.55, Nozzle: 3, MaxDistance: 4},
		N:          shot,
		NE:         shot,
		E:          shot,
		SE:         shot,
		S:          shot,
		SW:         shot,
		W:          shot,
		NW:         shot,
	}
}

// ShotAt is the shot one compass point fires. Headings off the
// compass hand back the zero shot.
func (c BlastConfig) ShotAt(h sprite.Heading) Shot {
	switch h {
	case sprite.N:
		return c.N
	case sprite.NE:
		return c.NE
	case sprite.E:
		return c.E
	case sprite.SE:
		return c.SE
	case sprite.S:
		return c.S
	case sprite.SW:
		return c.SW
	case sprite.W:
		return c.W
	case sprite.NW:
		return c.NW
	}
	return Shot{}
}

// SetShot retunes one compass point's shot, leaving the other seven
// alone. Headings off the compass set nothing.
func (c *BlastConfig) SetShot(h sprite.Heading, s Shot) {
	switch h {
	case sprite.N:
		c.N = s
	case sprite.NE:
		c.NE = s
	case sprite.E:
		c.E = s
	case sprite.SE:
		c.SE = s
	case sprite.S:
		c.S = s
	case sprite.SW:
		c.SW = s
	case sprite.W:
		c.W = s
	case sprite.NW:
		c.NW = s
	}
}

var activeBlast = DefaultBlast()

// UseBlast makes ActiveBlast — and every blast reading it — burn
// these settings. Invalid settings are rejected and the active blast
// holds.
func UseBlast(c BlastConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	activeBlast = c
	return nil
}

// ResetBlast restores the stock shotgun flame.
func ResetBlast() {
	_ = UseBlast(DefaultBlast())
}

// ActiveBlast is the blast settings now in effect.
func ActiveBlast() BlastConfig { return activeBlast }

// FindConfig locates the shipped components/gunfire/config.json — the
// same file the tuner saves and the shotgun fires.
func FindConfig() string {
	const file = "config.json"
	if _, src, _, ok := runtime.Caller(0); ok {
		cand := filepath.Join(filepath.Dir(src), file)
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	seen := map[string]bool{}
	var cands []string
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		cands = append(cands, p)
	}
	addFrom := func(start string) {
		dir := start
		for i := 0; i < 8; i++ {
			add(filepath.Join(dir, file))
			add(filepath.Join(dir, "gunfire", file))
			add(filepath.Join(dir, "components", "gunfire", file))
			parent := filepath.Dir(dir)
			if parent == dir {
				return
			}
			dir = parent
		}
	}
	if wd, err := os.Getwd(); err == nil {
		addFrom(wd)
	}
	for _, p := range cands {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join("components", "gunfire", file)
}

// nominal box for validating engine knobs without a real stage.
const (
	nominalW = 120
	nominalH = 60
)

// validHeading reports h sits on the compass.
func validHeading(h sprite.Heading) bool {
	for _, hh := range sprite.Headings {
		if h == hh {
			return true
		}
	}
	return false
}

// Validate reports the first thing wrong with c.
func (c BlastConfig) Validate() error {
	if c.MuzzleX < 0 || c.MuzzleX > 1 || c.MuzzleY < 0 || c.MuzzleY > 1 {
		return ErrMuzzle
	}
	if !validHeading(c.Heading) {
		return ErrHeading
	}
	if c.PulseDelay < 0 {
		return ErrDelay
	}
	if c.PulseFrac < 0 || c.PulseFrac > 1 {
		return ErrPulse
	}
	if c.EdgeAt < 1 || c.MidAt <= c.EdgeAt || c.CoreAt <= c.MidAt {
		return ErrLadder
	}
	core, flames := c.Engines(nominalW, nominalH)
	if err := core.Validate(); err != nil {
		return fmt.Errorf("gunfire: core layer: %w", err)
	}
	for i, h := range sprite.Headings {
		if err := flames[i].Validate(); err != nil {
			return fmt.Errorf("gunfire: %s shot: %w", h, err)
		}
		for _, stop := range c.ShotAt(h).Colors {
			if stop < 1 || stop > 255 {
				return fmt.Errorf("gunfire: %s shot: %w", h, ErrColor)
			}
		}
	}
	return nil
}

// Engines are the nine particle worlds this blast describes on a w×h
// unit stage: the shared white-hot core aimed along the active
// heading, and one flame per compass point — each leaving the same
// muzzle its own way, in straight flight, under its own lift and
// drag. Every world has Period 0 — a gunshot is a trigger, not a
// clock. The muzzle clamps inside stages smaller than its fractions
// reach. The flames come back in sprite.Headings order.
func (c BlastConfig) Engines(w, h float64) (core particle.Config, flames [8]particle.Config) {
	muzzle := particle.Vec2{
		X: clamp(c.MuzzleX*w, 0, w),
		Y: clamp(c.MuzzleY*h, 0, h),
	}
	base := func(l Layer, dir particle.Vec2) particle.Config {
		return particle.Config{
			Width:       w,
			Height:      h,
			Origin:      muzzle,
			Direction:   dir,
			Count:       l.Count,
			MinLife:     l.MinLife,
			MaxLife:     l.MaxLife,
			MinSpeed:    l.MinSpeed,
			MaxSpeed:    l.MaxSpeed,
			Spread:      l.Spread,
			Nozzle:      l.Nozzle,
			MaxDistance: l.MaxDistance,
			Lift:        l.Lift,
			Drag:        l.Drag,
		}
	}
	core = base(c.Core, dirOf(c.Heading))
	for i, heading := range sprite.Headings {
		flames[i] = base(c.ShotAt(heading).Layer, dirOf(heading))
	}
	return core, flames
}

// dirOf is the unit heading of one compass point on the screen: north
// is up (Y grows downward), the diagonals are unit-length.
func dirOf(h sprite.Heading) particle.Vec2 {
	d := math.Sqrt2 / 2
	switch h {
	case sprite.N:
		return particle.Vec2{X: 0, Y: -1}
	case sprite.NE:
		return particle.Vec2{X: d, Y: -d}
	case sprite.E:
		return particle.Vec2{X: 1, Y: 0}
	case sprite.SE:
		return particle.Vec2{X: d, Y: d}
	case sprite.S:
		return particle.Vec2{X: 0, Y: 1}
	case sprite.SW:
		return particle.Vec2{X: -d, Y: d}
	case sprite.W:
		return particle.Vec2{X: -1, Y: 0}
	case sprite.NW:
		return particle.Vec2{X: -d, Y: -d}
	}
	return particle.Vec2{}
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

// LoadBlast reads a blast-config JSON file.
func LoadBlast(path string) (BlastConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return BlastConfig{}, err
	}
	var c BlastConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return BlastConfig{}, fmt.Errorf("gunfire: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return BlastConfig{}, err
	}
	return c, nil
}

// Save writes the blast settings as JSON.
func (c BlastConfig) Save(path string) error {
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
