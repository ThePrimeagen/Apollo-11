// Package landing is the portable landing scene: a huge moon horizon
// painted as a colored floor and the north-facing lander coming down
// onto it. The fall eases out — fast off the top, then a long crawl
// that clinks onto the pad. The booster stays full until the last
// three stages, then steps ¾, ½, ¼, and cuts off on the pad. Pad dust
// is timed on its own: it starts at DustStart and blows for DustRun,
// independent of the booster.
//
// Three live knobs retune the scene 50ms at a time. s saves them to
// scenes/landing/config.json. 02. Walkthrough plays that same file.
package landing

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	// LandSeconds is how long the craft takes to come down from off
	// the top onto the pad. Smaller is a faster landing.
	LandSeconds = 5.0

	// StageSeconds is how long each booster step (¾, ½, ¼) lasts
	// before the next cut. Three stages, then off on the pad.
	StageSeconds = 0.4

	// DustStart is when the pad cloud kicks, measured from t=0.
	// Stock is the first booster step-down (LandSeconds minus three
	// stages).
	DustStart = LandSeconds - 3*StageSeconds

	// DustRun is how long the pad cloud keeps emitting after it
	// starts. Stock covers the throttle window plus the old two-
	// second linger after booster-off.
	DustRun = 3*StageSeconds + 2.0
)

// Show is the landing as a live scene: Cfg is the three knobs
// Assemble reads on each Start, so Play (Stop then Start) rebuilds
// the craft from whatever they hold now.
type Show struct {
	Cfg Config
	sky *stars.Continuity
	screenplay.Ensemble
}

// New is the landing scene. A non-nil sky seeds the still starfield
// so a cut into this scene freezes on the frame the last scene left;
// a nil sky is a fresh parked sky of its own.
func New(sky *stars.Continuity) *Show {
	s := &Show{Cfg: Active(), sky: sky}
	s.Assemble = s.assemble
	return s
}

func (s *Show) assemble() []screenplay.Component {
	field := stars.NewTunedStarfield()
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	return []screenplay.Component{
		field.Still(),
		moon.NewHorizon(),
		lander.NewShip(11).North().
			Land(s.Cfg.LandSeconds).
			ThrottleStage(StageSeconds).
			DustAt(s.Cfg.DustStart, s.Cfg.DustRun),
	}
}

// Bill is the landing as a one-scene screenplay, handy for the
// standalone runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "landing", Scene: New(nil)},
	}
}
