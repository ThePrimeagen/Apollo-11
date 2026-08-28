package main

// Demo harness tests, written first: cmd/inverse runs 03. Inverse
// Walkthrough — the composable three-scene bill from shows/inverse.
// The house opens on "liftoff": the lander parked on the moon floor
// ignites, kicks the mirrored dust, and climbs off the top; the empty
// moon holds. Space cuts to "engines on": the west-facing craft
// parked at center, tail fire burning, bobbling. Space cuts to
// "engines off": the same craft, engine out, bobbling ad infinitum.
// Space on the last scene ends the show. q and ctrl+c quit. The view
// is the sky plus one status line, always exactly window-height lines.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
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

func hasFire(v string) bool {
	return strings.ContainsAny(v, "⠁⠒⠶")
}

func cutOnce(t *testing.T, m model) model {
	t.Helper()
	mm, cmd := m.Update(space())
	if cmd != nil {
		t.Fatal("a cut with scenes left must not quit")
	}
	return mm.(model)
}

func TestInverseWalkthroughRunner(t *testing.T) {
	t.Run("happy: the house opens on scene 1/3 — liftoff on the moon floor", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"scene 1/3", "liftoff", "space next scene", "q quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.Contains(v, "48;5;") {
			t.Fatal("the liftoff must show the moon as a background floor")
		}
		if !strings.ContainsRune(v, '▟') {
			t.Fatal("at t=0 the north hull must already sit on the pad")
		}
		if strings.ContainsRune(v, '▌') {
			t.Fatal("the sideways craft is still two cuts away")
		}
	})
	t.Run("happy: the craft ignites, climbs off the top, and the empty moon holds for the cut", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, int(1.9*30))
		burning := m.View().Content
		if !strings.ContainsRune(burning, '▟') || !hasFire(burning) {
			t.Fatal("during ignition the hull must burn on the pad")
		}
		m = frames(m, int((liftoff.LiftAt+liftoff.RiseSeconds)*30))
		held := m.View().Content
		if strings.ContainsRune(held, '▟') {
			t.Fatal("past lift-at plus rise the craft must have cleared the top")
		}
		if !strings.Contains(held, "scene 1/3") {
			t.Fatal("the liftoff holds its empty moon until the audience cuts")
		}
	})
	t.Run("happy: space walks liftoff → engines on → engines off, then ends the show", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = cutOnce(t, m)
		v := m.View().Content
		if !strings.Contains(v, "scene 2/3") || !strings.Contains(v, "engines on") {
			t.Fatal("the first cut must land on engines on")
		}
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("engines on opens with the west hull parked")
		}
		m = frames(m, 20)
		if !hasFire(m.View().Content) {
			t.Fatal("engines on must burn the tail fire")
		}
		m = cutOnce(t, m)
		v = m.View().Content
		if !strings.Contains(v, "scene 3/3") || !strings.Contains(v, "engines off") {
			t.Fatal("the second cut must land on engines off")
		}
		m = frames(m, 30)
		v = m.View().Content
		if hasFire(v) {
			t.Fatal("engines off flies a cold engine")
		}
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("engines off holds the parked craft ad infinitum")
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
	t.Run("unhappy: the inverse walkthrough is not the forward one", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		if strings.Contains(v, "pause") || strings.Contains(v, "1/5") {
			t.Fatal("the inverse walkthrough must not open as the forward walkthrough")
		}
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in the inverse walkthrough")
		}
	})
}
