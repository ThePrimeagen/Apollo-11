package moonshow

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// OrbitConfig is the orbit scene's editable face: how long the
// arriving streak takes and how long one lap lasts. The numbers are
// the operator's, verbatim — Nudge steps and never clamps.
type OrbitConfig struct {
	ArriveSeconds float64 `json:"arriveSeconds"`
	LapSeconds    float64 `json:"lapSeconds"`
}

// orbitStepSeconds is one tick of either orbit knob: a quarter second.
const orbitStepSeconds = 0.25

// DefaultOrbitConfig is the stock pace — the moon component's consts.
func DefaultOrbitConfig() OrbitConfig {
	return OrbitConfig{ArriveSeconds: moon.ArriveSeconds, LapSeconds: moon.OrbitSeconds}
}

// KnobCount is how many knobs the orbit carries.
func (c OrbitConfig) KnobCount() int { return 2 }

// KnobLabel is the panel name of knob i.
func (c OrbitConfig) KnobLabel(i int) string {
	switch i {
	case 0:
		return "arrive"
	case 1:
		return "lap"
	default:
		return ""
	}
}

// Value reads one knob for display and tests.
func (c OrbitConfig) Value(i int) float64 {
	switch i {
	case 0:
		return c.ArriveSeconds
	case 1:
		return c.LapSeconds
	default:
		return 0
	}
}

// Nudge walks one knob by dir quarter-second steps, verbatim — no
// floors, no ceilings. A bad cursor is a no-op.
func (c *OrbitConfig) Nudge(i, dir int) {
	if c == nil || dir == 0 {
		return
	}
	switch i {
	case 0:
		c.ArriveSeconds += orbitStepSeconds * float64(dir)
	case 1:
		c.LapSeconds += orbitStepSeconds * float64(dir)
	}
}

// OrbitShow is the orbit scene wearing that face: the still sky, the
// moon, and the arriving craft flying whatever pace Cfg holds when
// the curtain rises. The knobs live on this instance alone — MAIN
// dresses them from its own config; the stock show flies the stock
// consts.
type OrbitShow struct {
	Cfg OrbitConfig
	screenplay.Ensemble
}

// NewOrbitShow is the orbit scene at the stock pace.
func NewOrbitShow() *OrbitShow {
	s := &OrbitShow{Cfg: DefaultOrbitConfig()}
	s.Assemble = s.assemble
	return s
}

func (s *OrbitShow) assemble() []screenplay.Component {
	return []screenplay.Component{
		stars.NewTunedStarfield().Still(),
		moon.New(),
		moon.NewOrbit().Arrive().Pace(s.Cfg.ArriveSeconds, s.Cfg.LapSeconds),
	}
}
