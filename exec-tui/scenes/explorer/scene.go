// Package explorer is the big Internet Explorer logo scene: the blue
// e under its golden swoosh, moon-sized, parked at center stage under
// a twinkling sky. The stars fly the twinkle mode — the sky holds
// where it scattered and some stars fade in and out on the four knobs
// the scene's config carries: the cycle range and the fade range,
// each a min and a max in seconds.
//
// Four live knobs retune the scene: min/max cycle (±250ms) and
// min/max fade (±50ms). The standalone runner walks them live —
// the sky reads the active twinkle every frame — and s saves them to
// scenes/explorer/config.json.
package explorer

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/ie"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the explorer as a live scene: Cfg is the four knobs
// Assemble pushes onto the sky on each Start, so Play (Stop then
// Start) rebuilds the breathing from whatever they hold now.
type Show struct {
	Cfg Config
	sky *stars.Continuity
	screenplay.Ensemble
}

// New is the explorer scene. A non-nil sky seeds the twinkling
// starfield's clock so a cut into this scene opens mid-breath where
// the last scene left; a nil sky is a fresh sky of its own.
func New(sky *stars.Continuity) *Show {
	s := &Show{Cfg: Active(), sky: sky}
	s.Assemble = s.assemble
	return s
}

func (s *Show) assemble() []screenplay.Component {
	// Broken knobs are refused and the sky keeps its last good
	// breath — the show goes on either way.
	_ = stars.UseTwinkle(s.Cfg.Twinkle())
	field := stars.NewStarfield(stars.Twinkle)
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	return []screenplay.Component{
		field,
		ie.NewBig(),
	}
}

// Bill is the explorer as a one-scene screenplay, handy for the
// standalone runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "explorer", Scene: New(nil)},
	}
}
