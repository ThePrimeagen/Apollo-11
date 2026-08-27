package main

// Demo harness tests, written first: cmd/skies runs the Skies scene
// standalone. The house opens on almost-pure light blue with the
// Skies marquee and the knob panel; the camera tilts up so the
// darker blue and the clouds come into view, then the eagle flies
// in. j/k select a knob, h/l nudge it, s saves, p replays.

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/skies"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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

func riseFrames() int { return int(skies.RiseSeconds*30) + 15 }

func TestSkiesDemoOpens(t *testing.T) {
	t.Cleanup(skies.Reset)
	t.Run("happy: the house opens on light blue with the Skies marquee", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"Skies", "replay", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
		if !strings.Contains(v, "48;5;153m") {
			t.Fatal("the stage must open painted light blue — the horizon shot")
		}
		if strings.Contains(v, "48;5;17m") {
			t.Fatal("no dark zenith yet — the camera tilts up first")
		}
	})
	t.Run("unhappy: unknown keys neither replay nor quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		mm, cmd := m.Update(runeKey('z'))
		if cmd != nil {
			t.Fatal("an unknown key must do nothing")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "48;5;153m") {
			t.Fatal("an unknown key must not rewind the show")
		}
	})
}

func TestSkiesDemoRise(t *testing.T) {
	t.Cleanup(skies.Reset)
	t.Run("happy: after the rise the darker blue is on stage", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, riseFrames())
		v := m.View().Content
		if !strings.Contains(v, "48;5;17m") {
			t.Fatal("after the rise the dark zenith must be on stage")
		}
	})
	t.Run("happy: then the eagle's brown ink is on stage", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		// delay 2s + a quarter of the 4s crossing
		m = frames(m, int((2.0+1.0)*30)+5)
		v := m.View().Content
		if !strings.Contains(v, "8;5;94m") {
			t.Fatal("a quarter into the crossing the eagle's brown ink must be on stage")
		}
		if !strings.Contains(v, "▄") && !strings.Contains(v, "▀") {
			t.Fatal("the eagle's half-block silhouette must be on stage")
		}
	})
	t.Run("unhappy: before the delay the eagle is off stage", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 20) // ~0.66s, well before the 2s delay
		v := m.View().Content
		if strings.Contains(v, "8;5;94m") {
			t.Fatal("the eagle must wait for its delay")
		}
	})
	t.Run("happy: after the bird flies in the talon guns fire", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		// delay 2s + first shell at 1/rate (0.5s) + a couple of frames
		m = frames(m, int((2.0+0.6)*30)+5)
		v := m.View().Content
		if !strings.Contains(v, "8;5;178m") {
			t.Fatal("the shotgun gold must ride the talons")
		}
		if !strings.Contains(v, "8;5;226m") && !strings.Contains(v, "8;5;208m") && !strings.Contains(v, "8;5;196m") {
			t.Fatal("past 1/rate of air time a muzzle blast must be on stage")
		}
	})
	t.Run("unhappy: before the bird flies the guns have not fired", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 20)
		v := m.View().Content
		if strings.Contains(v, "8;5;226m") || strings.Contains(v, "8;5;208m") || strings.Contains(v, "8;5;196m") {
			t.Fatal("muzzle flame must wait for the bird")
		}
	})
}

func TestSkiesDemoFlag(t *testing.T) {
	t.Cleanup(skies.Reset)
	t.Run("happy: after the flag fade the stars and a stripe ink are on stage", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, int((skies.FlagDelaySeconds+skies.FlagFadeSeconds)*30)+5)
		v := m.View().Content
		if !strings.Contains(v, "★") {
			t.Fatal("a finished flag fade must paint stars on the floor")
		}
		if !strings.Contains(v, "48;5;160m") && !strings.Contains(v, "48;5;18m") {
			t.Fatal("a finished flag fade must wear a stripe or canton background")
		}
	})
	t.Run("unhappy: before the flag delay the stars stay off the sky", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 20)
		v := m.View().Content
		if strings.Contains(v, "★") {
			t.Fatal("the flag must wait for its delay")
		}
	})
}

