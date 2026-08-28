package main

// Demo harness tests, written first: cmd/coreset2 runs the Core Sets
// Two scene standalone. The house opens on the pickup — scene one's
// held bits frame, the PRIORITY word over its fifteen bits — under a
// one-line marquee, and the scene plays itself: the six-job roster,
// the sweep, the check_for_higher_priority_jobs() function revealing
// on the right half, the five-set scan with the word math and the
// arrow while a cursor walks the code, then the redo with the
// duplicated SERVICER and the newest copy selected. space (or p, or
// enter) replays from the top, -seconds brings the curtain down on
// time, q and ctrl+c quit anywhere, and the view is always exactly
// window-height lines.

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset2"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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

func TestCoreset2DemoOpens(t *testing.T) {
	t.Run("happy: the house opens on the pickup with the marquee", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		for _, want := range []string{"PRIORITY — OCT 20", "VAC ADDRESS — OCT 400", "Core Sets Two", "replay", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
	})
	t.Run("unhappy: the code and the scan are nowhere before their acts", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		if strings.Contains(v, "EJSCAN") || strings.Contains(v, "check_for_higher_priority_jobs") {
			t.Fatal("the code must wait for its act")
		}
		if strings.Contains(v, "SELECTED") || strings.Contains(v, "◀") {
			t.Fatal("the scan must wait for its act")
		}
	})
}

func TestCoreset2DemoPlays(t *testing.T) {
	t.Run("happy: the acts advance on their own clock", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, toSeconds(coreset2.ScanOneStart-0.3))
		v := plain(m)
		if !strings.Contains(v, "EJSCAN") || !strings.Contains(v, strings.TrimSpace(coreset2.CodeLines()[3])) {
			t.Fatal("past the reveal the whole scan function must be on stage")
		}
		if strings.Contains(v, "VAC ADDRESS — OCT 400") {
			t.Fatal("the pickup must be long gone by the code act")
		}
		m = frames(m, toSeconds(coreset2.SelectOneStart-coreset2.ScanOneStart+0.3+0.5))
		v = plain(m)
		if !strings.Contains(v, "SELECTED") || !strings.Contains(v, "RR READ·32") {
			t.Fatal("scan one must end with the third box down selected")
		}
		if !strings.Contains(v, strings.TrimSpace(coreset2.CodeLines()[0])) {
			t.Fatal("the code must still stand beside the finished scan")
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
			m = frames(m, toSeconds(coreset2.ScanOneStart+1))
			if strings.Contains(plain(m), "VAC ADDRESS — OCT 400") {
				t.Fatal("test premise: the pickup act must be over before the replay")
			}
			m = press(m, msg)
			if !strings.Contains(plain(m), "VAC ADDRESS — OCT 400") {
				t.Fatalf("%v must rewind to the pickup", msg)
			}
		}
	})
	t.Run("unhappy: unknown keys neither replay nor quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, toSeconds(coreset2.ScanOneStart+1))
		mm, cmd := m.Update(runeKey('z'))
		if cmd != nil {
			t.Fatal("an unknown key must do nothing")
		}
		m = mm.(model)
		if strings.Contains(plain(m), "VAC ADDRESS — OCT 400") {
			t.Fatal("an unknown key must not rewind the show")
		}
	})
}

func TestCoreset2DemoHouseRules(t *testing.T) {
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
