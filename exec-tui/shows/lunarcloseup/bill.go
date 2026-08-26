// Package lunarcloseup is 02. Walkthrough, a composable five-scene
// bill. Scene one, "pause": the drifting sky alone — a blank stage
// the audience sits on for as long as it likes; only the cut moves
// the show along. Scene two, "Lunar Lander Close-Up": the zoomed-in
// Apollo craft slides in from the right the moment the curtain rises,
// hull only, cold engine. Scene three, "fire": the parked craft
// lights the booster and the stars slow by 60% over five seconds.
// Scene four, "fall": the north-facing lander, fire down, drops from
// the top of the stage to the bottom. Scene five, "landing": a huge
// moon horizon painted as a colored floor (five rows high in the
// middle, one row at the edges) and the north-facing lander coming
// down onto it. The fall eases out — fast off the top, then a long
// crawl that clinks onto the pad. The booster stays full until the
// last three seconds, then steps ¾, ½, ¼, and cuts off on the pad.
// The pad answers twice: the moment the booster starts slowing the
// craft, and again at touchdown, mirrored dust kicks blow out of the
// surface on both sides of the bell — leftward and rightward,
// climbing away at a shallow angle — and each kick counts its
// particles down from the full cloud to nothing over two seconds.
// After that there is nothing left — the runner ends the show.
//
// One stars.Continuity seeds every scene's sky, so a cut never jumps
// or skips a single star: each new starfield opens on the exact frame
// the last one left on screen, and the landing's still sky freezes
// right there.
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

// Bill is the walkthrough screenplay, in playing order. Each scene's
// cast is assembled when its curtain rises, not before; the shared
// continuity carries the sky across every cut.
func Bill() screenplay.Bill {
	sky := stars.NewContinuity()
	return screenplay.Bill{
		screenplay.Entry{Name: "pause", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Seed(sky),
				}
			},
		}},
		screenplay.Entry{Name: "Lunar Lander Close-Up", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Seed(sky).SlideIn(lander.FlyInSeconds, lander.BodyCols),
					lander.NewShip(11).Dark(),
				}
			},
		}},
		screenplay.Entry{Name: "fire", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Seed(sky).Slow(0.6, 5),
					lander.NewShip(11).Parked(),
				}
			},
		}},
		screenplay.Entry{Name: "fall", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Seed(sky),
					lander.NewShip(11).North().Drop(lander.DropSeconds),
				}
			},
		}},
		screenplay.Entry{Name: "landing", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Seed(sky).Still(),
					moon.NewHorizon(),
					lander.NewShip(11).North().Land(lander.LandSeconds),
				}
			},
		}},
	}
}
