// Package liftoff is 03. Inverse Walkthrough — 02. Walkthrough played
// backwards inside one portable scene. The curtain rises on the
// landing's final frame: the north-facing lander parked on the huge
// moon horizon, engine cold, under a still sky. The booster ignites
// and throttles up (¼, ½, ¾, full — the landing throttle run
// backwards), the pad blows its mirrored dust cloud, and at LiftAt
// the craft climbs on the landing's mirrored ease — a slow, heavy
// crawl off the pad that rockets off the top. The moment the hull
// clears the stage the scene cuts, exactly like the walkthrough's own
// cuts: the horizon is gone and the tilted-sideways west craft is
// revealed parked at center stage, tail fire on, under the very same
// sky. FireOff seconds later the fire cuts out, and the craft bobbles
// on the parked sine ad infinitum — the scene never ends on its own.
//
// Ten live knobs retune the scene: rise, lift at, the four ignition
// offsets, fire off, and the dust window. Time knobs move 50ms at a
// time; dust loss moves 0.005/ms. s saves them to
// scenes/liftoff/config.json.
package liftoff

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the inverse walkthrough as a live scene: Cfg is the ten
// knobs Start snapshots, so Play (Stop then Start) rebuilds the
// liftoff from whatever they hold now — and a mid-flight nudge never
// moves a cut already in the air. Inside, the show is its own tiny
// two-scene screenplay — the ground act and the space act — with the
// cut pulled by the clock instead of the spacebar.
type Show struct {
	Cfg      Config
	sky      *stars.Continuity
	play     *screenplay.Screenplay
	cutAt    float64
	clock    float64
	cut      bool
	rendered bool
}

// New is the inverse walkthrough scene. A non-nil sky seeds the still
// starfield so a cut into this scene freezes on the frame the last
// scene left; a nil sky is a fresh parked sky of its own.
func New(sky *stars.Continuity) *Show {
	return &Show{Cfg: Active(), sky: sky}
}

// seeded is a still tuned starfield on the show's continuity — the
// one sky both acts share, so the internal cut never moves a star.
func (s *Show) seeded() *stars.Starfield {
	field := stars.NewTunedStarfield()
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	return field.Still()
}

// Start snapshots the knobs and builds the two acts. The ground act:
// the horizon and the north craft igniting and lifting off. The space
// act: the tilted-sideways craft parked with its tail fire on for
// FireOff seconds, bobbling forever. Nothing plays until the first
// render — the stage size arrives there.
func (s *Show) Start() {
	if s == nil {
		return
	}
	cfg := s.Cfg
	if s.sky == nil {
		s.sky = stars.NewContinuity()
	}
	ground := &screenplay.Ensemble{Assemble: func() []screenplay.Component {
		return []screenplay.Component{
			s.seeded(),
			moon.NewHorizon(),
			lander.NewShip(11).North().
				Lift(cfg.LiftAt, cfg.RiseSeconds).
				IgniteAt(cfg.Fire25, cfg.Fire50, cfg.Fire75, cfg.FireFull).
				DustAt(cfg.DustStart, cfg.DustRun).
				DustLoss(cfg.DustLoss),
		}
	}}
	space := &screenplay.Ensemble{Assemble: func() []screenplay.Component {
		return []screenplay.Component{
			s.seeded(),
			lander.NewShip(11).Parked().FireFor(cfg.FireOff),
		}
	}}
	s.play = screenplay.New(
		screenplay.Entry{Name: "liftoff", Scene: ground},
		screenplay.Entry{Name: "aloft", Scene: space},
	)
	s.cutAt = cfg.CutSeconds()
	s.clock = 0
	s.cut = false
	s.rendered = false
	s.play.Start()
}

// Update advances the act now playing and pulls the cut the moment
// the climb has cleared the stage. Time before the first render holds
// the curtain — components only start when the stage size is known —
// and dt <= 0 holds everything.
func (s *Show) Update(dt float64) {
	if s == nil || s.play == nil || !s.rendered || dt <= 0 {
		return
	}
	s.play.Update(dt)
	s.clock += dt
	if !s.cut && s.clock >= s.cutAt {
		s.play.Next()
		s.cut = true
	}
}

// Render paints the act now playing onto the screen.
func (s *Show) Render(scr *screenplay.Screen) {
	if s == nil || s.play == nil || scr == nil {
		return
	}
	s.rendered = true
	s.play.Render(scr)
}

// Stop brings the whole show down and drops it for the collector; a
// fresh Start rebuilds both acts from the knobs of that moment.
func (s *Show) Stop() {
	if s == nil || s.play == nil {
		return
	}
	s.play.Stop()
	s.play = nil
}

// Bill is the inverse walkthrough as a one-scene screenplay, handy
// for the standalone runner. After it there is nothing left — the
// scene itself simply never ends.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "inverse walkthrough", Scene: New(nil)},
	}
}
