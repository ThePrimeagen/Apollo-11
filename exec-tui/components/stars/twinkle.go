package stars

import (
	"errors"
	"math"
)

// Twinkle is the breathing sky mode: every layer parks where it
// scattered and some of the stars fade in and out while the rest hold
// steady. Which stars breathe, how long one full breath takes, and
// how long each ramp lasts are all deterministic per star, drawn from
// the active TwinkleConfig — so the same sky always breathes the same
// way, and a knob change retunes every breath on the next frame.
var Twinkle = Strategy{Name: "twinkle"}

// The rails every twinkle knob lives between, in seconds. A cycle is
// one full fade-in / hold / fade-out / hold lap; a fade is one ramp.
const (
	MinTwinkleCycle = 0.5
	MaxTwinkleCycle = 30.0
	MinTwinkleFade  = 0.05
	MaxTwinkleFade  = 10.0
)

var (
	ErrTwinkleCycle = errors.New("stars: twinkle cycle range must sit in [0.5, 30]s with min <= max")
	ErrTwinkleFade  = errors.New("stars: twinkle fade range must sit in [0.05, 10]s with min <= max")
)

// TwinkleConfig is the four knobs of the breathing: each star picks
// its cycle from [MinCycleSeconds, MaxCycleSeconds] and its ramps
// from [MinFadeSeconds, MaxFadeSeconds], clamped so a fade never
// outlasts its half of the cycle.
type TwinkleConfig struct {
	MinCycleSeconds float64 `json:"minCycleSeconds"`
	MaxCycleSeconds float64 `json:"maxCycleSeconds"`
	MinFadeSeconds  float64 `json:"minFadeSeconds"`
	MaxFadeSeconds  float64 `json:"maxFadeSeconds"`
}

// DefaultTwinkle is the stock breathing: unhurried laps of two to
// seven seconds with ramps under two.
func DefaultTwinkle() TwinkleConfig {
	return TwinkleConfig{
		MinCycleSeconds: 2,
		MaxCycleSeconds: 7,
		MinFadeSeconds:  0.4,
		MaxFadeSeconds:  1.6,
	}
}

var activeTwinkle = DefaultTwinkle()

// UseTwinkle makes ActiveTwinkle — and every twinkling paint — breathe
// these ranges. Invalid ranges are rejected and the active ones hold.
func UseTwinkle(c TwinkleConfig) error {
	if err := c.Validate(); err != nil {
		return err
	}
	activeTwinkle = c
	return nil
}

// ResetTwinkle restores the stock breathing.
func ResetTwinkle() {
	activeTwinkle = DefaultTwinkle()
}

// ActiveTwinkle is the breathing now in effect.
func ActiveTwinkle() TwinkleConfig {
	return activeTwinkle
}

// Validate reports the first thing wrong with the ranges.
func (c TwinkleConfig) Validate() error {
	if !orderedRange(c.MinCycleSeconds, c.MaxCycleSeconds, MinTwinkleCycle, MaxTwinkleCycle) {
		return ErrTwinkleCycle
	}
	if !orderedRange(c.MinFadeSeconds, c.MaxFadeSeconds, MinTwinkleFade, MaxTwinkleFade) {
		return ErrTwinkleFade
	}
	return nil
}

// orderedRange reports lo <= hi with both finite and inside the rails.
func orderedRange(lo, hi, floor, ceil float64) bool {
	for _, v := range []float64{lo, hi} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < floor || v > ceil {
			return false
		}
	}
	return lo <= hi
}

// twinkleRamps are the dimmed grays a breathing star wears mid-fade,
// one ramp per layer, dark to bright — always dimmer than the layer's
// own tints so a fade reads as a fade.
var twinkleRamps = [4][]int{
	{233, 235, 237},
	{234, 237, 241},
	{235, 240, 246},
	{236, 242, 249},
}

// twinkleHash is a deterministic per-star stream: the same star and
// salt always draw the same 64 bits.
func twinkleHash(row, col, kind, salt int) uint64 {
	z := uint64(int64(row))*0x9E3779B97F4A7C15 ^
		uint64(int64(col))*0xC2B2AE3D27D4EB4F ^
		uint64(int64(kind))*0x165667B19E3779F9 ^
		uint64(int64(salt))*0x27D4EB2F165667C5
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// twinkleFrac is the star's deterministic draw in [0, 1) for one salt.
func twinkleFrac(row, col, kind, salt int) float64 {
	return float64(twinkleHash(row, col, kind, salt)%(1<<20)) / (1 << 20)
}

// Twinkles reports whether this star breathes — about one star in
// three — or holds steady. Deterministic per star.
func Twinkles(row, col, kind int) bool {
	return twinkleHash(row, col, kind, 0)%3 == 0
}

// TwinkleLevel is the star's brightness at t seconds into the
// breathing, 0 (out) to 1 (full). Steady stars are always 1; time
// before the curtain clamps to the start. One lap is fade in over the
// star's ramp, hold bright, fade out, hold dark — and around again.
func TwinkleLevel(row, col, kind int, t float64, c TwinkleConfig) float64 {
	if !Twinkles(row, col, kind) {
		return 1
	}
	if t < 0 {
		t = 0
	}
	cycle := pickSeconds(c.MinCycleSeconds, c.MaxCycleSeconds, row, col, kind, 1)
	if cycle <= 0 {
		return 1
	}
	fade := pickSeconds(c.MinFadeSeconds, c.MaxFadeSeconds, row, col, kind, 2)
	if fade > cycle/2 {
		fade = cycle / 2
	}
	local := math.Mod(t+twinkleFrac(row, col, kind, 3)*cycle, cycle)
	half := cycle / 2
	switch {
	case fade <= 0:
		if local < half {
			return 1
		}
		return 0
	case local < fade:
		return local / fade
	case local < half:
		return 1
	case local < half+fade:
		return 1 - (local-half)/fade
	default:
		return 0
	}
}

// pickSeconds is the star's deterministic draw from [lo, hi].
func pickSeconds(lo, hi float64, row, col, kind, salt int) float64 {
	if hi < lo {
		hi = lo
	}
	return lo + twinkleFrac(row, col, kind, salt)*(hi-lo)
}

// TwinkleInk is the ink the star wears at t seconds into the
// breathing: -1 when faded out, its own tint at full brightness, and
// a dimmed gray from its layer's ramp mid-fade.
func TwinkleInk(row, col, kind int, t float64, c TwinkleConfig) int {
	if kind < 0 || kind >= len(twinkleRamps) {
		return -1
	}
	lvl := TwinkleLevel(row, col, kind, t, c)
	if lvl <= 0 {
		return -1
	}
	if lvl >= 1 {
		return tint(kind, row, col)
	}
	ramp := twinkleRamps[kind]
	idx := int(lvl * float64(len(ramp)))
	if idx >= len(ramp) {
		idx = len(ramp) - 1
	}
	return ramp[idx]
}
