package director

// Tests written FIRST: the director is the screenplay editor. The
// whole bill plays on one stage; n and p walk the cuts both ways and
// never end the show. A knob panel rides over the scene now playing —
// row one is the editor's own hold (how long the scene plays in play
// mode before the cut), the rows under it are the scene's own knobs —
// j/k pick a row, h/l turn it, and the hold is never clamped: zero
// and negative are the operator's numbers. Space is the play button:
// the scene restarts and the bill cuts itself forward on each hold;
// on the last hold the play just stops — the editor never quits on
// its own. f is the fullscreen premiere: the chrome drops, the show
// rewinds to the top, and it plays through; f or esc hands the chrome
// back, and the end of the show does too. s saves the holds file and
// the current scene's knobs, syncing only the moonwalk's sibling
// beats. q and ctrl+c are the only ways out.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/fall"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// probe is a scene that counts its lifecycle, so a test can see cuts.
type probe struct {
	starts, stops int
	updated       float64
}

func (p *probe) Start()                        { p.starts++ }
func (p *probe) Update(dt float64)             { p.updated += dt }
func (p *probe) Render(scr *screenplay.Screen) {}
func (p *probe) Stop()                         { p.stops++ }

func probeBill() (screenplay.Bill, []*probe) {
	ps := []*probe{{}, {}, {}}
	return screenplay.Bill{
		screenplay.Entry{Name: "one", Scene: ps[0]},
		screenplay.Entry{Name: "two", Scene: ps[1]},
		screenplay.Entry{Name: "three", Scene: ps[2]},
	}, ps
}

func press(m Model, msg tea.Msg) Model {
	mm, _ := m.Update(msg)
	return mm.(Model)
}

func frames(m Model, n int) Model {
	for i := 0; i < n; i++ {
		mm, _ := m.Update(frameMsg{})
		m = mm.(Model)
	}
	return m
}

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func space() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "} }

func esc() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyEscape} }

