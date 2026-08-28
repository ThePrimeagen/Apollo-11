package main

// Demo harness tests, written first: cmd/interpreter runs the
// Interpreter scene standalone, and it is the scene's tuner too. The
// house opens on the slimmed-down walkthrough — the ΔV load under
// the spotlight: one plain comment on top that just says what the
// block does, the bare MUNRVG ops under it, and the whole DANZIG
// construction as the one pseudo call
// check_for_higher_priority_jobs() — over a marquee and the two
// knob rows. The scene plays itself: the camera glides
// stop to stop through the five blocks and holds on the V cross V.
// j/k walk the knob cursor with wrap, h/l nudge the selected knob
// one 50ms step and the panel shows the new reading, s saves the
// knobs to the config path (and installs them as the Active timing),
// space (or p, or enter) replays from the top with whatever the
// knobs hold, -seconds brings the curtain down on time, q and ctrl+c
// quit anywhere, and the view is always exactly window-height lines.

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/interpreter"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stock is the scene's default clock, the timeline every fresh model
// plays.
var stock = interpreter.DefaultConfig()

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

func TestInterpreterDemoOpens(t *testing.T) {
	t.Run("happy: the house opens on the spotlit ΔV load — its comment, its ops, its call, the knob rows", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		for _, want := range []string{"MUNRVG", "VLOAD", "KPIP2", "INTPRET", interpreter.Chunks()[0].Comment, "check_for_higher_priority_jobs()", "# DANZIG", "Interpreter", "replay", "quit", "save"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
		if strings.Contains(v, "THIS BLOCK") {
			t.Fatal("the comments must say what the block does — no narration")
		}
		for k := interpreter.Knob(0); k < interpreter.KnobCount; k++ {
			if !strings.Contains(v, interpreter.KnobLabel(k)) {
				t.Fatalf("the knob panel is missing %q", interpreter.KnobLabel(k))
			}
		}
	})
	t.Run("unhappy: the far blocks are past the vignette at the curtain, and no dress survives", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		if strings.Contains(v, "VXV") || strings.Contains(v, "DELVS") {
			t.Fatal("the cross product must wait behind the vignette")
		}
		for _, gone := range []string{"CARRY ON", "NEWJOB", "CHANG2"} {
			if strings.Contains(v, gone) {
				t.Fatalf("the old dressed-up check (%q) must be gone from the show", gone)
			}
		}
	})
}

func TestInterpreterDemoPlays(t *testing.T) {
	t.Run("happy: the spotlight advances on its own clock", func(t *testing.T) {
		m := newModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 110, Height: 45})
		m = mm.(model)
		_ = m.View()
		if strings.Contains(plain(m), "ABVEL") {
			t.Fatal("test premise: the velocity block waits past the opening vignette")
		}
		m = frames(m, toSeconds(stock.StopStart(1)+0.2))
		v := plain(m)
		if !strings.Contains(v, "ABVEL") {
			t.Fatal("by the second stop the velocity block must surface at the vignette's edge")
		}
		m = frames(m, toSeconds(stock.StopStart(4)-stock.StopStart(1)-0.2+0.5))
		v = plain(m)
		if !strings.Contains(v, "DELVS") || !strings.Contains(v, "HCALC") || !strings.Contains(v, "RVQ") {
			t.Fatal("the last stop must show the V cross V block over the fading tail")
		}
		if strings.Contains(v, "INTPRET") {
			t.Fatal("the prologue must be long gone by the last stop")
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
			m = frames(m, toSeconds(stock.StopStart(4)+0.5))
			if strings.Contains(plain(m), "INTPRET") {
				t.Fatal("test premise: the prologue must be gone before the replay")
			}
			m = press(m, msg)
			if !strings.Contains(plain(m), "INTPRET") {
				t.Fatalf("%v must rewind to the opening composition", msg)
			}
		}
	})
	t.Run("unhappy: unknown keys neither replay nor quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, toSeconds(stock.StopStart(4)+0.5))
		mm, cmd := m.Update(runeKey('z'))
		if cmd != nil {
			t.Fatal("an unknown key must do nothing")
		}
		m = mm.(model)
		if strings.Contains(plain(m), "INTPRET") {
			t.Fatal("an unknown key must not rewind the show")
		}
	})
}

