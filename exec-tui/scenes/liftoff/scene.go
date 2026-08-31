// Package liftoff is the portable liftoff scene: the landing played
// backwards, and nothing more. The curtain rises on the landing's
// final frame — the north-facing lander parked on the huge moon
// horizon, engine cold, under a still sky. The booster ignites and
// throttles up (¼, ½, ¾, full — the landing throttle run backwards),
// the pad blows its mirrored dust cloud, and at LiftAt the craft
// climbs on the landing's mirrored ease: a slow, heavy crawl off the
// pad that rockets off the top, hull and booster fire both gone.
// Then the scene simply holds the empty
// moon until the screenplay cuts away — on 04. Inverse Walkthrough's
// bill the next entry is the bobble scene, engines on.
//
// Nine live knobs retune the scene: rise, lift at, the four ignition
// offsets, and the dust window. Time knobs move 50ms at a time; dust
// loss moves 0.005/ms. s saves them to scenes/liftoff/config.json.
package liftoff

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the liftoff as a live scene: Cfg is the nine knobs Assemble
// reads on each Start, so Play (Stop then Start) rebuilds the craft
// from whatever they hold now.
type Show struct {
	Cfg Config
	sky *stars.Continuity
	screenplay.Ensemble
}

// New is the liftoff scene. A non-nil sky seeds the still starfield
// so a cut into this scene freezes on the frame the last scene left;
// a nil sky is a fresh parked sky of its own.
func New(sky *stars.Continuity) *Show {
	s := &Show{Cfg: Active(), sky: sky}
	s.Assemble = s.assemble
	return s
}

func (s *Show) craft() *lander.Ship {
	ship := lander.NewShip(11).North()
	if s.Cfg.WhiteOnly {
		ship = ship.WhiteOnly()
	}
	return ship.
		Lift(s.Cfg.LiftAt, s.Cfg.RiseSeconds).
		IgniteAt(s.Cfg.Fire25, s.Cfg.Fire50, s.Cfg.Fire75, s.Cfg.FireFull).
		DustAt(s.Cfg.DustStart, s.Cfg.DustRun).
		DustLoss(s.Cfg.DustLoss)
}

func (s *Show) assemble() []screenplay.Component {
	field := stars.NewTunedStarfield()
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	return []screenplay.Component{
		field.Still(),
		moon.NewHorizon(),
		s.craft(),
	}
}

// Bill is the liftoff as a one-scene screenplay, handy for the
// standalone runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "liftoff", Scene: New(nil)},
	}
}
