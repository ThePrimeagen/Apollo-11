package adjuststars

// Tests written FIRST: adjuststars is the sky tuner — a screenplay
// Scene playing the whole starfield behind a panel of eight numbers, a
// fly delay and a density for each of the four star layers. j/k pick a
// number, h/l change it, and the sky reacts on the next frame. The
// same lifecycle that runs the premiere runs the tool.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// The tuner must be castable on any screenplay bill.
var _ screenplay.Scene = (*Tuner)(nil)

func started() *Tuner {
	t := NewTuner()
	t.Start()
	return t
}

func contentGrid(scr *screenplay.Screen) []string {
	w, h := scr.Size()
	rows := make([]string, h)
	for y := 0; y < h; y++ {
		var b strings.Builder
		for x := 0; x < w; x++ {
			c := scr.Cell(x, y)
			if c == nil || c.Content == "" {
				b.WriteString(" ")
				continue
			}
			b.WriteString(c.Content)
		}
		rows[y] = b.String()
	}
	return rows
}

func countGlyph(scr *screenplay.Screen, glyph string) int {
	n := 0
	for _, row := range contentGrid(scr) {
		n += strings.Count(row, glyph)
	}
	return n
}

func render(t *Tuner) *screenplay.Screen {
	scr := screenplay.NewScreen(72, 28)
	t.Render(scr)
	return scr
}