func TestInterpreterKnobPanel(t *testing.T) {
	t.Run("happy: j and k walk the cursor with wrap", func(t *testing.T) {
		m := newModel(0)
		if !strings.Contains(plain(m), "> "+interpreter.KnobLabel(interpreter.Knob(0))) {
			t.Fatal("the cursor must open on the first knob")
		}
		m = press(m, runeKey('j'))
		if !strings.Contains(plain(m), "> "+interpreter.KnobLabel(interpreter.Knob(1))) {
			t.Fatal("j must select the second knob")
		}
		m = press(m, runeKey('j'))
		if !strings.Contains(plain(m), "> "+interpreter.KnobLabel(interpreter.Knob(0))) {
			t.Fatal("j past the bottom must wrap to the first knob")
		}
		m = press(m, runeKey('k'))
		if !strings.Contains(plain(m), "> "+interpreter.KnobLabel(interpreter.KnobCount-1)) {
			t.Fatal("k past the top must wrap to the last knob")
		}
	})
	t.Run("happy: l and h retune the selected knob and the panel shows the new reading", func(t *testing.T) {
		m := newModel(0)
		before := m.show.Cfg.Value(interpreter.Knob(0))
		m = press(m, runeKey('l'))
		after := m.show.Cfg.Value(interpreter.Knob(0))
		if after <= before {
			t.Fatalf("l must nudge the knob up: %v then %v", before, after)
		}
		if !strings.Contains(plain(m), m.show.Cfg.Display(interpreter.Knob(0))) {
			t.Fatalf("the panel must show the new reading %q", m.show.Cfg.Display(interpreter.Knob(0)))
		}
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.Value(interpreter.Knob(0)); got != before {
			t.Fatalf("h must bring the knob back to %v, got %v", before, got)
		}
	})
	t.Run("happy: a nudged knob retimes the next replay", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		for i := 0; i < 100; i++ {
			m = press(m, runeKey('h'))
		}
		if got := m.show.Cfg.HoldSeconds; got != 0 {
			t.Fatalf("test premise: a hundred h presses floor the hold, got %v", got)
		}
		m = press(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		_ = m.View() // the real loop renders every frame — restage the fresh cast
		m = frames(m, toSeconds(m.show.Cfg.StopStart(4)+0.3))
		v := plain(m)
		if !strings.Contains(v, "DELVS") {
			t.Fatal("with a zero hold the replay must already be on the V cross V")
		}
	})
	t.Run("unhappy: the floors hold on the panel too", func(t *testing.T) {
		m := newModel(0)
		m = press(m, runeKey('j'))
		if !strings.Contains(plain(m), "> "+interpreter.KnobLabel(interpreter.KnobGlide)) {
			t.Fatal("test premise: the cursor must sit on the glide knob")
		}
		for i := 0; i < 200; i++ {
			m = press(m, runeKey('h'))
		}
		if got := m.show.Cfg.GlideSeconds; got != interpreter.StepSeconds {
			t.Fatalf("the glide knob must floor at one step, got %v", got)
		}
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.GlideSeconds; got != interpreter.StepSeconds {
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
		if !strings.Contains(plain(m), "> "+interpreter.KnobLabel(interpreter.Knob(0))) {
			t.Fatal("z must leave the cursor on the first knob")
		}
	})
}

func TestInterpreterSave(t *testing.T) {
	t.Run("happy: s writes the knobs to the config path and installs them as Active", func(t *testing.T) {
		t.Cleanup(interpreter.Reset)
		m := newModel(0)
		m.path = filepath.Join(t.TempDir(), "config.json")
		m = press(m, runeKey('l'))
		m = press(m, runeKey('s'))
		if !strings.Contains(plain(m), "saved") {
			t.Fatal("a good save must say so on the marquee")
		}
		back, err := interpreter.LoadOrDefault(m.path)
		if err != nil {
			t.Fatalf("the saved file must load: %v", err)
		}
		if back != m.show.Cfg {
			t.Fatalf("the file must hold the panel's knobs:\nfile  %+v\npanel %+v", back, m.show.Cfg)
		}
		if interpreter.Active() != m.show.Cfg {
			t.Fatalf("a save must install the knobs as Active, got %+v", interpreter.Active())
		}
	})
	t.Run("unhappy: a failed save names the failure and the house stays open", func(t *testing.T) {
		t.Cleanup(interpreter.Reset)
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
		if !strings.Contains(plain(m), "VLOAD") {
			t.Fatal("the stage must survive a failed save")
		}
	})
}

func TestInterpreterDemoHouseRules(t *testing.T) {
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
