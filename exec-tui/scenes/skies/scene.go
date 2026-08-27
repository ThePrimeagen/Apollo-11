// Package skies is the portable blue-sky scene. The curtain rises on
// almost-pure light blue; over RiseSeconds the camera tilts up so the
// darker blue and the generated clouds come into view; then the eagle
// flies in from the right to its end point and the shotgun in each
// talon fires — after the bird is on stage, each gun on its own shot
// count and rate of fire. Every performer is a reusable component:
// components/sky, components/cloud, components/eagle, components/shotgun.
package skies

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/cloud"
	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sky"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the Skies scene as a live scene: Cfg is the knobs Assemble
// reads on each Start, so Play (Stop then Start) rebuilds the rise,
// the flyover and the guns from whatever they hold now.
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
	bird := eagle.New().Delay(s.Cfg.EagleDelay).Cross(s.Cfg.CrossSeconds).
		Path(s.Cfg.EagleStart, s.Cfg.EagleEnd)
	talons := eagle.Talons()
	return []screenplay.Component{
		sky.New().Rise(s.Cfg.RiseSeconds),
		cloud.NewField(11).Rise(s.Cfg.RiseSeconds),
		bird,
		newGunner(bird, talons[0], s.Cfg.LeftAim, s.Cfg.LeftShots, s.Cfg.LeftRate),
		newGunner(bird, talons[1], s.Cfg.RightAim, s.Cfg.RightShots, s.Cfg.RightRate),
	}
}

func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "Skies", Scene: New()},
	}
}
