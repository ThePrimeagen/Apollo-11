package adjustflame

// Tests written FIRST. The adjust-flame TUI loads heat thresholds from
// JSON, walks them with j/k, nudges the selected one with h/l (0..500),
// and s writes the file and quits.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/fire"
)

func writeCfg(t *testing.T, c fire.HeatConfig) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "flame.json")
	if err := c.Save(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func send(m Model, msg tea.Msg) (Model, tea.Cmd) {
	got, cmd := m.Update(msg)
	return got.(Model), cmd
}

func key(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func TestOpen(t *testing.T) {
	t.Run("happy: Open reads the JSON thresholds", func(t *testing.T) {
		c := fire.DefaultHeat()
		c.Thresholds[2] = 20
		path := writeCfg(t, c)
		m, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		if m.Path != path {
			t.Fatalf("path %q, want %q", m.Path, path)
		}
		if m.Cursor != 0 {
			t.Fatalf("cursor %d, want 0", m.Cursor)
		}
		if m.Thresholds[2] != 20 {
			t.Fatalf("threshold[2]=%d, want 20", m.Thresholds[2])
		}
	})
	t.Run("unhappy: a missing file is an error", func(t *testing.T) {
		if _, err := Open(filepath.Join(t.TempDir(), "missing.json")); err == nil {
			t.Fatal("missing config must fail")
		}
	})
	t.Run("unhappy: invalid JSON is an error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(path, []byte("not-json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(path); err == nil {
			t.Fatal("invalid JSON must fail")
		}
	})
}

func TestCursorJK(t *testing.T) {
	t.Run("happy: j and k walk the rungs", func(t *testing.T) {
		m, err := Open(writeCfg(t, fire.DefaultHeat()))
		if err != nil {
			t.Fatal(err)
		}
		m, _ = send(m, key('j'))
		if m.Cursor != 1 {
			t.Fatalf("j must move down, cursor=%d", m.Cursor)
		}
		m, _ = send(m, key('k'))
		if m.Cursor != 0 {
			t.Fatalf("k must move up, cursor=%d", m.Cursor)
		}
	})
	t.Run("unhappy: j/k clamp at the ends", func(t *testing.T) {
		m, err := Open(writeCfg(t, fire.DefaultHeat()))
		if err != nil {
			t.Fatal(err)
		}
		m, _ = send(m, key('k'))
		if m.Cursor != 0 {
			t.Fatalf("k at top must stay, cursor=%d", m.Cursor)
		}
		m.Cursor = len(m.Thresholds) - 1
		m, _ = send(m, key('j'))
		if m.Cursor != len(m.Thresholds)-1 {
			t.Fatalf("j at bottom must stay, cursor=%d", m.Cursor)
		}
	})
}

func TestNudgeHL(t *testing.T) {
	t.Run("happy: l raises and h lowers the selected threshold", func(t *testing.T) {
		c := fire.DefaultHeat()
		c.Thresholds[0] = 10
		m, err := Open(writeCfg(t, c))
		if err != nil {
			t.Fatal(err)
		}
		m, _ = send(m, key('l'))
		if m.Thresholds[0] != 11 {
			t.Fatalf("l must increment, got %d", m.Thresholds[0])
		}
		m, _ = send(m, key('h'))
		if m.Thresholds[0] != 10 {
			t.Fatalf("h must decrement, got %d", m.Thresholds[0])
		}
	})
	t.Run("unhappy: h at 0 and l at 500 stay put", func(t *testing.T) {
		c := fire.DefaultHeat()
		c.Thresholds[0] = 0
		c.Thresholds[1] = 500
		m, err := Open(writeCfg(t, c))
		if err != nil {
			t.Fatal(err)
		}
		m, _ = send(m, key('h'))
		if m.Thresholds[0] != 0 {
			t.Fatalf("h at 0 must clamp, got %d", m.Thresholds[0])
		}
		m.Cursor = 1
		m, _ = send(m, key('l'))
		if m.Thresholds[1] != 500 {
			t.Fatalf("l at 500 must clamp, got %d", m.Thresholds[1])
		}
	})
}

func TestSaveQuit(t *testing.T) {
	t.Run("happy: s writes JSON and quits", func(t *testing.T) {
		path := writeCfg(t, fire.DefaultHeat())
		m, err := Open(path)
		if err != nil {
			t.Fatal(err)
		}
		m, _ = send(m, key('l'))
		m, cmd := send(m, key('s'))
		if cmd == nil {
			t.Fatal("s must return a quit command")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var got fire.HeatConfig
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if got.Thresholds[0] != fire.DefaultHeat().Thresholds[0]+1 {
			t.Fatalf("saved %v, want first threshold bumped", got.Thresholds)
		}
		if !m.Saved {
			t.Fatal("s must mark the model saved")
		}
	})
	t.Run("unhappy: a failed save does not quit", func(t *testing.T) {
		m, err := Open(writeCfg(t, fire.DefaultHeat()))
		if err != nil {
			t.Fatal(err)
		}
		m.Path = filepath.Join(t.TempDir(), "no", "such", "flame.json")
		m, cmd := send(m, key('s'))
		if cmd != nil {
			t.Fatal("a failed save must not quit")
		}
		if m.Err == "" {
			t.Fatal("a failed save must set an error")
		}
	})
}

func TestView(t *testing.T) {
	t.Run("happy: the view lists every rung and marks the cursor", func(t *testing.T) {
		m, err := Open(writeCfg(t, fire.DefaultHeat()))
		if err != nil {
			t.Fatal(err)
		}
		v := m.View().Content
		for _, want := range []string{"single", "braille", "yellow", "230", "j/k", "h/l", "s"} {
			if !strings.Contains(v, want) {
				t.Fatalf("view missing %q\n%s", want, v)
			}
		}
		if !strings.Contains(v, ">") {
			t.Fatal("view must mark the selected rung")
		}
	})
	t.Run("unhappy: an empty model still renders", func(t *testing.T) {
		v := Model{}.View().Content
		if v == "" {
			t.Fatal("empty model must still render something")
		}
	})
}

func TestPage(t *testing.T) {
	t.Run("happy: the page is the sliders plus all eight headings", func(t *testing.T) {
		m, err := Open(writeCfg(t, fire.DefaultHeat()))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 24; i++ {
			m, _ = send(m, TickMsg{})
		}
		sp := m.Page()
		if sp.Width < fire.CompassCols || sp.Height < fire.CompassRows {
			t.Fatalf("page %dx%d is too small for the rose", sp.Width, sp.Height)
		}
		v := m.View().Content
		for _, want := range []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW", "230"} {
			if !strings.Contains(v, want) {
				t.Fatalf("page missing %q\n%s", want, v)
			}
		}
		var lit int
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				if !sp.At(r, c).Transparent() && sp.At(r, c).Ch != ' ' {
					lit++
				}
			}
		}
		if lit < 20 {
			t.Fatalf("expected a live rose on the page, lit=%d", lit)
		}
	})
	t.Run("unhappy: a tick with no rose does not panic", func(t *testing.T) {
		m := Model{}
		m, _ = send(m, TickMsg{})
		if m.View().Content == "" {
			t.Fatal("empty tick must still render")
		}
	})
}

func TestWriteTape(t *testing.T) {
	t.Run("happy: WriteTape writes n same-size frames of the page", func(t *testing.T) {
		m, err := Open(writeCfg(t, fire.DefaultHeat()))
		if err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		paths, err := m.WriteTape(dir, 2, 8)
		if err != nil {
			t.Fatal(err)
		}
		if len(paths) != 2 {
			t.Fatalf("paths %d, want 2", len(paths))
		}
		for i, p := range paths {
			st, err := os.Stat(p)
			if err != nil || st.Size() == 0 {
				t.Fatalf("frame %d missing: %v", i, err)
			}
		}
	})
	t.Run("unhappy: a zero-frame tape is an error", func(t *testing.T) {
		m, err := Open(writeCfg(t, fire.DefaultHeat()))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := m.WriteTape(t.TempDir(), 0, 8); err == nil {
			t.Fatal("n<=0 must fail")
		}
	})
}
