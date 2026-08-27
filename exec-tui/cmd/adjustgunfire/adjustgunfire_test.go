package adjustgunfire

// Tests written FIRST: adjustgunfire is the muzzle-flame tuner on the
// eight-point compass — the live one-shot flame burning behind a
// paged panel of every blast knob. Ten pages: aim (heading, muzzle,
// the two-frame pulse, the core brightness ladder — eight knobs), the
// shared core (ten engine knobs), then one page per direction — N,
// NE, E, SE, S, SW, W, NW — each carrying the ten engine knobs plus
// the five color stops its flame cools through. tab flips pages, j/k
// pick a knob, h/l turn it, [/] take bigger steps, f pulls the
// trigger now, and the tool re-fires on its own. s saves the gunfire
// component's config and quits. Every change goes live via UseBlast.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestTuner(t *testing.T) {
	t.Run("happy: NewTuner seeds every knob from the active blast, opening on the aim page", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		c := gunfire.DefaultBlast()
		c.Heading = sprite.SE
		if err := gunfire.UseBlast(c); err != nil {
			t.Fatalf("UseBlast: %v", err)
		}
		tu := NewTuner()
		if tu.Blast != c {
			t.Fatalf("tuner seeded %+v, want the active %+v", tu.Blast, c)
		}
		if tu.Page != 0 || tu.Cursor != 0 {
			t.Fatalf("tuner must open on the aim page's first knob, got page %d cursor %d", tu.Page, tu.Cursor)
		}
	})
	t.Run("happy: Flip walks aim, core, and the eight directions, wrapping and clamping the cursor", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		tu := NewTuner()
		for i := 1; i < nPages; i++ {
			tu.Flip(1)
			if tu.Page != i {
				t.Fatalf("flip %d landed on page %d", i, tu.Page)
			}
		}
		tu.Flip(1)
		if tu.Page != 0 {
			t.Fatalf("flipping past the last page must wrap to aim, got %d", tu.Page)
		}
		tu.Flip(-1)
		if tu.Page != nPages-1 {
			t.Fatalf("flipping back from aim must wrap to NW, got %d", tu.Page)
		}
		tu.Move(99) // the NW page's last knob: color 5
		if tu.Cursor != 14 {
			t.Fatalf("cursor %d, want a direction page's last knob 14", tu.Cursor)
		}
		tu.Flip(2) // wraps to core, which has only ten rows
		if tu.Page != 1 || tu.Cursor > 9 {
			t.Fatalf("flipping to a shorter page must clamp the cursor, page %d cursor %d", tu.Page, tu.Cursor)
		}
	})
	t.Run("happy: the heading knob steps the compass and Nudge turns whichever page is open", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		tu := NewTuner()
		tu.Nudge(1) // aim page, heading: N -> NE
		if tu.Blast.Heading != sprite.NE {
			t.Fatalf("heading %s after a nudge, want NE", tu.Blast.Heading)
		}
		tu.Nudge(99)
		if tu.Blast.Heading != sprite.NW {
			t.Fatalf("heading %s, want clamped at the compass end NW", tu.Blast.Heading)
		}
		tu.Nudge(-99)
		if tu.Blast.Heading != sprite.N {
			t.Fatalf("heading %s, want clamped back at N", tu.Blast.Heading)
		}
		tu.Flip(1) // core page, count
		before := tu.Blast.Core.Count
		tu.Nudge(2)
		if tu.Blast.Core.Count != before+2 {
			t.Fatalf("core count %d, want %d", tu.Blast.Core.Count, before+2)
		}
		tu.Flip(3) // the E page
		tu.Move(8) // lift
		lift := tu.Blast.ShotAt(sprite.E).Lift
		tu.Nudge(1)
		if got := tu.Blast.ShotAt(sprite.E).Lift; got != lift+1 {
			t.Fatalf("E lift %v, want %v", got, lift+1)
		}
		if tu.Blast.ShotAt(sprite.N).Lift != lift {
			t.Fatal("turning the E page must leave N alone")
		}
	})
	t.Run("happy: the five color stops turn one at a time and rail on the xterm cube", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		tu := NewTuner()
		tu.Flip(2)  // the N page
		tu.Move(10) // color 1
		tu.Nudge(1)
		if got := tu.Blast.ShotAt(sprite.N).Colors[0]; got != 227 {
			t.Fatalf("color 1 turned to %d, want 227", got)
		}
		tu.Move(4) // color 5
		tu.Nudge(-99999)
		if got := tu.Blast.ShotAt(sprite.N).Colors[4]; got != 1 {
			t.Fatalf("color 5 %d, want the cube floor 1", got)
		}
		tu.Nudge(99999)
		if got := tu.Blast.ShotAt(sprite.N).Colors[4]; got != 255 {
			t.Fatalf("color 5 %d, want the cube ceiling 255", got)
		}
		if tu.Blast.ShotAt(sprite.NE).Colors == tu.Blast.ShotAt(sprite.N).Colors {
			t.Fatal("turning N's colors must leave NE's ramp alone")
		}
		if err := gunfire.UseBlast(tu.Blast); err != nil {
			t.Fatalf("the railed ramp must still be a valid blast: %v", err)
		}
	})
	t.Run("happy: counts floor at zero with no ceiling, ranges swap, the ladder never folds", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		tu := NewTuner()
		tu.Flip(4) // the SE page, count
		start := tu.Blast.ShotAt(sprite.SE).Count
		tu.Nudge(100000)
		if got := tu.Blast.ShotAt(sprite.SE).Count; got != start+100000 {
			t.Fatalf("count %d after +100000, want %d — no artificial ceiling", got, start+100000)
		}
		tu.Nudge(-999999)
		if tu.Blast.ShotAt(sprite.SE).Count != 0 {
			t.Fatalf("count %d, want a hard stop at zero", tu.Blast.ShotAt(sprite.SE).Count)
		}
		tu.Move(3) // min speed
		tu.Nudge(999)
		if s := tu.Blast.ShotAt(sprite.SE); s.MinSpeed > s.MaxSpeed {
			t.Fatalf("speeds %v..%v must swap, never fold", s.MinSpeed, s.MaxSpeed)
		}
		tu = NewTuner()
		tu.Move(5) // aim page, edge at
		tu.Nudge(999)
		if e, m, c := tu.Blast.EdgeAt, tu.Blast.MidAt, tu.Blast.CoreAt; m <= e || c <= m {
			t.Fatalf("pushing the edge to %d must push the ladder past it, mid=%d core=%d", e, m, c)
		}
	})
	t.Run("unhappy: cursor and knob rails clamp and nil tuners skip their cue", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		tu := NewTuner()
		tu.Move(-9)
		if tu.Cursor != 0 {
			t.Fatalf("cursor %d, want clamped at 0", tu.Cursor)
		}
		tu.Move(999) // aim has eight knobs
		if tu.Cursor != 7 {
			t.Fatalf("cursor %d, want clamped at 7", tu.Cursor)
		}
		tu.Move(-99)
		tu.Move(1) // muzzle x
		tu.Nudge(9999)
		if tu.Blast.MuzzleX != 1 {
			t.Fatalf("muzzle x %v, want clamped at the right edge 1", tu.Blast.MuzzleX)
		}
		tu.Nudge(-99999)
		if tu.Blast.MuzzleX != 0 {
			t.Fatalf("muzzle x %v, want clamped at the left edge 0", tu.Blast.MuzzleX)
		}
		var ghost *Tuner
		ghost.Move(1)
		ghost.Flip(1)
		ghost.Nudge(1)
	})
}

