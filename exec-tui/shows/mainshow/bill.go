// Package mainshow is 05. Main — the one that puts everything
// together. Its bill is every numbered show's bill added together, in
// shelf order: 01. Moon Orbit (the bare moon, then the fly-in to
// orbit), 02. Walkthrough (pause, close-up, fire, fall, landing on
// the huge horizon), 03. Mario (run, flagpole, board), then 04.
// Inverse Walkthrough (liftoff, engines on, engines off) — thirteen
// scenes. Every entry is the same performer its home show casts, so
// the editor can reach the knobbed Shows and the bobble keeps the
// bill's word on the engine.
//
// The bill is the composable unit: this package adds nothing of its
// own — it is Compose's argument list written down, plus the home of
// the editor's hold file.
package mainshow

import (
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/inverse"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/lunarcloseup"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/mario"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/moonshow"
)

const (
	// Title is the marquee name of the whole show.
	Title = "MAIN"

	// HoldsPath is the editor's hold file — how long each scene
	// plays in play mode before the cut — relative to the module
	// root, beside this bill.
	HoldsPath = "shows/mainshow/config.json"
)

// Bill is MAIN, in playing order: every numbered show's bill, added
// together. Each call builds a fresh cast.
func Bill() screenplay.Bill {
	var all screenplay.Bill
	for _, b := range []screenplay.Bill{
		moonshow.Bill(),
		lunarcloseup.Bill(),
		mario.Bill(),
		inverse.Bill(),
	} {
		all = append(all, b...)
	}
	return all
}