// tmpResolve maps module-relative config paths into a temp dir, the
// parent folders made on the way, so a save never touches the repo.
func tmpResolve(t *testing.T) (string, func(string) string) {
	t.Helper()
	root := t.TempDir()
	return root, func(rel string) string {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
}

func TestDirectorWalk(t *testing.T) {
	t.Run("happy: the house opens on scene one with the marquee and the help", func(t *testing.T) {
		bill, ps := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		if ps[0].starts != 1 {
			t.Fatalf("the first curtain must rise once, rose %d", ps[0].starts)
		}
		v := m.View().Content
		for _, want := range []string{"TEST", "1/3", "one", "n/p scene", "space play", "f premiere", "s save", "q quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
	})
	t.Run("happy: New seeds a hold for every scene in bill order", func(t *testing.T) {
		bill, _ := probeBill()
		var seed Config
		seed.SetHold("two", 1.5)
		m := New("TEST", bill, seed, "holds.json", 0)
		if len(m.holds.Holds) != 3 {
			t.Fatalf("the editor carries %d holds, want one per scene", len(m.holds.Holds))
		}
		for i, e := range bill {
			if m.holds.Holds[i].Scene != e.Name {
				t.Fatalf("hold %d is %q, want %q", i, m.holds.Holds[i].Scene, e.Name)
			}
		}
		if m.holds.HoldFor("two") != 1.5 || m.holds.HoldFor("one") != DefaultHoldSeconds {
			t.Fatal("seeding must keep loaded holds and stock the rest")
		}
	})
	t.Run("happy: n cuts forward, p cuts back, and the clocks reset", func(t *testing.T) {
		bill, ps := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m.clock, m.cursor = 5, 0
		m = press(m, runeKey('n'))
		if m.play.SceneIndex() != 1 || ps[0].stops != 1 || ps[1].starts != 1 {
			t.Fatalf("n must cut to scene two: idx %d stops %d starts %d",
				m.play.SceneIndex(), ps[0].stops, ps[1].starts)
		}
		if m.clock != 0 {
			t.Fatalf("a cut must reset the hold clock, got %v", m.clock)
		}
		if !strings.Contains(m.View().Content, "2/3") {
			t.Fatal("the marquee must follow the cut")
		}
		m.clock = 5
		m = press(m, runeKey('p'))
		if m.play.SceneIndex() != 0 || ps[1].stops != 1 || ps[0].starts != 2 {
			t.Fatalf("p must cut back to scene one: idx %d stops %d starts %d",
				m.play.SceneIndex(), ps[1].stops, ps[0].starts)
		}
		if m.clock != 0 {
			t.Fatalf("a cut back must reset the hold clock, got %v", m.clock)
		}
	})
	t.Run("unhappy: the ends hold — p on the first and n on the last never quit", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		mm, cmd := m.Update(runeKey('p'))
		m = mm.(Model)
		if cmd != nil || m.play.SceneIndex() != 0 {
			t.Fatal("p on the first scene must hold quietly")
		}
		m = press(m, runeKey('n'))
		m = press(m, runeKey('n'))
		mm, cmd = m.Update(runeKey('n'))
		m = mm.(Model)
		if cmd != nil || m.play.SceneIndex() != 2 {
			t.Fatal("n on the last scene must hold — the editor never ends the show")
		}
	})
	t.Run("happy: r replays the scene from its top", func(t *testing.T) {
		bill, ps := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m.clock = 3
		m = press(m, runeKey('r'))
		if ps[0].stops != 1 || ps[0].starts != 2 {
			t.Fatalf("r must stop and start the scene: stops %d starts %d", ps[0].stops, ps[0].starts)
		}
		if m.clock != 0 {
			t.Fatalf("a replay must reset the hold clock, got %v", m.clock)
		}
	})
	t.Run("happy: frames reach the scene now playing and schedule the next tick", func(t *testing.T) {
		bill, ps := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		if m.Init() == nil {
			t.Fatal("Init must start the clock")
		}
		mm, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
		m = mm.(Model)
		if ps[0].updated <= 0 {
			t.Fatal("a frame must reach the scene now playing")
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0.05)
		mm, cmd := m.Update(frameMsg{})
		m = mm.(Model)
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("one frame is 0.033s — too early for a 0.05s curtain")
		}
		_, cmd = m.Update(frameMsg{})
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("two frames pass 0.05s — the curtain must fall")
		}
	})
	t.Run("unhappy: q and ctrl+c close the editor from anywhere", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			runeKey('q'),
			tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		} {
			bill, _ := probeBill()
			m := New("TEST", bill, Config{}, "holds.json", 0)
			_, cmd := m.Update(msg)
			if cmd == nil {
				t.Fatalf("%v must quit", msg)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("%v must issue tea.Quit", msg)
			}
		}
	})
	t.Run("unhappy: a tiny window never breaks the view", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m = press(m, tea.WindowSizeMsg{Width: 2, Height: 2})
		if got := len(strings.Split(m.View().Content, "\n")); got != 2 {
			t.Fatalf("view has %d lines for a 2-line window", got)
		}
		m = press(m, tea.WindowSizeMsg{Width: 90, Height: 32})
		if got := len(strings.Split(m.View().Content, "\n")); got != 32 {
			t.Fatalf("view has %d lines for a 32-line window", got)
		}
	})
}

