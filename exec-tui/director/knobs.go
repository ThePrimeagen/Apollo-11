package director

import (
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/fall"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// knobs is the tunable face one scene shows the editor: the scene
// kind, its own config file, and label/value/nudge bound to that one
// Show's Cfg — nudging never touches a sibling. save writes the file
// and makes the knobs Active, so future curtains play them. sync,
// when a kind volunteers one, pulls Active back into this Show — the
// moonwalk does, because its beats are one performance sharing one
// Cfg; the bobble must not, because Lit and Dark are the bill's word
// on each entry.
type knobs struct {
	kind  string
	path  string
	count int
	label func(i int) string
	value func(i int) float64
	nudge func(i, dir int)
	save  func(path string) error
	sync  func()
}

// knobsFor maps a scene to its adapter, or nil for a scene with no
// knobs at all — the inline ensembles play as staged.
func knobsFor(sc screenplay.Scene) *knobs {
	switch s := sc.(type) {
	case *fall.Show:
		return &knobs{
			kind:  "fall",
			path:  fall.DefaultConfigPath,
			count: int(fall.KnobCount),
			label: func(i int) string { return fall.KnobLabel(fall.Knob(i)) },
			value: func(i int) float64 { return s.Cfg.Value(fall.Knob(i)) },
			nudge: func(i, dir int) { s.Cfg.Nudge(fall.Knob(i), dir) },
			save: func(path string) error {
				if err := s.Cfg.Save(path); err != nil {
					return err
				}
				return fall.Use(s.Cfg)
			},
		}
	case *landing.Show:
		return &knobs{
			kind:  "landing",
			path:  landing.DefaultConfigPath,
			count: int(landing.KnobCount),
			label: func(i int) string { return landing.KnobLabel(landing.Knob(i)) },
			value: func(i int) float64 { return s.Cfg.Value(landing.Knob(i)) },
			nudge: func(i, dir int) { s.Cfg.Nudge(landing.Knob(i), dir) },
			save: func(path string) error {
				if err := s.Cfg.Save(path); err != nil {
					return err
				}
				return landing.Use(s.Cfg)
			},
		}
	case *liftoff.Show:
		return &knobs{
			kind:  "liftoff",
			path:  liftoff.DefaultConfigPath,
			count: int(liftoff.KnobCount),
			label: func(i int) string { return liftoff.KnobLabel(liftoff.Knob(i)) },
			value: func(i int) float64 { return s.Cfg.Value(liftoff.Knob(i)) },
			nudge: func(i, dir int) { s.Cfg.Nudge(liftoff.Knob(i), dir) },
			save: func(path string) error {
				if err := s.Cfg.Save(path); err != nil {
					return err
				}
				return liftoff.Use(s.Cfg)
			},
		}
	case *bobble.Show:
		return &knobs{
			kind:  "bobble",
			path:  bobble.DefaultConfigPath,
			count: int(bobble.KnobCount),
			label: func(i int) string { return bobble.KnobLabel(bobble.Knob(i)) },
			value: func(i int) float64 { return s.Cfg.Value(bobble.Knob(i)) },
			nudge: func(i, dir int) { s.Cfg.Nudge(bobble.Knob(i), dir) },
			save: func(path string) error {
				if err := s.Cfg.Save(path); err != nil {
					return err
				}
				return bobble.Use(s.Cfg)
			},
		}
	case *moonwalk.Show:
		return &knobs{
			kind:  "moonwalk",
			path:  moonwalk.DefaultConfigPath,
			count: int(moonwalk.KnobCount),
			label: func(i int) string { return moonwalk.Knob(i).String() },
			value: func(i int) float64 { return s.Cfg.Value(moonwalk.Knob(i)) },
			nudge: func(i, dir int) { s.Cfg.Nudge(moonwalk.Knob(i), dir) },
			save: func(path string) error {
				if err := s.Cfg.Save(path); err != nil {
					return err
				}
				return moonwalk.Use(s.Cfg)
			},
			sync: func() { s.Cfg = moonwalk.Active() },
		}
	}
	return nil
}
