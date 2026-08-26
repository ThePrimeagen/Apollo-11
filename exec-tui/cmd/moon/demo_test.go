package main

// Demo harness tests, written first: the moon demo plays the descent
// orbit as its own one-scene bill — the pixelated moon under the
// dotted descent path, the gold marker riding it westward, the tuned
// starfield behind — so the scene can be run (and taped) without the
// rest of the premiere. q and ctrl+c quit; space has no next scene to
// cut to, so the show holds. The view is the rendered screen plus one
// status line, always exactly window-height lines.

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
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

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func hasStar(v string) bool {
	for _, g := range stars.Glyphs {
		if strings.ContainsRune(v, g) {
			return true
		}
	}
	return false
}

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// markerCell locates the gold marker in a rendered view, ANSI stripped
// so escape bytes never shift the column count.
func markerCell(v string) (row, col int, ok bool) {
	for r, line := range strings.Split(ansiPat.ReplaceAllString(v, ""), "\n") {
		for c, ch := range []rune(line) {
			if ch == moon.MarkerGlyph {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

func TestMoonDemo(t *testing.T) {
	t.Run("happy: the house opens on the descent orbit under stars", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"descent orbit", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.ContainsRune(v, '▓') {
			t.Fatal("the moon must fill the middle of the stage")
		}
		if !strings.ContainsRune(v, moon.RingGlyph) {
			t.Fatal("the dotted descent path must circle the moon")
		}
		if !strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("the gold marker must ride the descent path")
		}
		if !hasStar(v) {
			t.Fatal("the orbit plays under the stars")
		}
	})
	t.Run("happy: frames fly the marker along the ring", func(t *testing.T) {
		m := newModel(0)
		r0, c0, ok := markerCell(m.View().Content)
		if !ok {
			t.Fatal("no marker on the opening frame")
		}
		m = frames(m, 90) // three seconds: a quarter lap
		r1, c1, ok := markerCell(m.View().Content)
		if !ok {
			t.Fatal("the marker left the stage")
		}
		if r0 == r1 && c0 == c1 {
			t.Fatal("frames must fly the marker along the ring")
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
		mm, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 32})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 32 {
			t.Fatalf("view has %d lines for a 32-line window", got)
		}
	})
	t.Run("unhappy: space has no next scene — the show holds", func(t *testing.T) {
		m := newModel(0)
		before := m.View().Content
		m = press(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		m = press(m, runeKey(' '))
		if got := m.View().Content; got != before {
			t.Fatal("space must hold the one-scene bill exactly where it is")
		}
		if !strings.Contains(m.View().Content, "descent orbit") {
			t.Fatal("the orbit must still be on the marquee")
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
}
