// Package adjustparticle is the particle-trail tuner: a live nyan cat
// over the sky and a panel of the engine knobs (band width, life,
// spawn, speed, spread, nozzle). j/k pick a number, h/l change it,
// [/] take bigger steps, and the plume reacts live. s saves the
// nyan component's config and quits.
package adjustparticle

import (
	"fmt"

	"github.com/theprimeagen/apollo-11/exec-tui/components/nyan"
)

// DefaultConfigPath is the nyan component's own config file — the
// JSON lives with the component it tunes, relative to the module root.
const DefaultConfigPath = "components/nyan/config.json"

const (
	nKnobs = 9

	minBandWidth = 0.4
	maxBandWidth = 8
	minCount     = 1
	maxCount     = 40
	minPeriod    = 0.001
	maxPeriod    = 0.050
	minLife      = 0.05
	maxLife      = 4
	minSpeed     = 0
	maxSpeedVal  = 60
	minSpread    = 0
	maxSpread    = 1.2
	minNozzle    = 0
	maxNozzle    = 24
)

// knob is one editable number on the panel.
type knob int

const (
	knobBand knob = iota
	knobCount
	knobPeriod
	knobMinLife
	knobMaxLife
	knobMinSpeed
	knobMaxSpeed
	knobSpread
	knobNozzle
)

var knobMeta = []struct {
	label string
	step  float64
	lo    float64
	hi    float64
}{
	{"band width", 0.2, minBandWidth, maxBandWidth},
	{"count", 1, minCount, maxCount},
	{"period", 0.001, minPeriod, maxPeriod},
	{"min life", 0.05, minLife, maxLife},
	{"max life", 0.05, minLife, maxLife},
	{"min speed", 1, minSpeed, maxSpeedVal},
	{"max speed", 1, minSpeed, maxSpeedVal},
	{"spread", 0.02, minSpread, maxSpread},
	{"nozzle", 0.2, minNozzle, maxNozzle},
}

// Tuner is the knob cursor over a TrailConfig.
type Tuner struct {
	Trail  nyan.TrailConfig
	Cursor int
}

// NewTuner seeds the knobs from the active trail.
func NewTuner() *Tuner {
	return &Tuner{Trail: nyan.ActiveTrail()}
}

// Move slides the cursor delta rows, clamped to the knobs.
func (t *Tuner) Move(delta int) {
	if t == nil {
		return
	}
	t.Cursor += delta
	if t.Cursor < 0 {
		t.Cursor = 0
	}
	if t.Cursor >= nKnobs {
		t.Cursor = nKnobs - 1
	}
}

// Nudge changes the selected number by steps (usually ±1 or ±10).
func (t *Tuner) Nudge(steps int) {
	if t == nil || t.Cursor < 0 || t.Cursor >= nKnobs {
		return
	}
	meta := knobMeta[t.Cursor]
	v := t.get(knob(t.Cursor)) + meta.step*float64(steps)
	if v < meta.lo {
		v = meta.lo
	}
	if v > meta.hi {
		v = meta.hi
	}
	t.set(knob(t.Cursor), v)
	t.fixLife()
	t.fixSpeed()
}

func (t *Tuner) get(k knob) float64 {
	c := t.Trail
	switch k {
	case knobBand:
		return c.BandWidth
	case knobCount:
		return float64(c.Count)
	case knobPeriod:
		return c.Period
	case knobMinLife:
		return c.MinLife
	case knobMaxLife:
		return c.MaxLife
	case knobMinSpeed:
		return c.MinSpeed
	case knobMaxSpeed:
		return c.MaxSpeed
	case knobSpread:
		return c.Spread
	case knobNozzle:
		return c.Nozzle
	}
	return 0
}

func (t *Tuner) set(k knob, v float64) {
	switch k {
	case knobBand:
		t.Trail.BandWidth = v
	case knobCount:
		t.Trail.Count = int(v + 0.5)
	case knobPeriod:
		t.Trail.Period = v
	case knobMinLife:
		t.Trail.MinLife = v
	case knobMaxLife:
		t.Trail.MaxLife = v
	case knobMinSpeed:
		t.Trail.MinSpeed = v
	case knobMaxSpeed:
		t.Trail.MaxSpeed = v
	case knobSpread:
		t.Trail.Spread = v
	case knobNozzle:
		t.Trail.Nozzle = v
	}
}

func (t *Tuner) fixLife() {
	if t.Trail.MinLife > t.Trail.MaxLife {
		t.Trail.MinLife, t.Trail.MaxLife = t.Trail.MaxLife, t.Trail.MinLife
	}
}

func (t *Tuner) fixSpeed() {
	if t.Trail.MinSpeed > t.Trail.MaxSpeed {
		t.Trail.MinSpeed, t.Trail.MaxSpeed = t.Trail.MaxSpeed, t.Trail.MinSpeed
	}
}

func formatKnob(k knob, v float64) string {
	switch k {
	case knobCount:
		return fmt.Sprintf("%3.0f", v)
	case knobPeriod:
		return fmt.Sprintf("%5.3f", v)
	case knobMinSpeed, knobMaxSpeed:
		return fmt.Sprintf("%5.1f", v)
	default:
		return fmt.Sprintf("%5.2f", v)
	}
}
