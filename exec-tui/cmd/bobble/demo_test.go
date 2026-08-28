package main

// Demo harness tests, written first: cmd/bobble runs the portable
// bobble scene from scenes/bobble — the west-facing lander parked at
// center stage, bobbling on a sine, with or without its engine on.
// Three live knobs: engine (l on, h off), period (±50ms), amplitude
// (±1 cell). p / enter / space replay from the top with the current
// knobs, s saves, q and ctrl+c quit. The view is the rendered screen
// plus the knob panel, always exactly window-height lines.

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
)

func frames(m model, n int) model {
	for i := 0; i < n; i++ {
		mm, _ := m.Update(frameMsg{})
		m = mm.(model)
	}
	return m
}

func press(m model, msg tea.Msg) model {
	mm, _ := m.Update(msg)
	return mm.(model)
}

func space() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "} }

func enter() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func hasFire(v string) bool {
	return strings.ContainsAny(v, "⠁⠒⠶")
}

func TestBobbleSceneRunner(t *testing.T) {
	t.Cleanup(bobble.Reset)
	t.Run("happy: the house opens on the parked west hull with its engine burning", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"bobble", "play", "tune", "save", "quit", "engine", "period", "amplitude"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("the bobble must open with the west hull already parked")
		}
		if !strings.Contains(v, "on") {
			t.Fatal("the engine knob must read on")
		}
		m = frames(m, 15)
		if !hasFire(m.View().Content) {
			t.Fatal("half a second in the tail fire must be burning")
		}
	})
	t.Run("happy: h switches the engine off and the replay parks a cold hull", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey('h'))
		if m.show.Cfg.Engine {
			t.Fatal("h on the engine knob must switch it off")
		}
		if !strings.Contains(m.View().Content, "off") {
			t.Fatal("the engine knob must read off")
		}
		m = press(m, space())
		m = frames(m, 30)
		v := m.View().Content
		if hasFire(v) {
			t.Fatal("after the replay the cold engine must burn nothing")
		}
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("the cold hull still parks on stage")
		}
		m = press(m, runeKey('l'))
		if !m.show.Cfg.Engine {
			t.Fatal("l on the engine knob must switch it back on")
		}
	})
	t.Run("happy: j/k select a knob and h/l walk the ride", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.PeriodSeconds; math.Abs(got-(bobble.DefaultConfig().PeriodSeconds+bobble.StepSeconds)) > 1e-9 {
			t.Fatalf("period after +50ms is %v, want %v", got, bobble.DefaultConfig().PeriodSeconds+bobble.StepSeconds)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.AmplitudeCells; got != bobble.DefaultConfig().AmplitudeCells+1 {
			t.Fatalf("amplitude after +1 is %v, want %v", got, bobble.DefaultConfig().AmplitudeCells+1)
		}
		if !strings.Contains(m.View().Content, ">") {
			t.Fatal("the selected knob must be marked")
		}
	})
	t.Run("unhappy: the period floor is 50ms, the amplitude floor is zero, and space does not quit", func(t *testing.T) {
		m := newModel(0)
		m.show.Cfg.PeriodSeconds = bobble.StepSeconds
		m.show.Cfg.AmplitudeCells = 0
		m.cursor = bobble.KnobPeriod
		m = press(m, runeKey('h'))
		if m.show.Cfg.PeriodSeconds != bobble.StepSeconds {
			t.Fatalf("period %v, want the 50ms floor", m.show.Cfg.PeriodSeconds)
		}
		m.cursor = bobble.KnobAmplitude
		m = press(m, runeKey('h'))
		if m.show.Cfg.AmplitudeCells != 0 {
			t.Fatalf("amplitude %v, want 0", m.show.Cfg.AmplitudeCells)
		}
		_, cmd := m.Update(space())
		if cmd != nil {
			t.Fatal("space must replay, not quit")
		}
	})
	t.Run("happy: each frame schedules the next", func(t *testing.T) {
		m := newModel(0)
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: Init schedules the first frame", func(t *testing.T) {
		if newModel(0).Init() == nil {
			t.Fatal("Init must start the clock")
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
		m := newModel(0.05)
		mm, cmd := m.Update(frameMsg{})
		m = mm.(model)
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("one frame is 0.033s — too early for a 0.05s curtain")
		}
		_, cmd = m.Update(frameMsg{})
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("two frames pass 0.05s — the curtain must fall")
		}
	})
	t.Run("happy: the view fills the window even when the sky runs short", func(t *testing.T) {
		m := newModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 20 {
			t.Fatalf("view has %d lines for a 20-line window", got)
		}
		mm, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 34})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 34 {
			t.Fatalf("view has %d lines for a 34-line window", got)
		}
	})
	t.Run("happy: p and enter replay and do not quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		for _, msg := range []tea.Msg{runeKey('p'), enter()} {
			mm, cmd := m.Update(msg)
			if cmd != nil {
				t.Fatalf("%v must replay, not quit", msg)
			}
			m = mm.(model)
			if !strings.ContainsRune(m.View().Content, '▌') {
				t.Fatalf("after %v the hull must still be parked", msg)
			}
		}
	})
	t.Run("happy: s saves the knobs and does not quit", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bobble.json")
		m := newModel(0)
		m.path = path
		m.show.Cfg.Engine = false
		m.show.Cfg.PeriodSeconds = 4.25
		m.show.Cfg.AmplitudeCells = 3
		mm, cmd := m.Update(runeKey('s'))
		if cmd != nil {
			t.Fatal("s must save, not quit")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "saved") {
			t.Fatal("a successful save must say so")
		}
		got, err := bobble.Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got != m.show.Cfg {
			t.Fatalf("saved %+v, want the live knobs %+v", got, m.show.Cfg)
		}
	})
	t.Run("unhappy: s with no path or a stuck file does not quit and says so", func(t *testing.T) {
		m := newModel(0)
		m.path = ""
		mm, cmd := m.Update(runeKey('s'))
		if cmd != nil {
			t.Fatal("a failed save must not quit")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "no config") {
			t.Fatal("a missing path must tell the operator it could not save")
		}
		stuck := filepath.Join(t.TempDir(), "stuck")
		if err := os.Mkdir(stuck, 0o755); err != nil {
			t.Fatal(err)
		}
		m.path = stuck
		mm, cmd = m.Update(runeKey('s'))
		if cmd != nil {
			t.Fatal("a stuck file must not quit")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "failed") {
			t.Fatal("a stuck file must tell the operator the save failed")
		}
	})
	t.Run("unhappy: q and ctrl+c close the house", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			runeKey('q'),
			tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		} {
			_, cmd := newModel(0).Update(msg)
			if cmd == nil {
				t.Fatalf("%v must quit", msg)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("%v must issue tea.Quit", msg)
			}
		}
	})
	t.Run("unhappy: the bobble plays in open space — no moon floor, no north hull", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		if strings.Contains(v, "48;5;25") || strings.Contains(v, "48;5;24") {
			t.Fatal("the bobble has no moon floor")
		}
		if strings.ContainsRune(v, '▟') && !strings.ContainsRune(v, '▌') {
			t.Fatal("the bobble flies the west hull, not the north one")
		}
	})
}
