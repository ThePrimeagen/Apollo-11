package main

// Demo harness tests, written first: screenplay-lab premieres a
// two-scene bill on the shared screen. Scene one, "arrival": a drifting
// starfield with the westbound craft sliding in from the right wing to
// park and bobble at center stage. Space cuts to scene two, "the end":
// the height-5 banner card. Space on the final scene holds; q and
// ctrl+c close the house. The view is the rendered screen plus one
// status line, always exactly window-height lines.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/stars-lab/stars"
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

func TestPremiere(t *testing.T) {
	t.Run("happy: the house opens on scene 1/2, arrival, under stars", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"1/2", "arrival", "space", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !hasStar(v) {
			t.Fatal("the opening scene must show the starfield")
		}
		if strings.ContainsRune(v, '▓') {
			t.Fatal("the craft is still off the right wing at t=0")
		}
	})
	t.Run("happy: frames run the clock and fly the craft in, fire and all", func(t *testing.T) {
		m := frames(newModel(0), 90) // three seconds
		if m.elapsed < 2.9 || m.elapsed > 3.1 {
			t.Fatalf("elapsed %f after 90 frames, want ~3.0", m.elapsed)
		}
		v := m.View().Content
		if !strings.ContainsRune(v, '▓') {
			t.Fatal("three seconds in, the hull must be on screen")
		}
		if !strings.ContainsAny(v, "⠁⠒⠶▒") {
			t.Fatal("the booster fire must be burning behind the craft")
		}
	})
	t.Run("happy: each frame schedules the next", func(t *testing.T) {
		m := newModel(0)
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: space cuts to scene 2/2 — THE END, centered", func(t *testing.T) {
		m := frames(newModel(0), 30)
		m = press(m, space())
		v := m.View().Content
		for _, want := range []string{"2/2", "the end", "___"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the end card is missing %q", want)
			}
		}
		if !hasStar(v) {
			t.Fatal("the end card still plays under the stars")
		}
		if strings.ContainsRune(v, '▓') {
			t.Fatal("the craft does not appear in the end card")
		}
	})
	t.Run("happy: the cut restarts the clock for the new scene's cast", func(t *testing.T) {
		m := frames(newModel(0), 30)
		m = press(m, space())
		before := m.View().Content
		m = frames(m, 30)
		if m.View().Content == before {
			t.Fatal("the end card's sky must drift on after the cut")
		}
	})
	t.Run("unhappy: space on the final scene holds the card", func(t *testing.T) {
		m := press(newModel(0), space())
		m = press(m, space())
		m = press(m, runeKey(' '))
		v := m.View().Content
		if !strings.Contains(v, "2/2") || !strings.Contains(v, "the end") {
			t.Fatal("extra spaces must hold on the final scene")
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
	t.Run("happy: Init schedules the first frame", func(t *testing.T) {
		if newModel(0).Init() == nil {
			t.Fatal("Init must start the clock")
		}
	})
}

func TestApplySky(t *testing.T) {
	t.Run("happy: a tuned stars.json is applied as the active sky", func(t *testing.T) {
		t.Cleanup(stars.ResetSky)
		path := filepath.Join(t.TempDir(), "stars.json")
		cfg := stars.SkyConfig{Delay: []int{1, 1, 1, 1}, Density: []int{99, 99, 99, 99}}
		if err := cfg.Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		used, err := applySky(path)
		if err != nil || !used {
			t.Fatalf("applySky = %v/%v, want used and no error", used, err)
		}
		if stars.ActiveSky().DensityLayers() != [4]int{99, 99, 99, 99} {
			t.Fatal("the premiere must fly the tuned sky")
		}
	})
	t.Run("happy: a missing file quietly keeps the stock sky", func(t *testing.T) {
		used, err := applySky(filepath.Join(t.TempDir(), "nowhere.json"))
		if err != nil || used {
			t.Fatalf("applySky = %v/%v, want quietly unused", used, err)
		}
	})
	t.Run("unhappy: a broken file is an error worth stopping for", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "stars.json")
		if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := applySky(path); err == nil {
			t.Fatal("a broken sky file must surface its error")
		}
	})
}

func TestForcedColorProfile(t *testing.T) {
	t.Run("happy: CLICOLOR_FORCE forces ANSI256 for tapes and CI", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "1")
		p, ok := forcedColorProfile()
		if !ok || p != colorprofile.ANSI256 {
			t.Fatalf("got %v/%v, want ANSI256 forced", p, ok)
		}
	})
	t.Run("unhappy: without the flag, detection is left alone", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "")
		if _, ok := forcedColorProfile(); ok {
			t.Fatal("an empty CLICOLOR_FORCE must not force a profile")
		}
	})
}
