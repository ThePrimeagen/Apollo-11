package adjustgunfire

// Tests written FIRST: adjustgunfire is the muzzle-flame tuner — the
// live one-shot flame burning behind a paged panel of every blast
// knob. Three pages: aim (angle, muzzle, the two-frame pulse, the
// core brightness ladder — eight knobs) and one page per layer — core
// and flame — each carrying count, life, speed, spread, nozzle, max
// distance, lift, and drag (ten knobs). tab flips pages, j/k pick a
// knob, h/l turn it, [/] take bigger steps, f pulls the trigger now,
// and the tool re-fires on its own so the flame is always burning.
// s saves the gunfire component's config and quits. Every change
// goes live via UseBlast.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
)

func TestTuner(t *testing.T) {
	t.Run("happy: NewTuner seeds every knob from the active blast, opening on the aim page", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		c := gunfire.DefaultBlast()
		c.AngleDeg = 45
		c.Flame.Count = 50
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
	t.Run("happy: Flip walks the three pages, wraps both ways, and clamps the cursor", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		tu := NewTuner()
		tu.Flip(1)
		if tu.Page != 1 {
			t.Fatalf("flip landed on page %d, want core", tu.Page)
		}
		tu.Flip(1)
		if tu.Page != 2 {
			t.Fatalf("flip landed on page %d, want flame", tu.Page)
		}
		tu.Flip(1)
		if tu.Page != 0 {
			t.Fatalf("flipping past the last page must wrap to aim, got %d", tu.Page)
		}
		tu.Flip(-1)
		if tu.Page != nPages-1 {
			t.Fatalf("flipping back from aim must wrap to flame, got %d", tu.Page)
		}
		tu.Move(99) // drag, the flame page's tenth row
		if tu.Cursor != 9 {
			t.Fatalf("cursor %d, want the flame page's last knob 9", tu.Cursor)
		}
		tu.Flip(1) // aim has only eight rows
		if tu.Cursor > 7 {
			t.Fatalf("flipping to a shorter page must clamp the cursor, got %d", tu.Cursor)
		}
	})
	t.Run("happy: Nudge turns the knob under the cursor on whichever page is open", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		tu := NewTuner()
		tu.Nudge(-1) // aim page, angle: stock 90 comes down
		if tu.Blast.AngleDeg != gunfire.DefaultBlast().AngleDeg-1 {
			t.Fatalf("angle %v after a nudge, want %v", tu.Blast.AngleDeg, gunfire.DefaultBlast().AngleDeg-1)
		}
		tu.Flip(1) // core page, count
		before := tu.Blast.Core.Count
		tu.Nudge(2)
		if tu.Blast.Core.Count != before+2 {
			t.Fatalf("core count %d, want %d", tu.Blast.Core.Count, before+2)
		}
		tu.Flip(1) // flame page
		tu.Move(8) // lift
		lift := tu.Blast.Flame.Lift
		tu.Nudge(1)
		if tu.Blast.Flame.Lift != lift+1 {
			t.Fatalf("flame lift %v, want %v", tu.Blast.Flame.Lift, lift+1)
		}
		tu.Move(1) // drag
		drag := tu.Blast.Flame.Drag
		tu.Nudge(1)
		if tu.Blast.Flame.Drag <= drag {
			t.Fatalf("flame drag %v must climb above %v", tu.Blast.Flame.Drag, drag)
		}
	})
	t.Run("happy: counts have a hard zero floor and no ceiling", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		tu := NewTuner()
		tu.Flip(2) // flame count
		start := tu.Blast.Flame.Count
		tu.Nudge(100000)
		if got := tu.Blast.Flame.Count; got != start+100000 {
			t.Fatalf("count %d after +100000, want %d — no artificial ceiling", got, start+100000)
		}
		tu.Nudge(-999999)
		if tu.Blast.Flame.Count != 0 {
			t.Fatalf("count %d, want a hard stop at zero", tu.Blast.Flame.Count)
		}
		if err := gunfire.UseBlast(tu.Blast); err != nil {
			t.Fatalf("a silent flame must still be a valid blast: %v", err)
		}
	})
	t.Run("happy: reversed ranges swap and the core ladder never folds", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		tu := NewTuner()
		tu.Flip(2) // flame page
		tu.Move(3) // min speed
		tu.Nudge(999)
		if tu.Blast.Flame.MinSpeed > tu.Blast.Flame.MaxSpeed {
			t.Fatalf("speeds %v..%v must swap, never fold", tu.Blast.Flame.MinSpeed, tu.Blast.Flame.MaxSpeed)
		}
		tu = NewTuner()
		tu.Flip(1) // core page
		tu.Move(2) // max life
		tu.Nudge(-999)
		if tu.Blast.Core.MinLife > tu.Blast.Core.MaxLife {
			t.Fatalf("lives %v..%v must swap, never fold", tu.Blast.Core.MinLife, tu.Blast.Core.MaxLife)
		}
		tu = NewTuner()
		tu.Move(5) // edge at
		tu.Nudge(999)
		if e, m, c := tu.Blast.EdgeAt, tu.Blast.MidAt, tu.Blast.CoreAt; m <= e || c <= m {
			t.Fatalf("pushing the edge to %d must push the ladder past it, mid=%d core=%d", e, m, c)
		}
		tu.Move(2) // core at
		tu.Nudge(-999)
		if m, c := tu.Blast.MidAt, tu.Blast.CoreAt; c <= m {
			t.Fatalf("dropping the core must stop above mid %d, core=%d", m, c)
		}
	})
	t.Run("unhappy: cursor, page rails, and knob rails clamp", func(t *testing.T) {
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
		tu.Move(-99) // angle
		tu.Nudge(999)
		if tu.Blast.AngleDeg != 90 {
			t.Fatalf("angle %v, want clamped at straight up 90", tu.Blast.AngleDeg)
		}
		tu.Nudge(-9999)
		if tu.Blast.AngleDeg != -90 {
			t.Fatalf("angle %v, want clamped at straight down -90", tu.Blast.AngleDeg)
		}
		tu.Move(1) // muzzle x
		tu.Nudge(9999)
		if tu.Blast.MuzzleX != 1 {
			t.Fatalf("muzzle x %v, want clamped at the right edge 1", tu.Blast.MuzzleX)
		}
		tu.Nudge(-99999)
		if tu.Blast.MuzzleX != 0 {
			t.Fatalf("muzzle x %v, want clamped at the left edge 0", tu.Blast.MuzzleX)
		}
		tu.Move(3) // pulse frac
		tu.Nudge(9999)
		if tu.Blast.PulseFrac != 1 {
			t.Fatalf("pulse frac %v, want clamped at a full second frame 1", tu.Blast.PulseFrac)
		}
	})
	t.Run("unhappy: nil tuners skip their cue", func(t *testing.T) {
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
	return len(m.blast.Core.Particles) + len(m.blast.Flame.Particles)
}

func TestModel(t *testing.T) {
	t.Run("happy: the tool boots the aim page over the stage", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := NewModel(0)
		v := m.View().Content
		if !strings.Contains(v, "adjust gunfire") {
			t.Fatal("the view must show the panel")
		}
		for _, page := range []string{"aim", "core", "flame"} {
			if !strings.Contains(v, page) {
				t.Fatalf("the page bar is missing %q", page)
			}
		}
		for _, label := range []string{
			"angle", "muzzle x", "muzzle y", "pulse delay", "pulse frac",
			"edge at", "mid at", "core at",
		} {
			if !strings.Contains(v, label) {
				t.Fatalf("the aim page is missing the %q knob", label)
			}
		}
		if got := len(strings.Split(v, "\n")); got != defaultH {
			t.Fatalf("view has %d lines, want %d", got, defaultH)
		}
	})
	t.Run("happy: tab flips to the core page and shift+tab flips back", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := NewModel(0)
		m = press(m, tab())
		v := m.View().Content
		for _, label := range []string{
			"count", "min life", "max life", "min speed", "max speed",
			"spread", "nozzle", "max dist", "lift", "drag",
		} {
			if !strings.Contains(v, label) {
				t.Fatalf("the core page is missing the %q knob", label)
			}
		}
		m = press(m, shiftTab())
		if !strings.Contains(m.View().Content, "angle") {
			t.Fatal("shift+tab must flip back to the aim page")
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
		want := gunfire.DefaultBlast().AngleDeg - 1
		m = press(m, runeKey('h'))
		if gunfire.ActiveBlast().AngleDeg != want {
			t.Fatalf("active angle %v after h, want %v — the flame must follow the knobs", gunfire.ActiveBlast().AngleDeg, want)
		}
		m = press(m, runeKey('['))
		if gunfire.ActiveBlast().AngleDeg != want-10 {
			t.Fatalf("active angle %v after [, want %v", gunfire.ActiveBlast().AngleDeg, want-10)
		}
		m = press(m, runeKey(']'))
		m = press(m, runeKey('l'))
		if gunfire.ActiveBlast().AngleDeg != want+1 {
			t.Fatalf("active angle %v after ] and l, want %v", gunfire.ActiveBlast().AngleDeg, want+1)
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
		saved.AngleDeg = 60
		saved.Flame.Lift = 44
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
	t.Run("happy: s saves the file, quits, and the file round-trips", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		path := filepath.Join(t.TempDir(), "blast.json")
		if err := gunfire.DefaultBlast().Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		m, err := Open(path, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m = press(m, runeKey('h')) // angle -1: the stock 90 comes down
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
		if got.AngleDeg != gunfire.DefaultBlast().AngleDeg-1 {
			t.Fatalf("saved angle %v, want the nudged %v", got.AngleDeg, gunfire.DefaultBlast().AngleDeg-1)
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
