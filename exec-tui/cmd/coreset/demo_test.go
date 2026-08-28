package main

// Demo harness tests, written first: cmd/coreset runs the Core Set
// scene standalone. The house opens on the memory unit — both panels,
// tops aligned — under a one-line marquee, and the scene plays itself:
// the drain, the move, the twelve-word anatomy, the priority zoom,
// the fifteen bits. space (or p, or enter) replays from the top,
// -seconds brings the curtain down on time, q and ctrl+c quit
// anywhere, and the view is always exactly window-height lines.

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// plain is the view with the color codes stripped, so spaced phrases
// read as they do on the terminal.
func plain(m model) string { return ansiPat.ReplaceAllString(m.View().Content, "") }

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

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func toSeconds(s float64) int { return int(s*30) + 3 }

func TestCoresetDemoOpens(t *testing.T) {
	t.Run("happy: the house opens on the memory unit with the marquee", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		for _, want := range []string{"CORE SETS", "VAC AREAS", "Core Set", "replay", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
	})
	t.Run("unhappy: the anatomy is nowhere before its act", func(t *testing.T) {
		m := newModel(0)
		if strings.Contains(plain(m), "MPAC") {
			t.Fatal("the twelve words must wait for the anatomy act")
		}
	})
}

func TestCoresetDemoPlays(t *testing.T) {
	t.Run("happy: the acts advance on their own clock", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, toSeconds(coreset.WordsStart+12*coreset.WordBeat+0.3))
		v := plain(m)
		if !strings.Contains(v, "MPAC") || !strings.Contains(v, "PRIO") {
			t.Fatal("past the reveal the twelve-word bar must be on stage")
		}
		if strings.Contains(v, "VAC AREAS") {
			t.Fatal("the memory unit must be long gone by the anatomy act")
		}
		m = frames(m, toSeconds(coreset.BitsStart-coreset.WordsStart-12*coreset.WordBeat-0.3+1))
		v = plain(m)
		if !strings.Contains(v, "VAC ADDRESS") || !strings.Contains(v, "OCT 20") {
			t.Fatal("the bits act must break the priority word open")
		}
	})
	t.Run("happy: space, p and enter replay from the top", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			tea.KeyPressMsg{Code: tea.KeySpace, Text: " "},
			runeKey('p'),
			tea.KeyPressMsg{Code: tea.KeyEnter},
		} {
			m := newModel(0)
			_ = m.View()
			m = frames(m, toSeconds(coreset.MoveStart+0.5))
			if strings.Contains(plain(m), "VAC AREAS") {
				t.Fatal("test premise: the unit act must be over before the replay")
			}
			m = press(m, msg)
			if !strings.Contains(plain(m), "VAC AREAS") {
				t.Fatalf("%v must rewind to the memory unit", msg)
			}
		}
	})
	t.Run("unhappy: unknown keys neither replay nor quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, toSeconds(coreset.MoveStart+0.5))
		mm, cmd := m.Update(runeKey('z'))
		if cmd != nil {
			t.Fatal("an unknown key must do nothing")
		}
		m = mm.(model)
		if strings.Contains(plain(m), "VAC AREAS") {
			t.Fatal("an unknown key must not rewind the show")
		}
	})
}

func TestCoresetDemoHouseRules(t *testing.T) {
	t.Run("happy: Init schedules the first frame and each frame the next", func(t *testing.T) {
		m := newModel(0)
		if m.Init() == nil {
			t.Fatal("Init must start the clock")
		}
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
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
	t.Run("happy: the view fills the window exactly", func(t *testing.T) {
		m := newModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 20 {
			t.Fatalf("view has %d lines for a 20-line window", got)
		}
		mm, _ = m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 34 {
			t.Fatalf("view has %d lines for a 34-line window", got)
		}
	})
	t.Run("unhappy: q and ctrl+c close the house from any point", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			runeKey('q'),
			tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		} {
			m := newModel(0)
			_ = m.View()
			m = frames(m, 10)
			_, cmd := m.Update(msg)
			if cmd == nil {
				t.Fatalf("%v must quit", msg)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("%v must issue tea.Quit", msg)
			}
		}
	})
}
