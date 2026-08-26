package main

// Demo harness tests, written first: cmd/america runs the America
// scene standalone. The house opens on pure black with the America
// marquee on the status line; the full-screened flag fades in slowly,
// and once it is fully in, the very large eagle crosses the stage
// right to left with the flag flying beneath. p (or space, or enter)
// replays from the top — back to black. -seconds brings the curtain
// down on time, q and ctrl+c quit anywhere, and the view is always
// exactly window-height lines.

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/america"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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

func fadeFrames() int { return int(america.FadeSeconds*30) + 15 }

func quarterCross() int { return int(america.CrossSeconds / 4 * 30) }

// eagleLeft finds the leftmost stage cell that is neither flag field
// (space) nor a canton star — the eagle — across every line above the
// status row, ANSI stripped.
func eagleLeft(v string) (int, bool) {
	lines := strings.Split(ansiPat.ReplaceAllString(v, ""), "\n")
	if len(lines) < 2 {
		return 0, false
	}
	left, ok := 1<<30, false
	for _, line := range lines[:len(lines)-1] {
		for c, ch := range []rune(line) {
			if ch == ' ' || ch == '★' {
				continue
			}
			if c < left {
				left = c
			}
			ok = true
		}
	}
	return left, ok
}

func TestAmericaDemoOpens(t *testing.T) {
	t.Run("happy: the house opens on pure black with the America marquee", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"America", "replay", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
		if !strings.Contains(v, "48;5;16m") {
			t.Fatal("the stage must open painted black — the fade starts from black, not from nothing")
		}
		if strings.Contains(v, "48;5;160m") {
			t.Fatal("no red yet — the flag fades in from black")
		}
		if _, ok := eagleLeft(v); ok {
			t.Fatal("no eagle yet — the flag comes first")
		}
	})
	t.Run("unhappy: waiting a moment is still black — the fade is slow", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 3)
		if strings.Contains(m.View().Content, "48;5;160m") {
			t.Fatal("three frames in, the flag must not already be red")
		}
	})
}

func TestAmericaDemoFade(t *testing.T) {
	t.Run("happy: the flag is fully in after the fade", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, fadeFrames())
		v := m.View().Content
		for _, want := range []string{"48;5;160m", "48;5;18m", "★"} {
			if !strings.Contains(v, want) {
				t.Fatalf("after the fade the view is missing %q — red stripes, blue canton, stars", want)
			}
		}
	})
	t.Run("happy: then the eagle crosses, right to left", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, fadeFrames()+quarterCross())
		l1, ok := eagleLeft(m.View().Content)
		if !ok {
			t.Fatal("a quarter into the crossing the eagle must be on stage")
		}
		m = frames(m, quarterCross())
		l2, ok := eagleLeft(m.View().Content)
		if !ok {
			t.Fatal("halfway into the crossing the eagle must still be on stage")
		}
		if l2 >= l1 {
			t.Fatalf("the eagle must fly leftward: leftmost went %d -> %d", l1, l2)
		}
		if !strings.Contains(m.View().Content, "48;5;160m") {
			t.Fatal("the flag must keep flying beneath the eagle")
		}
	})
}

func TestAmericaDemoReplay(t *testing.T) {
	t.Run("happy: p replays from the top — back to black", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, fadeFrames())
		if !strings.Contains(m.View().Content, "48;5;160m") {
			t.Fatal("test premise: the flag must be in before the replay")
		}
		m = press(m, runeKey('p'))
		v := m.View().Content
		if strings.Contains(v, "48;5;160m") {
			t.Fatal("p must rewind the fade to black")
		}
		if !strings.Contains(v, "48;5;16m") {
			t.Fatal("after the replay the stage must be painted black again")
		}
	})
	t.Run("happy: space and enter replay too", func(t *testing.T) {
		for _, msg := range []tea.Msg{space(), runeKey(' '), tea.KeyPressMsg{Code: tea.KeyEnter}} {
			m := newModel(0)
			_ = m.View()
			m = frames(m, fadeFrames())
			m = press(m, msg)
			if strings.Contains(m.View().Content, "48;5;160m") {
				t.Fatalf("%v must replay from the top", msg)
			}
		}
	})
	t.Run("unhappy: unknown keys neither replay nor quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, fadeFrames())
		mm, cmd := m.Update(runeKey('z'))
		if cmd != nil {
			t.Fatal("an unknown key must do nothing")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "48;5;160m") {
			t.Fatal("an unknown key must not rewind the show")
		}
	})
}

func TestAmericaDemoHouseRules(t *testing.T) {
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
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 20 {
			t.Fatalf("view has %d lines for a 20-line window", got)
		}
		mm, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 32})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 32 {
			t.Fatalf("view has %d lines for a 32-line window", got)
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
