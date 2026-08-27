// Package america is the portable patriot scene. The curtain rises on
// pure black; the full-screened American flag fades in slowly — every
// cell of the stage walking its own ramp from black to the finished
// red, white and blue over FadeSeconds — and only once the flag is
// fully in does the very large bald eagle enter off the right wing
// and cross the whole stage leftward over CrossSeconds, the flag
// still flying beneath it. After the flyover the flag flies alone,
// and the scene holds there until the cut.
package america

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/flag"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	// FadeSeconds is how long the flag takes to fade in from black.
	// Slow on purpose: the reveal is the scene.
	FadeSeconds = 8.0

	// CrossSeconds is the eagle's flyover, off one wing of the stage
	// and off the other. It starts the moment the fade lands.
	CrossSeconds = 12.0
)

// Show is the America scene: the fading flag below, the crossing
// eagle above, assembled fresh at every curtain so a replay starts
// from black again.
type Show struct {
	screenplay.Ensemble
}

// New is the America scene, ready for its curtain.
func New() *Show {
	s := &Show{}
	s.Assemble = func() []screenplay.Component {
		return []screenplay.Component{
			flag.New(FadeSeconds),
			eagle.New().Delay(FadeSeconds).Cross(CrossSeconds),
		}
	}
	return s
}

// Bill is America as a one-scene screenplay, handy for the standalone
// runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "America", Scene: New()},
	}
}