func TestSkiesDemoHouseRules(t *testing.T) {
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
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 20 {
			t.Fatalf("view has %d lines for a 20-line window", got)
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

func TestSkiesDemoKnobs(t *testing.T) {
	t.Cleanup(skies.Reset)
	t.Run("happy: the panel opens on the stock knobs", func(t *testing.T) {
		v := ansiPat.ReplaceAllString(newModel(0).View().Content, "")
		for _, want := range []string{
			"sky rise", "flag delay", "flag fade", "eagle delay", "eagle cross", "eagle start", "eagle end",
			"left on", "left shots", "left rate", "left aim", "right on", "right shots", "right rate", "right aim",
			fmt.Sprintf("%7.3f", skies.RiseSeconds),
			fmt.Sprintf("%7d", skies.StockShots),
			fmt.Sprintf("%7.2f/s", skies.StockRate),
			fmt.Sprintf("%7s", "on"),
		} {
			if !strings.Contains(v, want) {
				t.Fatalf("the knob panel is missing %q:\n%s", want, v)
			}
		}
		marked := false
		for _, line := range strings.Split(v, "\n") {
			if strings.Contains(line, ">") && strings.Contains(line, "sky rise") {
				marked = true
			}
		}
		if !marked {
			t.Fatal("the cursor must open on the sky rise knob")
		}
	})
	t.Run("happy: j and k walk the cursor over the fifteen knobs with wrap", func(t *testing.T) {
		m := newModel(0)
		m = press(m, runeKey('j'))
		if m.cursor != skies.KnobFlagDelay {
			t.Fatalf("j must land on flag delay, got %d", m.cursor)
		}
		m = press(m, runeKey('k'))
		m = press(m, runeKey('k'))
		if m.cursor != skies.KnobRightAim {
			t.Fatalf("k from the top must wrap to the right aim, got %d", m.cursor)
		}
	})
	t.Run("happy: h and l nudge the selected knob one step at a time", func(t *testing.T) {
		m := newModel(0)
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.RiseSeconds; math.Abs(got-(skies.RiseSeconds+skies.StepSeconds)) > 1e-9 {
			t.Fatalf("l must add 50ms to the rise, got %v", got)
		}
	})
	t.Run("happy: h and l switch each shotgun on and off", func(t *testing.T) {
		m := newModel(0)
		for i := 0; i < int(skies.KnobLeftOn); i++ {
			m = press(m, runeKey('j'))
		}
		if m.cursor != skies.KnobLeftOn {
			t.Fatalf("j must land on left on, got %d", m.cursor)
		}
		m = press(m, runeKey('h'))
		if m.show.Cfg.LeftOn {
			t.Fatal("h must switch the left gun off")
		}
		v := ansiPat.ReplaceAllString(m.View().Content, "")
		if !strings.Contains(v, "left on") || !strings.Contains(v, "off") {
			t.Fatalf("the panel must read the left gun off:\n%s", v)
		}
		m = press(m, runeKey('l'))
		if !m.show.Cfg.LeftOn {
			t.Fatal("l must switch the left gun back on")
		}
	})
	t.Run("unhappy: h cannot switch an already-off gun any further off", func(t *testing.T) {
		m := newModel(0)
		for i := 0; i < int(skies.KnobRightOn); i++ {
			m = press(m, runeKey('j'))
		}
		if m.cursor != skies.KnobRightOn {
			t.Fatalf("j must land on right on, got %d", m.cursor)
		}
		m = press(m, runeKey('h'))
		m = press(m, runeKey('h'))
		if m.show.Cfg.RightOn {
			t.Fatal("h on an already-off gun must leave it off")
		}
		if !m.show.Cfg.LeftOn {
			t.Fatal("switching the right gun must not touch the left")
		}
	})
	t.Run("happy: s saves the knobs to the config path", func(t *testing.T) {
		t.Cleanup(skies.Reset)
		m := newModel(0)
		m.path = filepath.Join(t.TempDir(), "skies.json")
		m = press(m, runeKey('l'))
		m = press(m, runeKey('s'))
		if m.note != "saved" {
			t.Fatalf("note %q, want saved", m.note)
		}
		got, err := skies.Load(m.path)
		if err != nil {
			t.Fatalf("the saved file must load: %v", err)
		}
		if math.Abs(got.RiseSeconds-(skies.RiseSeconds+skies.StepSeconds)) > 1e-9 {
			t.Fatalf("saved rise %v, want the nudged %v", got.RiseSeconds, skies.RiseSeconds+skies.StepSeconds)
		}
	})
	t.Run("unhappy: s without a config path says so instead of writing", func(t *testing.T) {
		m := newModel(0)
		m.path = ""
		m = press(m, runeKey('s'))
		if m.note != "no config path" {
			t.Fatalf("note %q, want no config path", m.note)
		}
	})
}
