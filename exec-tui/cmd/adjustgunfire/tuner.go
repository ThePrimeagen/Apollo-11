// Package adjustgunfire is the muzzle-flame tuner on the eight-point
// compass: the live one-shot flame burning behind a paged panel of
// every blast knob. Ten pages — aim (heading, muzzle, the two-frame
// pulse, the core brightness ladder), the shared core, then one page
// per direction: N, NE, E, SE, S, SW, W, NW — each direction carrying
// its ten engine knobs plus the five color stops its flame cools
// through. tab flips pages, j/k pick a knob, h/l turn it, [/] take
// bigger steps, f pulls the trigger now, and the tool re-fires on its
// own so the flame is always burning. s saves the gunfire component's
// config and quits.
package adjustgunfire

import (
	"fmt"
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// DefaultConfigPath is the gunfire component's own config file — the
// JSON lives with the component it tunes, relative to the module root.
const DefaultConfigPath = "components/gunfire/config.json"

// The pages, in tab order: aim, the shared core, then the compass.
const (
	pageAim = iota
	pageCore
	firstHeadingPage
)

const nPages = firstHeadingPage + 8

var pageNames = [nPages]string{
	"aim", "core", "N", "NE", "E", "SE", "S", "SW", "W", "NW",
}

// meta is one knob's rails: label, step, floor, ceiling.
type meta struct {
	label string
	step  float64
	lo    float64
	hi    float64
}

// aimMeta is the aim page: the compass heading the trigger fires,
// where the muzzle sits, the two-frame pulse, and the core ladder.
var aimMeta = []meta{
	{"heading", 1, 0, 7},
	{"muzzle x", 0.01, 0, 1},
	{"muzzle y", 0.01, 0, 1},
	{"pulse delay", 0.01, 0, 1},
	{"pulse frac", 0.05, 0, 1},
	{"edge at", 1, 1, 30},
	{"mid at", 1, 2, 40},
	{"core at", 1, 3, 60},
}

// layerMeta is the engine-knob page. Count has no artificial
// ceiling — zero is a silent layer and the top is whatever the
// terminal holds.
var layerMeta = []meta{
	{"count", 1, 0, math.Inf(1)},
	{"min life", 0.01, 0.01, 6},
	{"max life", 0.01, 0.01, 6},
	{"min speed", 1, 0, 120},
	{"max speed", 1, 0, 120},
	{"spread", 0.02, 0, 1.2},
	{"nozzle", 0.2, 0, 24},
	{"max dist", 0.5, 0, 80},
	{"lift", 1, 0, 200},
	{"drag", 0.1, 0, 12},
}

// shotMeta is a direction page: the engine knobs plus the five color
// stops of that direction's cooling ramp, freshest first.
var shotMeta = append(append([]meta{}, layerMeta...),
	meta{"color 1", 1, 1, 255},
	meta{"color 2", 1, 1, 255},
	meta{"color 3", 1, 1, 255},
	meta{"color 4", 1, 1, 255},
	meta{"color 5", 1, 1, 255},
)

// pageMeta is the knob table of one page.
func pageMeta(page int) []meta {
	switch {
	case page == pageAim:
		return aimMeta
	case page == pageCore:
		return layerMeta
	default:
		return shotMeta
	}
}

// Tuner is the page + knob cursor over a BlastConfig.
type Tuner struct {
	Blast  gunfire.BlastConfig
	Page   int
	Cursor int
}

// NewTuner seeds the knobs from the active blast, opening on aim.
func NewTuner() *Tuner {
	return &Tuner{Blast: gunfire.ActiveBlast()}
}

// shot is the Shot the current page edits, or nil off the compass
// pages.
func (t *Tuner) shot() *gunfire.Shot {
	switch t.Page {
	case firstHeadingPage + 0:
		return &t.Blast.N
	case firstHeadingPage + 1:
		return &t.Blast.NE
	case firstHeadingPage + 2:
		return &t.Blast.E
	case firstHeadingPage + 3:
		return &t.Blast.SE
	case firstHeadingPage + 4:
		return &t.Blast.S
	case firstHeadingPage + 5:
		return &t.Blast.SW
	case firstHeadingPage + 6:
		return &t.Blast.W
	case firstHeadingPage + 7:
		return &t.Blast.NW
	}
	return nil
}

// layer is the Layer the current page edits, or nil on the aim page.
func (t *Tuner) layer() *gunfire.Layer {
	if t.Page == pageCore {
		return &t.Blast.Core
	}
	if s := t.shot(); s != nil {
		return &s.Layer
	}
	return nil
}

// Flip turns delta pages, wrapping around the ten; the cursor clamps
// into the new page's rows.
func (t *Tuner) Flip(delta int) {
	if t == nil {
		return
	}
	t.Page = ((t.Page+delta)%nPages + nPages) % nPages
	if last := len(pageMeta(t.Page)) - 1; t.Cursor > last {
		t.Cursor = last
	}
}

// Move slides the cursor delta rows, clamped to the page's knobs.
func (t *Tuner) Move(delta int) {
	if t == nil {
		return
	}
	t.Cursor += delta
	if t.Cursor < 0 {
		t.Cursor = 0
	}
	if last := len(pageMeta(t.Page)) - 1; t.Cursor > last {
		t.Cursor = last
	}
}

// Nudge changes the selected number by steps (usually ±1 or ±10).
func (t *Tuner) Nudge(steps int) {
	if t == nil || t.Cursor < 0 || t.Cursor >= len(pageMeta(t.Page)) {
		return
	}
	m := pageMeta(t.Page)[t.Cursor]
	v := t.get(t.Cursor) + m.step*float64(steps)
	if v < m.lo {
		v = m.lo
	}
	if v > m.hi {
		v = m.hi
	}
	t.set(t.Cursor, v)
	t.fixRanges()
	t.fixLadder()
}

// headingIdx is the active heading's slot on the compass.
func (t *Tuner) headingIdx() int {
	for i, h := range sprite.Headings {
		if t.Blast.Heading == h {
			return i
		}
	}
	return 0
}

// get reads knob at the given row of the current page.
func (t *Tuner) get(cursor int) float64 {
	if l := t.layer(); l != nil {
		if s := t.shot(); s != nil && cursor >= len(layerMeta) {
			return float64(s.Colors[cursor-len(layerMeta)])
		}
		switch cursor {
		case 0:
			return float64(l.Count)
		case 1:
			return l.MinLife
		case 2:
			return l.MaxLife
		case 3:
			return l.MinSpeed
		case 4:
			return l.MaxSpeed
		case 5:
			return l.Spread
		case 6:
			return l.Nozzle
		case 7:
			return l.MaxDistance
		case 8:
			return l.Lift
		case 9:
			return l.Drag
		}
		return 0
	}
	c := t.Blast
	switch cursor {
	case 0:
		return float64(t.headingIdx())
	case 1:
		return c.MuzzleX
	case 2:
		return c.MuzzleY
	case 3:
		return c.PulseDelay
	case 4:
		return c.PulseFrac
	case 5:
		return float64(c.EdgeAt)
	case 6:
		return float64(c.MidAt)
	case 7:
		return float64(c.CoreAt)
	}
	return 0
}

// set writes knob at the given row of the current page.
func (t *Tuner) set(cursor int, v float64) {
	if l := t.layer(); l != nil {
		if s := t.shot(); s != nil && cursor >= len(layerMeta) {
			s.Colors[cursor-len(layerMeta)] = int(v + 0.5)
			return
		}
		switch cursor {
		case 0:
			l.Count = int(v + 0.5)
		case 1:
			l.MinLife = v
		case 2:
			l.MaxLife = v
		case 3:
			l.MinSpeed = v
		case 4:
			l.MaxSpeed = v
		case 5:
			l.Spread = v
		case 6:
			l.Nozzle = v
		case 7:
			l.MaxDistance = v
		case 8:
			l.Lift = v
		case 9:
			l.Drag = v
		}
		return
	}
	switch cursor {
	case 0:
		t.Blast.Heading = sprite.Headings[int(v+0.5)]
	case 1:
		t.Blast.MuzzleX = v
	case 2:
		t.Blast.MuzzleY = v
	case 3:
		t.Blast.PulseDelay = v
	case 4:
		t.Blast.PulseFrac = v
	case 5:
		t.Blast.EdgeAt = int(v + 0.5)
	case 6:
		t.Blast.MidAt = int(v + 0.5)
	case 7:
		t.Blast.CoreAt = int(v + 0.5)
	}
}

// fixRanges keeps every layer's min/max pairs ordered by swapping,
// never folding — the core and all eight shots.
func (t *Tuner) fixRanges() {
	layers := []*gunfire.Layer{
		&t.Blast.Core,
		&t.Blast.N.Layer, &t.Blast.NE.Layer, &t.Blast.E.Layer, &t.Blast.SE.Layer,
		&t.Blast.S.Layer, &t.Blast.SW.Layer, &t.Blast.W.Layer, &t.Blast.NW.Layer,
	}
	for _, l := range layers {
		if l.MinLife > l.MaxLife {
			l.MinLife, l.MaxLife = l.MaxLife, l.MinLife
		}
		if l.MinSpeed > l.MaxSpeed {
			l.MinSpeed, l.MaxSpeed = l.MaxSpeed, l.MinSpeed
		}
	}
}

// fixLadder keeps the core ladder climbing: the core always needs
// more concentration than the mid, the mid more than the edge.
func (t *Tuner) fixLadder() {
	if t.Blast.EdgeAt < 1 {
		t.Blast.EdgeAt = 1
	}
	if t.Blast.MidAt <= t.Blast.EdgeAt {
		t.Blast.MidAt = t.Blast.EdgeAt + 1
	}
	if t.Blast.CoreAt <= t.Blast.MidAt {
		t.Blast.CoreAt = t.Blast.MidAt + 1
	}
}

func formatKnob(page, cursor int, v float64) string {
	if page == pageAim {
		switch cursor {
		case 0: // heading, by name
			i := int(v + 0.5)
			if i < 0 || i >= len(sprite.Headings) {
				i = 0
			}
			return fmt.Sprintf("%5s", string(sprite.Headings[i]))
		case 5, 6, 7: // ladder
			return fmt.Sprintf("%5.0f", v)
		default: // muzzle fractions, pulse delay and frac
			return fmt.Sprintf("%5.2f", v)
		}
	}
	switch cursor {
	case 0, 8, 10, 11, 12, 13, 14: // count, lift, colors
		return fmt.Sprintf("%5.0f", v)
	case 1, 2, 5, 9: // lives, spread, drag
		return fmt.Sprintf("%5.2f", v)
	default: // speeds, nozzle, max dist
		return fmt.Sprintf("%5.1f", v)
	}
}
