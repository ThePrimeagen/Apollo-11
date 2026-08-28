// Package landing is the portable landing scene: a huge moon horizon
// painted as a colored floor and the north-facing lander coming down
// onto it. The fall eases out — fast off the top, then a long crawl
// that clinks onto the pad. The booster stays full until Fire75, then
// steps ¾, ½, ¼ at the fire knobs, and cuts off at FireOff. Pad dust
// is timed on its own: it starts at DustStart and blows for DustRun,
// independent of the booster. Once the engines start cutting, the
// cloud drains at DustLoss specks per millisecond instead of blinking
// out.
//
// Fourteen live knobs retune the scene. Time knobs move 50ms at a time;
// dust loss moves 0.005/ms. s saves them to scenes/landing/config.json.
// 02. Walkthrough plays that same file.
package landing

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/caption"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/shootingstar"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	// LandSeconds is how long the craft takes to come down from off
	// the top onto the pad. Smaller is a faster landing.
	LandSeconds = 5.0

	// StageSeconds is how long each booster step (¾, ½, ¼) lasts
	// before the next cut. Three stages, then off on the pad.
	StageSeconds = 0.4

	// DustStart is when the pad cloud kicks, measured from t=0.
	// Stock is the first booster step-down (LandSeconds minus three
	// stages).
	DustStart = LandSeconds - 3*StageSeconds

	// DustRun is how long the pad cloud keeps emitting after it
	// starts. Stock covers the throttle window plus the old two-
	// second linger after booster-off.
	DustRun = 3*StageSeconds + 2.0

	// Fire75/Fire50/Fire25/FireOff are when each booster step
	// kicks, measured from t=0. Stock matches the old equal
	// stages: ¾ at first step-down, then ½, ¼, off on the pad.
	Fire75  = LandSeconds - 3*StageSeconds
	Fire50  = LandSeconds - 2*StageSeconds
	Fire25  = LandSeconds - 1*StageSeconds
	FireOff = LandSeconds

	// DustLoss is how many pad specks leave per millisecond once
	// the engines start cutting. Stock matches the dust component's
	// LossPerMs (50 specks a second).
	DustLoss = 0.05

	// Code1At / Code1Hold is the first 1202, Code2At / Code2Hold the
	// second — both right before touchdown. LandCaptionAt / Hold is
	// LAND on the pad. Stock order is 1202, 1202, LAND.
	Code1At         = LandSeconds - 1.80
	Code1Hold       = 0.80
	Code2At         = LandSeconds - 0.90
	Code2Hold       = 0.70
	LandCaptionAt   = LandSeconds
	LandCaptionHold = 3.0
)

// Show is the landing as a live scene: Cfg is the fourteen knobs
// Assemble reads on each Start, so Play (Stop then Start) rebuilds
// the craft from whatever they hold now.
type Show struct {
	Cfg Config
	sky *stars.Continuity
	screenplay.Ensemble
}

// New is the landing scene. A non-nil sky seeds the twinkling
// starfield so a cut into this scene opens mid-breath where the last
// scene left; a nil sky is a fresh sky of its own.
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
	ship := lander.NewShip(11).North().
		Land(s.Cfg.LandSeconds).
		ThrottleAt(s.Cfg.Fire75, s.Cfg.Fire50, s.Cfg.Fire25, s.Cfg.FireOff).
		DustAt(s.Cfg.DustStart, s.Cfg.DustRun).
		DustLoss(s.Cfg.DustLoss)
	cast := []screenplay.Component{
		field,
		shootingstar.NewMeteor(),
		&hullMask{ship: ship, land: s.Cfg.LandSeconds},
		moon.NewHorizon(),
		ship,
	}
	if board := caption.New(
		caption.Cue{Text: "1202", At: s.Cfg.Code1At, Hold: s.Cfg.Code1Hold},
		caption.Cue{Text: "1202", At: s.Cfg.Code2At, Hold: s.Cfg.Code2Hold},
		caption.Cue{Text: "LAND", At: s.Cfg.LandCaptionAt, Hold: s.Cfg.LandCaptionHold},
	); board != nil {
		cast = append(cast, board)
	}
	return cast
}

// hullMask paints an opaque pad the size of the LM bounding box so a
// meteor crossing the craft sits behind it, even in the hull's
// transparent gaps.
type hullMask struct {
	ship *lander.Ship
	land float64
	w, h int
}

func (m *hullMask) Start(w, h int) {
	if m == nil {
		return
	}
	m.w, m.h = w, h
}

func (m *hullMask) Update(dt float64) {}

func (m *hullMask) Render() sprite.Sprite {
	if m == nil || m.ship == nil || m.w < 1 || m.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(m.w, m.h)
	row, col := lander.LandPath(m.w, m.h, m.ship.Clock(), m.land)
	for r := 0; r < lander.BodyRows; r++ {
		for c := 0; c < lander.BodyCols; c++ {
			stage.Set(row+r, col+c, sprite.Cell{Ch: ' ', FG: 0, BG: 0})
		}
	}
	return stage
}

func (m *hullMask) Stop() {}

// Bill is the landing as a one-scene screenplay, handy for the
// standalone runner. After it there is nothing left.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "landing", Scene: New(nil)},
	}
}
