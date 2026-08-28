// Package prog is the portable program-alarm drop: the north-facing
// lander falling under a twinkling sky and pausing three times —
// 1202 on the right, then 1202 again, then 1201. That is the first
// three flight alarms (two 1202s in P63, then the 1201 in P64). The
// last drop carries the craft off the bottom. Seven live knobs
// retune the four drops and three holds 50ms at a time. s saves
// them to scenes/prog/config.json.
package prog

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/caption"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the prog as a live scene: Cfg is the seven knobs Assemble
// reads on each Start, so Play (Stop then Start) rebuilds the craft
// from whatever they hold now.
type Show struct {
	Cfg Config
	sky *stars.Continuity
	screenplay.Ensemble
}

// New is the prog scene. A non-nil sky seeds the twinkling starfield
// so a cut into this scene opens mid-breath where the last scene
// left; a nil sky is a fresh sky of its own.
func New(sky *stars.Continuity) *Show {
	s := &Show{Cfg: Active(), sky: sky}
	s.Assemble = s.assemble
	return s
}

func (s *Show) assemble() []screenplay.Component {
	field := stars.NewStarfield(stars.Twinkle)
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	cast := []screenplay.Component{
		field,
		lander.NewShip(11).North().DropBeats(s.Cfg.Beats()),
	}
	if board := caption.New(s.Cfg.cues()...); board != nil {
		cast = append(cast, board)
	}
	return cast
}

func (c Config) cues() []caption.Cue {
	codes := Codes()
	at1 := c.Drop1
	at2 := at1 + c.Hold1 + c.Drop2
	at3 := at2 + c.Hold2 + c.Drop3
	return []caption.Cue{
		{Text: codes[0], At: at1, Hold: c.Hold1},
		{Text: codes[1], At: at2, Hold: c.Hold2},
		{Text: codes[2], At: at3, Hold: c.Hold3},
	}
}

// Bill is the prog as a one-scene screenplay, handy for the
// standalone runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "prog", Scene: New(nil)},
	}
}
