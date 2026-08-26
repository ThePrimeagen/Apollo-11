package main

// Demo harness tests, written first: cmd/lunarcloseup runs the lunar
// lander close-up screenplay — the composable four-scene bill from
// shows/lunarcloseup. The house opens on "Lunar Lander Close-Up"
// (arrival copy, cold engine). Space cuts to "fire" (booster on,
// stars slow 60% over 5s), then "fall" (north-facing drop), then
// "landing" (huge moon horizon, lander comes down onto it). Space on
// the last scene ends the show. q and ctrl+c quit anywhere. The view
// is the rendered screen plus one status line, always exactly
// window-height lines.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
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

func hasFire(v string) bool {
	return strings.ContainsAny(v, "⠁⠒⠶")
}

func TestLunarCloseUpScreenplay(t *testing.T) {
	t.Run("happy: the house opens on scene 1/4 — Lunar Lander Close-Up, under stars", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"1/4", "Lunar Lander Close-Up", "space", "quit"} {
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
		_ = m.View()
		m = frames(m, 60)
		if strings.ContainsRune(m.View().Content, '▌') {
			t.Fatal("the hold is still running — the craft must stay offstage")
		}
		m = frames(m, 120)
		v := m.View().Content
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("after the hold the hull must be on screen")
		}
		if hasFire(v) {
			t.Fatal("the close-up must fly a dark engine — no booster fire yet")
		}
	})
	t.Run("happy: space cuts to scene 2/4 — fire, booster on", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 30)
		m = press(m, space())
		v := m.View().Content
		for _, want := range []string{"2/4", "fire"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the fire scene is missing %q", want)
			}
		}
		m = frames(m, 15)
		v = m.View().Content
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("the fire scene parks the west-facing craft")
		}
		if !hasFire(v) {
			t.Fatal("the fire scene must light the booster")
		}
	})
	t.Run("happy: space cuts to scene 3/4 — north-facing fall", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, space())
		m = press(m, space())
		opening := m.View().Content
		for _, want := range []string{"3/4", "fall"} {
			if !strings.Contains(opening, want) {
				t.Fatalf("the fall scene is missing %q", want)
			}
		}
		if strings.ContainsRune(opening, '▌') {
			t.Fatal("the fall must not open on the west-facing hull")
		}
		m = frames(m, int(lander.DropSeconds*30/2))
		mid := m.View().Content
		if !strings.ContainsRune(mid, '▟') {
			t.Fatal("mid-fall the north hull must be on stage")
		}
		if !hasFire(mid) {
			t.Fatal("the falling craft must keep the booster lit")
		}
	})
	t.Run("happy: space cuts to scene 4/4 — landing on the moon horizon", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, space())
		m = press(m, space())
		m = press(m, space())
		v := m.View().Content
		for _, want := range []string{"4/4", "landing"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the landing scene is missing %q", want)
			}
		}
		if !strings.Contains(v, "48;5;") {
			t.Fatal("the landing scene must show the moon as a background floor")
		}
		if strings.ContainsRune(v, '▓') {
			t.Fatal("the moon floor must be a background color, not terrain glyphs covering the fire")
		}
		m = frames(m, int((lander.LandSeconds-0.5)*30))
		near := m.View().Content
		if !strings.ContainsRune(near, '▟') {
			t.Fatal("near the pad the north hull must be on stage")
		}
		if !hasFire(near) {
			t.Fatal("the plume must still be lit as the craft comes in")
		}
		m = frames(m, 30)
		landed := m.View().Content
		if !strings.ContainsRune(landed, '▟') {
			t.Fatal("at touchdown the north hull must sit on the surface")
		}
		if hasFire(landed) {
			t.Fatal("at touchdown the booster must cut off")
		}
	})
	t.Run("happy: space on the last scene ends the show — nothing left", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, space())
		m = press(m, space())
		m = press(m, space())
		mm, cmd := m.Update(space())
		if cmd == nil {
			t.Fatal("space after the last scene must end the show")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("the end of the bill must issue tea.Quit")
		}
		_ = mm
	})
	t.Run("happy: the plain-rune spacebar cuts and ends the show too", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey(' '))
		if !strings.Contains(m.View().Content, "2/4") {
			t.Fatal("a rune spacebar must cut to the fire scene")
		}
		m = press(m, runeKey(' '))
		m = press(m, runeKey(' '))
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
		v := m.View().Content
		if strings.Contains(v, "arrival") {
			t.Fatal("the close-up must not open as the premiere's arrival card")
		}
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in the close-up")
		}
		m = press(m, space())
		m = press(m, space())
		m = press(m, space())
		v = m.View().Content
		if strings.Contains(v, "the end") || strings.Contains(v, "THE END") {
			t.Fatal("the end card does not appear in the close-up")
		}
		if strings.Contains(v, "dsky") {
			t.Fatal("the DSKY scene does not appear in the close-up")
		}
	})
}
