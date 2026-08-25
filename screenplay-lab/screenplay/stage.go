// Package screenplay is the theater: a Stage to paint on, Actors that
// play over time, Scenes that gather a cast, and a Screenplay that runs
// scenes in order, cutting on demand. The package composes sprites; it
// draws nothing itself and owns no clock but the dt handed to it.
package screenplay

import (
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

// Stage is the fixed board one frame is composed on. Paint back-to-front:
// whatever blits last sits on top.
type Stage struct {
	Board sprite.Sprite
}

// NewStage allocates a transparent w×h stage.
func NewStage(w, h int) *Stage {
	return &Stage{}
}

// Size is the stage dimensions in terminal cells.
func (st *Stage) Size() (w, h int) {
	return 0, 0
}

// Put writes one cell. Out of bounds is ignored.
func (st *Stage) Put(row, col int, c sprite.Cell) {
}

// Blit composes a sprite with its top-left at (row, col). Transparent
// cells do not overwrite; anything past an edge is clipped.
func (st *Stage) Blit(row, col int, sp sprite.Sprite) {
}

// Render is the ANSI view of the stage.
func (st *Stage) Render() string {
	return ""
}
