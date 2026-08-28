package main

// Demo harness tests, written first: cmd/climb runs the portable
// spacelander climb from scenes/climb — the north-facing lander
// rising bottom to top under twinkling stars. One live knob: climb
// duration (±50ms). p / enter / space replay from the top, s saves,
// q quits.

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/climb"
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

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestClimbRunner(t *testing.T) {
	t.Cleanup(climb.Reset)
	t.Run("happy: the house opens on the climb, the stars, and the climb knob", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"climb", "play", "50ms", "save", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.ContainsAny(v, "·˚*✦") {
			t.Fatal("the climb plays under the stars")
		}
		if strings.ContainsRune(v, '▟') {
			t.Fatal("at t=0 the lander must still be off the bottom")
		}
	})
	t.Run("happy: h/l walk the climb 50ms; p replays from off the bottom", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey('l'))
		if math.Abs(m.show.Cfg.ClimbSeconds-(climb.DefaultConfig().ClimbSeconds+climb.StepSeconds)) > 1e-9 {
			t.Fatalf("climb after +50ms is %v", m.show.Cfg.ClimbSeconds)
		}
		m.show.Cfg.ClimbSeconds = 0.4
		m = frames(m, 20)
		if !strings.ContainsRune(m.View().Content, '▟') {
			t.Fatal("a short climb must put the hull on stage")
		}
		m = press(m, runeKey('p'))
		if strings.ContainsRune(m.View().Content, '▟') {
			t.Fatal("p must rewind the craft off the bottom")
		}
	})
	t.Run("unhappy: the climb floor is 50ms, space does not quit, and q does", func(t *testing.T) {
		m := newModel(0)
		m.show.Cfg.ClimbSeconds = climb.StepSeconds
		m = press(m, runeKey('h'))
		if m.show.Cfg.ClimbSeconds != climb.StepSeconds {
			t.Fatalf("climb %v, want the 50ms floor", m.show.Cfg.ClimbSeconds)
		}
		_, cmd := m.Update(space())
		if cmd != nil {
			t.Fatal("space must replay, not quit")
		}
		_, cmd = newModel(0).Update(runeKey('q'))
		if cmd == nil {
			t.Fatal("q must quit")
		}
	})
	t.Run("happy: s saves the knobs; the view fills the window", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "climb.json")
		m := newModel(0)
		m.path = path
		m.show.Cfg.ClimbSeconds = 4.25
		m = press(m, runeKey('s'))
		if !strings.Contains(m.View().Content, "saved") {
			t.Fatal("a successful save must say so")
		}
		got, err := climb.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got.ClimbSeconds-4.25) > 1e-9 {
			t.Fatalf("saved %v, want 4.25", got.ClimbSeconds)
		}
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 20 {
			t.Fatalf("view has %d lines for a 20-line window", got)
		}
		m.path = ""
		m = press(m, runeKey('s'))
		if !strings.Contains(m.View().Content, "no config") {
			t.Fatal("a missing path must say so")
		}
	})
}