func frames(m Model, n int) Model {
	for i := 0; i < n; i++ {
		mm, _ := m.Update(FrameMsg{})
		m = mm.(Model)
	}
	return m
}

func press(m Model, msg tea.Msg) Model {
	mm, _ := m.Update(msg)
	return mm.(Model)
}

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func tab() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyTab} }

func shiftTab() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift} }

func liveCount(m Model) int {
	n := len(m.blast.Core.Particles)
	for _, e := range m.blast.Flames {
		n += len(e.Particles)
	}
	return n
}

func TestModel(t *testing.T) {
	t.Run("happy: the tool boots the aim page over the stage with the compass on the page bar", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := NewModel(0)
		v := m.View().Content
		if !strings.Contains(v, "adjust gunfire") {
			t.Fatal("the view must show the panel")
		}
		for _, page := range []string{"aim", "core", "NE", "SE", "SW", "NW"} {
			if !strings.Contains(v, page) {
				t.Fatalf("the page bar is missing %q", page)
			}
		}
		for _, label := range []string{
			"heading", "muzzle x", "muzzle y", "pulse delay", "pulse frac",
			"edge at", "mid at", "core at",
		} {
			if !strings.Contains(v, label) {
				t.Fatalf("the aim page is missing the %q knob", label)
			}
		}
		if strings.Contains(v, "color 1") {
			t.Fatal("the aim page must not show color knobs")
		}
		if got := len(strings.Split(v, "\n")); got != defaultH {
			t.Fatalf("view has %d lines, want %d", got, defaultH)
		}
	})
	t.Run("happy: tab reaches a direction page with its ten engine knobs and five colors", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := NewModel(0)
		m = press(m, tab())
		v := m.View().Content
		for _, label := range []string{"count", "min life", "max speed", "lift", "drag"} {
			if !strings.Contains(v, label) {
				t.Fatalf("the core page is missing the %q knob", label)
			}
		}
		if strings.Contains(v, "color 1") {
			t.Fatal("the shared core has no color ramp")
		}
		m = press(m, tab())
		v = m.View().Content
		for _, label := range []string{"count", "lift", "drag", "color 1", "color 2", "color 3", "color 4", "color 5"} {
			if !strings.Contains(v, label) {
				t.Fatalf("the N page is missing the %q knob", label)
			}
		}
		m = press(m, shiftTab())
		m = press(m, shiftTab())
		if !strings.Contains(m.View().Content, "heading") {
			t.Fatal("shift+tab twice must flip back to the aim page")
		}
	})
	t.Run("happy: j/k walk the knobs and h/l retune the live blast", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := NewModel(0)
		m = press(m, runeKey('j'))
		if m.tuner.Cursor != 1 {
			t.Fatalf("cursor %d after j, want 1", m.tuner.Cursor)
		}
		m = press(m, runeKey('k'))
		if m.tuner.Cursor != 0 {
			t.Fatalf("cursor %d after k, want 0", m.tuner.Cursor)
		}
		m = press(m, runeKey('l')) // heading N -> NE
		if gunfire.ActiveBlast().Heading != sprite.NE {
			t.Fatalf("active heading %s after l, want NE — the flame must follow the knobs", gunfire.ActiveBlast().Heading)
		}
		m = press(m, runeKey('h'))
		if gunfire.ActiveBlast().Heading != sprite.N {
			t.Fatalf("active heading %s after h, want N", gunfire.ActiveBlast().Heading)
		}
	})
	t.Run("happy: f pulls the trigger right now", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := NewModel(0)
		_ = m.View()
		if liveCount(m) != 0 {
			t.Fatal("the tool must boot with the flame holding fire")
		}
		m = press(m, runeKey('f'))
		if liveCount(m) == 0 {
			t.Fatal("f must fire the flame")
		}
	})
	t.Run("happy: the tool re-fires on its own so the flame keeps burning", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := NewModel(0)
		_ = m.View()
		m = frames(m, 18) // 0.6s: past the first auto trigger
		if liveCount(m) == 0 {
			t.Fatal("the tuner must keep the flame burning without a keypress")
		}
	})
	t.Run("happy: -seconds brings the curtain down", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		timed := NewModel(0.05)
		mm, cmd := timed.Update(FrameMsg{})
		timed = mm.(Model)
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("one frame is 0.033s — too early for a 0.05s curtain")
		}
		_, cmd = timed.Update(FrameMsg{})
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("two frames pass 0.05s — the curtain must fall")
		}
	})
	t.Run("happy: the view follows the window", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		m := NewModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
		m = mm.(Model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 40 {
			t.Fatalf("view has %d lines for a 40-line window", got)
		}
	})
	t.Run("unhappy: q and ctrl+c close the tool", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		for _, msg := range []tea.Msg{
			runeKey('q'),
			tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		} {
			_, cmd := NewModel(0).Update(msg)
			if cmd == nil {
				t.Fatalf("%v must quit", msg)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("%v must issue tea.Quit", msg)
			}
		}
	})
	t.Run("unhappy: stray keys change nothing", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := NewModel(0)
		before := *m.tuner
		m = press(m, runeKey('x'))
		if *m.tuner != before {
			t.Fatal("a stray key must change nothing")
		}
	})
}

