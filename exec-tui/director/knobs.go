package director

import (
	"encoding/json"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/fall"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/lunarcloseup"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/moonshow"
)

// knobs is the tunable face one scene shows the editor: the scene
// kind and label/value/nudge bound to that one Show's Cfg — nudging
// never touches a sibling. marshal and apply carry the Cfg through
// the scene's own JSON shape, so MAIN's config file can own every
// scene's knobs without ever touching a scene package's file or its
// Active. syncs marks a kind whose sibling entries are one
// performance — the moonwalk's beats — so a save copies the edited
// Cfg across them; the bobble must not sync, because Lit and Dark are
// the bill's word on each entry.
type knobs struct {
	kind    string
	count   int
	syncs   bool
	label   func(i int) string
	value   func(i int) float64
	nudge   func(i, dir int)
	marshal func() (json.RawMessage, error)
	apply   func(json.RawMessage) error
}

// bind carries any Cfg through its JSON shape: marshal snapshots it,
// apply lays a blob over it — partial blobs keep every unnamed knob,
// and a blob that does not fit leaves it untouched.
func bind[T any](cfg *T) (func() (json.RawMessage, error), func(json.RawMessage) error) {
	return func() (json.RawMessage, error) {
			return json.Marshal(cfg)
		}, func(raw json.RawMessage) error {
			next := *cfg
			if err := json.Unmarshal(raw, &next); err != nil {
				return err
			}
			*cfg = next
			return nil
		}
}

// knobsFor maps a scene to its adapter, or nil for a scene with no
// knobs at all — the still ensembles play as staged.
func knobsFor(sc screenplay.Scene) *knobs {
	switch s := sc.(type) {
	case *fall.Show:
		m, a := bind(&s.Cfg)
		return &knobs{
			kind:    "fall",
			count:   int(fall.KnobCount),
			label:   func(i int) string { return fall.KnobLabel(fall.Knob(i)) },
			value:   func(i int) float64 { return s.Cfg.Value(fall.Knob(i)) },
			nudge:   func(i, dir int) { s.Cfg.Nudge(fall.Knob(i), dir) },
			marshal: m,
			apply:   a,
		}
	case *landing.Show:
		m, a := bind(&s.Cfg)
		return &knobs{
			kind:    "landing",
			count:   int(landing.KnobCount),
			label:   func(i int) string { return landing.KnobLabel(landing.Knob(i)) },
			value:   func(i int) float64 { return s.Cfg.Value(landing.Knob(i)) },
			nudge:   func(i, dir int) { s.Cfg.Nudge(landing.Knob(i), dir) },
			marshal: m,
			apply:   a,
		}
	case *liftoff.Show:
		m, a := bind(&s.Cfg)
		return &knobs{
			kind:    "liftoff",
			count:   int(liftoff.KnobCount),
			label:   func(i int) string { return liftoff.KnobLabel(liftoff.Knob(i)) },
			value:   func(i int) float64 { return s.Cfg.Value(liftoff.Knob(i)) },
			nudge:   func(i, dir int) { s.Cfg.Nudge(liftoff.Knob(i), dir) },
			marshal: m,
			apply:   a,
		}
	case *bobble.Show:
		m, a := bind(&s.Cfg)
		return &knobs{
			kind:    "bobble",
			count:   int(bobble.KnobCount),
			label:   func(i int) string { return bobble.KnobLabel(bobble.Knob(i)) },
			value:   func(i int) float64 { return s.Cfg.Value(bobble.Knob(i)) },
			nudge:   func(i, dir int) { s.Cfg.Nudge(bobble.Knob(i), dir) },
			marshal: m,
			apply:   a,
		}
	case *moonwalk.Show:
		m, a := bind(&s.Cfg)
		return &knobs{
			kind:    "moonwalk",
			count:   int(moonwalk.KnobCount),
			syncs:   true,
			label:   func(i int) string { return moonwalk.Knob(i).String() },
			value:   func(i int) float64 { return s.Cfg.Value(moonwalk.Knob(i)) },
			nudge:   func(i, dir int) { s.Cfg.Nudge(moonwalk.Knob(i), dir) },
			marshal: m,
			apply:   a,
		}
	case *moonshow.OrbitShow:
		m, a := bind(&s.Cfg)
		return &knobs{
			kind:    "orbit",
			count:   s.Cfg.KnobCount(),
			label:   s.Cfg.KnobLabel,
			value:   func(i int) float64 { return s.Cfg.Value(i) },
			nudge:   func(i, dir int) { s.Cfg.Nudge(i, dir) },
			marshal: m,
			apply:   a,
		}
	case *lunarcloseup.CloseupShow:
		m, a := bind(&s.Cfg)
		return &knobs{
			kind:    "closeup",
			count:   s.Cfg.KnobCount(),
			label:   s.Cfg.KnobLabel,
			value:   func(i int) float64 { return s.Cfg.Value(i) },
			nudge:   func(i, dir int) { s.Cfg.Nudge(i, dir) },
			marshal: m,
			apply:   a,
		}
	case *lunarcloseup.FireShow:
		m, a := bind(&s.Cfg)
		return &knobs{
			kind:    "fire",
			count:   s.Cfg.KnobCount(),
			label:   s.Cfg.KnobLabel,
			value:   func(i int) float64 { return s.Cfg.Value(i) },
			nudge:   func(i, dir int) { s.Cfg.Nudge(i, dir) },
			marshal: m,
			apply:   a,
		}
	}
	return nil
}
