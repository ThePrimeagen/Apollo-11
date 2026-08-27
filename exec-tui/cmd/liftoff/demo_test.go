package main

// Demo harness tests, written first: cmd/liftoff runs 03. Inverse
// Walkthrough from scenes/liftoff — the walkthrough played backwards.
// The house opens on the landing's final frame: the north lander
// parked on the moon floor, engine cold. The booster ignites, the
// craft climbs off the top, the scene cuts to the tilted-sideways
// west craft with its tail fire on, the fire cuts, and the craft
// bobbles for the rest of the scene. p / enter / space replay from
// the top with the current knobs. j/k select a knob, h/l walk it
// 50ms (dust loss 0.005/ms). s saves. q and ctrl+c quit. The view is
// the rendered screen plus the knob panel, always exactly
// window-height lines.

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
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

func cutFrames() int {
	return int(liftoff.DefaultConfig().CutSeconds()*30) + 3
}

func TestLiftoffSceneRunner(t *testing.T) {
	t.Cleanup(liftoff.Reset)
	t.Run("happy: the house opens on the pad — moon floor, north hull parked, engine cold", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"inverse walkthrough", "play", "50ms", "save", "quit", "rise", "lift at", "fire full", "fire off", "dust start", "dust loss"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.Contains(v, "48;5;") {
			t.Fatal("the liftoff scene must show the moon as a background floor")
		}
		if !strings.ContainsRune(v, '▟') {
			t.Fatal("at t=0 the north hull must already sit on the pad")
		}
		if strings.ContainsRune(v, '▌') {
			t.Fatal("at t=0 the tilted-sideways hull is still a scene away")
		}
		if hasFire(v) {
			t.Fatal("at t=0 the booster must be cold")
		}
	})
	t.Run("happy: Use is the first play, not only a later replay", func(t *testing.T) {
		t.Cleanup(liftoff.Reset)
		c := liftoff.DefaultConfig()
		c.LiftAt = 0
		c.RiseSeconds = 0.2
		c.Fire25 = 0
		c.Fire50 = 0
		c.Fire75 = 0
		c.FireFull = 0
		if err := liftoff.Use(c); err != nil {
			t.Fatal(err)
		}
		m := newModel(0)
		_ = m.View()
		m = frames(m, int(0.3*30+0.5))
		if !strings.ContainsRune(m.View().Content, '▌') {
			t.Fatal("the opening play must already use the saved knobs — a 0.2s climb has cut by now")
		}
	})
	t.Run("happy: ignition burns on the pad, the cut reveals the sideways craft, then the fire goes out", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, int(1.9*30))
		burning := m.View().Content
		if !strings.ContainsRune(burning, '▟') {
			t.Fatal("during ignition the north hull must be on stage")
		}
		if !hasFire(burning) {
			t.Fatal("past full power the booster must be lit")
		}
		if strings.ContainsRune(burning, '▌') {
			t.Fatal("the cut must not play before the climb is over")
		}
		m = frames(m, cutFrames()-int(1.9*30))
		revealed := m.View().Content
		if !strings.ContainsRune(revealed, '▌') {
			t.Fatal("past the cut the tilted-sideways hull must be parked on stage")
		}
		m = frames(m, int((liftoff.FireOff+0.4)*30))
		doused := m.View().Content
		if hasFire(doused) {
			t.Fatal("FireOff seconds after the reveal the tail fire must be out")
		}
		if !strings.ContainsRune(doused, '▌') {
			t.Fatal("the doused craft holds the park for the rest of the scene")
		}
	})
	t.Run("happy: p, enter, and space replay from the pad and do not quit", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, cutFrames())
		if !strings.ContainsRune(m.View().Content, '▌') {
			t.Fatal("test premise: the cut must have played")
		}
		for _, msg := range []tea.Msg{runeKey('p'), enter(), space()} {
			mm, cmd := m.Update(msg)
			if cmd != nil {
				t.Fatalf("%v must replay, not quit", msg)
			}
			m = mm.(model)
			v := m.View().Content
			if strings.ContainsRune(v, '▌') {
				t.Fatalf("%v must rewind the cut away", msg)
			}
			if !strings.ContainsRune(v, '▟') {
				t.Fatalf("%v must rewind the craft onto the pad", msg)
			}
			m = frames(m, cutFrames())
			if !strings.ContainsRune(m.View().Content, '▌') {
				t.Fatalf("after %v the next play must still lift off", msg)
			}
		}
	})
	t.Run("happy: j/k select a knob and h/l walk it by 50ms", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.RiseSeconds; math.Abs(got-(liftoff.RiseSeconds+liftoff.StepSeconds)) > 1e-9 {
			t.Fatalf("rise after +50ms is %v, want %v", got, liftoff.RiseSeconds+liftoff.StepSeconds)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('h'))
		if got := m.show.Cfg.LiftAt; math.Abs(got-(liftoff.LiftAt-liftoff.StepSeconds)) > 1e-9 {
			t.Fatalf("lift at after -50ms is %v, want %v", got, liftoff.LiftAt-liftoff.StepSeconds)
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.Fire25; math.Abs(got-(liftoff.Fire25+liftoff.StepSeconds)) > 1e-9 {
			t.Fatalf("fire ¼ after +50ms is %v, want %v", got, liftoff.Fire25+liftoff.StepSeconds)
		}
		m.cursor = liftoff.KnobDustLoss
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.DustLoss; math.Abs(got-(liftoff.DustLoss+liftoff.StepLoss)) > 1e-9 {
			t.Fatalf("dust loss after +1 step is %v, want %v", got, liftoff.DustLoss+liftoff.StepLoss)
		}
		if !strings.Contains(m.View().Content, "/ms") {
			t.Fatal("dust loss must show as particles per millisecond")
		}
		if !strings.Contains(m.View().Content, ">") {
			t.Fatal("the selected knob must be marked")
		}
		m = press(m, runeKey('k'))
		m = press(m, runeKey('l'))
		if got := m.show.Cfg.DustRun; math.Abs(got-(liftoff.DustRun+liftoff.StepSeconds)) > 1e-9 {
			t.Fatalf("dust run after +50ms is %v, want %v", got, liftoff.DustRun+liftoff.StepSeconds)
		}
	})
	t.Run("unhappy: the rise floor is 50ms, nothing else goes negative, and space does not quit", func(t *testing.T) {
		m := newModel(0)
		m.show.Cfg.RiseSeconds = liftoff.StepSeconds
		m.show.Cfg.LiftAt = 0
		m.show.Cfg.DustStart = 0
		m.show.Cfg.DustRun = 0
		m.show.Cfg.DustLoss = 0
		m.show.Cfg.Fire25 = 0
		m.show.Cfg.FireOff = 0
		m.cursor = liftoff.KnobRise
		m = press(m, runeKey('h'))
		if m.show.Cfg.RiseSeconds != liftoff.StepSeconds {
			t.Fatalf("rise %v, want the 50ms floor", m.show.Cfg.RiseSeconds)
		}
		m.cursor = liftoff.KnobLiftAt
		m = press(m, runeKey('h'))
		if m.show.Cfg.LiftAt != 0 {
			t.Fatalf("lift at %v, want 0", m.show.Cfg.LiftAt)
		}
		m.cursor = liftoff.KnobDustStart
		m = press(m, runeKey('h'))
		if m.show.Cfg.DustStart != 0 {
			t.Fatalf("dust start %v, want 0", m.show.Cfg.DustStart)
		}
		m.cursor = liftoff.KnobDustLoss
		m = press(m, runeKey('h'))
		if m.show.Cfg.DustLoss != 0 {
			t.Fatalf("dust loss %v, want 0", m.show.Cfg.DustLoss)
		}
		m.cursor = liftoff.KnobFire25
		m = press(m, runeKey('h'))
		if m.show.Cfg.Fire25 != 0 {
			t.Fatalf("fire ¼ %v, want 0", m.show.Cfg.Fire25)
		}
		m.cursor = liftoff.KnobFireOff
		m = press(m, runeKey('h'))
		if m.show.Cfg.FireOff != 0 {
			t.Fatalf("fire off %v, want 0", m.show.Cfg.FireOff)
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
		mm, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 34})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 34 {
			t.Fatalf("view has %d lines for a 34-line window", got)
		}
	})
	t.Run("happy: s saves the knobs and does not quit", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "liftoff.json")
		m := newModel(0)
		m.path = path
		m.show.Cfg.RiseSeconds = 4.25
		m.show.Cfg.LiftAt = 1.0
		m.show.Cfg.Fire25 = 0.2
		m.show.Cfg.Fire50 = 0.45
		m.show.Cfg.Fire75 = 0.7
		m.show.Cfg.FireFull = 0.95
		m.show.Cfg.FireOff = 2.5
		m.show.Cfg.DustStart = 0.5
		m.show.Cfg.DustRun = 1.5
		m.show.Cfg.DustLoss = 0.075
		mm, cmd := m.Update(runeKey('s'))
		if cmd != nil {
			t.Fatal("s must save, not quit")
		}
		m = mm.(model)
		if !strings.Contains(m.View().Content, "saved") {
			t.Fatal("a successful save must say so")
		}
		got, err := liftoff.Load(path)
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
	t.Run("unhappy: the inverse walkthrough is not the landing runner", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		if strings.Contains(v, "landing") {
			t.Fatal("the runner must announce the inverse walkthrough, not the landing")
		}
		if strings.Contains(v, "land ") {
			t.Fatal("the knob panel flies a liftoff — there is no land knob")
		}
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in the inverse walkthrough")
		}
		_ = press(m, space())
	})
}
