// Package america is the portable patriot scene. The curtain rises on
// pure black; the full-screened American flag fades in fast — every
// cell of the stage walking its own ramp from black to the finished
// red, white and blue — and once the fade lands, the very large bald
// eagle enters off the right wing and crosses the whole stage
// leftward, the flag still flying beneath it, a shotgun component
// painted onto each talon firing the gunfire particle blast on its
// own schedule. After the flyover the flag flies alone, and the scene
// holds there until the cut. The stock show is quick — the whole beat
// lands inside six seconds — and the knobs stay live for anyone who
// wants it slower or louder.
//
// Nine live knobs retune it, the same way the landing scene tunes:
// FadeSeconds (the flag's fade-in), EagleDelay (when the eagle
// enters, measured from t=0), CrossSeconds (how long the crossing
// takes — the eagle's speed), EagleStart / EagleEnd (where the flight
// begins and ends, as fractions of the full off-right-to-off-left
// span), LeftShots / RightShots (how many shells each talon's gun
// fires across one crossing), and LeftAim / RightAim (which of the
// eight compass points each barrel faces). The runner nudges the time
// knobs 50ms, the path knobs 0.05 of the span, the shots one shell,
// the aims one compass point at a time, and s saves them to
// scenes/america/config.json. Every performer is a reusable component
// on its own: components/flag and components/armed (eagle + shotgun
// + gunfire as one performer) carry all of this as plain constructor
// knobs.
package america

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/armed"
	"github.com/theprimeagen/apollo-11/exec-tui/components/flag"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	// FadeSeconds is the stock fade-in from black. Fast on purpose:
	// two seconds from black to full color.
	FadeSeconds = 2.0

	// CrossSeconds is the stock flyover, off one wing of the stage
	// and off the other in four seconds. The stock delay starts it
	// the moment the fade lands.
	CrossSeconds = 4.0

	// StartPoint and EndPoint are the stock flight path, as
	// fractions of the full off-right-to-off-left span: the eagle
	// enters off the right wing and exits off the left.
	StartPoint = 0.0
	EndPoint   = 1.0

	// StockShots is how many shells each talon's shotgun fires
	// across one crossing.
	StockShots = 3

	// StockLeftAim and StockRightAim are the barrels' stock compass
	// points — the clean side-on frames: the leading talon rakes
	// ahead of the flight, the trailing one covers the rear.
	StockLeftAim  = sprite.W
	StockRightAim = sprite.E
)

// Show is the America scene as a live scene: Cfg is the nine knobs
// Assemble reads on each Start, so Play (Stop then Start) rebuilds
// the fade, the flyover and the guns from whatever they hold now.
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
		armed.New().Delay(s.Cfg.EagleDelay).Cross(s.Cfg.CrossSeconds).
			Path(s.Cfg.EagleStart, s.Cfg.EagleEnd).
			LeftEven(s.Cfg.LeftAim, s.Cfg.LeftShots).
			RightEven(s.Cfg.RightAim, s.Cfg.RightShots),
	}
}

// Bill is America as a one-scene screenplay, handy for the standalone
// runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "America", Scene: New()},
	}
}
