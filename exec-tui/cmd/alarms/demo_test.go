package main

// Demo harness tests, written first: cmd/alarms runs the Alarms
// scene standalone. The house opens on find_free_core_set()'s first
// line under a one-line marquee, and the scene plays itself: the
// core-set allocation revealed and walked to a free set, then to the
// 1202 throw under its PROG ALARM chip; the vac-area allocation
// walked the same way to 1201; the final hold naming both codes.
// space (or p, or enter) replays from the top, -seconds brings the
// curtain down on time, q and ctrl+c quit anywhere, and the view is
// always exactly window-height lines.

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/alarms"
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

func TestAlarmsDemoOpens(t *testing.T) {
	t.Run("happy: the house opens on the core-set function with the marquee", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		for _, want := range []string{"find_free_core_set()", "Alarms", "replay", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
	})
	t.Run("unhappy: the vac act and the alarm chips are nowhere at the curtain", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		if strings.Contains(v, "find_free_vac_area()") {
			t.Fatal("the vac act must wait its turn")
		}
		if strings.Contains(v, "PROG ALARM") {
			t.Fatal("no alarm before the loop falls off its end")
		}
	})
}

func TestAlarmsDemoPlays(t *testing.T) {
	t.Run("happy: the acts advance on their own clock — 1202, then the vac act, then 1201", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, toSeconds(alarms.CoreAlarmAt()+0.3))
		v := plain(m)
		if !strings.Contains(v, "PROG ALARM 1202") {
			t.Fatal("the full core pass must raise the 1202 chip")
		}
		m = frames(m, toSeconds(alarms.VACAlarmAt()-alarms.CoreAlarmAt()-0.3+0.3))
		v = plain(m)
		if !strings.Contains(v, "PROG ALARM 1201") || !strings.Contains(v, "find_free_vac_area()") {
			t.Fatal("the full vac pass must raise the 1201 chip")
		}
		if strings.Contains(v, "find_free_core_set()") {
			t.Fatal("the core card must be long gone by the vac alarm")
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
			m = frames(m, toSeconds(alarms.VACStart()+1))
			if strings.Contains(plain(m), "find_free_core_set()") {
				t.Fatal("test premise: the core act must be over before the replay")
			}
			m = press(m, msg)
			if !strings.Contains(plain(m), "find_free_core_set()") {
				t.Fatalf("%v must rewind to the core act", msg)
			}
		}
	})
	t.Run("unhappy: unknown keys neither replay nor quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, toSeconds(alarms.VACStart()+1))
		mm, cmd := m.Update(runeKey('z'))
		if cmd != nil {
			t.Fatal("an unknown key must do nothing")
		}
		m = mm.(model)
		if strings.Contains(plain(m), "find_free_core_set()") {
			t.Fatal("an unknown key must not rewind the show")
		}
	})
}

func TestAlarmsDemoHouseRules(t *testing.T) {
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
