package director

// Tests written FIRST: knobsFor is how the editor reads a scene's
// tunable face. Every knobbed Show on the four bills maps to an
// adapter carrying its kind and label/value/nudge bound to that one
// Show's Cfg — the five tuner scenes (fall, landing, liftoff, bobble,
// moonwalk) and the three shows that grew faces for MAIN (the orbit,
// the close-up, the fire). marshal and apply carry a Cfg through the
// scene's own JSON shape, so MAIN's config file can own every scene's
// knobs without ever touching a scene package's file or its Active.
// Only the moonwalk syncs siblings — its beats are one performance —
// and a scene without knobs maps to nothing at all.

import (
	"encoding/json"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/fall"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/lunarcloseup"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/moonshow"
)

func TestKnobsFor(t *testing.T) {
	t.Run("happy: every knobbed show on the bills maps to its adapter", func(t *testing.T) {
		cases := []struct {
			scene screenplay.Scene
			kind  string
			count int
			syncs bool
		}{
			{fall.New(nil), "fall", int(fall.KnobCount), false},
			{landing.New(nil), "landing", int(landing.KnobCount), false},
			{liftoff.New(nil), "liftoff", int(liftoff.KnobCount), false},
			{bobble.New(nil), "bobble", int(bobble.KnobCount), false},
			{moonwalk.New(moonwalk.BeatRun), "moonwalk", int(moonwalk.KnobCount), true},
			{moonshow.NewOrbitShow(), "orbit", 2, false},
			{lunarcloseup.NewCloseupShow(nil), "closeup", 1, false},
			{lunarcloseup.NewFireShow(nil), "fire", 2, false},
		}
		for _, tc := range cases {
			k := knobsFor(tc.scene)
			if k == nil {
				t.Fatalf("%s must map to an adapter", tc.kind)
			}
			if k.kind != tc.kind || k.count != tc.count || k.syncs != tc.syncs {
				t.Fatalf("%s adapter = kind %q count %d syncs %v, want %q %d %v",
					tc.kind, k.kind, k.count, k.syncs, tc.kind, tc.count, tc.syncs)
			}
			for i := 0; i < k.count; i++ {
				if k.label(i) == "" {
					t.Fatalf("%s knob %d has no label", tc.kind, i)
				}
			}
		}
	})
	t.Run("happy: value and nudge are bound to that one show's Cfg", func(t *testing.T) {
		s := moonshow.NewOrbitShow()
		k := knobsFor(s)
		if k.label(1) != "lap" {
			t.Fatalf("the orbit's second knob is %q, want lap", k.label(1))
		}
		before := s.Cfg.LapSeconds
		if k.value(1) != before {
			t.Fatalf("value reads %v, want the show's %v", k.value(1), before)
		}
		k.nudge(1, -2)
		if got := s.Cfg.LapSeconds; got != before-0.5 {
			t.Fatalf("two nudges down read %v, want %v", got, before-0.5)
		}
		if other := moonshow.NewOrbitShow(); other.Cfg.LapSeconds != before {
			t.Fatal("nudging one show must not touch another")
		}
	})
	t.Run("happy: marshal and apply carry a Cfg through the scene's own JSON shape", func(t *testing.T) {
		edited := fall.New(nil)
		ke := knobsFor(edited)
		ke.nudge(0, 4)
		raw, err := ke.marshal()
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		fresh := fall.New(nil)
		if fresh.Cfg.DropSeconds == edited.Cfg.DropSeconds {
			t.Fatal("test premise: the fresh show must open at stock")
		}
		if err := knobsFor(fresh).apply(raw); err != nil {
			t.Fatalf("apply: %v", err)
		}
		if fresh.Cfg != edited.Cfg {
			t.Fatalf("the applied show carries %+v, want %+v", fresh.Cfg, edited.Cfg)
		}
	})
	t.Run("happy: a partial apply keeps every unnamed knob", func(t *testing.T) {
		s := liftoff.New(nil)
		stock := s.Cfg
		if err := knobsFor(s).apply(json.RawMessage(`{"riseSeconds":9}`)); err != nil {
			t.Fatalf("apply: %v", err)
		}
		if s.Cfg.RiseSeconds != 9 {
			t.Fatalf("the named knob reads %v, want 9", s.Cfg.RiseSeconds)
		}
		if s.Cfg.LiftAt != stock.LiftAt || s.Cfg.DustRun != stock.DustRun {
			t.Fatal("knobs the JSON does not name must stand")
		}
	})
	t.Run("happy: applying never touches the scene package's Active", func(t *testing.T) {
		t.Cleanup(fall.Reset)
		s := fall.New(nil)
		if err := knobsFor(s).apply(json.RawMessage(`{"dropSeconds":9}`)); err != nil {
			t.Fatalf("apply: %v", err)
		}
		if s.Cfg.DropSeconds != 9 {
			t.Fatalf("the show must wear the applied drop, got %v", s.Cfg.DropSeconds)
		}
		if fall.Active().DropSeconds == 9 {
			t.Fatal("MAIN's knobs are its own — Active must not move")
		}
	})
	t.Run("unhappy: a mismatched blob is an error and the knobs stand", func(t *testing.T) {
		s := lunarcloseup.NewFireShow(nil)
		stock := s.Cfg
		if err := knobsFor(s).apply(json.RawMessage(`[1,2]`)); err == nil {
			t.Fatal("a JSON array is not a knob set — apply must error")
		}
		if s.Cfg != stock {
			t.Fatalf("a failed apply moved the knobs to %+v", s.Cfg)
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
}
