package lunarcloseup

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Steps of the walkthrough faces' knobs: times move a quarter second,
// the brake's depth a twentieth.
const (
	closeupStepSeconds = 0.25
	fireByStep         = 0.05
	fireOverStep       = 0.25
)

// CloseupConfig is the close-up scene's editable face: how long the
// craft's slide from the right wing takes — the sky translates on the
// same clock. The number is the operator's, verbatim.
type CloseupConfig struct {
	FlyInSeconds float64 `json:"flyInSeconds"`
}

// DefaultCloseupConfig is the stock slide — the lander const.
func DefaultCloseupConfig() CloseupConfig {
	return CloseupConfig{FlyInSeconds: lander.FlyInSeconds}
}

// KnobCount is how many knobs the close-up carries.
func (c CloseupConfig) KnobCount() int { return 1 }

// KnobLabel is the panel name of knob i.
func (c CloseupConfig) KnobLabel(i int) string {
	if i == 0 {
		return "fly-in"
	}
	return ""
}

// Value reads one knob for display and tests.
func (c CloseupConfig) Value(i int) float64 {
	if i == 0 {
		return c.FlyInSeconds
	}
	return 0
}

// Nudge walks the fly-in by dir quarter-second steps, verbatim — no
// floors, no ceilings. A bad cursor is a no-op.
func (c *CloseupConfig) Nudge(i, dir int) {
	if c == nil || dir == 0 || i != 0 {
		return
	}
	c.FlyInSeconds += closeupStepSeconds * float64(dir)
}

// CloseupShow is the close-up wearing that face: the seeded sky
// sliding with a dark hull gliding in from the right wing, both on
// whatever clock Cfg holds when the curtain rises. The knob lives on
// this instance alone.
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
		field.SlideIn(s.Cfg.FlyInSeconds, lander.BodyCols),
		lander.NewShip(11).Dark().FlyIn(s.Cfg.FlyInSeconds),
	}
}

// FireConfig is the fire scene's editable face: how far the stars
// brake and how long the brake takes. The numbers are the operator's,
// verbatim.
type FireConfig struct {
	SlowBy          float64 `json:"slowBy"`
	SlowOverSeconds float64 `json:"slowOverSeconds"`
}

// DefaultFireConfig is the stock brake: the stars finish 60% slower
// over five seconds.
func DefaultFireConfig() FireConfig {
	return FireConfig{SlowBy: 0.6, SlowOverSeconds: 5}
}

// KnobCount is how many knobs the fire carries.
func (c FireConfig) KnobCount() int { return 2 }

// KnobLabel is the panel name of knob i.
func (c FireConfig) KnobLabel(i int) string {
	switch i {
	case 0:
		return "slow by"
	case 1:
		return "slow over"
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
	default:
		return 0
	}
}

// Nudge walks one knob by dir steps — the brake's depth a twentieth,
// its window a quarter second — verbatim, no floors, no ceilings. A
// bad cursor is a no-op.
func (c *FireConfig) Nudge(i, dir int) {
	if c == nil || dir == 0 {
		return
	}
	switch i {
	case 0:
		c.SlowBy += fireByStep * float64(dir)
	case 1:
		c.SlowOverSeconds += fireOverStep * float64(dir)
	}
}

// FireShow is the fire wearing that face: the parked craft lights the
// booster while the seeded sky brakes on whatever numbers Cfg holds
// when the curtain rises. The knobs live on this instance alone.
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
	return []screenplay.Component{
		field.Slow(s.Cfg.SlowBy, s.Cfg.SlowOverSeconds),
		lander.NewShip(11).Parked(),
	}
}
