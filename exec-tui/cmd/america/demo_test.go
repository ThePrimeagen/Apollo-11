package main

// Demo harness tests, written first: cmd/america runs the America
// scene standalone, tuned live the way the landing runner tunes. The
// house opens on pure black with the America marquee and the knob
// panel — flag fade, eagle delay, eagle cross, eagle start, eagle
// end — under the stage; the full-screened flag fades in fast, and
// once it is fully in, the very large eagle crosses the stage right
// to left with the flag flying beneath, from its start point to its
// end point of the span. j/k select a knob, h/l nudge it 50ms, s saves to
// the scene's config JSON, and p (or space, or enter) replays from
// the top — back to black — on the current knobs. -seconds brings the
// curtain down on time, q and ctrl+c quit anywhere, and the view is
// always exactly window-height lines.

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/america"
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

func space() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "} }

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func fadeFrames() int { return int(america.FadeSeconds*30) + 15 }

func quarterCross() int { return int(america.CrossSeconds / 4 * 30) }

// eagleSeen reports the eagle's brown ink anywhere in the raw view.
// The flag never wears 94 at any point of its fade, so the ink is the
// bird.
func eagleSeen(v string) bool { return strings.Contains(v, "8;5;94m") }

// eagleLeft finds the leftmost lower-half block above the status
// panel, ANSI stripped. The flag draws its stripe boundaries with
// upper-half blocks only, so every '▄' on stage is the eagle's
// silhouette.
func eagleLeft(v string) (int, bool) {
	lines := strings.Split(ansiPat.ReplaceAllString(v, ""), "\n")
	if len(lines) < statusRows+1 {
		return 0, false
	}
	left, ok := 1<<30, false
	for _, line := range lines[:len(lines)-statusRows] {
		for c, ch := range []rune(line) {
			if ch != '▄' {
				continue
			}
			if c < left {
				left = c
			}
			ok = true
		}
	}
	return left, ok
}

func TestAmericaDemoOpens(t *testing.T) {
	t.Run("happy: the house opens on pure black with the America marquee", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"America", "replay", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the opening view is missing %q", want)
			}
		}
		if !strings.Contains(v, "48;5;16m") {
			t.Fatal("the stage must open painted black — the fade starts from black, not from nothing")
		}
		if strings.Contains(v, "48;5;160m") {
			t.Fatal("no red yet — the flag fades in from black")
		}
		if eagleSeen(v) {
			t.Fatal("no eagle ink yet — the flag comes first")
		}
		if _, ok := eagleLeft(v); ok {
			t.Fatal("no eagle silhouette yet — the flag comes first")
		}
	})
	t.Run("unhappy: waiting a moment is still black — even a fast fade ramps from black", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 3)
		if strings.Contains(m.View().Content, "48;5;160m") {
			t.Fatal("three frames in, the flag must not already be red")
		}
	})
}

func TestAmericaDemoFade(t *testing.T) {
	t.Run("happy: the flag is fully in after the fade", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, fadeFrames())
		v := m.View().Content
		for _, want := range []string{"48;5;160m", "48;5;18m", "★"} {
			if !strings.Contains(v, want) {
				t.Fatalf("after the fade the view is missing %q — red stripes, blue canton, stars", want)
			}
		}
	})
	t.Run("happy: then the eagle crosses, right to left", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, fadeFrames()+quarterCross())
		v := m.View().Content
		if !eagleSeen(v) {
			t.Fatal("a quarter into the crossing the eagle's ink must be on stage")
		}
		l1, ok := eagleLeft(v)
		if !ok {
			t.Fatal("a quarter into the crossing the eagle's silhouette must be on stage")
		}
		m = frames(m, quarterCross())
		v = m.View().Content
		l2, ok := eagleLeft(v)
		if !ok {
			t.Fatal("halfway into the crossing the eagle must still be on stage")
		}
		if l2 >= l1 {
			t.Fatalf("the eagle must fly leftward: leftmost went %d -> %d", l1, l2)
		}
		if !strings.Contains(v, "48;5;160m") {
			t.Fatal("the flag must keep flying beneath the eagle")
		}
	})
}

