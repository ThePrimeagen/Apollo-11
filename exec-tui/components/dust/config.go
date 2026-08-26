// Package dust is the landing kick-up: something drops, and dust blows
// out of the floor to both sides — two mirrored particle engines in
// swirl mode climbing away from a shared point at a shallow angle,
// with a still gap of columns between the nozzles. Concentration picks
// the symbol: heavy cells wear shade blocks in brightening grays, and
// the thin fringe is braille with the exact dots computed from where
// each speck sits inside its cell. The knobs live in this component's
// config file and become the active puff via UsePuff, so the editor
// and the demo stay on the same values.
package dust

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
)

var (
	ErrAngle  = errors.New("dust: angle must be 0..89 degrees above horizontal")
	ErrGap    = errors.New("dust: gap must be non-negative")
	ErrLadder = errors.New("dust: thresholds must climb: 2 <= quarterAt < halfAt")
	ErrGray   = errors.New("dust: grays must sit on the xterm gray ramp 232..255")
)

// The xterm-256 grayscale ramp, near-black to near-white.
const (
	GrayMin = 232
	GrayMax = 255
)

// PuffConfig is the JSON that tunes the dust-off: the engine knobs both
// mirrored engines share, the geometry of the kick, and the gray ladder
// that decides which symbol a cell's concentration earns.
type PuffConfig struct {
	Count    int     `json:"count"`
	Period   float64 `json:"period"`
	MinLife  float64 `json:"minLife"`
	MaxLife  float64 `json:"maxLife"`
	MinSpeed float64 `json:"minSpeed"`
	MaxSpeed float64 `json:"maxSpeed"`
	Spread   float64 `json:"spread"`
	Nozzle   float64 `json:"nozzle"`

	AngleDeg float64 `json:"angleDeg"` // climb above horizontal, degrees
	Gap      float64 `json:"gap"`      // still columns between the two nozzles
	LoopUp   bool    `json:"loopUp"`   // swirl loops curl up (true) or down

	QuarterAt int `json:"quarterAt"` // concentration that earns ░
	HalfAt    int `json:"halfAt"`    // concentration that earns ▒
	BrailleFG int `json:"brailleFG"` // deep gray for the braille fringe
	QuarterFG int `json:"quarterFG"` // mid gray for ░
	HalfFG    int `json:"halfFG"`    // light gray for ▒
}

// DefaultPuff is the stock landing kick: 15° above horizontal, an
// 8-column still gap, upward loops, and a deep-to-light gray ladder.
func DefaultPuff() PuffConfig {
	return PuffConfig{
		Count:     26,
		Period:    1.4,
		MinLife:   1.1,
		MaxLife:   2.1,
		MinSpeed:  5,
		MaxSpeed:  9,
		Spread:    0.10,
		Nozzle:    3,
		AngleDeg:  15,
		Gap:       8,
		LoopUp:    true,
		QuarterAt: 3,
		HalfAt:    6,
		BrailleFG: 240,
		QuarterFG: 246,
		HalfFG:    252,
	}
}

var activePuff = DefaultPuff()

// UsePuff makes ActivePuff — and every cloud reading it — kick these
// settings. Invalid settings are rejected and the active puff holds.
func UsePuff(c PuffConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	activePuff = c
	return nil
}

// ResetPuff restores the stock kick.
func ResetPuff() {
	_ = UsePuff(DefaultPuff())
}

// ActivePuff is the kick settings now in effect.
func ActivePuff() PuffConfig { return activePuff }

// nominal box for validating engine knobs without a real stage.
const (
	nominalW = 120
	nominalH = 60
)

// Validate reports the first thing wrong with c.
func (c PuffConfig) Validate() error {
	if c.AngleDeg < 0 || c.AngleDeg >= 90 {
		return ErrAngle
	}
	if c.Gap < 0 {
		return ErrGap
	}
	if c.QuarterAt < 2 || c.HalfAt <= c.QuarterAt {
		return ErrLadder
	}
	for _, g := range []int{c.BrailleFG, c.QuarterFG, c.HalfFG} {
		if g < GrayMin || g > GrayMax {
			return ErrGray
		}
	}
	left, _ := c.Engines(nominalW, nominalH)
	return left.Validate()
}

// floorUnits is how far above the bottom edge the nozzles sit.
const floorUnits = 4

// Engines are the two mirrored particle worlds this puff describes on
// a w×h unit stage: both climb AngleDeg above horizontal out of a
// shared floor point — one leftward, one rightward — with the gap of
// still columns between the nozzles, and both swirling to the puff's
// loop side. Origins clamp inside stages smaller than the gap.
func (c PuffConfig) Engines(w, h float64) (left, right particle.Config) {
	sin, cos := math.Sincos(c.AngleDeg * math.Pi / 180)
	base := particle.Config{
		Width:    w,
		Height:   h,
		Count:    c.Count,
		Period:   c.Period,
		MinLife:  c.MinLife,
		MaxLife:  c.MaxLife,
		MinSpeed: c.MinSpeed,
		MaxSpeed: c.MaxSpeed,
		Spread:   c.Spread,
		Nozzle:   c.Nozzle,
	}.SideSwirl(c.LoopUp)
	floor := clamp(h-floorUnits, 0, h)
	left = base
	left.Direction = particle.Vec2{X: -cos, Y: -sin}
	left.Origin = particle.Vec2{X: clamp(w/2-c.Gap/2, 0, w), Y: floor}
	right = base
	right.Direction = particle.Vec2{X: cos, Y: -sin}
	right.Origin = particle.Vec2{X: clamp(w/2+c.Gap/2, 0, w), Y: floor}
	return left, right
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

// LoadPuff reads a puff-config JSON file.
func LoadPuff(path string) (PuffConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return PuffConfig{}, err
	}
	var c PuffConfig
	if err := json.Unmarshal(raw, &c); err != nil {
		return PuffConfig{}, fmt.Errorf("dust: %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return PuffConfig{}, err
	}
	return c, nil
}

// Save writes the puff settings as JSON.
func (c PuffConfig) Save(path string) error {
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