func TestTunerLifecycle(t *testing.T) {
	t.Run("happy: Start deals the drift delays and the stock densities", func(t *testing.T) {
		tu := started()
		if tu.Delays != stars.Drift.Delay {
			t.Fatalf("delays %v, want drift's %v", tu.Delays, stars.Drift.Delay)
		}
		if tu.Densities != stars.DefaultDensity {
			t.Fatalf("densities %v, want the stock %v", tu.Densities, stars.DefaultDensity)
		}
		if tu.Cursor != 0 {
			t.Fatalf("cursor %d, want the first number", tu.Cursor)
		}
	})
	t.Run("happy: the sky fills the screen and drifts as time runs", func(t *testing.T) {
		tu := started()
		before := contentGrid(render(tu))
		for _, g := range stars.Glyphs {
			if !strings.Contains(strings.Join(before, "\n"), string(g)) {
				t.Fatalf("sky missing star %q", string(g))
			}
		}
		tu.Update(2.0)
		after := contentGrid(render(tu))
		if strings.Join(before, "\n") == strings.Join(after, "\n") {
			t.Fatal("two seconds must drift the sky")
		}
	})
	t.Run("happy: the panel shows all eight numbers over the sky", func(t *testing.T) {
		grid := contentGrid(render(started()))
		if !strings.HasPrefix(grid[0], "adjust stars") {
			t.Fatalf("header row %q, want the tool's name at top-left", grid[0])
		}
		panel := strings.Join(grid[:1+Rows], "\n")
		for _, want := range []string{"dust", "spark", "mid", "near", "delay", "density", ">"} {
			if !strings.Contains(panel, want) {
				t.Fatalf("panel is missing %q:\n%s", want, panel)
			}
		}
		for kind := 0; kind < 4; kind++ {
			if !strings.Contains(grid[1+kind*2], "delay") {
				t.Fatalf("row %d must be a delay row: %q", 1+kind*2, grid[1+kind*2])
			}
			if !strings.Contains(grid[2+kind*2], "density") {
				t.Fatalf("row %d must be a density row: %q", 2+kind*2, grid[2+kind*2])
			}
		}
	})
	t.Run("happy: a density nudge thickens that layer on the very next render", func(t *testing.T) {
		tu := started()
		before := countGlyph(render(tu), string(stars.Glyphs[3]))
		tu.Move(7) // near density
		tu.Nudge(200)
		after := countGlyph(render(tu), string(stars.Glyphs[3]))
		if after <= before*3 {
			t.Fatalf("near density %d -> %d painted ✦%d -> ✦%d; the sky must react",
				stars.DefaultDensity[3], tu.Densities[3], before, after)
		}
	})
	t.Run("unhappy: dt<=0 holds the sky exactly", func(t *testing.T) {
		tu := started()
		before := contentGrid(render(tu))
		tu.Update(0)
		tu.Update(-3)
		after := contentGrid(render(tu))
		if strings.Join(before, "\n") != strings.Join(after, "\n") {
			t.Fatal("a held clock must hold every star")
		}
	})
	t.Run("unhappy: cursor and knobs clamp at their rails", func(t *testing.T) {
		tu := started()
		tu.Move(-5)
		if tu.Cursor != 0 {
			t.Fatalf("cursor %d, want clamped at 0", tu.Cursor)
		}
		tu.Move(99)
		if tu.Cursor != Rows-1 {
			t.Fatalf("cursor %d, want clamped at %d", tu.Cursor, Rows-1)
		}
		tu.Move(-99)
		tu.Nudge(-99)
		if tu.Delays[0] != 0 {
			t.Fatalf("dust delay %d, want the floor 0 — zero movement parks the layer", tu.Delays[0])
		}
		tu.Nudge(999)
		if tu.Delays[0] != MaxDelay {
			t.Fatalf("dust delay %d, want the ceiling %d", tu.Delays[0], MaxDelay)
		}
		tu.Move(1)
		tu.Nudge(-999)
		if tu.Densities[0] != MinDensity {
			t.Fatalf("dust density %d, want the floor %d", tu.Densities[0], MinDensity)
		}
		tu.Nudge(9999)
		if tu.Densities[0] != MaxDensity {
			t.Fatalf("dust density %d, want the ceiling %d", tu.Densities[0], MaxDensity)
		}
	})
	t.Run("unhappy: nil tuners and nil screens skip their cue", func(t *testing.T) {
		var ghost *Tuner
		ghost.Start()
		ghost.Update(0.1)
		ghost.Render(screenplay.NewScreen(4, 2))
		ghost.Stop()
		ghost.Move(1)
		ghost.Nudge(1)
		started().Render(nil)
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
	t.Run("happy: the tool boots panel-over-sky at full window height", func(t *testing.T) {
		m := NewModel(0)
		v := m.View().Content
		if !strings.Contains(v, "adjust stars") {
			t.Fatal("the view must show the panel")
		}
		star := false
		for _, g := range stars.Glyphs {
			if strings.ContainsRune(v, g) {
				star = true
			}
		}
		if !star {
			t.Fatal("the view must show the sky")
		}
		if got := len(strings.Split(v, "\n")); got != defaultH {
			t.Fatalf("view has %d lines, want %d", got, defaultH)
		}
	})
	t.Run("happy: j/k and the arrows walk the eight numbers", func(t *testing.T) {
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
	})
	t.Run("happy: h/l turn the selected number and the view shows it", func(t *testing.T) {
		m := NewModel(0)
		m = press(m, runeKey('j')) // dust density
		m = press(m, runeKey('l'))
		want := stars.DefaultDensity[0] + 1
		if m.tuner.Densities[0] != want {
			t.Fatalf("dust density %d after l, want %d", m.tuner.Densities[0], want)
		}
		m = press(m, runeKey('h'))
		m = press(m, tea.KeyPressMsg{Code: tea.KeyLeft})
		if m.tuner.Densities[0] != want-2 {
			t.Fatalf("dust density %d after h+left, want %d", m.tuner.Densities[0], want-2)
		}
		m = press(m, tea.KeyPressMsg{Code: tea.KeyRight})
		if m.tuner.Densities[0] != want-1 {
			t.Fatalf("dust density %d after right, want %d", m.tuner.Densities[0], want-1)
		}
	})
	t.Run("happy: frames run the clock and schedule the next", func(t *testing.T) {
		m := NewModel(0)
		mm, cmd := m.Update(FrameMsg{})
		m = mm.(Model)
		if m.elapsed <= 0 {
			t.Fatal("a frame must advance the clock")
		}
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
		m := NewModel(0.05)
		mm, cmd := m.Update(FrameMsg{})
		m = mm.(Model)
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("one frame is 0.033s — too early for a 0.05s curtain")
		}
		_, cmd = m.Update(FrameMsg{})
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("two frames pass 0.05s — the curtain must fall")
		}
	})
	t.Run("happy: the view follows the window", func(t *testing.T) {
		m := NewModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
		m = mm.(Model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 40 {
			t.Fatalf("view has %d lines for a 40-line window", got)
		}
	})
	t.Run("unhappy: q and ctrl+c close the tool", func(t *testing.T) {
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
	t.Run("unhappy: nudging past a rail sticks, stray keys change nothing", func(t *testing.T) {
		m := NewModel(0)
		for i := 0; i < MaxDelay+9; i++ {
			m = press(m, runeKey('h'))
		}
		if m.tuner.Delays[0] != 0 {
			t.Fatalf("dust delay %d, want stuck at the 0 floor", m.tuner.Delays[0])
		}
		before := *m.tuner
		m = press(m, runeKey('x'))
		if *m.tuner != before {
			t.Fatal("a stray key must change nothing")
		}
	})
}

func TestFileLifecycle(t *testing.T) {
	t.Run("happy: Open seeds the eight knobs from the file and makes it the active sky", func(t *testing.T) {
		t.Cleanup(stars.ResetSky)
		path := filepath.Join(t.TempDir(), "sky.json")
		saved := stars.SkyConfig{Delay: []int{2, 3, 4, 5}, Density: []int{10, 20, 30, 40}}
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
		if m.tuner.Delays != [4]int{2, 3, 4, 5} || m.tuner.Densities != [4]int{10, 20, 30, 40} {
			t.Fatalf("knobs %v/%v, want the file's", m.tuner.Delays, m.tuner.Densities)
		}
		if stars.ActiveSky().DensityLayers() != [4]int{10, 20, 30, 40} {
			t.Fatal("Open must put the file's sky in effect")
		}
	})
	t.Run("happy: every nudge is applied as the active sky, live", func(t *testing.T) {
		t.Cleanup(stars.ResetSky)
		m := NewModel(0)
		m = press(m, runeKey('j')) // dust density
		m = press(m, runeKey('l'))
		want := stars.DefaultDensity[0] + 1
		if got := stars.ActiveSky().DensityLayers()[0]; got != want {
			t.Fatalf("active dust density %d after l, want %d — the sky must follow the knobs", got, want)
		}
	})
	t.Run("happy: s saves the file and quits, and the file round-trips", func(t *testing.T) {
		t.Cleanup(stars.ResetSky)
		path := filepath.Join(t.TempDir(), "sky.json")
		if err := stars.DefaultSky().Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		m, err := Open(path, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m = press(m, runeKey('l')) // dust delay +1
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
		got, err := stars.LoadSky(path)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		if got.FlyStrategy().Delay[0] != stars.Drift.Delay[0]+1 {
			t.Fatalf("saved dust delay %d, want the nudged %d", got.FlyStrategy().Delay[0], stars.Drift.Delay[0]+1)
		}
	})
	t.Run("unhappy: Open on a missing or broken file is an error", func(t *testing.T) {
		if _, err := Open(filepath.Join(t.TempDir(), "nope.json"), 0); err == nil {
			t.Fatal("a missing file must error")
		}
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte(`{"delay":[0,0,0,0],"density":[0,0,0,0]}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(bad, 0); err == nil {
			t.Fatal("an out-of-range file must error")
		}
	})
	t.Run("unhappy: a failed save keeps the tool open and says so", func(t *testing.T) {
		t.Cleanup(stars.ResetSky)
		path := filepath.Join(t.TempDir(), "sky.json")
		if err := stars.DefaultSky().Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		m, err := Open(path, 0)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		m.Path = filepath.Join(t.TempDir(), "no", "such", "dir", "sky.json")
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