func TestAmericaDemoReplay(t *testing.T) {
	t.Run("happy: p replays from the top — back to black", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, fadeFrames())
		if !strings.Contains(m.View().Content, "48;5;160m") {
			t.Fatal("test premise: the flag must be in before the replay")
		}
		m = press(m, runeKey('p'))
		v := m.View().Content
		if strings.Contains(v, "48;5;160m") {
			t.Fatal("p must rewind the fade to black")
		}
		if !strings.Contains(v, "48;5;16m") {
			t.Fatal("after the replay the stage must be painted black again")
		}
	})
	t.Run("happy: space and enter replay too", func(t *testing.T) {
		for _, msg := range []tea.Msg{space(), runeKey(' '), tea.KeyPressMsg{Code: tea.KeyEnter}} {
			m := newModel(0)
			_ = m.View()
			m = frames(m, fadeFrames())
			m = press(m, msg)
			if strings.Contains(m.View().Content, "48;5;160m") {
				t.Fatalf("%v must replay from the top", msg)
			}
		}
	})
	t.Run("unhappy: unknown keys neither replay nor quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, fadeFrames())
		mm, cmd := m.Update(runeKey('z'))
		if cmd != nil {
			t.Fatal("an unknown key must do nothing")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "48;5;160m") {
			t.Fatal("an unknown key must not rewind the show")
		}
	})
}

