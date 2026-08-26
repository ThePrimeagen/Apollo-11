// Package moonshow is the moon screenplay, a composable two-scene
// bill. Scene one, "the moon": the bare pixelated disc alone under a
// parked sky — nothing on stage moves. Scene two, "orbit": the wide
// dotted descent ring rises and a spaceship streaks in fast off the
// left wing, merges onto the top of the ring, and orbits clockwise,
// lap after lap, until the next cut. After that there is nothing
// left — the runner ends the show.
//
// The bill is the composable unit: append it to other shows' bills
// and hand the lot to screenplay.Compose for the one big screenplay.
package moonshow

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Bill is the moon screenplay, in playing order. Each scene's cast is
// assembled when its curtain rises, not before.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "the moon", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Still(),
					moon.New().Bare(),
				}
			},
		}},
		screenplay.Entry{Name: "orbit", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Still(),
					moon.New().Arrive(),
				}
			},
		}},
	}
}