func TestDirectorKnobPanel(t *testing.T) {
	editorBill := func() screenplay.Bill {
		return screenplay.Bill{
			screenplay.Entry{Name: "blank", Scene: &screenplay.Ensemble{}},
			screenplay.Entry{Name: "drop", Scene: fall.New(nil)},
		}
	}
	t.Run("happy: the panel is the hold plus the scene's own knobs", func(t *testing.T) {
		m := New("TEST", editorBill(), Config{}, "holds.json", 0)
		v := m.View().Content
		if !strings.Contains(v, "hold") {
			t.Fatal("the panel must carry the editor's hold knob")
		}
		if strings.Contains(v, "drop") {
			t.Fatal("the blank scene has no drop knob")
		}
		m = press(m, runeKey('n'))
		v = m.View().Content
		if !strings.Contains(v, "hold") || !strings.Contains(v, "drop") {
			t.Fatal("the fall scene's panel is the hold then its drop knob")
		}
	})
	t.Run("happy: h and l turn the hold — never clamped, zero and negative stand", func(t *testing.T) {
		var seed Config
		seed.SetHold("blank", 0.5)
		m := New("TEST", editorBill(), seed, "holds.json", 0)
		m = press(m, runeKey('l'))
		if got := m.holds.HoldFor("blank"); got != 0.5+HoldStepSeconds {
			t.Fatalf("one l reads %v, want %v", got, 0.5+HoldStepSeconds)
		}
		m = press(m, runeKey('h'))
		m = press(m, runeKey('h'))
		if got := m.holds.HoldFor("blank"); got != 0 {
			t.Fatalf("back to zero reads %v, want exactly 0", got)
		}
		m = press(m, runeKey('h'))
		if got := m.holds.HoldFor("blank"); got != -HoldStepSeconds {
			t.Fatalf("one more h reads %v, want %v — the floor is the operator's", got, -HoldStepSeconds)
		}
	})
	t.Run("happy: j walks onto the scene's knob and h/l turn that knob", func(t *testing.T) {
		t.Cleanup(fall.Reset)
		bill := editorBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m = press(m, runeKey('n'))
		m = press(m, runeKey('j'))
		if m.cursor != 1 {
			t.Fatalf("j must move to the drop knob, cursor %d", m.cursor)
		}
		show := bill[1].Scene.(*fall.Show)
		before := show.Cfg.DropSeconds
		m = press(m, runeKey('l'))
		if show.Cfg.DropSeconds != before+fall.StepSeconds {
			t.Fatalf("l on the drop knob reads %v, want %v", show.Cfg.DropSeconds, before+fall.StepSeconds)
		}
		hold := m.holds.HoldFor("drop")
		m = press(m, runeKey('k'))
		m = press(m, runeKey('l'))
		if got := m.holds.HoldFor("drop"); got != hold+HoldStepSeconds {
			t.Fatalf("k then l must turn the hold again, %v want %v", got, hold+HoldStepSeconds)
		}
	})
	t.Run("happy: the cursor wraps around the panel", func(t *testing.T) {
		m := New("TEST", editorBill(), Config{}, "holds.json", 0)
		m = press(m, runeKey('n'))
		m = press(m, runeKey('k'))
		if m.cursor != 1 {
			t.Fatalf("k off the top must wrap to the last knob, cursor %d", m.cursor)
		}
		m = press(m, runeKey('j'))
		if m.cursor != 0 {
			t.Fatalf("j off the bottom must wrap to the hold, cursor %d", m.cursor)
		}
	})
	t.Run("happy: a cut resets the cursor to the hold row", func(t *testing.T) {
		m := New("TEST", editorBill(), Config{}, "holds.json", 0)
		m = press(m, runeKey('n'))
		m = press(m, runeKey('j'))
		m = press(m, runeKey('p'))
		if m.cursor != 0 {
			t.Fatalf("a cut must land the cursor on the hold, got %d", m.cursor)
		}
	})
	t.Run("unhappy: a knobless scene keeps the cursor on its one row", func(t *testing.T) {
		m := New("TEST", editorBill(), Config{}, "holds.json", 0)
		m = press(m, runeKey('j'))
		m = press(m, runeKey('k'))
		if m.cursor != 0 {
			t.Fatalf("the blank scene has only the hold, cursor %d", m.cursor)
		}
	})
}

