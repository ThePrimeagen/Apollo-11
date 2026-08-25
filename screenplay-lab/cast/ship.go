// Package cast is the screenplay-lab troupe: the actors the demo puts on
// stage. Each one wraps an existing lab component — the LM sprite atlas
// and booster fire from lander-lab, the starfield from stars-lab, the
// banner font from terminal-fonts — behind the screenplay.Actor face.
package cast

import (
	"github.com/theprimeagen/apollo-11/lander-lab/components/fire"
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

const (
	// BodyCols/BodyRows is the size-4 frame: the full zoomed-in craft.
	BodyCols = 26
	BodyRows = 10
	// FlyInSeconds is how long the slide from the right wing to center
	// stage takes.
	FlyInSeconds = 4.0
	// BobPeriodSeconds is one full up-and-down of the parked bobble.
	BobPeriodSeconds = 10.0
	// BobAmplitudeCells is how far the bobble rides from center: one
	// full cell up and one down. (Half a cell needs half-shifted art
	// the atlas doesn't have yet.)
	BobAmplitudeCells = 1
	// FlameRow/FlameCol hang the booster box off the tail, relative to
	// the hull's top-left, so the plume trails right of the craft.
	FlameRow = 2
	FlameCol = 19
)

// Ship is the Apollo craft flying west: the size-4 W-heading frame with
// its baked tilde plume stripped and a live left-to-right booster fire
// trailing from the tail. It enters from the right wing, parks at center
// stage, and bobs on a slow sine.
type Ship struct {
	Body  sprite.Sprite
	Flame *fire.Flame
}

// NewShip builds the westbound craft. The fire has not yet emitted.
func NewShip(seed int64) *Ship {
	return &Ship{}
}

// Advance moves the ship's clock and burns the fire. dt <= 0 holds.
func (s *Ship) Advance(dt float64) {
}

// Clock is how many seconds of scene time the ship has played.
func (s *Ship) Clock() float64 {
	return 0
}

// Paint composes flame then hull at this instant's spot — the hull always
// wins the overlap at the tail.
func (s *Ship) Paint(st *screenplay.Stage) {
}

// FlightPath is the hull's top-left at t seconds into the scene, on a
// stageW×stageH stage: offscreen right at t=0, an eased slide to center
// by FlyInSeconds, then a ±BobAmplitudeCells sine bob with a
// BobPeriodSeconds period.
func FlightPath(stageW, stageH int, t float64) (row, col int) {
	return 0, 0
}
