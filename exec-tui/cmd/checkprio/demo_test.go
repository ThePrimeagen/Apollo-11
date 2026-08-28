package main

// Demo harness tests, written first: cmd/checkprio runs the Check
// Priority scene standalone. The house opens on the function's first
// line under a one-line marquee, and the scene plays itself: the
// C-style check_for_higher_priority_jobs() reveals one line per
// beat, then the gold cursor walks it — the read of data[11], the
// new-against-old compare, the win — and rests on the run line.
// space (or p, or enter) replays from the top, -seconds brings the
// curtain down on time, q and ctrl+c quit anywhere, and the view is
// always exactly window-height lines.

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/checkprio"
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

func TestCheckprioDemoOpens(t *testing.T) {
	t.Run("happy: the house opens on the function's name with the marquee", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		for _, want := range []string{"check_for_higher_priority_jobs()", "Check Priority", "replay", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
	})
	t.Run("unhappy: the walk and the far lines are nowhere before their beats", func(t *testing.T) {
		m := newModel(0)
		v := plain(m)
		if strings.Contains(v, "▸") {
			t.Fatal("the cursor must wait for the walk")
		}
		if strings.Contains(v, "run(core_sets[winner])") {
			t.Fatal("the run line must wait for its reveal beat")
		}
	})
}

func TestCheckprioDemoPlays(t *testing.T) {
	t.Run("happy: the reveal and the walk advance on their own clock", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, toSeconds(checkprio.WalkStart()-0.3))
		v := plain(m)
		if !strings.Contains(v, "run(core_sets[winner])") {
			t.Fatal("past the reveal the whole function must be on stage")
		}
		if strings.Contains(v, "▸") {
			t.Fatal("the cursor must still be waiting")
		}
		m = frames(m, toSeconds(checkprio.StepAt(2)+0.4-(checkprio.WalkStart()-0.3)))
		v = plain(m)
		if !strings.Contains(v, "▸") || !strings.Contains(v, checkprio.WalkSteps()[2].Text) {
			t.Fatal("by the read step the cursor and its caption must be up")
		}
		m = frames(m, toSeconds(checkprio.HoldStart()-checkprio.StepAt(2)-0.4+0.6))
		if !strings.Contains(plain(m), checkprio.CaptionHold) {
			t.Fatal("the walk must end on the hold caption")
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
			m = frames(m, toSeconds(checkprio.HoldStart()+1))
			if !strings.Contains(plain(m), "run(core_sets[winner])") {
				t.Fatal("test premise: the whole function must be up before the replay")
			}
			m = press(m, msg)
			if strings.Contains(plain(m), "run(core_sets[winner])") {
				t.Fatalf("%v must rewind to the bare opening", msg)
			}
		}
	})
	t.Run("unhappy: unknown keys neither replay nor quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, toSeconds(checkprio.HoldStart()+1))
		mm, cmd := m.Update(runeKey('z'))
		if cmd != nil {
			t.Fatal("an unknown key must do nothing")
		}
		m = mm.(model)
		if !strings.Contains(plain(m), "run(core_sets[winner])") {
			t.Fatal("an unknown key must not rewind the show")
		}
	})
}

func TestCheckprioDemoHouseRules(t *testing.T) {
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