func TestDirectorPlayMode(t *testing.T) {
	shortHolds := func(bill screenplay.Bill, seconds float64) Config {
		var c Config
		for _, e := range bill {
			c.SetHold(e.Name, seconds)
		}
		return c
	}
	t.Run("happy: space plays — the scene restarts and the bill cuts itself on the hold", func(t *testing.T) {
		bill, ps := probeBill()
		m := New("TEST", bill, shortHolds(bill, 0.05), "holds.json", 0)
		m = press(m, space())
		if !m.playing {
			t.Fatal("space must start the play")
		}
		if ps[0].starts != 2 || ps[0].stops != 1 {
			t.Fatalf("play must restart the scene from its top: starts %d stops %d", ps[0].starts, ps[0].stops)
		}
		if !strings.Contains(m.View().Content, "▶") {
			t.Fatal("the marquee must show the play")
		}
		m = frames(m, 1)
		if m.play.SceneIndex() != 0 {
			t.Fatal("one frame is 0.033s — the 0.05s hold has not elapsed")
		}
		m = frames(m, 1)
		if m.play.SceneIndex() != 1 {
			t.Fatalf("two frames pass the hold — the bill must cut, idx %d", m.play.SceneIndex())
		}
		m = frames(m, 2)
		if m.play.SceneIndex() != 2 {
			t.Fatalf("the next hold must cut again, idx %d", m.play.SceneIndex())
		}
	})
	t.Run("happy: past the last hold the play stops and the editor stays", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, shortHolds(bill, 0.05), "holds.json", 0)
		m = press(m, space())
		m = frames(m, 8)
		if m.playing {
			t.Fatal("the last hold must stop the play")
		}
		if m.play.SceneIndex() != 2 {
			t.Fatalf("the editor must rest on the last scene, idx %d", m.play.SceneIndex())
		}
		mm, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("the editor must keep ticking after the show ends")
		}
		_ = mm
	})
	t.Run("happy: a zero hold cuts on the very next frame — never rewritten", func(t *testing.T) {
		bill, _ := probeBill()
		holds := shortHolds(bill, 5)
		holds.SetHold("one", 0)
		m := New("TEST", bill, holds, "holds.json", 0)
		m = press(m, space())
		m = frames(m, 1)
		if m.play.SceneIndex() != 1 {
			t.Fatalf("a zero hold plays zero seconds, idx %d", m.play.SceneIndex())
		}
	})
	t.Run("happy: a negative hold is the operator's — it cuts immediately too", func(t *testing.T) {
		bill, _ := probeBill()
		holds := shortHolds(bill, 5)
		holds.SetHold("one", -3)
		m := New("TEST", bill, holds, "holds.json", 0)
		m = press(m, space())
		m = frames(m, 1)
		if m.play.SceneIndex() != 1 {
			t.Fatalf("a negative hold must not be clamped into a wait, idx %d", m.play.SceneIndex())
		}
	})
	t.Run("unhappy: space again pauses — the bill stops cutting itself", func(t *testing.T) {
		bill, ps := probeBill()
		m := New("TEST", bill, shortHolds(bill, 0.05), "holds.json", 0)
		m = press(m, space())
		m = press(m, space())
		if m.playing {
			t.Fatal("space must toggle the play off")
		}
		starts := ps[0].starts
		m = frames(m, 10)
		if m.play.SceneIndex() != 0 {
			t.Fatalf("a paused editor must hold the scene, idx %d", m.play.SceneIndex())
		}
		if ps[0].starts != starts {
			t.Fatal("pausing must not restart the scene")
		}
		if ps[0].updated <= 0 {
			t.Fatal("a paused editor still animates the scene now playing")
		}
	})
	t.Run("unhappy: manual cuts while playing keep the play and reset the hold", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, shortHolds(bill, 5), "holds.json", 0)
		m = press(m, space())
		m.clock = 4.9
		m = press(m, runeKey('n'))
		if !m.playing {
			t.Fatal("a manual cut must not stop the play")
		}
		if m.clock != 0 {
			t.Fatalf("a manual cut must reset the hold clock, got %v", m.clock)
		}
	})
}

