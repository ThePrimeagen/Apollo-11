// Package gunfire is the one-shot Doom muzzle flame: the red flame
// that comes out when the shotgun goes off. One squeeze and the flame
// leaps up from the muzzle — a white-hot heart wrapped in tongues
// that cool bright yellow through orange and red down to a maroon
// ember as they rise, slow, and die. Doom's flash is two sprite
// frames, so a dimmer second pulse follows the first on a short
// fuse. There is no period clock anywhere: the blast holds fire until
// Fire, burns out, and leaves the stage exactly as it found it.
//
// Every number is data. BlastConfig carries the aim (angle and muzzle
// position — straight up by default, the way the flash sits in the
// first-person view), the two-frame pulse, the core brightness
// ladder, and one engine-knob Layer each for the core and the flame —
// the flame carrying lift (hot gas rises) and drag (the eruption dies
// down). The knobs live in this component's config file and become
// the active blast via UseBlast, so the tuner and the demo stay on
// the same values.
package gunfire

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

var (
	ErrMuzzle = errors.New("gunfire: muzzle must sit on the stage (fractions 0..1)")
	ErrAngle  = errors.New("gunfire: angle must be -180..180 degrees around level")
	ErrDelay  = errors.New("gunfire: pulse delay must be non-negative")
	ErrPulse  = errors.New("gunfire: pulse fraction must be 0..1")
	ErrLadder = errors.New("gunfire: core ladder must climb: 1 <= edgeAt < midAt < coreAt")
)

// Layer is the engine-knob bundle one part of the flame burns under:
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

// BlastConfig is the JSON that tunes the flame: where the muzzle sits
// (fractions of the stage), where it aims (degrees above level; 90 is
// straight up), Doom's two-frame pulse (the dimmer re-burst that
// follows the first on a short fuse), the concentration ladder that
// decides how bright a core cell burns, and the two layers.
type BlastConfig struct {
	AngleDeg   float64 `json:"angleDeg"`   // aim, any which direction: 0 right, 90 up, ±180 left, -90 down
	MuzzleX    float64 `json:"muzzleX"`    // muzzle, as a fraction of stage width
	MuzzleY    float64 `json:"muzzleY"`    // muzzle, as a fraction of stage height
	PulseDelay float64 `json:"pulseDelay"` // seconds between Doom's two flash frames; 0 = one frame
	PulseFrac  float64 `json:"pulseFrac"`  // second frame size, as a fraction of the first; 0 = one frame
	EdgeAt     int     `json:"edgeAt"`     // core concentration that earns the star
	MidAt      int     `json:"midAt"`      // … the yellow shade block
	CoreAt     int     `json:"coreAt"`     // … the white-hot core block

	Core  Layer `json:"core"`
	Flame Layer `json:"flame"`
}

// DefaultBlast is the stock Doom shotgun flame: the muzzle low at
// center screen like the gun in first person, the flame leaping
// straight up — fast out of the barrel, dragged to a stall as lift
// carries the cooling tongues — and a 0.11s fuse to the dimmer
// second frame, the way the flash sprite plays twice.
func DefaultBlast() BlastConfig {
	return BlastConfig{
		AngleDeg:   90,
		MuzzleX:    0.5,
		MuzzleY:    0.85,
		PulseDelay: 0.11,
		PulseFrac:  0.6,
		EdgeAt:     2,
		MidAt:      4,
		CoreAt:     7,
		Core:  Layer{Count: 80, MinLife: 0.04, MaxLife: 0.12, MinSpeed: 8, MaxSpeed: 20, Spread: 0.55, Nozzle: 3, MaxDistance: 4},
		Flame: Layer{Count: 140, MinLife: 0.14, MaxLife: 0.62, MinSpeed: 16, MaxSpeed: 36, Spread: 0.24, Nozzle: 2.4, Lift: 46, Drag: 3},
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

// ResetBlast restores the stock flame.
func ResetBlast() {
	_ = UseBlast(DefaultBlast())
}

// ActiveBlast is the blast settings now in effect.
func ActiveBlast() BlastConfig { return activeBlast }

// nominal box for validating engine knobs without a real stage.
const (
	nominalW = 120
	nominalH = 60
)

// Validate reports the first thing wrong with c.
func (c BlastConfig) Validate() error {
	if c.MuzzleX < 0 || c.MuzzleX > 1 || c.MuzzleY < 0 || c.MuzzleY > 1 {
		return ErrMuzzle
	}
	if c.AngleDeg < -180 || c.AngleDeg > 180 {
		return ErrAngle
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
	core, flame := c.Engines(nominalW, nominalH)
	for _, layer := range []struct {
		name string
		cfg  particle.Config
	}{
		{"core", core}, {"flame", flame},
	} {
		if err := layer.cfg.Validate(); err != nil {
			return fmt.Errorf("gunfire: %s layer: %w", layer.name, err)
		}
	}
	return nil
}

// Engines are the two particle worlds this blast describes on a w×h
// unit stage: the white-hot core and the red flame, both leaving the
// muzzle along the aim in straight flight, each under its own lift
// and drag. Every world has Period 0 — a gunshot is a trigger, not a
// clock. The muzzle clamps inside stages smaller than its fractions
// reach.
func (c BlastConfig) Engines(w, h float64) (core, flame particle.Config) {
	muzzle := particle.Vec2{
		X: clamp(c.MuzzleX*w, 0, w),
		Y: clamp(c.MuzzleY*h, 0, h),
	}
	aim := dirAt(c.AngleDeg)
	base := func(l Layer) particle.Config {
		return particle.Config{
			Width:       w,
			Height:      h,
			Origin:      muzzle,
			Direction:   aim,
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
	return base(c.Core), base(c.Flame)
}

// dirAt is the unit heading deg degrees around level, covering the
// whole circle: 0 fires rightward, 90 straight up, ±180 leftward,
// -90 straight down. Y grows downward, so climbing means a negative Y.
func dirAt(deg float64) particle.Vec2 {
	sin, cos := math.Sincos(deg * math.Pi / 180)
	return particle.Vec2{X: cos, Y: -sin}
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
