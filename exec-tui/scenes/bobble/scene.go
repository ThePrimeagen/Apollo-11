// Package bobble is the portable bobble scene: the west-facing lander
// alone, parked at center stage under the drifting sky, bobbling up
// and down on a sine — with or without its engine on. It is the
// reusable middle of two shows: 02. Walkthrough plays this state
// engine off and then engine on across its cuts; 04. Inverse
// Walkthrough plays it engine on ("engines on") and then engine off
// ("engines off"), holding the dark ride ad infinitum. Each flip is a
// cut on the bill, not a timer in here.
//
// Three live knobs retune the scene: the engine (l on, h off), the
// ride's period (±50ms), and its amplitude (±1 cell) — throb or
// undulate, whichever way the operator sets it. s saves them to
// scenes/bobble/config.json.
package bobble

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the bobble as a live scene: Cfg is the three knobs Assemble
// reads on each Start, so Play (Stop then Start) rebuilds the ride
// from whatever they hold now.
type Show struct {
	Cfg Config
	sky *stars.Continuity
	screenplay.Ensemble
}

// New is the bobble scene. A non-nil sky seeds the drifting starfield
// so a cut into this scene opens on the frame the last scene left; a
// nil sky is a fresh sky of its own.
func New(sky *stars.Continuity) *Show {
	s := &Show{Cfg: Active(), sky: sky}
	s.Assemble = s.assemble
	return s
}

// Lit burns the tail fire, whatever the active config says — the
// bill's word wins. Nil-safe.
func (s *Show) Lit() *Show {
	if s == nil {
		return nil
	}
	s.Cfg.Engine = true
	return s
}

// Dark flies the hull cold, whatever the active config says — the
// bill's word wins. Nil-safe.
func (s *Show) Dark() *Show {
	if s == nil {
		return nil
	}
	s.Cfg.Engine = false
	return s
}

func (s *Show) assemble() []screenplay.Component {
	field := stars.NewTunedStarfield()
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	ship := lander.NewShip(11).Parked().
		Bob(s.Cfg.PeriodSeconds, s.Cfg.AmplitudeCells)
	if !s.Cfg.Engine {
		ship = ship.Dark()
	}
	return []screenplay.Component{
		field,
		ship,
	}
}

// Bill is the bobble as a one-scene screenplay, handy for the
// standalone runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "bobble", Scene: New(nil)},
	}
}
