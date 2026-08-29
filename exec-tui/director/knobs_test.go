package director

// Tests written FIRST: knobsFor is how the editor reads a scene's
// tunable face. Every knobbed Show on the four bills — fall, landing,
// liftoff, bobble, moonwalk — maps to an adapter carrying its kind,
// its config file, and label/value/nudge bound to that one Show's
// Cfg. Saving writes the file and makes the knobs Active, so future
// curtains play them. Only the moonwalk syncs siblings: its three
// beats are one performance sharing one Cfg, so a sync pulls Active
// into another beat. The bobble must never sync — Lit and Dark are
// the bill's word on each entry, and a blanket copy would relight a
// dark engine. A scene without knobs (the inline ensembles) maps to
// nothing at all.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/fall"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

func TestKnobsFor(t *testing.T) {
	t.Run("happy: every knobbed show on the bills maps to its adapter", func(t *testing.T) {
		cases := []struct {
			scene screenplay.Scene
			kind  string
			count int
			path  string
		}{
			{fall.New(nil), "fall", int(fall.KnobCount), fall.DefaultConfigPath},
			{landing.New(nil), "landing", int(landing.KnobCount), landing.DefaultConfigPath},
			{liftoff.New(nil), "liftoff", int(liftoff.KnobCount), liftoff.DefaultConfigPath},
			{bobble.New(nil), "bobble", int(bobble.KnobCount), bobble.DefaultConfigPath},
			{moonwalk.New(moonwalk.BeatRun), "moonwalk", int(moonwalk.KnobCount), moonwalk.DefaultConfigPath},
		}
		for _, tc := range cases {
			k := knobsFor(tc.scene)
			if k == nil {
				t.Fatalf("%s must map to an adapter", tc.kind)
			}
			if k.kind != tc.kind || k.count != tc.count || k.path != tc.path {
				t.Fatalf("%s adapter = kind %q count %d path %q, want %q %d %q",
					tc.kind, k.kind, k.count, k.path, tc.kind, tc.count, tc.path)
			}
			for i := 0; i < k.count; i++ {
				if k.label(i) == "" {
					t.Fatalf("%s knob %d has no label", tc.kind, i)
				}
			}
		}
	})
	t.Run("happy: value and nudge are bound to that one show's Cfg", func(t *testing.T) {
		t.Cleanup(fall.Reset)
		s := fall.New(nil)
		k := knobsFor(s)
		if k.label(0) != "drop" {
			t.Fatalf("the fall knob is %q, want drop", k.label(0))
		}
		before := s.Cfg.DropSeconds
		if k.value(0) != before {
			t.Fatalf("value reads %v, want the show's %v", k.value(0), before)
		}
		k.nudge(0, 1)
		if got := s.Cfg.DropSeconds; got != before+fall.StepSeconds {
			t.Fatalf("one nudge moved the drop to %v, want %v", got, before+fall.StepSeconds)
		}
		other := fall.New(nil)
		if other.Cfg.DropSeconds != before {
			t.Fatal("nudging one show must not touch another")
		}
	})
	t.Run("happy: the moonwalk labels are its own knob names", func(t *testing.T) {
		k := knobsFor(moonwalk.New(moonwalk.BeatPole))
		for i := 0; i < k.count; i++ {
			if want := moonwalk.Knob(i).String(); k.label(i) != want {
				t.Fatalf("moonwalk knob %d labeled %q, want %q", i, k.label(i), want)
			}
		}
	})
	t.Run("happy: save writes the file and makes the knobs active", func(t *testing.T) {
		t.Cleanup(fall.Reset)
		s := fall.New(nil)
		k := knobsFor(s)
		k.nudge(0, 2)
		path := filepath.Join(t.TempDir(), "fall.json")
		if err := k.save(path); err != nil {
			t.Fatalf("save: %v", err)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("save left no file: %v", err)
		}
		if got := fall.Active().DropSeconds; got != s.Cfg.DropSeconds {
			t.Fatalf("save must activate the knobs: active %v, show %v", got, s.Cfg.DropSeconds)
		}
	})
	t.Run("happy: the moonwalk beats are one performance — sync pulls Active in", func(t *testing.T) {
		t.Cleanup(moonwalk.Reset)
		run := moonwalk.New(moonwalk.BeatRun)
		pole := moonwalk.New(moonwalk.BeatPole)
		kr := knobsFor(run)
		if kr.sync == nil {
			t.Fatal("the moonwalk adapter must sync its siblings")
		}
		run.Cfg.RunSpeed += 7
		if err := kr.save(filepath.Join(t.TempDir(), "walk.json")); err != nil {
			t.Fatalf("save: %v", err)
		}
		if pole.Cfg.RunSpeed == run.Cfg.RunSpeed {
			t.Fatal("an unsynced sibling must still hold its own knobs")
		}
		knobsFor(pole).sync()
		if pole.Cfg.RunSpeed != run.Cfg.RunSpeed {
			t.Fatalf("after the sync the pole beat runs %v, want %v", pole.Cfg.RunSpeed, run.Cfg.RunSpeed)
		}
		if pole.Beat() != moonwalk.BeatPole {
			t.Fatal("a sync must not change which beat a show plays")
		}
	})
	t.Run("unhappy: the bobble never syncs — the bill's word on the engine wins", func(t *testing.T) {
		for _, sc := range []screenplay.Scene{
			bobble.New(nil).Lit(),
			bobble.New(nil).Dark(),
			fall.New(nil),
			landing.New(nil),
			liftoff.New(nil),
		} {
			if k := knobsFor(sc); k.sync != nil {
				t.Fatalf("%s must not volunteer a sync", k.kind)
			}
		}
	})
	t.Run("unhappy: a scene without knobs maps to nothing", func(t *testing.T) {
		if k := knobsFor(&screenplay.Ensemble{}); k != nil {
			t.Fatalf("a bare ensemble has no knobs, got %+v", k)
		}
		if k := knobsFor(nil); k != nil {
			t.Fatalf("a nil scene has no knobs, got %+v", k)
		}
	})
	t.Run("unhappy: save into a missing folder reports the error and stays inactive", func(t *testing.T) {
		t.Cleanup(bobble.Reset)
		s := bobble.New(nil)
		k := knobsFor(s)
		k.nudge(int(bobble.KnobPeriod), 3)
		if err := k.save(filepath.Join(t.TempDir(), "missing-dir", "bobble.json")); err == nil {
			t.Fatal("an unwritable path must error")
		}
		if bobble.Active().PeriodSeconds == s.Cfg.PeriodSeconds {
			t.Fatal("a failed save must not activate the knobs")
		}
	})
}