func TestDirectorPremiere(t *testing.T) {
	shortHolds := func(bill screenplay.Bill, seconds float64) Config {
		var c Config
		for _, e := range bill {
			c.SetHold(e.Name, seconds)
		}
		return c
	}
	t.Run("happy: f drops the chrome and plays the show from the top", func(t *testing.T) {
		bill, ps := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m = press(m, runeKey('n'))
		m = press(m, runeKey('n'))
		m = press(m, runeKey('f'))
		if !m.full || !m.playing {
			t.Fatalf("f must go fullscreen and play: full %v playing %v", m.full, m.playing)
		}
		if m.play.SceneIndex() != 0 || ps[0].starts != 2 || ps[2].stops != 1 {
			t.Fatalf("the premiere must rewind to a fresh scene one: idx %d starts %d stops %d",
				m.play.SceneIndex(), ps[0].starts, ps[2].stops)
		}
		v := m.View().Content
		for _, chrome := range []string{"TEST", "1/3", "n/p scene", "hold"} {
			if strings.Contains(v, chrome) {
				t.Fatalf("fullscreen must drop the chrome, found %q", chrome)
			}
		}
		if got, want := len(strings.Split(v, "\n")), m.h; got != want {
			t.Fatalf("the premiere owns %d lines of a %d-line window", got, want)
		}
	})
	t.Run("happy: f again hands the chrome back and stops the play", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m = press(m, runeKey('f'))
		m = press(m, runeKey('f'))
		if m.full || m.playing {
			t.Fatalf("f must toggle back to the editor: full %v playing %v", m.full, m.playing)
		}
		if !strings.Contains(m.View().Content, "n/p scene") {
			t.Fatal("the chrome must return")
		}
	})
	t.Run("happy: esc leaves the premiere too", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m = press(m, runeKey('f'))
		m = press(m, esc())
		if m.full || m.playing {
			t.Fatal("esc must leave the premiere")
		}
	})
	t.Run("happy: the end of the show hands the chrome back on its own", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, shortHolds(bill, 0.05), "holds.json", 0)
		m = press(m, runeKey('f'))
		m = frames(m, 10)
		if m.full || m.playing {
			t.Fatalf("the show's end must exit the premiere: full %v playing %v", m.full, m.playing)
		}
		if m.play.SceneIndex() != 2 {
			t.Fatalf("the editor must rest on the last scene, idx %d", m.play.SceneIndex())
		}
	})
	t.Run("happy: the premiere screen owns the whole window", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m = press(m, tea.WindowSizeMsg{Width: 100, Height: 40})
		if w, h := m.screen.Size(); w != 100 || h != 39 {
			t.Fatalf("the editor screen is %dx%d, want 100x39 over the status line", w, h)
		}
		m = press(m, runeKey('f'))
		if w, h := m.screen.Size(); w != 100 || h != 40 {
			t.Fatalf("the premiere screen is %dx%d, want the whole 100x40", w, h)
		}
		m = press(m, esc())
		if w, h := m.screen.Size(); w != 100 || h != 39 {
			t.Fatalf("the editor screen is %dx%d after the premiere, want 100x39", w, h)
		}
	})
	t.Run("unhappy: esc outside the premiere is a quiet no-op", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		mm, cmd := m.Update(esc())
		m = mm.(Model)
		if cmd != nil || m.full || m.playing {
			t.Fatal("esc with the chrome up must do nothing")
		}
	})
	t.Run("unhappy: q quits straight out of the premiere", func(t *testing.T) {
		bill, _ := probeBill()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m = press(m, runeKey('f'))
		_, cmd := m.Update(runeKey('q'))
		if cmd == nil {
			t.Fatal("q must quit from the premiere")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("q must issue tea.Quit")
		}
	})
}

