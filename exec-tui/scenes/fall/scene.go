// Package fall is the portable spacelander fall: the north-facing
// lander dropping from the top of the stage to the bottom under a
// twinkling sky. One live drop knob retunes the drop duration 50ms at
// a time. Three hold knobs, stock at zero, let MAIN pause 1202, 1202,
// then 1201 about a third of the way down. s saves it to
// scenes/fall/config.json.
package fall

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/caption"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the fall as a live scene: Cfg is the drop knob Assemble
// reads on each Start, so Play (Stop then Start) rebuilds the craft
// from whatever it holds now.
type Show struct {
	Cfg Config
	sky *stars.Continuity
	screenplay.Ensemble
}

// New is the fall scene. A non-nil sky seeds the twinkling starfield
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
	if !s.Cfg.Armed() {
		return []screenplay.Component{
			field,
			lander.NewShip(11).North().Drop(s.Cfg.DropSeconds),
		}
	}
	act := newAct(s.Cfg)
	return []screenplay.Component{
		&gate{inner: field, skip: act.Holding},
		act,
	}
}

// Bill is the fall as a one-scene screenplay, handy for the
// standalone runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "fall", Scene: New(nil)},
	}
}

// gate skips Update while skip reports a freeze — stars hold their
// last frame so only the alarm card blinks.
type gate struct {
	inner screenplay.Component
	skip  func() bool
}

func (g *gate) Start(w, h int) {
	if g != nil && g.inner != nil {
		g.inner.Start(w, h)
	}
}

func (g *gate) Update(dt float64) {
	if g == nil || g.inner == nil || (g.skip != nil && g.skip()) {
		return
	}
	g.inner.Update(dt)
}

func (g *gate) Render() sprite.Sprite {
	if g == nil || g.inner == nil {
		return sprite.Sprite{}
	}
	return g.inner.Render()
}

func (g *gate) Stop() {
	if g != nil && g.inner != nil {
		g.inner.Stop()
	}
}

// act is the MAIN-armed fall: a pausing drop, blinking caption, and
// the top-left elevation HUD.
type act struct {
	cfg   Config
	ship  *lander.Ship
	board *caption.Board
	beats []lander.DropBeat
	clock float64
	w, h  int
}

func newAct(cfg Config) *act {
	return &act{cfg: cfg}
}

func (a *act) Holding() bool {
	if a == nil || !a.cfg.Armed() {
		return false
	}
	return lander.DropBeatHold(a.clock, a.beats) >= 0
}

func (a *act) Start(w, h int) {
	if a == nil {
		return
	}
	a.w, a.h = w, h
	a.beats = AlarmBeats(h, a.cfg)
	if a.ship == nil {
		a.ship = lander.NewShip(11).North()
	}
	a.ship.DropBeats(a.beats)
	a.ship.Start(w, h)
	a.clock = a.ship.Clock()
	a.board = caption.New(a.cues()...)
	if a.board != nil {
		a.board.Start(w, h)
	}
}

func (a *act) cues() []caption.Cue {
	codes := Codes()
	at := 0.0
	out := make([]caption.Cue, 0, 3)
	for i := 0; i < 3 && i < len(a.beats); i++ {
		at += a.beats[i].Drop
		out = append(out, caption.Cue{
			Text:  codes[i],
			At:    at,
			Hold:  a.beats[i].Hold,
			Blink: AlarmBlink,
		})
		at += a.beats[i].Hold
	}
	return out
}

func (a *act) Update(dt float64) {
	if a == nil || dt <= 0 {
		return
	}
	a.clock += dt
	if a.Holding() {
		a.ship.AdvanceClock(dt)
	} else if a.ship != nil {
		a.ship.Update(dt)
	}
	if a.board != nil {
		a.board.Update(dt)
	}
}

func (a *act) Render() sprite.Sprite {
	if a == nil || a.ship == nil {
		return sprite.Sprite{}
	}
	stage := a.ship.Render()
	if a.board != nil {
		sprite.Blit(stage, 0, 0, a.board.Render())
	}
	if a.cfg.Armed() && stage.Width > 0 && stage.Height > 0 {
		row, _ := lander.DropBeatPath(a.w, a.h, a.clock, a.beats)
		paintElevation(stage, ElevationAt(row, a.h))
	}
	return stage
}

func paintElevation(stage sprite.Sprite, alt float64) {
	for i, r := range FormatElevation(alt) {
		// A space with no background is transparent and lets stars
		// break "ALT  33500ft". The HUD keeps a floor so the face
		// stays one token.
		stage.Set(0, i, sprite.Cell{Ch: r, FG: ElevInk, BG: 0})
	}
}

func (a *act) Stop() {
	if a == nil {
		return
	}
	if a.ship != nil {
		a.ship.Stop()
	}
	if a.board != nil {
		a.board.Stop()
	}
}
