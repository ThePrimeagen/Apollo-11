// Package fall is the portable spacelander fall: the north-facing
// lander dropping from the top of the stage to the bottom under a
// twinkling sky. One live knob retunes the drop duration 50ms at a
// time. s saves it to scenes/fall/config.json.
package fall

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the fall as a live scene: Cfg is the drop knob Assemble
// reads on each Start, so Play (Stop then Start) rebuilds the craft
// from whatever it holds now.
type Show struct {
	Cfg Config
	sky *stars.Continuity
	screenplay.Ensemble
}

// New is the fall scene. A non-nil sky seeds the twinkling starfield
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
		lander.NewShip(11).North().Drop(s.Cfg.DropSeconds),
	}
}

// Bill is the fall as a one-scene screenplay, handy for the
// standalone runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "fall", Scene: New(nil)},
	}
}
