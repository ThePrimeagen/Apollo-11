// Package america is the portable patriot scene. The curtain rises on
// pure black; the full-screened American flag fades in slowly — every
// cell of the stage walking its own ramp from black to the finished
// red, white and blue — and once the fade lands, the very large bald
// eagle enters off the right wing and crosses the whole stage
// leftward, the flag still flying beneath it. After the flyover the
// flag flies alone, and the scene holds there until the cut.
//
// Three live knobs retune it, the same way the landing scene tunes:
// FadeSeconds (the flag's fade-in), EagleDelay (when the eagle
// enters, measured from t=0), and CrossSeconds (how long the crossing
// takes — the eagle's speed). The runner nudges them 50ms at a time
// and s saves them to scenes/america/config.json. Both performers are
// reusable components on their own: components/flag and
// components/eagle carry all of this as plain constructor knobs.
package america

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/flag"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	// FadeSeconds is the stock fade-in from black. Slow on purpose:
	// the reveal is the scene.
	FadeSeconds = 8.0

	// CrossSeconds is the stock flyover, off one wing of the stage
	// and off the other. The stock delay starts it the moment the
	// fade lands.
	CrossSeconds = 12.0
)

// Show is the America scene as a live scene: Cfg is the three knobs
// Assemble reads on each Start, so Play (Stop then Start) rebuilds
// the fade and the flyover from whatever they hold now.
type Show struct {
	Cfg Config
	screenplay.Ensemble
}

// New is the America scene, playing the Active knobs, ready for its
// curtain.
func New() *Show {
	s := &Show{Cfg: Active()}
	s.Assemble = s.assemble
	return s
}

func (s *Show) assemble() []screenplay.Component {
	return []screenplay.Component{
		flag.New(s.Cfg.FadeSeconds),
		eagle.New().Delay(s.Cfg.EagleDelay).Cross(s.Cfg.CrossSeconds),
	}
}

// Bill is America as a one-scene screenplay, handy for the standalone
// runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "America", Scene: New()},
	}
}
