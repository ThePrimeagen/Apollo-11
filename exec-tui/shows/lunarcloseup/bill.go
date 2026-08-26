// Package lunarcloseup is the lunar lander close-up screenplay, a
// composable one-scene bill — a copy of the premiere's arrival. Scene
// one, "Lunar Lander Close-Up": three seconds of drifting sky, then
// the zoomed-in Apollo craft slides in from the right wing over a
// starfield that translates with it — every star speeds up on the
// same ease-out cubic the hull flies, so the whole scene rushes left
// as the ship comes in, then the sky settles back into its own drift
// once the craft parks. Hull only, cold engine. More scenes will join
// later; for now this is the whole show. After that there is nothing
// left — the runner ends the show.
//
// The bill is the composable unit: append it to other shows' bills
// and hand the lot to screenplay.Compose for the one big screenplay.
package lunarcloseup

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
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
	}
}
