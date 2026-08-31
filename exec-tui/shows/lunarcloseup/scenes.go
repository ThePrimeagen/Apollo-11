package lunarcloseup

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Steps of the walkthrough faces' knobs: times move a quarter second,
// the brake's depth and the rush peak a twentieth.
const (
	closeupStepSeconds = 0.25
	closeupRushStep    = 0.05
	fireByStep         = 0.05
	fireOverStep       = 0.25
)

// CloseupRushPeak is the stock crest: the sky flies 25% faster than
// cruise at mid-slide, then settles so the hull holds center.
const CloseupRushPeak = 1.25

// CloseupConfig is the close-up scene's editable face: how long the
// craft's slide from the right wing takes — the sky translates on the
// same clock — and how far past cruise the sky rushes on the way in.
// The numbers are the operator's, verbatim.
type CloseupConfig struct {
	FlyInSeconds float64 `json:"flyInSeconds"`
	RushPeak     float64 `json:"rushPeak"`
}

// DefaultCloseupConfig is the stock slide and the stock 1.25 rush.
func DefaultCloseupConfig() CloseupConfig {
	return CloseupConfig{FlyInSeconds: lander.FlyInSeconds, RushPeak: CloseupRushPeak}
}

// KnobCount is how many knobs the close-up carries.
func (c CloseupConfig) KnobCount() int { return 2 }

// KnobLabel is the panel name of knob i.
func (c CloseupConfig) KnobLabel(i int) string {
	switch i {
	case 0:
		return "fly-in"
	case 1:
		return "rush"
	default:
		return ""
	}
}

// Value reads one knob for display and tests.
func (c CloseupConfig) Value(i int) float64 {
	switch i {
	case 0:
		return c.FlyInSeconds
	case 1:
		return c.RushPeak
	default:
		return 0
	}
}

// Nudge walks one knob by dir steps — the fly-in a quarter second,
// the rush a twentieth — verbatim, no floors, no ceilings. A bad
// cursor is a no-op.
func (c *CloseupConfig) Nudge(i, dir int) {
	if c == nil || dir == 0 {
		return
	}
	switch i {
	case 0:
		c.FlyInSeconds += closeupStepSeconds * float64(dir)
	case 1:
		c.RushPeak += closeupRushStep * float64(dir)
	}
}

// CloseupShow is the close-up wearing that face: the seeded sky
// sliding with a dark hull gliding in from the right wing, the sky
// surging from rest to Cfg.RushPeak then settling to cruise, both
// on whatever clock Cfg holds when the curtain rises. The knobs
// live on this instance alone.
type CloseupShow struct {
	Cfg CloseupConfig
	sky *stars.Continuity
	screenplay.Ensemble
}

// NewCloseupShow is the close-up at the stock slide. A non-nil sky
// seeds the starfield so a cut into this scene opens on the frame the
// last scene left; a nil sky is a fresh sky of its own.
func NewCloseupShow(sky *stars.Continuity) *CloseupShow {
	s := &CloseupShow{Cfg: DefaultCloseupConfig(), sky: sky}
	s.Assemble = s.assemble
	return s
}

func (s *CloseupShow) assemble() []screenplay.Component {
	field := stars.NewTunedStarfield()
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	return []screenplay.Component{
		field.SlideIn(s.Cfg.FlyInSeconds, lander.BodyCols).Surge(s.Cfg.RushPeak, s.Cfg.FlyInSeconds),
		lander.NewShip(11).Dark().FlyIn(s.Cfg.FlyInSeconds),
	}
}

// FireConfig is the fire scene's editable face: how far the stars
// brake, how long the brake takes, and how long the lit hull falls
// from the park off the bottom (SinkSeconds, panel label "fall").
// Hold is a different number — how long the scene plays before the
// cut. If hold is 7 and sink is 12 the cut can happen mid-fall; that
// is the operator's choice, never clamped. Stock sink is zero — the
// walkthrough stays parked. MAIN's saved knobs turn the fall on.
type FireConfig struct {
	SlowBy          float64 `json:"slowBy"`
	SlowOverSeconds float64 `json:"slowOverSeconds"`
	SinkSeconds     float64 `json:"sinkSeconds"`
}

// DefaultFireConfig is the stock brake with the sink off: the stars
// finish 60% slower over five seconds, and the hull holds the park
// until a show's knobs set a sink window.
func DefaultFireConfig() FireConfig {
	return FireConfig{SlowBy: 0.6, SlowOverSeconds: 5}
}

// KnobCount is how many knobs the fire carries.
func (c FireConfig) KnobCount() int { return 3 }

// KnobLabel is the panel name of knob i.
func (c FireConfig) KnobLabel(i int) string {
	switch i {
	case 0:
		return "slow by"
	case 1:
		return "slow over"
	case 2:
		return "fall"
	default:
		return ""
	}
}

// Value reads one knob for display and tests.
func (c FireConfig) Value(i int) float64 {
	switch i {
	case 0:
		return c.SlowBy
	case 1:
		return c.SlowOverSeconds
	case 2:
		return c.SinkSeconds
	default:
		return 0
	}
}

// Nudge walks one knob by dir steps — the brake's depth a twentieth,
// its window and the sink a quarter second — verbatim, no floors, no
// ceilings. A bad cursor is a no-op.
func (c *FireConfig) Nudge(i, dir int) {
	if c == nil || dir == 0 {
		return
	}
	switch i {
	case 0:
		c.SlowBy += fireByStep * float64(dir)
	case 1:
		c.SlowOverSeconds += fireOverStep * float64(dir)
	case 2:
		c.SinkSeconds += fireOverStep * float64(dir)
	}
}

// FireShow is the fire wearing that face: the parked craft lights the
// booster while the seeded sky brakes on whatever numbers Cfg holds
// when the curtain rises. A non-zero Cfg.SinkSeconds eases the lit
// hull downward — MAIN turns that on; stock walkthrough stays
// parked. The knobs live on this instance alone.
type FireShow struct {
	Cfg FireConfig
	sky *stars.Continuity
	screenplay.Ensemble
}

// NewFireShow is the fire at the stock brake. A non-nil sky seeds the
// starfield so a cut into this scene opens on the frame the last
// scene left; a nil sky is a fresh sky of its own.
func NewFireShow(sky *stars.Continuity) *FireShow {
	s := &FireShow{Cfg: DefaultFireConfig(), sky: sky}
	s.Assemble = s.assemble
	return s
}

func (s *FireShow) assemble() []screenplay.Component {
	field := stars.NewTunedStarfield()
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	craft := lander.NewShip(11).Parked()
	// Zero keeps the park (the walkthrough's stock). A non-zero
	// window rides SinkPath — the operator's number, verbatim,
	// including a negative, which the path treats as already gone.
	if s.Cfg.SinkSeconds != 0 {
		craft = craft.Sink(s.Cfg.SinkSeconds)
	}
	return []screenplay.Component{
		field.Slow(s.Cfg.SlowBy, s.Cfg.SlowOverSeconds),
		craft,
	}
}
