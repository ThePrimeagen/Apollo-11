package main

// Demo harness tests, written first: cmd/astronaut is now the moonwalk
// tuner — it plays the scenes/moonwalk show (crates, the pole-top
// landing, the rising flag, the closing pan to the rover) and exposes
// every scene knob: j/k select a knob, h/l nudge it live, s saves the
// config JSON, space replays from the top, q quits. The stage fills
// the window above a compact knob footer.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
)

const (
	tw = 84
	th = 30
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

func newTestModel(t *testing.T, seconds float64) model {
	t.Helper()
	m, err := newModel(seconds)
	if err != nil {
		t.Fatalf("newModel: %v", err)
	}
	mm, _ := m.Update(tea.WindowSizeMsg{Width: tw, Height: th})
	return mm.(model)
}

func TestMoonwalkTuner(t *testing.T) {
	t.Run("happy: the opening view shows the set and fills the window", func(t *testing.T) {
		m := newTestModel(t, 0)
		v := m.View().Content
		if got := len(strings.Split(v, "\n")); got != th {
			t.Fatalf("view has %d lines for a %d-line window", got, th)
		}
		if !strings.Contains(v, "│") {
			t.Fatal("the flagpole must be on stage from the first frame")
		}
		if !strings.Contains(strings.ToLower(v), "moonwalk") {
			t.Fatal("the footer must name the show")
		}
	})
	t.Run("happy: the scene animates with the ticks", func(t *testing.T) {
		m := newTestModel(t, 0)
		m = frames(m, 20)
		before := m.View().Content
		m = frames(m, 8)
		if m.View().Content == before {
			t.Fatal("eight ticks must advance the scene")
		}
	})
	t.Run("happy: j and k walk the knob list and wrap", func(t *testing.T) {
		m := newTestModel(t, 0)
		if m.knob != 0 {
			t.Fatalf("the first knob starts selected, got %d", m.knob)
		}
		m = press(m, runeKey('j'))
		if m.knob != 1 {
			t.Fatalf("j must select the next knob, got %d", m.knob)
		}
		m = press(m, runeKey('k'))
		m = press(m, runeKey('k'))
		if m.knob != moonwalk.KnobCount-1 {
			t.Fatalf("k past the top must wrap to the last knob, got %d", m.knob)
		}
	})
	t.Run("happy: h and l nudge the selected knob live", func(t *testing.T) {
		m := newTestModel(t, 0)
		before := m.cfg.Value(m.knob)
		m = press(m, runeKey('l'))
		if m.cfg.Value(m.knob) <= before {
			t.Fatalf("l must nudge %s up from %v", m.knob, before)
		}
		m = press(m, runeKey('h'))
		if m.cfg.Value(m.knob) != before {
			t.Fatalf("h must bring %s back to %v", m.knob, before)
		}
	})
	t.Run("happy: the footer shows every knob and marks the selected one", func(t *testing.T) {
		m := newTestModel(t, 0)
		v := m.View().Content
		for k := moonwalk.Knob(0); k < moonwalk.KnobCount; k++ {
			if !strings.Contains(v, k.String()) {
				t.Fatalf("footer is missing knob %q", k)
			}
		}
		if !strings.Contains(v, ">") {
			t.Fatal("the selected knob must carry a marker")
		}
	})
	t.Run("happy: s saves the knobs to the config path", func(t *testing.T) {
		m := newTestModel(t, 0)
		m.path = t.TempDir() + "/config.json"
		m = press(m, runeKey('l'))
		want := m.cfg
		m = press(m, runeKey('s'))
		got, err := moonwalk.LoadOrDefault(m.path)
		if err != nil {
			t.Fatalf("saved config does not load: %v", err)
		}
		if got != want {
			t.Fatalf("saved %+v, want %+v", got, want)
		}
		if !strings.Contains(strings.ToLower(m.View().Content), "saved") {
			t.Fatal("the footer must confirm the save")
		}
	})
	t.Run("happy: space replays from the top", func(t *testing.T) {
		m := newTestModel(t, 0)
		m = frames(m, 60)
		if m.clock == 0 {
			t.Fatal("sixty frames must move the clock")
		}
		m = press(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
		if m.clock != 0 {
			t.Fatalf("space must rewind the clock, got %v", m.clock)
		}
	})
	t.Run("happy: each frame schedules the next; Init starts the clock", func(t *testing.T) {
		m := newTestModel(t, 0)
		if m.Init() == nil {
			t.Fatal("Init must start the ticker")
		}
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
		m := newTestModel(t, 0.05)
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
	t.Run("unhappy: q and ctrl+c quit from any point", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			runeKey('q'),
			tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		} {
			m := newTestModel(t, 0)
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
	t.Run("unhappy: an unbound key changes nothing", func(t *testing.T) {
		m := newTestModel(t, 0)
		before := m.View().Content
		cfg := m.cfg
		m = press(m, runeKey('x'))
		if m.cfg != cfg {
			t.Fatal("an unbound key must not touch the knobs")
		}
		if m.View().Content != before {
			t.Fatal("an unbound key must not move the scene")
		}
	})
	t.Run("unhappy: a tiny window never panics and still fills its height", func(t *testing.T) {
		m := newTestModel(t, 0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 6, Height: 4})
		m = mm.(model)
		m = frames(m, 40)
		if got := len(strings.Split(m.View().Content, "\n")); got != 4 {
			t.Fatalf("view has %d lines for a 4-line window", got)
		}
	})
}
