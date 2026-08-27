// Package skies is the portable blue-sky scene. The curtain rises on
// almost-pure light blue; over RiseSeconds the camera tilts up so the
// darker blue and the generated clouds come into view; then the
// American flag crossfades in as the new floor — background coloring,
// so the eagle and the talon shotguns sit on top. The bird flies in
// from the right to its end point and the shotgun in each talon
// fires — after the bird is on stage, each gun on its own shot count
// and rate of fire, unless that talon is switched off. Every
// performer is a reusable component:
// components/sky, components/cloud, components/flag,
// components/transition, components/armed.
package skies

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/armed"
	"github.com/theprimeagen/apollo-11/exec-tui/components/cloud"
	"github.com/theprimeagen/apollo-11/exec-tui/components/flag"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sky"
	"github.com/theprimeagen/apollo-11/exec-tui/components/transition"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the Skies scene as a live scene: Cfg is the knobs Assemble
// reads on each Start, so Play (Stop then Start) rebuilds the rise,
// the flag walk, the flyover and the guns from whatever they hold now.
type Show struct {
	Cfg Config
	screenplay.Ensemble
}

func New() *Show {
	s := &Show{Cfg: Active()}
	s.Assemble = s.assemble
	return s
}

func (s *Show) assemble() []screenplay.Component {
	floor := transition.Between(
		transition.Stack(
			sky.New().Rise(s.Cfg.RiseSeconds),
			cloud.NewField(11).Rise(s.Cfg.RiseSeconds),
		),
		flag.New(0),
	).Delay(s.Cfg.FlagDelay).Over(s.Cfg.FlagFade)
	bird := armed.New().Delay(s.Cfg.EagleDelay).Cross(s.Cfg.CrossSeconds).
		Path(s.Cfg.EagleStart, s.Cfg.EagleEnd)
	if s.Cfg.LeftOn {
		bird.LeftGun(s.Cfg.LeftAim, s.Cfg.LeftShots, s.Cfg.LeftRate)
	} else {
		bird.UnmountLeft()
	}
	if s.Cfg.RightOn {
		bird.RightGun(s.Cfg.RightAim, s.Cfg.RightShots, s.Cfg.RightRate)
	} else {
		bird.UnmountRight()
	}
	return []screenplay.Component{
		floor,
		bird,
	}
}

func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "Skies", Scene: New()},
	}
}
