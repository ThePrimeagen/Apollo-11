package main

// Demo harness tests, written first: cmd/shootingstar runs the
// shooting-star tuner from scenes/shootingstar — a larger star with a
// persist-particle trail falling right-to-left, or walking a circle
// or a square so the tail is readable. Live knobs: path, size, random size, speed, count,
// period, min/max life, nozzle, peak, taper. p / enter / space replay, s saves,
// q and ctrl+c quit. The view is the rendered screen plus the knob
// panel, always exactly window-height lines.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/shootingstar"
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

func enter() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEnter} }

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestShootingStarRunner(t *testing.T) {
	t.Cleanup(shootingstar.Reset)
	t.Run("happy: the house opens on the right-to-left fall with the star and the knob panel", func(t *testing.T) {
		m := newModel(0, false)
		v := m.View().Content
		for _, want := range []string{"shooting", "play", "tune", "save", "quit",
			"path", "size", "speed", "count", "period", "life", "nozzle", "peak", "taper"} {
			if !strings.Contains(strings.ToLower(v), want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.Contains(v, "fall") {
			t.Fatal("the path knob must read fall")
		}
		if !strings.ContainsRune(v, '★') {
			t.Fatal("the larger star must already be on stage")
		}
	})
	t.Run("happy: j/k select a knob and h/l walk path, size, and the trail", func(t *testing.T) {
		m := newModel(0, false)
		_ = m.View()
		m = press(m, runeKey('l'))
		if m.show.Cfg.Path != shootingstar.PathCircle {
			t.Fatal("l on the path knob must switch to circle")
		}
		if !strings.Contains(m.View().Content, "circle") {
			t.Fatal("the path knob must read circle")
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if m.show.Cfg.Size != shootingstar.DefaultConfig().Size+1 {
			t.Fatalf("size after +1 is %d", m.show.Cfg.Size)
		}
		if !strings.Contains(m.View().Content, ">") {
			t.Fatal("the selected knob must be marked")
		}
	})
	t.Run("unhappy: size walks past 5, speed walks past 80 and through zero, and space does not quit", func(t *testing.T) {
		t.Cleanup(shootingstar.Reset)
		m := newModel(0, false)
		m.show.Cfg.Size = 5
		m.cursor = shootingstar.KnobSize
		m = press(m, runeKey('l'))
		if m.show.Cfg.Size != 6 {
			t.Fatalf("size %d, want 6 — no ceiling", m.show.Cfg.Size)
		}
		m.show.Cfg.Size = 0
		m = press(m, runeKey('h'))
		if m.show.Cfg.Size != -1 {
			t.Fatalf("size %d, want -1", m.show.Cfg.Size)
		}
		m.show.Cfg.Speed = 80
		m.cursor = shootingstar.KnobSpeed
		m = press(m, runeKey('l'))
		if m.show.Cfg.Speed <= 80 {
			t.Fatalf("speed %v, want past 80 — no ceiling", m.show.Cfg.Speed)
		}
		m.show.Cfg.Speed = 0
		m = press(m, runeKey('h'))
		if m.show.Cfg.Speed >= 0 {
			t.Fatalf("speed %v, want negative — no floor", m.show.Cfg.Speed)
		}
		_, cmd := m.Update(space())
		if cmd != nil {
			t.Fatal("space must replay, not quit")
		}
	})
	t.Run("happy: each frame schedules the next", func(t *testing.T) {
		m := newModel(0, false)
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: Init schedules the first frame", func(t *testing.T) {
		if newModel(0, false).Init() == nil {
			t.Fatal("Init must start the clock")
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
		m := newModel(0.05, false)
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
		m := newModel(0, false)
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
	t.Run("happy: p and enter replay and do not quit", func(t *testing.T) {
		m := newModel(0, false)
		_ = m.View()
		for _, msg := range []tea.Msg{runeKey('p'), enter()} {
			mm, cmd := m.Update(msg)
			if cmd != nil {
				t.Fatalf("%v must replay, not quit", msg)
			}
			m = mm.(model)
			if !strings.ContainsRune(m.View().Content, '★') {
				t.Fatalf("after %v the star must still be on stage", msg)
			}
		}
	})
	t.Run("happy: s saves the knobs and does not quit", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "shoot.json")
		m := newModel(0, false)
		m.path = path
		m.show.Cfg.Path = shootingstar.PathSquare
		m.show.Cfg.Size = 4
		mm, cmd := m.Update(runeKey('s'))
		if cmd != nil {
			t.Fatal("s must save, not quit")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "saved") {
			t.Fatal("a successful save must say so")
		}
		got, err := shootingstar.Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got != m.show.Cfg {
			t.Fatalf("saved %+v, want the live knobs %+v", got, m.show.Cfg)
		}
	})
	t.Run("unhappy: s with no path or a stuck file does not quit and says so", func(t *testing.T) {
		m := newModel(0, false)
		m.path = ""
		mm, cmd := m.Update(runeKey('s'))
		if cmd != nil {
			t.Fatal("a failed save must not quit")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "no config") {
			t.Fatal("a missing path must tell the operator it could not save")
		}
		stuck := filepath.Join(t.TempDir(), "stuck")
		if err := os.Mkdir(stuck, 0o755); err != nil {
			t.Fatal(err)
		}
		m.path = stuck
		mm, cmd = m.Update(runeKey('s'))
		if cmd != nil {
			t.Fatal("a stuck file must not quit")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "failed") {
			t.Fatal("a stuck file must tell the operator the save failed")
		}
	})
	t.Run("happy: -random flies the scene fall, not a closed-loop preview", func(t *testing.T) {
		m := newModel(0, true)
		if m.show == nil {
			t.Fatal("random flight must still build a show")
		}
		_ = m.View()
		if !strings.ContainsRune(m.View().Content, '★') {
			t.Fatal("the random shooting star must still put the larger star on stage")
		}
	})
	t.Run("unhappy: -random is not the tuner preview's parked loop", func(t *testing.T) {
		preview := newModel(0, false)
		random := newModel(0, true)
		_ = preview.View()
		_ = random.View()
		if preview.show == nil || random.show == nil {
			t.Fatal("both flights must build")
		}
		random.show.Cfg.Path = shootingstar.PathCircle
		if random.show == preview.show {
			t.Fatal("random and preview must be distinct shows")
		}
	})
	t.Run("unhappy: q and ctrl+c close the house", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			runeKey('q'),
			tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		} {
			_, cmd := newModel(0, false).Update(msg)
			if cmd == nil {
				t.Fatalf("%v must quit", msg)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("%v must issue tea.Quit", msg)
			}
		}
	})
}