func TestDirectorSave(t *testing.T) {
	saveBill := func() screenplay.Bill {
		return screenplay.Bill{
			screenplay.Entry{Name: "blank", Scene: &screenplay.Ensemble{}},
			screenplay.Entry{Name: "drop", Scene: fall.New(nil)},
			screenplay.Entry{Name: "run", Scene: moonwalk.New(moonwalk.BeatRun)},
			screenplay.Entry{Name: "pole", Scene: moonwalk.New(moonwalk.BeatPole)},
			screenplay.Entry{Name: "engines on", Scene: bobble.New(nil).Lit()},
			screenplay.Entry{Name: "engines off", Scene: bobble.New(nil).Dark()},
		}
	}
	t.Run("happy: s writes the holds and the scene's knobs, and says so", func(t *testing.T) {
		t.Cleanup(fall.Reset)
		bill := saveBill()
		root, resolve := tmpResolve(t)
		m := New("TEST", bill, Config{}, "shows/mainshow/config.json", 0)
		m.resolve = resolve
		m = press(m, runeKey('n'))
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		m = press(m, runeKey('s'))
		if m.note != "saved" {
			t.Fatalf("the status must read saved, got %q", m.note)
		}
		holds, err := Load(filepath.Join(root, "shows", "mainshow", "config.json"))
		if err != nil {
			t.Fatalf("the holds file must exist: %v", err)
		}
		if len(holds.Holds) != len(bill) {
			t.Fatalf("the holds file carries %d scenes, want every one of the %d", len(holds.Holds), len(bill))
		}
		for i, e := range bill {
			if holds.Holds[i].Scene != e.Name {
				t.Fatalf("holds file row %d is %q, want %q", i, holds.Holds[i].Scene, e.Name)
			}
		}
		saved, err := fall.Load(filepath.Join(root, filepath.FromSlash(fall.DefaultConfigPath)))
		if err != nil {
			t.Fatalf("the fall config must exist: %v", err)
		}
		show := bill[1].Scene.(*fall.Show)
		if saved.DropSeconds != show.Cfg.DropSeconds {
			t.Fatalf("the file carries drop %v, want the show's %v", saved.DropSeconds, show.Cfg.DropSeconds)
		}
		if fall.Active().DropSeconds != show.Cfg.DropSeconds {
			t.Fatal("s must make the scene's knobs active")
		}
	})
	t.Run("happy: saving the moonwalk syncs its sibling beats", func(t *testing.T) {
		t.Cleanup(moonwalk.Reset)
		bill := saveBill()
		_, resolve := tmpResolve(t)
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m.resolve = resolve
		run := bill[2].Scene.(*moonwalk.Show)
		pole := bill[3].Scene.(*moonwalk.Show)
		m = press(m, runeKey('n'))
		m = press(m, runeKey('n'))
		run.Cfg.RunSpeed += 7
		m = press(m, runeKey('s'))
		if pole.Cfg.RunSpeed != run.Cfg.RunSpeed {
			t.Fatalf("the pole beat must pick up the saved sprint: %v want %v", pole.Cfg.RunSpeed, run.Cfg.RunSpeed)
		}
		if pole.Beat() != moonwalk.BeatPole {
			t.Fatal("a sync must not change which beat a show plays")
		}
	})
	t.Run("happy: saving one bobble leaves the other's ride and engine alone", func(t *testing.T) {
		t.Cleanup(bobble.Reset)
		bill := saveBill()
		_, resolve := tmpResolve(t)
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m.resolve = resolve
		lit := bill[4].Scene.(*bobble.Show)
		dark := bill[5].Scene.(*bobble.Show)
		darkPeriod := dark.Cfg.PeriodSeconds
		for i := 0; i < 4; i++ {
			m = press(m, runeKey('n'))
		}
		m = press(m, runeKey('j'))
		m = press(m, runeKey('j'))
		m = press(m, runeKey('l'))
		m = press(m, runeKey('s'))
		if lit.Cfg.PeriodSeconds == darkPeriod {
			t.Fatal("the nudge must land on the lit ride's period")
		}
		if dark.Cfg.PeriodSeconds != darkPeriod {
			t.Fatal("the dark ride keeps its own period — the bobble never syncs")
		}
		if dark.Cfg.Engine {
			t.Fatal("the bill said dark — a save must never relight the engine")
		}
	})
	t.Run("happy: a knobless scene saves just the holds", func(t *testing.T) {
		bill := saveBill()
		root, resolve := tmpResolve(t)
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m.resolve = resolve
		m = press(m, runeKey('l'))
		m = press(m, runeKey('s'))
		if m.note != "saved" {
			t.Fatalf("the status must read saved, got %q", m.note)
		}
		holds, err := Load(filepath.Join(root, "holds.json"))
		if err != nil {
			t.Fatalf("the holds file must exist: %v", err)
		}
		if got := holds.HoldFor("blank"); got != DefaultHoldSeconds+HoldStepSeconds {
			t.Fatalf("the file carries hold %v, want %v", got, DefaultHoldSeconds+HoldStepSeconds)
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(fall.DefaultConfigPath))); !os.IsNotExist(err) {
			t.Fatal("a knobless scene must not write any scene config")
		}
	})
	t.Run("happy: the note is transient — the next key hands the help line back", func(t *testing.T) {
		bill := saveBill()
		_, resolve := tmpResolve(t)
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m.resolve = resolve
		m = press(m, runeKey('s'))
		if m.note != "saved" {
			t.Fatalf("the status must read saved, got %q", m.note)
		}
		m = press(m, runeKey('n'))
		if m.note != "" {
			t.Fatalf("the next action must clear the note, got %q", m.note)
		}
		if !strings.Contains(m.View().Content, "n/p scene") {
			t.Fatal("the help line must return once the note clears")
		}
	})
	t.Run("unhappy: a failed save lands on the status line and the editor keeps going", func(t *testing.T) {
		bill := saveBill()
		root := t.TempDir()
		m := New("TEST", bill, Config{}, "holds.json", 0)
		m.resolve = func(rel string) string {
			return filepath.Join(root, "missing-dir", filepath.FromSlash(rel))
		}
		m = press(m, runeKey('s'))
		if !strings.Contains(m.note, "save failed") {
			t.Fatalf("the status must carry the failure, got %q", m.note)
		}
		if !strings.Contains(m.View().Content, "save failed") {
			t.Fatal("the failure must reach the status line")
		}
		mm, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a failed save must not stop the editor")
		}
		_ = mm
	})
}
