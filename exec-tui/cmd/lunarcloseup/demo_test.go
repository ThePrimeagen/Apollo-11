package main

// Demo harness tests, written first: cmd/lunarcloseup runs the lunar
// lander close-up screenplay — the composable one-scene bill from
// shows/lunarcloseup, a copy of the premiere's arrival. The house
// opens on "Lunar Lander Close-Up": three seconds of drifting sky,
// then the zoomed-in craft slides in from the right wing over a
// starfield that translates with it — hull only, cold engine — parks
// and bobbles. Space on the last (only) scene ends the show — there
// is nothing left, so the program quits. q and ctrl+c quit anywhere.
// The view is the rendered screen plus one status line, always
// exactly window-height lines.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
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

func space() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "} }

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func hasStar(v string) bool {
	for _, g := range stars.Glyphs {
		if strings.ContainsRune(v, g) {
			return true
		}
	}
	return false
}

func TestLunarCloseUpScreenplay(t *testing.T) {
	t.Run("happy: the house opens on scene 1/1 — Lunar Lander Close-Up, under stars", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"1/1", "Lunar Lander Close-Up", "space", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !hasStar(v) {
			t.Fatal("the opening scene must show the starfield")
		}
		if strings.ContainsRune(v, '▌') {
			t.Fatal("the craft is still off the right wing at t=0")
		}
	})
	t.Run("happy: frames fly the craft in with a cold engine — hull, no fire", func(t *testing.T) {
		m := newModel(0)
		_ = m.View() // the opening paint stages the cast, as bubbletea does
		m = frames(m, 60)
		if strings.ContainsRune(m.View().Content, '▌') {
			t.Fatal("the hold is still running — the craft must stay offstage")
		}
		m = frames(m, 120) // three more seconds: hold ends and the fly-in is well under way
		if m.elapsed < 5.9 || m.elapsed > 6.1 {
			t.Fatalf("elapsed %f after 180 frames, want ~6.0", m.elapsed)
		}
		v := m.View().Content
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("after the hold the hull must be on screen")
		}
		if strings.ContainsAny(v, "⠁⠒⠶▒") {
			t.Fatal("the close-up must fly a dark engine — no booster fire yet")
		}
	})
	t.Run("happy: the arrival sky slides with the craft — same cells, same ease", func(t *testing.T) {
		hold := lander.FlyInHoldSeconds
		for _, w := range []int{40, 72, 120} {
			for _, sceneT := range []float64{0, 2, hold, hold + 1, hold + lander.FlyInSeconds, hold + lander.FlyInSeconds + 3} {
				flyT := sceneT - hold
				_, c0 := lander.FlightPath(w, 28, 0)
				_, c := lander.FlightPath(w, 28, flyT)
				got := stars.SlideOffset(w, lander.BodyCols, flyT, lander.FlyInSeconds)
				if c0-c != got {
					t.Fatalf("w=%d scene t=%.1f (fly t=%.1f) ship traveled %d, sky slide %d", w, sceneT, flyT, c0-c, got)
				}
			}
		}
	})
	t.Run("happy: space on the last scene ends the show — nothing left", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		mm, cmd := m.Update(space())
		if cmd == nil {
			t.Fatal("space after the last scene must end the show")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("the end of the bill must issue tea.Quit")
		}
		_ = mm
	})
	t.Run("happy: the plain-rune spacebar ends the show too", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		_, cmd := m.Update(runeKey(' '))
		if cmd == nil {
			t.Fatal("a rune spacebar must also end the show on the last scene")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("the end of the bill must issue tea.Quit")
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
	t.Run("unhappy: q and ctrl+c close the house from any scene", func(t *testing.T) {
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
	t.Run("unhappy: the close-up never plays the premiere's other cards", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 180)
		v := m.View().Content
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in the close-up")
		}
		if strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("the moon's craft does not appear in the close-up")
		}
		if strings.Contains(v, "2/1") || strings.Contains(v, "1/4") {
			t.Fatal("the close-up is a one-scene bill, not the four-scene premiere")
		}
	})
}
