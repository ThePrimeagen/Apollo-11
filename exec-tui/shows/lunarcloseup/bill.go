// Package lunarcloseup is the lunar lander close-up screenplay, a
// composable four-scene bill. Scene one, "Lunar Lander Close-Up": a
// copy of the premiere's arrival — three seconds of drifting sky,
// then the zoomed-in Apollo craft slides in from the right, hull
// only, cold engine. Scene two, "fire": the parked craft lights the
// booster and the stars slow by 60% over five seconds. Scene three,
// "fall": the north-facing lander, fire down, drops from the top of
// the stage to the bottom. Scene four, "landing": a huge moon
// horizon (five rows high in the middle, one row at the edges) and
// the north-facing lander coming down onto it. After that there is
// nothing left — the runner ends the show.
//
// The bill is the composable unit: append it to other shows' bills
// and hand the lot to screenplay.Compose for the one big screenplay.
package lunarcloseup

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Bill is the lunar lander close-up screenplay, in playing order.
// Each scene's cast is assembled when its curtain rises, not before.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "Lunar Lander Close-Up", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().SlideIn(lander.FlyInSeconds, lander.BodyCols).Hold(lander.FlyInHoldSeconds),
					lander.NewShip(11).Dark().Hold(lander.FlyInHoldSeconds),
				}
			},
		}},
		screenplay.Entry{Name: "fire", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Slow(0.6, 5),
					lander.NewShip(11).Parked(),
				}
			},
		}},
		screenplay.Entry{Name: "fall", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield(),
					lander.NewShip(11).North().Drop(lander.DropSeconds),
				}
			},
		}},
		screenplay.Entry{Name: "landing", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Still(),
					moon.NewHorizon(),
					lander.NewShip(11).North().Land(lander.LandSeconds),
				}
			},
		}},
	}
}
