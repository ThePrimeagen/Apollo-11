// Package adjustdust is the dust-off tuner: the live mirrored kick
// playing behind a panel of the puff knobs — the engine numbers, the
// kick geometry (angle, gap, loop side), and the gray ladder that maps
// concentration onto braille, ░, and ▒. j/k pick a knob, h/l change
// it, [/] take bigger steps, and the dust reacts live. s saves the
// dust component's config and quits.
package adjustdust

import (
	"fmt"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
)

// DefaultConfigPath is the dust component's own config file — the
// JSON lives with the component it tunes, relative to the module root.
const DefaultConfigPath = "components/dust/config.json"

const nKnobs = 16

// knob is one editable number on the panel.
type knob int

const (
	knobCount knob = iota
	knobPeriod
	knobMinLife
	knobMaxLife
	knobMinSpeed
	knobMaxSpeed
	knobSpread
	knobNozzle
	knobAngle
	knobGap
	knobLoop
	knobQuarterAt
	knobHalfAt
	knobBrailleFG
	knobQuarterFG
	knobHalfFG
)

var knobMeta = []struct {
	label string
	step  float64
	lo    float64
	hi    float64
}{
	{"count", 1, 1, 60},
	{"period", 0.05, 0.05, 4},
	{"min life", 0.05, 0.05, 5},
	{"max life", 0.05, 0.05, 5},
	{"min speed", 0.5, 0, 30},
	{"max speed", 0.5, 0, 30},
	{"spread", 0.02, 0, 1.2},
	{"nozzle", 0.5, 0, 24},
	{"angle", 1, 0, 85},
	{"gap", 1, 0, 40},
	{"loop", 1, 0, 1},
	{"quarter at", 1, 2, 30},
	{"half at", 1, 3, 40},
	{"braille gray", 1, dust.GrayMin, dust.GrayMax},
	{"quarter gray", 1, dust.GrayMin, dust.GrayMax},
	{"half gray", 1, dust.GrayMin, dust.GrayMax},
}

// Tuner is the knob cursor over a PuffConfig.
type Tuner struct {
	Puff   dust.PuffConfig
	Cursor int
}

// NewTuner seeds the knobs from the active puff.
func NewTuner() *Tuner {
	return &Tuner{Puff: dust.ActivePuff()}
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
	t.fixLadder()
}

func (t *Tuner) get(k knob) float64 {
	c := t.Puff
	switch k {
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
	case knobAngle:
		return c.AngleDeg
	case knobGap:
		return c.Gap
	case knobLoop:
		if c.LoopUp {
			return 1
		}
		return 0
	case knobQuarterAt:
		return float64(c.QuarterAt)
	case knobHalfAt:
		return float64(c.HalfAt)
	case knobBrailleFG:
		return float64(c.BrailleFG)
	case knobQuarterFG:
		return float64(c.QuarterFG)
	case knobHalfFG:
		return float64(c.HalfFG)
	}
	return 0
}

func (t *Tuner) set(k knob, v float64) {
	switch k {
	case knobCount:
		t.Puff.Count = int(v + 0.5)
	case knobPeriod:
		t.Puff.Period = v
	case knobMinLife:
		t.Puff.MinLife = v
	case knobMaxLife:
		t.Puff.MaxLife = v
	case knobMinSpeed:
		t.Puff.MinSpeed = v
	case knobMaxSpeed:
		t.Puff.MaxSpeed = v
	case knobSpread:
		t.Puff.Spread = v
	case knobNozzle:
		t.Puff.Nozzle = v
	case knobAngle:
		t.Puff.AngleDeg = v
	case knobGap:
		t.Puff.Gap = v
	case knobLoop:
		t.Puff.LoopUp = v >= 0.5
	case knobQuarterAt:
		t.Puff.QuarterAt = int(v + 0.5)
	case knobHalfAt:
		t.Puff.HalfAt = int(v + 0.5)
	case knobBrailleFG:
		t.Puff.BrailleFG = int(v + 0.5)
	case knobQuarterFG:
		t.Puff.QuarterFG = int(v + 0.5)
	case knobHalfFG:
		t.Puff.HalfFG = int(v + 0.5)
	}
}

func (t *Tuner) fixLife() {
	if t.Puff.MinLife > t.Puff.MaxLife {
		t.Puff.MinLife, t.Puff.MaxLife = t.Puff.MaxLife, t.Puff.MinLife
	}
}

func (t *Tuner) fixSpeed() {
	if t.Puff.MinSpeed > t.Puff.MaxSpeed {
		t.Puff.MinSpeed, t.Puff.MaxSpeed = t.Puff.MaxSpeed, t.Puff.MinSpeed
	}
}

// fixLadder keeps the shade ladder climbing: ▒ always needs more
// concentration than ░.
func (t *Tuner) fixLadder() {
	if t.Puff.HalfAt <= t.Puff.QuarterAt {
		t.Puff.HalfAt = t.Puff.QuarterAt + 1
	}
}

func formatKnob(k knob, v float64) string {
	switch k {
	case knobLoop:
		if v >= 0.5 {
			return "   up"
		}
		return " down"
	case knobCount, knobQuarterAt, knobHalfAt, knobBrailleFG, knobQuarterFG, knobHalfFG, knobAngle, knobGap:
		return fmt.Sprintf("%5.0f", v)
	case knobMinSpeed, knobMaxSpeed:
		return fmt.Sprintf("%5.1f", v)
	default:
		return fmt.Sprintf("%5.2f", v)
	}
}
