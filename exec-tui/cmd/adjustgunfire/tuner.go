// Package adjustgunfire is the shotgun-blast tuner: the live one-shot
// blast playing behind a paged panel of every blast knob. Five pages
// of eight knobs each — aim (angle, muzzle, smoke fuse and rise, the
// flash ladder), then one page per layer: flash, pellets, sparks,
// smoke. tab flips pages, j/k pick a knob, h/l turn it, [/] take
// bigger steps, f pulls the trigger now, and the tool re-fires on its
// own so the blast is always in the air. s saves the gunfire
// component's config and quits.
package adjustgunfire

import (
	"fmt"
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
)

// DefaultConfigPath is the gunfire component's own config file — the
// JSON lives with the component it tunes, relative to the module root.
const DefaultConfigPath = "components/gunfire/config.json"

const (
	nPages       = 5
	knobsPerPage = 8
)

// The pages, in tab order.
const (
	pageAim = iota
	pageFlash
	pagePellets
	pageSparks
	pageSmoke
)

var pageNames = [nPages]string{"aim", "flash", "pellets", "sparks", "smoke"}

// meta is one knob's rails: label, step, floor, ceiling.
type meta struct {
	label string
	step  float64
	lo    float64
	hi    float64
}

// aimMeta is the aim page: where the gun points and sits, the smoke
// fuse and climb, and the flash brightness ladder.
var aimMeta = [knobsPerPage]meta{
	{"angle", 1, -80, 80},
	{"muzzle x", 0.01, 0, 1},
	{"muzzle y", 0.01, 0, 1},
	{"smoke delay", 0.01, 0, 2},
	{"smoke rise", 1, 0, 89},
	{"edge at", 1, 1, 30},
	{"mid at", 1, 2, 40},
	{"core at", 1, 3, 60},
}

// layerMeta is every layer page. Count has no artificial ceiling —
// zero is a silent layer and the top is whatever the terminal holds.
var layerMeta = [knobsPerPage]meta{
	{"count", 1, 0, math.Inf(1)},
	{"min life", 0.01, 0.01, 6},
	{"max life", 0.01, 0.01, 6},
	{"min speed", 1, 0, 120},
	{"max speed", 1, 0, 120},
	{"spread", 0.02, 0, 1.2},
	{"nozzle", 0.2, 0, 24},
	{"max dist", 0.5, 0, 80},
}

// pageMeta is the knob table of one page.
func pageMeta(page int) [knobsPerPage]meta {
	if page == pageAim {
		return aimMeta
	}
	return layerMeta
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

// layer is the Layer the current page edits, or nil on the aim page.
func (t *Tuner) layer() *gunfire.Layer {
	switch t.Page {
	case pageFlash:
		return &t.Blast.Flash
	case pagePellets:
		return &t.Blast.Pellets
	case pageSparks:
		return &t.Blast.Sparks
	case pageSmoke:
		return &t.Blast.Smoke
	}
	return nil
}

// Flip turns delta pages, wrapping around the five.
func (t *Tuner) Flip(delta int) {
	if t == nil {
		return
	}
	t.Page = ((t.Page+delta)%nPages + nPages) % nPages
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
	if t.Cursor >= knobsPerPage {
		t.Cursor = knobsPerPage - 1
	}
}

// Nudge changes the selected number by steps (usually ±1 or ±10).
func (t *Tuner) Nudge(steps int) {
	if t == nil || t.Cursor < 0 || t.Cursor >= knobsPerPage {
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

// get reads knob at the given row of the current page.
func (t *Tuner) get(cursor int) float64 {
	if l := t.layer(); l != nil {
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
		}
		return 0
	}
	c := t.Blast
	switch cursor {
	case 0:
		return c.AngleDeg
	case 1:
		return c.MuzzleX
	case 2:
		return c.MuzzleY
	case 3:
		return c.SmokeDelay
	case 4:
		return c.SmokeRiseDeg
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
		}
		return
	}
	switch cursor {
	case 0:
		t.Blast.AngleDeg = v
	case 1:
		t.Blast.MuzzleX = v
	case 2:
		t.Blast.MuzzleY = v
	case 3:
		t.Blast.SmokeDelay = v
	case 4:
		t.Blast.SmokeRiseDeg = v
	case 5:
		t.Blast.EdgeAt = int(v + 0.5)
	case 6:
		t.Blast.MidAt = int(v + 0.5)
	case 7:
		t.Blast.CoreAt = int(v + 0.5)
	}
}

// fixRanges keeps every layer's min/max pairs ordered by swapping,
// never folding.
func (t *Tuner) fixRanges() {
	for _, l := range []*gunfire.Layer{
		&t.Blast.Flash, &t.Blast.Pellets, &t.Blast.Sparks, &t.Blast.Smoke,
	} {
		if l.MinLife > l.MaxLife {
			l.MinLife, l.MaxLife = l.MaxLife, l.MinLife
		}
		if l.MinSpeed > l.MaxSpeed {
			l.MinSpeed, l.MaxSpeed = l.MaxSpeed, l.MinSpeed
		}
	}
}

// fixLadder keeps the flash ladder climbing: the core always needs
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
		case 0, 4, 5, 6, 7: // angle, rise, ladder
			return fmt.Sprintf("%5.0f", v)
		default: // muzzle fractions, smoke delay
			return fmt.Sprintf("%5.2f", v)
		}
	}
	switch cursor {
	case 0: // count
		return fmt.Sprintf("%5.0f", v)
	case 1, 2, 5: // lives, spread
		return fmt.Sprintf("%5.2f", v)
	default: // speeds, nozzle, max dist
		return fmt.Sprintf("%5.1f", v)
	}
}
