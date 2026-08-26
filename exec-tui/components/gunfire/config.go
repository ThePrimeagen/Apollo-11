// Package gunfire is the one-shot shotgun blast, tuned to read like
// the Doom shotgun: one squeeze blooms a white-hot muzzle flash,
// throws seven pellets in a tight fan, sprays sparks that cool
// through the fire ramp, and — a beat later, on a short fuse — curls
// gray gunsmoke up out of the barrel. There is no period clock
// anywhere: the blast holds fire until Fire, plays the shot out, and
// leaves the stage exactly as it found it.
//
// Every number is data. BlastConfig carries the aim (angle, muzzle
// position, the smoke fuse and its climb), the flash brightness
// ladder, and one engine-knob Layer each for the flash, the pellets,
// the sparks, and the smoke. The knobs live in this component's
// config file and become the active blast via UseBlast, so the tuner
// and the demo stay on the same values.
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
	ErrAngle  = errors.New("gunfire: angle must be -80..80 degrees around level")
	ErrDelay  = errors.New("gunfire: smoke delay must be non-negative")
	ErrRise   = errors.New("gunfire: smoke rise must be 0..89 degrees above the aim")
	ErrLadder = errors.New("gunfire: flash ladder must climb: 1 <= edgeAt < midAt < coreAt")
)

// Layer is the engine-knob bundle one part of the blast fires under:
// how many specks a squeeze throws, how long and how fast they fly,
// how wide the fan is, how thick the bore is, and how far from the
// muzzle they are allowed to get (0 is unleashed).
type Layer struct {
	Count       int     `json:"count"`
	MinLife     float64 `json:"minLife"`
	MaxLife     float64 `json:"maxLife"`
	MinSpeed    float64 `json:"minSpeed"`
	MaxSpeed    float64 `json:"maxSpeed"`
	Spread      float64 `json:"spread"`
	Nozzle      float64 `json:"nozzle"`
	MaxDistance float64 `json:"maxDistance"`
}

// BlastConfig is the JSON that tunes the shot: where the muzzle sits
// (fractions of the stage), where it aims (degrees above level), the
// smoke fuse and how steeply the smoke climbs past the aim, the
// concentration ladder that decides how bright a flash cell burns,
// and the four layers.
type BlastConfig struct {
	AngleDeg     float64 `json:"angleDeg"`     // aim above level, degrees; negative dips
	MuzzleX      float64 `json:"muzzleX"`      // muzzle, as a fraction of stage width
	MuzzleY      float64 `json:"muzzleY"`      // muzzle, as a fraction of stage height
	SmokeDelay   float64 `json:"smokeDelay"`   // seconds after the trigger before the smoke
	SmokeRiseDeg float64 `json:"smokeRiseDeg"` // how far above the aim the smoke climbs
	EdgeAt       int     `json:"edgeAt"`       // flash concentration that earns the star
	MidAt        int     `json:"midAt"`        // … the yellow shade block
	CoreAt       int     `json:"coreAt"`       // … the white-hot core block

	Flash   Layer `json:"flash"`
	Pellets Layer `json:"pellets"`
	Sparks  Layer `json:"sparks"`
	Smoke   Layer `json:"smoke"`
}

// DefaultBlast is the stock Doom shotgun: a fat, brief flash leashed
// to a tight ball at the muzzle, the classic seven pellets in a
// ±5.7° fan (Doom's own shotgun spread), a fist of sparks that cool
// as they fall away, and a lazy curl of gunsmoke on a 0.12s fuse.
func DefaultBlast() BlastConfig {
	return BlastConfig{
		AngleDeg:     0,
		MuzzleX:      0.18,
		MuzzleY:      0.52,
		SmokeDelay:   0.12,
		SmokeRiseDeg: 42,
		EdgeAt:       2,
		MidAt:        4,
		CoreAt:       7,
		Flash:   Layer{Count: 90, MinLife: 0.05, MaxLife: 0.16, MinSpeed: 16, MaxSpeed: 44, Spread: 0.48, Nozzle: 1.2, MaxDistance: 7.5},
		Pellets: Layer{Count: 7, MinLife: 0.55, MaxLife: 0.75, MinSpeed: 58, MaxSpeed: 74, Spread: 0.07, Nozzle: 0.6},
		Sparks:  Layer{Count: 26, MinLife: 0.18, MaxLife: 0.5, MinSpeed: 9, MaxSpeed: 26, Spread: 0.5, Nozzle: 0.8},
		Smoke:   Layer{Count: 16, MinLife: 0.9, MaxLife: 1.8, MinSpeed: 2.5, MaxSpeed: 6, Spread: 0.55, Nozzle: 2.2},
	}
}

var activeBlast = DefaultBlast()

// UseBlast makes ActiveBlast — and every blast reading it — fire
// these settings. Invalid settings are rejected and the active blast
// holds.
func UseBlast(c BlastConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	activeBlast = c
	return nil
}

// ResetBlast restores the stock shotgun.
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
	if c.AngleDeg < -80 || c.AngleDeg > 80 {
		return ErrAngle
	}
	if c.SmokeDelay < 0 {
		return ErrDelay
	}
	if c.SmokeRiseDeg < 0 || c.SmokeRiseDeg > 89 {
		return ErrRise
	}
	if c.EdgeAt < 1 || c.MidAt <= c.EdgeAt || c.CoreAt <= c.MidAt {
		return ErrLadder
	}
	flash, pellets, sparks, smoke := c.Engines(nominalW, nominalH)
	for _, layer := range []struct {
		name string
		cfg  particle.Config
	}{
		{"flash", flash}, {"pellets", pellets}, {"sparks", sparks}, {"smoke", smoke},
	} {
		if err := layer.cfg.Validate(); err != nil {
			return fmt.Errorf("gunfire: %s layer: %w", layer.name, err)
		}
	}
	return nil
}

// Engines are the four particle worlds this blast describes on a w×h
// unit stage. Flash, pellets, and sparks all leave the muzzle along
// the aim and fly straight; the smoke leaves the same muzzle climbing
// SmokeRiseDeg past the aim and curls with the cartoon-wind swirl.
// Every world has Period 0 — a gunshot is a trigger, not a clock.
// The muzzle clamps inside stages smaller than its fractions reach.
func (c BlastConfig) Engines(w, h float64) (flash, pellets, sparks, smoke particle.Config) {
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
		}
	}
	aim := dirAt(c.AngleDeg)
	flash = base(c.Flash, aim)
	pellets = base(c.Pellets, aim)
	sparks = base(c.Sparks, aim)
	smoke = base(c.Smoke, dirAt(c.AngleDeg+c.SmokeRiseDeg)).SideSwirl(true)
	return flash, pellets, sparks, smoke
}

// dirAt is the unit heading deg degrees above level, firing rightward.
// Y grows downward, so climbing means a negative Y.
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
