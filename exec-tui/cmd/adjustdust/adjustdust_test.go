package adjustdust

// Tests written FIRST: adjustdust is the dust-off tuner — the live
// mirrored kick playing behind a panel of the puff knobs: engine
// numbers, the kick geometry (angle, gap, loop side), and the gray
// ladder that maps concentration onto braille, ░, and ▒. j/k pick a
// knob, h/l change it, [/] take bigger steps, s saves the dust
// component's config and quits. Every change goes live via UsePuff.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
)

func TestTuner(t *testing.T) {
	t.Run("happy: NewTuner seeds every knob from the active puff", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		c := dust.DefaultPuff()
		c.Count = 9
		c.AngleDeg = 30
		if err := dust.UsePuff(c); err != nil {
			t.Fatalf("UsePuff: %v", err)
		}
		tu := NewTuner()
		if tu.Puff != c {
			t.Fatalf("tuner seeded %+v, want the active %+v", tu.Puff, c)
		}
		if tu.Cursor != 0 {
			t.Fatalf("cursor %d, want the first knob", tu.Cursor)
		}
	})
	t.Run("happy: move and nudge walk and turn the knobs", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		tu := NewTuner()
		before := tu.Puff.Count
		tu.Nudge(1)
		if tu.Puff.Count != before+1 {
			t.Fatalf("count %d after a nudge, want %d", tu.Puff.Count, before+1)
		}
		tu.Move(8) // angle
		a := tu.Puff.AngleDeg
		tu.Nudge(-1)
		if tu.Puff.AngleDeg != a-1 {
			t.Fatalf("angle %v after -1, want %v", tu.Puff.AngleDeg, a-1)
		}
		tu.Move(2) // loop side
		tu.Nudge(-1)
		if tu.Puff.LoopUp {
			t.Fatal("nudging the loop knob down must flip it to a downward loop")
		}
		tu.Nudge(1)
		if !tu.Puff.LoopUp {
			t.Fatal("nudging the loop knob up must flip it back upward")
		}
	})
	t.Run("happy: count climbs as high as you can hold — no ceiling", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		tu := NewTuner()
		start := tu.Puff.Count
		tu.Nudge(100000)
		if got := tu.Puff.Count; got != start+100000 {
			t.Fatalf("count %d after +100000 nudges, want %d — no artificial ceiling", got, start+100000)
		}
		if err := dust.UsePuff(tu.Puff); err != nil {
			t.Fatalf("a huge count must be a valid puff: %v", err)
		}
	})
	t.Run("happy: count reaches exactly zero and the silent puff applies", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		tu := NewTuner()
		tu.Nudge(-tu.Puff.Count)
		if tu.Puff.Count != 0 {
			t.Fatalf("count %d, want exactly zero", tu.Puff.Count)
		}
		if err := dust.UsePuff(tu.Puff); err != nil {
			t.Fatalf("a zero count must be a valid puff: %v", err)
		}
		if dust.ActivePuff().Count != 0 {
			t.Fatalf("active count %d, want the zero we set", dust.ActivePuff().Count)
		}
	})
	t.Run("happy: the ladder never folds and reversed ranges swap", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		tu := NewTuner()
		tu.Move(11) // quarter at
		tu.Nudge(999)
		if q, h := tu.Puff.QuarterAt, tu.Puff.HalfAt; h <= q {
			t.Fatalf("pushing quarter to %d must push half past it, half=%d", q, h)
		}
		tu.Move(1) // half at
		tu.Nudge(-999)
		if q, h := tu.Puff.QuarterAt, tu.Puff.HalfAt; h <= q {
			t.Fatalf("dropping half must stop above quarter %d, half=%d", q, h)
		}
		tu = NewTuner()
		tu.Move(4) // min speed
		tu.Nudge(999)
		if tu.Puff.MinSpeed > tu.Puff.MaxSpeed {
			t.Fatalf("speeds %v..%v must swap, never fold", tu.Puff.MinSpeed, tu.Puff.MaxSpeed)
		}
		tu = NewTuner()
		tu.Move(3) // max life
		tu.Nudge(-999)
		if tu.Puff.MinLife > tu.Puff.MaxLife {
			t.Fatalf("lives %v..%v must swap, never fold", tu.Puff.MinLife, tu.Puff.MaxLife)
		}
	})
	t.Run("unhappy: cursor and knobs clamp at their rails", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		tu := NewTuner()
		tu.Move(-9)
		if tu.Cursor != 0 {
			t.Fatalf("cursor %d, want clamped at 0", tu.Cursor)
		}
		tu.Move(999)
		if tu.Cursor != nKnobs-1 {
			t.Fatalf("cursor %d, want clamped at %d", tu.Cursor, nKnobs-1)
		}
		tu.Nudge(999) // half gray tops out on the gray ramp
		if tu.Puff.HalfFG != dust.GrayMax {
			t.Fatalf("half gray %d, want the ramp's end %d", tu.Puff.HalfFG, dust.GrayMax)
		}
		tu.Move(-999)
		tu.Nudge(-999)
		if tu.Puff.Count != 0 {
			t.Fatalf("count %d, want a hard stop at zero — never negative, no floor above it", tu.Puff.Count)
		}
		tu.Move(8) // angle
		tu.Nudge(999)
		if tu.Puff.AngleDeg > 85 {
			t.Fatalf("angle %v blew past its ceiling", tu.Puff.AngleDeg)
		}
	})
	t.Run("unhappy: nil tuners skip their cue", func(t *testing.T) {
		var ghost *Tuner
		ghost.Move(1)
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

func TestModel(t *testing.T) {
	t.Run("happy: the tool boots the panel over live dust", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		m := NewModel(0)
		v := m.View().Content
		if !strings.Contains(v, "adjust dustoff") {
			t.Fatal("the view must show the panel")
		}
		for _, label := range []string{
			"count", "period", "min life", "max life", "min speed", "max speed",
			"spread", "nozzle", "angle", "gap", "loop", "quarter at", "half at",
			"braille gray", "quarter gray", "half gray",
		} {
			if !strings.Contains(v, label) {
				t.Fatalf("panel is missing the %q knob", label)
			}
		}
		if !strings.Contains(v, "up") {
			t.Fatal("the loop knob must read its side by name")
		}
		dusty := strings.ContainsAny(v, "░▒")
		for _, r := range v {
			if r >= '⠀' && r <= '⣿' {
				dusty = true
			}
		}
		if !dusty {
			t.Fatal("the view must show live dust behind the panel")
		}
		if got := len(strings.Split(v, "\n")); got != defaultH {
			t.Fatalf("view has %d lines, want %d", got, defaultH)
		}
	})
	t.Run("happy: j/k walk the knobs and h/l retune the live puff", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		m := NewModel(0)
		m = press(m, runeKey('j'))
		if m.tuner.Cursor != 1 {
			t.Fatalf("cursor %d after j, want 1", m.tuner.Cursor)
		}
		m = press(m, tea.KeyPressMsg{Code: tea.KeyDown})
		if m.tuner.Cursor != 2 {
			t.Fatalf("cursor %d after down, want 2", m.tuner.Cursor)
		}
		m = press(m, runeKey('k'))
		m = press(m, tea.KeyPressMsg{Code: tea.KeyUp})
		if m.tuner.Cursor != 0 {
			t.Fatalf("cursor %d after k+up, want 0", m.tuner.Cursor)
		}
		want := dust.ActivePuff().Count + 1
		m = press(m, runeKey('l'))
		if dust.ActivePuff().Count != want {
			t.Fatalf("active count %d after l, want %d — the dust must follow the knobs", dust.ActivePuff().Count, want)
		}
		m = press(m, runeKey('h'))
		m = press(m, runeKey(']'))
		if got := dust.ActivePuff().Count; got != want-1+10 {
			t.Fatalf("active count %d after ] , want %d", got, want-1+10)
		}
		m = press(m, runeKey('['))
		if got := dust.ActivePuff().Count; got != want-1 {
			t.Fatalf("active count %d after [ , want %d", got, want-1)
		}
	})
	t.Run("happy: frames run the clock and -seconds brings the curtain down", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		m := NewModel(0)
		mm, cmd := m.Update(FrameMsg{})
		m = mm.(Model)
		if m.elapsed <= 0 {
			t.Fatal("a frame must advance the clock")
		}
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
		timed := NewModel(0.05)
		mm, cmd = timed.Update(FrameMsg{})
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
		t.Cleanup(dust.ResetPuff)
		m := NewModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
		m = mm.(Model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 40 {
			t.Fatalf("view has %d lines for a 40-line window", got)
		}
	})
	t.Run("unhappy: q and ctrl+c close the tool", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
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
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		m := NewModel(0)
		before := *m.tuner
		m = press(m, runeKey('x'))
		if *m.tuner != before {
			t.Fatal("a stray key must change nothing")
		}
	})
}