func TestAmericaDemoHouseRules(t *testing.T) {
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
		mm, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 32})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 32 {
			t.Fatalf("view has %d lines for a 32-line window", got)
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

func TestAmericaDemoKnobs(t *testing.T) {
	t.Run("happy: the panel opens on the fast stock — timing, flight path, and the talon guns", func(t *testing.T) {
		v := ansiPat.ReplaceAllString(newModel(0).View().Content, "")
		for _, want := range []string{
			"flag fade", "eagle delay", "eagle cross", "eagle start", "eagle end",
			"left shots", "left aim", "right shots", "right aim",
			"  2.000", "  4.000", "  0.000", "  1.000",
			fmt.Sprintf("%7d", america.StockShots),
			fmt.Sprintf("%7s", string(america.StockLeftAim)),
			fmt.Sprintf("%7s", string(america.StockRightAim)),
		} {
			if !strings.Contains(v, want) {
				t.Fatalf("the knob panel is missing %q:\n%s", want, v)
			}
		}
		for _, slow := range []string{"8.000", "12.000"} {
			if strings.Contains(v, slow) {
				t.Fatalf("the slow stock %q is still on the panel:\n%s", slow, v)
			}
		}
		for _, unitless := range []string{"0.000s", "1.000s"} {
			if strings.Contains(v, unitless) {
				t.Fatalf("the path knobs are fractions of the span, not seconds — found %q:\n%s", unitless, v)
			}
		}
		marked := false
		for _, line := range strings.Split(v, "\n") {
			if strings.Contains(line, ">") && strings.Contains(line, "flag fade") {
				marked = true
			}
		}
		if !marked {
			t.Fatal("the cursor must open on the flag fade knob")
		}
	})
	t.Run("happy: j and k walk the cursor over the nine knobs with wrap", func(t *testing.T) {
		m := newModel(0)
		walk := []america.Knob{
			america.KnobDelay, america.KnobCross, america.KnobStart, america.KnobEnd,
			america.KnobLeftShots, america.KnobLeftAim, america.KnobRightShots, america.KnobRightAim,
			america.KnobFade,
		}
		for _, want := range walk {
			m = press(m, runeKey('j'))
			if m.cursor != want {
				t.Fatalf("j must land on knob %d, got %d", want, m.cursor)
			}
		}
		m = press(m, runeKey('k'))
		if m.cursor != america.KnobRightAim {
			t.Fatalf("k from the top must wrap to the right aim, got %d", m.cursor)
		}
	})
	t.Run("happy: h and l nudge the selected knob one step at a time", func(t *testing.T) {
		m := newModel(0)
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.FadeSeconds; math.Abs(got-(america.FadeSeconds+america.StepSeconds)) > 1e-9 {
			t.Fatalf("l must add 50ms to the fade, got %v", got)
		}
		m = press(m, runeKey('h'))
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.FadeSeconds; math.Abs(got-(america.FadeSeconds-america.StepSeconds)) > 1e-9 {
			t.Fatalf("h twice must take 50ms off the fade, got %v", got)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.CrossSeconds; math.Abs(got-(america.CrossSeconds+america.StepSeconds)) > 1e-9 {
			t.Fatalf("l on the cross knob must add 50ms, got %v", got)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.EagleStart; math.Abs(got-(america.StartPoint+america.StepPoint)) > 1e-9 {
			t.Fatalf("l on the start knob must add 0.05 of the span, got %v", got)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.EagleEnd; math.Abs(got-(america.EndPoint-america.StepPoint)) > 1e-9 {
			t.Fatalf("h on the end knob must take 0.05 of the span off, got %v", got)
		}
	})
	t.Run("unhappy: the path knobs never leave the span", func(t *testing.T) {
		m := newModel(0)
		m = press(m, runeKey('j'))
		m = press(m, runeKey('j'))
		m = press(m, runeKey('j'))
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.EagleStart; got != america.StartPoint {
			t.Fatalf("h on the stock start must hold at %v, got %v", america.StartPoint, got)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.EagleEnd; got != america.EndPoint {
			t.Fatalf("l on the stock end must hold at %v, got %v", america.EndPoint, got)
		}
	})
	t.Run("happy: the gun knobs count shells and walk the compass", func(t *testing.T) {
		m := newModel(0)
		for i := 0; i < 5; i++ {
			m = press(m, runeKey('j'))
		}
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.LeftShots; got != america.StockShots+1 {
			t.Fatalf("l on the left shots must add a shell, got %d", got)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.LeftAim; got != sprite.W {
			t.Fatalf("l on the stock %s aim must step clockwise to W, got %s", america.StockLeftAim, got)
		}
		m = press(m, runeKey('h'))
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.LeftAim; got != sprite.S {
			t.Fatalf("h twice must step back to S, got %s", got)
		}
	})
	t.Run("unhappy: the shell counts never go negative", func(t *testing.T) {
		m := newModel(0)
		for i := 0; i < 7; i++ {
			m = press(m, runeKey('j'))
		}
		for i := 0; i < america.StockShots+2; i++ {
			m = press(m, runeKey('h'))
		}
		if got := m.show.Cfg.RightShots; got != 0 {
			t.Fatalf("h below zero must hold the right shots at 0, got %d", got)
		}
	})
	t.Run("happy: a nudged replay plays the new knobs", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m.show.Cfg.FadeSeconds = 0.2
		m = press(m, runeKey('p'))
		_ = m.View()
		m = frames(m, 30)
		if !strings.Contains(m.View().Content, "48;5;160m") {
			t.Fatal("a second after a 0.2s-fade replay the flag must be red")
		}
	})
	t.Run("happy: s saves the knobs to the config path", func(t *testing.T) {
		t.Cleanup(america.Reset)
		m := newModel(0)
		m.path = filepath.Join(t.TempDir(), "america.json")
		m = press(m, runeKey('l'))
		m = press(m, runeKey('s'))
		if m.note != "saved" {
			t.Fatalf("note %q, want saved", m.note)
		}
		got, err := america.Load(m.path)
		if err != nil {
			t.Fatalf("the saved file must load: %v", err)
		}
		if math.Abs(got.FadeSeconds-(america.FadeSeconds+america.StepSeconds)) > 1e-9 {
			t.Fatalf("saved fade %v, want the nudged %v", got.FadeSeconds, america.FadeSeconds+america.StepSeconds)
		}
		if !strings.Contains(m.View().Content, "saved") {
			t.Fatal("the status line must show the save")
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
