package main

// Demo harness tests, written first: cmd/explorer runs the explorer
// scene from scenes/explorer — the big IE logo parked at center stage
// under the twinkling sky, plus the scene's own shooting star — and
// is the editable screen for its knobs: min/max cycle (250ms steps),
// min/max fade (50ms steps), and the ten star knobs the shooting-star
// tuner walks (size, random size, speed, count, period, min/max life,
// nozzle, peak, taper). h/l retune LIVE — the sky and the flying
// meteor read the knobs on the next frame, no replay needed — while
// p / enter / space replay from the top, s saves to
// scenes/explorer/config.json, q and ctrl+c quit. The view is the
// rendered stage plus the knob panel, always exactly window-height
// lines.

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/explorer"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/shootingstar"
)

func cleanup() {
	explorer.Reset()
	stars.ResetTwinkle()
	shootingstar.Reset()
}

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

func TestExplorerRunner(t *testing.T) {
	t.Cleanup(cleanup)
	t.Run("happy: the house opens on the big logo, the stars, and the four knobs", func(t *testing.T) {
		cleanup()
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"big e", "play", "tune", "save", "quit",
			"min cycle", "max cycle", "min fade", "max fade",
			"star size", "star random size", "star speed", "star count",
			"star period", "star min life", "star max life", "star nozzle",
			"star peak", "star taper"} {
			if !strings.Contains(strings.ToLower(v), want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.Contains(v, "38;5;33") {
			t.Fatal("the blue e must be on stage")
		}
		if !strings.Contains(v, "38;5;220") {
			t.Fatal("the golden swoosh must be on stage")
		}
		if !strings.ContainsAny(v, "·˚*✦") {
			t.Fatal("the logo plays under the stars")
		}
		if !strings.Contains(v, "★") {
			t.Fatal("one shooting star must already be on stage")
		}
	})
	t.Run("happy: the shooting star flies once and does not return", func(t *testing.T) {
		cleanup()
		m := newModel(0)
		_ = m.View()
		seen := strings.Contains(m.View().Content, "★")
		m = frames(m, 40)
		if strings.Contains(m.View().Content, "★") {
			seen = true
		}
		if !seen {
			t.Fatal("test premise: the shooting star must fly")
		}
		m = frames(m, 200)
		if strings.Contains(m.View().Content, "★") {
			t.Fatal("after the crossing the shooting star must leave the stage")
		}
		m = frames(m, 90)
		if strings.Contains(m.View().Content, "★") {
			t.Fatal("a second shooting star must not appear")
		}
	})
	t.Run("happy: j/k select a knob and h/l walk it by its step", func(t *testing.T) {
		cleanup()
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey('l'))
		want := explorer.DefaultConfig().MinCycleSeconds + explorer.CycleStepSeconds
		if got := m.show.Cfg.MinCycleSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("min cycle after +1 is %v, want %v", got, want)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		want = explorer.DefaultConfig().MaxCycleSeconds + explorer.CycleStepSeconds
		if got := m.show.Cfg.MaxCycleSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("max cycle after +1 is %v, want %v", got, want)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('h'))
		want = explorer.DefaultConfig().MinFadeSeconds - explorer.FadeStepSeconds
		if got := m.show.Cfg.MinFadeSeconds; math.Abs(got-want) > 1e-9 {
			t.Fatalf("min fade after -1 is %v, want %v", got, want)
		}
		m = press(m, runeKey('k'))
		m = press(m, runeKey('k'))
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.MinCycleSeconds; math.Abs(got-explorer.DefaultConfig().MinCycleSeconds) > 1e-9 {
			t.Fatalf("k must walk back to the min cycle knob, it reads %v", got)
		}
		if !strings.Contains(m.View().Content, ">") {
			t.Fatal("the selected knob must be marked")
		}
	})
	t.Run("happy: a nudge retunes the sky live — no replay needed", func(t *testing.T) {
		cleanup()
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey('l'))
		if got := stars.ActiveTwinkle(); got != m.show.Cfg.Twinkle() {
			t.Fatalf("after the nudge the sky breathes %+v, want the live knobs %+v", got, m.show.Cfg.Twinkle())
		}
	})
	t.Run("happy: j/k reach the star knobs and h/l walk them by the shooting-star steps", func(t *testing.T) {
		cleanup()
		m := newModel(0)
		_ = m.View()
		for i := 0; i < int(explorer.KnobStarSpeed); i++ {
			m = press(m, runeKey('j'))
		}
		m = press(m, runeKey('l'))
		want := explorer.DefaultConfig().Star.Speed + shootingstar.StepSpeed
		if got := m.show.Cfg.Star.Speed; math.Abs(got-want) > 1e-9 {
			t.Fatalf("star speed after +1 is %v, want %v", got, want)
		}
		m = press(m, runeKey('k'))
		m = press(m, runeKey('k'))
		m = press(m, runeKey('l'))
		if got, want := m.show.Cfg.Star.Size, explorer.DefaultConfig().Star.Size+1; got != want {
			t.Fatalf("star size after +1 is %v, want %v", got, want)
		}
		if !strings.Contains(strings.ToLower(m.View().Content), "star size") {
			t.Fatal("the panel must show the selected star knob")
		}
	})
	t.Run("happy: a star nudge retunes the flying meteor live — no replay needed", func(t *testing.T) {
		cleanup()
		m := newModel(0)
		_ = m.View()
		if !strings.Contains(m.View().Content, "★") {
			t.Fatal("test premise: the meteor opens on stage")
		}
		for i := 0; i < int(explorer.KnobStarSpeed); i++ {
			m = press(m, runeKey('j'))
		}
		for i := 0; i < 13; i++ {
			m = press(m, runeKey('h'))
		}
		if got := m.show.Cfg.Star.Speed; math.Abs(got-2) > 1e-9 {
			t.Fatalf("test premise: thirteen h presses walk the stock speed to 2, got %v", got)
		}
		m = frames(m, 90)
		if !strings.Contains(m.View().Content, "★") {
			t.Fatal("slowed to speed 2 the meteor must still be crossing three seconds in — the nudge must reach the flying star")
		}
	})
	t.Run("unhappy: k off the top wraps onto the last star knob", func(t *testing.T) {
		cleanup()
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey('k'))
		m = press(m, runeKey('l'))
		want := explorer.DefaultConfig().Star.Taper + shootingstar.StepTaper
		if got := m.show.Cfg.Star.Taper; math.Abs(got-want) > 1e-9 {
			t.Fatalf("star taper after wrap and +1 is %v, want %v", got, want)
		}
		if got := m.show.Cfg.Twinkle(); got != explorer.DefaultConfig().Twinkle() {
			t.Fatalf("the wrap must not touch the twinkle, got %+v", got)
		}
	})
	t.Run("happy: p, enter and space replay and do not quit", func(t *testing.T) {
		cleanup()
		m := newModel(0)
		_ = m.View()
		for _, msg := range []tea.Msg{runeKey('p'), enter(), space()} {
			mm, cmd := m.Update(msg)
			if cmd != nil {
				t.Fatalf("%v must replay, not quit", msg)
			}
			m = mm.(model)
			if !strings.Contains(m.View().Content, "38;5;33") {
				t.Fatalf("after %v the logo must still be on stage", msg)
			}
		}
	})
	t.Run("happy: s saves the knobs and does not quit", func(t *testing.T) {
		cleanup()
		path := filepath.Join(t.TempDir(), "explorer.json")
		m := newModel(0)
		m.path = path
		m.show.Cfg.MinCycleSeconds = 1.5
		m.show.Cfg.MaxCycleSeconds = 9.25
		m.show.Cfg.MinFadeSeconds = 0.25
		m.show.Cfg.MaxFadeSeconds = 2.5
		m.show.Cfg.Star.Size = 3
		m.show.Cfg.Star.Speed = 44
		m.show.Cfg.Star.Taper = 0.5
		mm, cmd := m.Update(runeKey('s'))
		if cmd != nil {
			t.Fatal("s must save, not quit")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "saved") {
			t.Fatal("a successful save must say so")
		}
		got, err := explorer.Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got != m.show.Cfg {
			t.Fatalf("saved %+v, want the live knobs %+v", got, m.show.Cfg)
		}
		if explorer.Active() != m.show.Cfg {
			t.Fatal("s must also make the saved knobs active")
		}
	})
	t.Run("unhappy: s with no path or a stuck file does not quit and says so", func(t *testing.T) {
		cleanup()
		m := newModel(0)
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
	t.Run("happy: the view fills the window even when the stage runs short", func(t *testing.T) {
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
	t.Run("happy: the twinkle plays on the frame clock — stars fade as the frames burn", func(t *testing.T) {
		cleanup()
		m := newModel(0)
		m.show.Cfg.MinCycleSeconds, m.show.Cfg.MaxCycleSeconds = 1, 1
		m.show.Cfg.MinFadeSeconds, m.show.Cfg.MaxFadeSeconds = 0.25, 0.25
		m = press(m, space())
		before := m.View().Content
		changed := false
		for i := 0; i < 8 && !changed; i++ {
			m = frames(m, 8)
			if m.View().Content != before {
				changed = true
			}
		}
		if !changed {
			t.Fatal("a second of one-second cycles must move some star")
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
