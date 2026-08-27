package moonwalk

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// blockRows is one crate's height in cell rows.
const blockRows = 4

// groundRow is the ground line's stage row.
func groundRow(stageH int) int { return 0 }

// groundedY is the sprite top row that parks the boots on the floor.
func groundedY(stageH int) int { return 0 }

// poleCol is the flagpole's world column.
func poleCol(stageW int) int { return 0 }

// poleTopRow is the pole tip's stage row for a config.
func poleTopRow(cfg Config, stageH int) int { return 0 }

// blockAX is the low crate's left world column.
func blockAX(stageW int) int { return 0 }

// roverX is the rover's left world column — beyond the viewport until
// the pan.
func roverX(cfg Config, stageW int) int { return 0 }

// CycleSeconds is one full loop of the show.
func CycleSeconds(cfg Config, stageW, stageH int) float64 { return 1 }

// timelineAt is the choreography: which pose plays and where its
// sprite's top-left sits in world coordinates at t.
func timelineAt(cfg Config, stageW, stageH int, t float64) (pose sprite.Heading, x, y int) {
	return "", 0, 0
}

// cameraAt is how far the viewport has panned right at t.
func cameraAt(cfg Config, stageW, stageH int, t float64) int { return 0 }

// flagTopAt is the flag sprite's top row at t.
func flagTopAt(cfg Config, stageW, stageH int, t float64) int { return 0 }

// Frame renders the visible viewport of the show at t.
func Frame(cfg Config, atlas *sprite.Atlas, stageW, stageH int, t float64) sprite.Sprite {
	return sprite.Sprite{}
}
