package main

// Demo harness tests, written first: cmd/prog runs the portable
// program-alarm drop from scenes/prog — the north-facing lander
// falling under twinkling stars, pausing 1202, 1202, then 1201.
// Seven live knobs (four drops, three holds) walk 50ms. p replays,
// s saves, q quits.

import (
	"math"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/prog"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
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

func hasBanner(v, text string) bool {
	lines, err := termfont.Lines(3, text)
	if err != nil {
		return false
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if !strings.Contains(v, trim) {
			return false
		}
	}
	return true
}

func TestProgRunner(t *testing.T) {
	t.Cleanup(prog.Reset)
	t.Run("happy: the house opens on the prog drop and the seven knobs", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"prog", "play", "50ms", "save", "quit",
			"drop 1", "hold 1", "drop 2", "hold 2", "drop 3", "hold 3", "drop 4"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.ContainsAny(v, "·˚*✦") {
			t.Fatal("the prog drop plays under the stars")
		}
	})
	t.Run("happy: the first hold paints 1202; j walks to hold 3", func(t *testing.T) {
		m := newModel(0)
		m.show.Cfg.Drop1, m.show.Cfg.Hold1 = 0.2, 0.5
		m.show.Cfg.Drop2, m.show.Cfg.Hold2 = 0.2, 0.2
		m.show.Cfg.Drop3, m.show.Cfg.Hold3 = 0.2, 0.2
		m.show.Cfg.Drop4 = 0.2
		m = press(m, runeKey('p'))
		m = frames(m, 12)
		if !hasBanner(m.View().Content, "1202") {
			t.Fatal("the runner must show 1202 on the first hold")
		}
		m = press(m, runeKey('j'))
		if m.cursor != prog.KnobHold1 {
			t.Fatalf("j from drop 1 must land on hold 1, got %d", m.cursor)
		}
		m = press(m, runeKey('l'))
		if math.Abs(m.show.Cfg.Hold1-(0.5+prog.StepSeconds)) > 1e-9 {
			t.Fatalf("hold 1 after +50ms is %v, want %v", m.show.Cfg.Hold1, 0.5+prog.StepSeconds)
		}
	})
	t.Run("unhappy: a hold will not go negative, space does not quit, and q does", func(t *testing.T) {
		m := newModel(0)
		m.cursor = prog.KnobHold1
		m.show.Cfg.Hold1 = 0
		m = press(m, runeKey('h'))
		if m.show.Cfg.Hold1 != 0 {
			t.Fatalf("hold 1 %v, want 0", m.show.Cfg.Hold1)
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
		path := filepath.Join(t.TempDir(), "prog.json")
		m := newModel(0)
		m.path = path
		m.show.Cfg.Hold3 = 2.0
		m = press(m, runeKey('s'))
		if !strings.Contains(m.View().Content, "saved") {
			t.Fatal("a successful save must say so")
		}
		got, err := prog.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(got.Hold3-2.0) > 1e-9 {
			t.Fatalf("saved hold 3 %v, want 2.0", got.Hold3)
		}
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 22})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 22 {
			t.Fatalf("view has %d lines for a 22-line window", got)
		}
		m.path = ""
		m = press(m, runeKey('s'))
		if !strings.Contains(m.View().Content, "no config") {
			t.Fatal("a missing path must say so")
		}
	})
}
