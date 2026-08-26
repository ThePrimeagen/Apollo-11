package main

// Demo harness tests, written first: cmd/landing runs the portable
// landing scene from scenes/landing. The house opens on "landing": a
// huge moon horizon and the north-facing lander coming down onto it.
// p / enter / space replay from the top with the current knobs.
// j/k select a knob, h/l walk it 50ms. q and ctrl+c quit. The view
// is the rendered screen plus the knob panel, always exactly
// window-height lines.

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
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

func hasFire(v string) bool {
	return strings.ContainsAny(v, "⠁⠒⠶")
}

func hotBraille(v string) bool {
	for _, ink := range []string{"38;5;88m", "38;5;124m", "38;5;160m"} {
		if strings.Contains(v, ink) {
			return true
		}
	}
	return false
}

func TestLandingSceneRunner(t *testing.T) {
	t.Cleanup(landing.Reset)
	t.Run("happy: the house opens on scene 1/1 — landing, moon floor, craft off the top", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"landing", "play", "50ms", "save", "quit", "land", "dust", "fire", "loss"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.Contains(v, "48;5;") {
			t.Fatal("the landing scene must show the moon as a background floor")
		}
		if strings.ContainsRune(v, '▟') {
			t.Fatal("at t=0 the lander must still be off the top")
		}
		if strings.ContainsRune(v, '▓') {
			t.Fatal("the moon floor must be a background color, not terrain glyphs covering the fire")
		}
	})
	t.Run("happy: Use is the first play, not only a later replay", func(t *testing.T) {
		t.Cleanup(landing.Reset)
		c := landing.DefaultConfig()
		c.LandSeconds = 0.2
		if err := landing.Use(c); err != nil {
			t.Fatal(err)
		}
		m := newModel(0)
		_ = m.View()
		m = frames(m, int(0.25*30+0.5))
		if !strings.ContainsRune(m.View().Content, '▟') {
			t.Fatal("the opening play must already use the saved land duration")
		}
	})
	t.Run("happy: the lander comes down, booster still lit, then cuts off on the pad", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, int((landing.LandSeconds-0.5)*30))
		near := m.View().Content
		if !strings.ContainsRune(near, '▟') {
			t.Fatal("near the pad the north hull must be on stage")
		}
		if !hasFire(near) {
			t.Fatal("the plume must still be lit as the craft comes in")
		}
		m = frames(m, 30)
		landed := m.View().Content
		if !strings.ContainsRune(landed, '▟') {
			t.Fatal("at touchdown the north hull must sit on the surface")
		}
		if hotBraille(landed) {
			t.Fatal("at touchdown the booster must cut off — only gray pad dust may remain")
		}
	})
	t.Run("happy: p, enter, and space replay from the top and do not quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, int((landing.LandSeconds-0.2)*30))
		if !strings.ContainsRune(m.View().Content, '▟') {
			t.Fatal("test premise: near the pad the hull must be on stage")
		}
		for _, msg := range []tea.Msg{runeKey('p'), enter(), space()} {
			mm, cmd := m.Update(msg)
			if cmd != nil {
				t.Fatalf("%v must replay, not quit", msg)
			}
			m = mm.(model)
			if strings.ContainsRune(m.View().Content, '▟') {
				t.Fatalf("%v must rewind the craft off the top", msg)
			}
			m = frames(m, int((landing.LandSeconds-0.2)*30))
			if !strings.ContainsRune(m.View().Content, '▟') {
				t.Fatalf("after %v the next play must still land", msg)
			}
		}
	})
	t.Run("happy: j/k select a knob and h/l walk it by 50ms", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.LandSeconds; math.Abs(got-(landing.LandSeconds+landing.StepSeconds)) > 1e-9 {
			t.Fatalf("land after +50ms is %v, want %v", got, landing.LandSeconds+landing.StepSeconds)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.DustStart; math.Abs(got-(landing.DustStart-landing.StepSeconds)) > 1e-9 {
			t.Fatalf("dust start after -50ms is %v, want %v", got, landing.DustStart-landing.StepSeconds)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.DustRun; math.Abs(got-(landing.DustRun+landing.StepSeconds)) > 1e-9 {
			t.Fatalf("dust run after +50ms is %v, want %v", got, landing.DustRun+landing.StepSeconds)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.DustLoss; math.Abs(got-(landing.DustLoss+landing.StepLoss)) > 1e-9 {
			t.Fatalf("dust loss after +1 step is %v, want %v", got, landing.DustLoss+landing.StepLoss)
		}
		if !strings.Contains(m.View().Content, "/ms") {
			t.Fatal("dust loss must show as particles per millisecond")
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.Fire75; math.Abs(got-(landing.Fire75+landing.StepSeconds)) > 1e-9 {
			t.Fatalf("fire ¾ after +50ms is %v, want %v", got, landing.Fire75+landing.StepSeconds)
		}
		if !strings.Contains(m.View().Content, ">") {
			t.Fatal("the selected knob must be marked")
		}
	})
	t.Run("unhappy: the land floor is 50ms, dust start and run will not go negative, and space does not quit", func(t *testing.T) {
		m := newModel(0)
		m.show.Cfg.LandSeconds = landing.StepSeconds
		m.show.Cfg.DustStart = 0
		m.show.Cfg.DustRun = 0
		m.cursor = landing.KnobLand
		m = press(m, runeKey('h'))
		if m.show.Cfg.LandSeconds != landing.StepSeconds {
			t.Fatalf("land %v, want the 50ms floor", m.show.Cfg.LandSeconds)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('h'))
		if m.show.Cfg.DustStart != 0 {
			t.Fatalf("dust start %v, want 0", m.show.Cfg.DustStart)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('h'))
		if m.show.Cfg.DustRun != 0 {
			t.Fatalf("dust run %v, want 0", m.show.Cfg.DustRun)
		}
		m.show.Cfg.DustLoss = 0
		m = press(m, runeKey('j'))
		m = press(m, runeKey('h'))
		if m.show.Cfg.DustLoss != 0 {
			t.Fatalf("dust loss %v, want 0", m.show.Cfg.DustLoss)
		}
		m.show.Cfg.Fire75 = 0
		m.cursor = landing.KnobFire75
		m = press(m, runeKey('h'))
		if m.show.Cfg.Fire75 != 0 {
			t.Fatalf("fire ¾ %v, want 0", m.show.Cfg.Fire75)
		}
		_, cmd := m.Update(space())
		if cmd != nil {
			t.Fatal("space must replay, not quit")
		}
	})
	t.Run("happy: each frame schedules the next", func(t *testing.T) {
		m := newModel(0)
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: Init schedules the first frame", func(t *testing.T) {
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
	t.Run("happy: s saves the knobs and does not quit", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "landing.json")
		m := newModel(0)
		m.path = path
		m.show.Cfg.LandSeconds = 4.25
		m.show.Cfg.DustStart = 2.0
		m.show.Cfg.DustRun = 1.5
		m.show.Cfg.Fire75 = 1.0
		m.show.Cfg.Fire50 = 2.0
		m.show.Cfg.Fire25 = 3.0
		m.show.Cfg.FireOff = 4.0
		m.show.Cfg.DustLoss = 0.075
		mm, cmd := m.Update(runeKey('s'))
		if cmd != nil {
			t.Fatal("s must save, not quit")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "saved") {
			t.Fatal("a successful save must say so")
		}
		got, err := landing.Load(path)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got != m.show.Cfg {
			t.Fatalf("saved %+v, want the live knobs %+v", got, m.show.Cfg)
		}
	})
	t.Run("unhappy: s with no path or a stuck file does not quit and says so", func(t *testing.T) {
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
	t.Run("unhappy: the landing runner is not the five-scene walkthrough", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		if strings.Contains(v, "pause") || strings.Contains(v, "1/5") {
			t.Fatal("the landing scene must not open as the walkthrough")
		}
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in the landing scene")
		}
		if strings.ContainsRune(v, '▌') {
			t.Fatal("the landing craft is north-facing, not the west hull")
		}
		_ = press(m, space())
	})
}
