// Package inverse is 03. Inverse Walkthrough, a composable
// three-scene bill: the walkthrough played backwards. Scene one,
// "liftoff": the lander parked on the huge moon horizon ignites (¼,
// ½, ¾, full — the landing throttle run backwards), the pad blows its
// mirrored dust cloud, and the craft climbs off the top on the
// landing's mirrored ease; the empty moon then holds for the cut.
// Scene two, "engines on": the west-facing craft parked at center
// stage, tail fire burning, bobbling on its sine. Scene three,
// "engines off": the very same bobble scene, engine out — it bobbles
// ad infinitum, and only the cut ends the show.
//
// The bobble scene is the walkthrough's parked craft made reusable:
// 02 plays that state engine off and then engine on across its cuts;
// this bill plays it engine on and then engine off — the same scene,
// twice, on two different screenplays. One stars.Continuity seeds
// every scene's sky, so a cut never jumps or skips a single star: the
// still liftoff sky hands the drifting bobble sky the exact frame it
// held.
//
// The bill is the composable unit: append it to other shows' bills
// and hand the lot to screenplay.Compose for the one big screenplay.
package inverse

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Bill is the inverse walkthrough, in playing order. Each scene's
// cast is assembled when its curtain rises, not before; the shared
// continuity carries the sky across every cut.
func Bill() screenplay.Bill {
	sky := stars.NewContinuity()
	return screenplay.Bill{
		screenplay.Entry{Name: "liftoff", Scene: liftoff.New(sky)},
		screenplay.Entry{Name: "engines on", Scene: bobble.New(sky).Lit()},
		screenplay.Entry{Name: "engines off", Scene: bobble.New(sky).Dark()},
	}
}
