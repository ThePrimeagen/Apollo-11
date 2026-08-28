// Package climb is the portable spacelander climb: the north-facing
// lander rising from the bottom of the stage to the top under a
// twinkling sky. One live knob retunes the climb duration 50ms at a
// time. s saves it to scenes/climb/config.json.
package climb

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the climb as a live scene: Cfg is the climb knob Assemble
// reads on each Start, so Play (Stop then Start) rebuilds the craft
// from whatever it holds now.
type Show struct {
	Cfg Config
	sky *stars.Continuity
	screenplay.Ensemble
}

// New is the climb scene. A non-nil sky seeds the twinkling starfield
// so a cut into this scene opens mid-breath where the last scene
// left; a nil sky is a fresh sky of its own.
func New(sky *stars.Continuity) *Show {
	s := &Show{Cfg: Active(), sky: sky}
	s.Assemble = s.assemble
	return s
}

func (s *Show) assemble() []screenplay.Component {
	field := stars.NewStarfield(stars.Twinkle)
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	return []screenplay.Component{
		field,
		lander.NewShip(11).North().Climb(s.Cfg.ClimbSeconds),
	}
}

// Bill is the climb as a one-scene screenplay, handy for the
// standalone runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "climb", Scene: New(nil)},
	}
}