func TestOpenSave(t *testing.T) {
	t.Run("happy: Open seeds the knobs from the file and makes it the active puff", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		path := filepath.Join(t.TempDir(), "dust.json")
		saved := dust.DefaultPuff()
		saved.Count = 4
		saved.Gap = 14
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
		if m.tuner.Puff != saved {
			t.Fatalf("knobs %+v, want the file's %+v", m.tuner.Puff, saved)
		}
		if dust.ActivePuff() != saved {
			t.Fatal("Open must put the file's puff in effect")
		}
	})
	t.Run("happy: s saves the file, quits, and the file round-trips", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		path := filepath.Join(t.TempDir(), "dust.json")
		if err := dust.DefaultPuff().Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		m, err := Open(path, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m = press(m, runeKey('l')) // count +1
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
		got, err := dust.LoadPuff(path)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if got.Count != dust.DefaultPuff().Count+1 {
			t.Fatalf("saved count %d, want the nudged %d", got.Count, dust.DefaultPuff().Count+1)
		}
	})
	t.Run("unhappy: Open on a missing or broken file is an error", func(t *testing.T) {
		if _, err := Open(filepath.Join(t.TempDir(), "nope.json"), 0); err == nil {
			t.Fatal("a missing file must error")
		}
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte(`{"count":-2}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(bad, 0); err == nil {
			t.Fatal("an out-of-range file must error")
		}
	})
	t.Run("unhappy: a failed save keeps the tool open and says so", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		path := filepath.Join(t.TempDir(), "dust.json")
		if err := dust.DefaultPuff().Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		m, err := Open(path, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m.Path = filepath.Join(t.TempDir(), "no", "such", "dir", "dust.json")
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
		t.Cleanup(dust.ResetPuff)
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
	t.Run("happy: the default path points into the dust component", func(t *testing.T) {
		if DefaultConfigPath != filepath.Join("components", "dust", "config.json") {
			t.Fatalf("DefaultConfigPath = %q, want components/dust/config.json", DefaultConfigPath)
		}
	})
	t.Run("happy: the shipped config opens from the module root", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		m, err := Open(filepath.Join("..", "..", DefaultConfigPath), 0)
		if err != nil {
			t.Fatalf("the component's shipped config must open: %v", err)
		}
		if err := m.tuner.Puff.Validate(); err != nil {
			t.Fatalf("the opened config must validate: %v", err)
		}
	})
	t.Run("unhappy: no config hides next to the tuner", func(t *testing.T) {
		if _, err := os.Stat("config.json"); err == nil {
			t.Fatal("config.json must not live in cmd/adjustdust — it belongs to components/dust")
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
