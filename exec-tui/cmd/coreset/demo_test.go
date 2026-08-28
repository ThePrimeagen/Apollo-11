package main

// Demo harness tests, written first: cmd/coreset runs the Core Set
// scene standalone, and now it is the scene's tuner too. The house
// opens on the memory unit — both panels, tops aligned — over a
// marquee and the nine knob rows, and the scene plays itself: the
// drain, the move, the settle, the twelve-word anatomy, the priority
// zoom, the fifteen bits. j/k walk the knob cursor with wrap, h/l
// nudge the selected knob one 50ms step and the panel shows the new
// reading, s saves the knobs to the config path (and installs them as
// the Active timing), space (or p, or enter) replays from the top
// with whatever the knobs hold, -seconds brings the curtain down on
// time, q and ctrl+c quit anywhere, and the view is always exactly
// window-height lines.

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stock is the scene's default clock, the timeline every fresh model
// plays.
var stock = coreset.DefaultConfig()

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
	t.Run("happy: the house opens on the memory unit with the marquee and the knob rows", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		for _, want := range []string{"CORE SETS", "VAC AREAS", "Core Set", "replay", "quit", "save"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
		for k := coreset.Knob(0); k < coreset.KnobCount; k++ {
			if !strings.Contains(v, coreset.KnobLabel(k)) {
				t.Fatalf("the knob panel is missing %q", coreset.KnobLabel(k))
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
		m = frames(m, toSeconds(stock.WordsStart()+12*stock.WordBeat+0.3))
		v := plain(m)
		if !strings.Contains(v, "MPAC") || !strings.Contains(v, "PRIO") {
			t.Fatal("past the reveal the twelve-word bar must be on stage")
		}
		if strings.Contains(v, "VAC AREAS") {
			t.Fatal("the memory unit must be long gone by the anatomy act")
		}
		m = frames(m, toSeconds(stock.BitsStart()-stock.WordsStart()-12*stock.WordBeat-0.3+1))
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
			m = frames(m, toSeconds(stock.MoveStart()+0.5))
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
		m = frames(m, toSeconds(stock.MoveStart()+0.5))
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

func TestCoresetKnobPanel(t *testing.T) {
	t.Run("happy: j and k walk the cursor with wrap", func(t *testing.T) {
		m := newModel(0)
		if !strings.Contains(plain(m), "> "+coreset.KnobLabel(coreset.Knob(0))) {
			t.Fatal("the cursor must open on the first knob")
		}
		m = press(m, runeKey('j'))
		if !strings.Contains(plain(m), "> "+coreset.KnobLabel(coreset.Knob(1))) {
			t.Fatal("j must select the second knob")
		}
		m = press(m, runeKey('k'))
		m = press(m, runeKey('k'))
		if !strings.Contains(plain(m), "> "+coreset.KnobLabel(coreset.KnobCount-1)) {
			t.Fatal("k past the top must wrap to the last knob")
		}
		m = press(m, runeKey('j'))
		if !strings.Contains(plain(m), "> "+coreset.KnobLabel(coreset.Knob(0))) {
			t.Fatal("j past the bottom must wrap to the first knob")
		}
	})
	t.Run("happy: l and h retune the selected knob and the panel shows the new reading", func(t *testing.T) {
		m := newModel(0)
		before := m.show.Cfg.Value(coreset.Knob(0))
		m = press(m, runeKey('l'))
		after := m.show.Cfg.Value(coreset.Knob(0))
		if after <= before {
			t.Fatalf("l must nudge the knob up: %v then %v", before, after)
		}
		if !strings.Contains(plain(m), m.show.Cfg.Display(coreset.Knob(0))) {
			t.Fatalf("the panel must show the new reading %q", m.show.Cfg.Display(coreset.Knob(0)))
		}
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.Value(coreset.Knob(0)); got != before {
			t.Fatalf("h must bring the knob back to %v, got %v", before, got)
		}
	})
	t.Run("happy: a nudged knob retimes the next replay", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		for i := 0; i < 100; i++ {
			m = press(m, runeKey('h'))
		}
		if got := m.show.Cfg.UnitSeconds; got != 0 {
			t.Fatalf("test premise: a hundred h presses floor the unit hold, got %v", got)
		}
		m = press(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		m = frames(m, toSeconds(0.3))
		v := plain(m)
		if !strings.Contains(v, coreset.CaptionFade) || strings.Contains(v, coreset.CaptionUnit) {
			t.Fatal("with a zero unit hold the replay must open straight on the drain")
		}
	})
	t.Run("unhappy: the floors hold on the panel too", func(t *testing.T) {
		m := newModel(0)
		for i := 0; i < 3; i++ {
			m = press(m, runeKey('j'))
		}
		if !strings.Contains(plain(m), "> "+coreset.KnobLabel(coreset.KnobMove)) {
			t.Fatal("test premise: the cursor must sit on the move knob")
		}
		for i := 0; i < 200; i++ {
			m = press(m, runeKey('h'))
		}
		if got := m.show.Cfg.MoveSeconds; got != coreset.StepSeconds {
			t.Fatalf("the move knob must floor at one step, got %v", got)
		}
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.MoveSeconds; got != coreset.StepSeconds {
			t.Fatalf("one more h must hold the floor, got %v", got)
		}
	})
	t.Run("unhappy: unknown keys change neither the cursor nor the knobs", func(t *testing.T) {
		m := newModel(0)
		before := m.show.Cfg
		m = press(m, runeKey('z'))
		if m.show.Cfg != before {
			t.Fatalf("z warped the knobs: %+v", m.show.Cfg)
		}
		if !strings.Contains(plain(m), "> "+coreset.KnobLabel(coreset.Knob(0))) {
			t.Fatal("z must leave the cursor on the first knob")
		}
	})
}

func TestCoresetSave(t *testing.T) {
	t.Run("happy: s writes the knobs to the config path and installs them as Active", func(t *testing.T) {
		t.Cleanup(coreset.Reset)
		m := newModel(0)
		m.path = filepath.Join(t.TempDir(), "config.json")
		m = press(m, runeKey('l'))
		m = press(m, runeKey('s'))
		if !strings.Contains(plain(m), "saved") {
			t.Fatal("a good save must say so on the marquee")
		}
		back, err := coreset.LoadOrDefault(m.path)
		if err != nil {
			t.Fatalf("the saved file must load: %v", err)
		}
		if back != m.show.Cfg {
			t.Fatalf("the file must hold the panel's knobs:\nfile  %+v\npanel %+v", back, m.show.Cfg)
		}
		if coreset.Active() != m.show.Cfg {
			t.Fatalf("a save must install the knobs as Active, got %+v", coreset.Active())
		}
	})
	t.Run("unhappy: a failed save names the failure and the house stays open", func(t *testing.T) {
		t.Cleanup(coreset.Reset)
		m := newModel(0)
		m.path = filepath.Join(t.TempDir(), "no", "dir", "config.json")
		m = press(m, runeKey('s'))
		if !strings.Contains(plain(m), "save failed") {
			t.Fatal("a failed save must name itself on the marquee")
		}
		mm, cmd := m.Update(frameMsg{})
		m = mm.(model)
		if cmd == nil {
			t.Fatal("the show must keep playing after a failed save")
		}
		if !strings.Contains(plain(m), "CORE SET") {
			t.Fatal("the stage must survive a failed save")
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
