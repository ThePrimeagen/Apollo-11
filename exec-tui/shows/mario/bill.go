// Package mario is 03. Mario, a composable three-scene bill: the
// astronaut's flagpole run. Scene one, "run": he sprints in from the
// left wing and climbs three crate stacks — one, two, three high —
// then holds on the top crate. Scene two, "flagpole": the leap onto
// the gold ball, a beat at the top, the slide down while the flag
// appears at the base and flies up past him, then the bow. Scene
// three, "board": the camera pans right to the lunar module, he runs
// over, jumps the hatch, and vanishes; the empty pad holds. After
// that there is nothing left — the runner ends the show.
//
// The moonwalk scene plays all three beats. The bill is the
// composable unit: append it to other shows' bills and hand the lot
// to screenplay.Compose for the one big screenplay.
package mario

import (
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Bill is the Mario screenplay, in playing order. Each scene's cast
// is assembled when its curtain rises, not before.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "run", Scene: moonwalk.New(moonwalk.BeatRun)},
		screenplay.Entry{Name: "flagpole", Scene: moonwalk.New(moonwalk.BeatPole)},
		screenplay.Entry{Name: "board", Scene: moonwalk.New(moonwalk.BeatBoard)},
	}
}
