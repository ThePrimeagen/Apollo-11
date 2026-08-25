// Package screenplay is the theater: a Stage to compose on, Actors that
// play over time, Scenes that gather a cast, and a Screenplay that runs
// scenes in order, cutting on demand. The package composes sprites; it
// owns no clock but the dt handed to it and draws nothing on its own.
package screenplay

import (
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

// Stage is the fixed board one frame is composed on. Paint back-to-front:
// whatever lands last sits on top.
type Stage struct {
	Board sprite.Sprite
}

// NewStage allocates a transparent w×h stage. Negative dimensions clamp
// to zero — a stage that takes nothing and renders empty.
func NewStage(w, h int) *Stage {
	return &Stage{Board: sprite.New(w, h)}
}

// Size is the stage dimensions in terminal cells.
func (st *Stage) Size() (w, h int) {
	if st == nil {
		return 0, 0
	}
	return st.Board.Width, st.Board.Height
}

// Put writes one cell. Out of bounds is ignored.
func (st *Stage) Put(row, col int, c sprite.Cell) {
	if st == nil {
		return
	}
	st.Board.Set(row, col, c)
}

// Blit composes a sprite with its top-left at (row, col). Transparent
// cells do not overwrite what is already there; anything past an edge
// is clipped, never wrapped.
func (st *Stage) Blit(row, col int, sp sprite.Sprite) {
	if st == nil {
		return
	}
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			cell := sp.At(r, c)
			if cell.Transparent() {
				continue
			}
			st.Board.Set(row+r, col+c, cell)
		}
	}
}

// Render is the ANSI view of the stage.
func (st *Stage) Render() string {
	if st == nil {
		return ""
	}
	return sprite.Render(st.Board)
}
