package main

// Demo harness tests, written first: cmd/mario runs 03. Mario — the
// composable three-scene bill from shows/mario. The house opens on
// "run": the astronaut sprints in and climbs the crates. Space cuts
// to "flagpole": the leap, the slide, the flag going up. Space cuts
// to "board": the camera pans to the lunar module and he jumps the
// hatch. Space on the last scene ends the show. q and ctrl+c quit.
// The view is the sky plus one status line, always exactly
// window-height lines.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func frames(m model, n int) model {
	for i := 0; i < n; i++ {
		mm, _ := m.Update(frameMsg{})
		m = mm.(model)
	}
	return m
}

func space() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "} }

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func cutOnce(t *testing.T, m model) model {
	t.Helper()
	mm, cmd := m.Update(space())
	if cmd != nil {
		t.Fatal("a cut with scenes left must not quit")
	}
	return mm.(model)
}

func TestMarioRunner(t *testing.T) {
	t.Run("happy: the house opens on scene 1/3 — run, the flagpole set", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"scene 1/3", "run", "space next scene", "q quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.ContainsRune(v, '│') || !strings.ContainsRune(v, '●') {
			t.Fatal("the run must stage the flagpole")
		}
		if strings.ContainsRune(v, '▟') {
			t.Fatal("the lunar module waits for the board")
		}
	})
	t.Run("happy: space walks run → flagpole → board, then ends the show", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = cutOnce(t, m)
		v := m.View().Content
		if !strings.Contains(v, "scene 2/3") || !strings.Contains(v, "flagpole") {
			t.Fatal("the first cut must land on flagpole")
		}
		if !strings.ContainsRune(v, '│') {
			t.Fatal("the flagpole scene keeps the pole")
		}
		m = cutOnce(t, m)
		v = m.View().Content
		if !strings.Contains(v, "scene 3/3") || !strings.Contains(v, "board") {
			t.Fatal("the second cut must land on board")
		}
		m = frames(m, int(4*30))
		if !strings.ContainsRune(m.View().Content, '▟') {
			t.Fatal("the board must pan the lunar module on stage")
		}
		_, cmd := m.Update(space())
		if cmd == nil {
			t.Fatal("space past the last scene must end the show")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("the end of the bill must issue tea.Quit")
		}
	})
	t.Run("happy: each frame schedules the next, and Init starts the clock", func(t *testing.T) {
		m := newModel(0)
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
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
	t.Run("happy: the view is always exactly window-height lines", func(t *testing.T) {
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
	t.Run("unhappy: mario is not the inverse walkthrough and not the premiere", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		if strings.Contains(v, "liftoff") || strings.Contains(v, "1/5") {
			t.Fatal("mario must not open as the inverse or the forward walkthrough")
		}
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in mario")
		}
		if strings.Contains(v, "THE END") {
			t.Fatal("the end card does not appear in mario")
		}
	})
}