func TestOpenSave(t *testing.T) {
	t.Run("happy: Open seeds the knobs from the file and makes it the active blast", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		path := filepath.Join(t.TempDir(), "blast.json")
		saved := gunfire.DefaultBlast()
		saved.Heading = sprite.SW
		shot := saved.ShotAt(sprite.SW)
		shot.Colors = [5]int{21, 27, 33, 39, 45}
		saved.SetShot(sprite.SW, shot)
		if err := saved.Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		m, err := Open(path, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		if m.Path != path {
			t.Fatalf("path %q, want %q", m.Path, path)
		}
		if m.tuner.Blast != saved {
			t.Fatalf("knobs %+v, want the file's %+v", m.tuner.Blast, saved)
		}
		if gunfire.ActiveBlast() != saved {
			t.Fatal("Open must put the file's blast in effect")
		}
	})
	t.Run("happy: s saves the file, quits, and a retuned color round-trips", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		path := filepath.Join(t.TempDir(), "blast.json")
		if err := gunfire.DefaultBlast().Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		m, err := Open(path, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m = press(m, tab())
		m = press(m, tab()) // the N page
		for i := 0; i < 10; i++ {
			m = press(m, runeKey('j'))
		}
		m = press(m, runeKey('l')) // color 1: 226 -> 227
		mm, cmd := m.Update(runeKey('s'))
		m = mm.(Model)
		if cmd == nil {
			t.Fatal("s must quit after a good save")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("s must issue tea.Quit")
		}
		if !m.Saved {
			t.Fatal("a good save must be marked saved")
		}
		got, err := gunfire.LoadBlast(path)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if got.ShotAt(sprite.N).Colors[0] != 227 {
			t.Fatalf("saved N color 1 is %d, want the nudged 227", got.ShotAt(sprite.N).Colors[0])
		}
	})
	t.Run("unhappy: Open on a missing or broken file is an error", func(t *testing.T) {
		if _, err := Open(filepath.Join(t.TempDir(), "nope.json"), 0); err == nil {
			t.Fatal("a missing file must error")
		}
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte(`{"muzzleX":-2}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(bad, 0); err == nil {
			t.Fatal("an out-of-range file must error")
		}
	})
	t.Run("unhappy: a failed save keeps the tool open and says so", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		path := filepath.Join(t.TempDir(), "blast.json")
		if err := gunfire.DefaultBlast().Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		m, err := Open(path, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m.Path = filepath.Join(t.TempDir(), "no", "such", "dir", "blast.json")
		mm, cmd := m.Update(runeKey('s'))
		m = mm.(Model)
		if cmd != nil {
			t.Fatal("a failed save must not quit")
		}
		if m.Saved {
			t.Fatal("a failed save must not be marked saved")
		}
		if !strings.Contains(m.View().Content, "save failed") {
			t.Fatal("the view must say the save failed")
		}
	})
	t.Run("unhappy: s with no config path stays open and says so", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		m := NewModel(0)
		mm, cmd := m.Update(runeKey('s'))
		m = mm.(Model)
		if cmd != nil || m.Saved {
			t.Fatal("saving nowhere must do nothing")
		}
		if !strings.Contains(m.View().Content, "no config") {
			t.Fatal("the view must say there is no config path")
		}
	})
}

func TestConfigHome(t *testing.T) {
	t.Run("happy: the default path points into the gunfire component", func(t *testing.T) {
		if DefaultConfigPath != filepath.Join("components", "gunfire", "config.json") {
			t.Fatalf("DefaultConfigPath = %q, want components/gunfire/config.json", DefaultConfigPath)
		}
	})
	t.Run("happy: the shipped config opens from the module root", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		m, err := Open(filepath.Join("..", "..", DefaultConfigPath), 0)
		if err != nil {
			t.Fatalf("the component's shipped config must open: %v", err)
		}
		if err := m.tuner.Blast.Validate(); err != nil {
			t.Fatalf("the opened config must validate: %v", err)
		}
	})
	t.Run("unhappy: no config hides next to the tuner", func(t *testing.T) {
		if _, err := os.Stat("config.json"); err == nil {
			t.Fatal("config.json must not live in cmd/adjustgunfire — it belongs to components/gunfire")
		}
	})
}

func TestForcedColorProfile(t *testing.T) {
	t.Run("happy: CLICOLOR_FORCE forces ANSI256 for tapes and CI", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "1")
		p, ok := ForcedColorProfile()
		if !ok || p != colorprofile.ANSI256 {
			t.Fatalf("got %v/%v, want ANSI256 forced", p, ok)
		}
	})
	t.Run("unhappy: without the flag, detection is left alone", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "")
		if _, ok := ForcedColorProfile(); ok {
			t.Fatal("an empty CLICOLOR_FORCE must not force a profile")
		}
	})
}
