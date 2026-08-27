package main

// Demo harness tests, written first: gunfire is the one-shot Doom
// muzzle-flame demo — an empty stage and the trigger on the space bar
// (f works too). One squeeze and the red flame leaps up from the
// muzzle: a white-hot heart, tongues cooling yellow through orange to
// red, and Doom's second flash frame pulsing a beat later — then the
// stage goes dark again, because a gunshot is a trigger, not a clock.
// The demo auto-fires once shortly after boot so tapes show the
// flame, and it reads the same JSON the gunfire tuner saves.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiPat.ReplaceAllString(s, "") }

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

func TestGunfireDemo(t *testing.T) {
	t.Run("happy: the curtain rises on a dark, empty stage — just the status line", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		v := m.View().Content
		if !strings.Contains(v, "gunfire") || !strings.Contains(v, "space fire") {
			t.Fatal("the status line must name the demo and the trigger")
		}
		rows := strings.Split(stripAnsi(v), "\n")
		for i, row := range rows[:len(rows)-1] {
			if strings.TrimSpace(row) != "" {
				t.Fatalf("an untriggered stage must be empty, row %d shows %q", i, row)
			}
		}
	})
	t.Run("happy: space pulls the trigger and the flame blooms white-hot at the muzzle", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		_ = m.View()
		m = press(m, space())
		v := m.View().Content
		if !strings.Contains(v, "█") {
			t.Fatal("the trigger must bloom the white-hot flash core")
		}
	})
	t.Run("happy: f is the second trigger", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey('f'))
		if !strings.Contains(m.View().Content, "█") {
			t.Fatal("f must fire too")
		}
	})
	t.Run("happy: the demo auto-fires once for tapes, then the stage goes dark again", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		idle := m.View().Content
		m = frames(m, 15) // 0.5s: past the 0.4s auto-fire
		if len(m.blast.Flame.Particles) == 0 {
			t.Fatal("0.5s in, the auto-shot's flame must be burning")
		}
		m = frames(m, 165) // 6s total: every life and the pulse fuse long spent
		if got := m.View().Content; got != idle {
			t.Fatal("a burnt-out one-shot must leave the stage exactly as it found it")
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
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
	t.Run("unhappy: q and ctrl+c quit", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		for _, msg := range []tea.Msg{
			tea.KeyPressMsg{Code: 'q', Text: "q"},
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
	t.Run("unhappy: a tiny window still renders", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		m := newModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 8, Height: 3})
		m = mm.(model)
		if m.View().Content == "" {
			t.Fatal("tiny terminals must still render")
		}
	})
}

func left() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyLeft} }

func right() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyRight} }

func TestAim(t *testing.T) {
	t.Run("happy: left/h swing the aim counterclockwise, right/l swing it back, and the status reads it out", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		_ = m.View()
		if !strings.Contains(m.View().Content, "aim 90°") {
			t.Fatal("the status line must read out the stock aim")
		}
		m = press(m, left())
		if got := gunfire.ActiveBlast().AngleDeg; got != 105 {
			t.Fatalf("one left press aims %v°, want 105", got)
		}
		m = press(m, runeKey('h'))
		if got := gunfire.ActiveBlast().AngleDeg; got != 120 {
			t.Fatalf("h aims %v°, want 120", got)
		}
		m = press(m, runeKey('l'))
		m = press(m, right())
		if got := gunfire.ActiveBlast().AngleDeg; got != 90 {
			t.Fatalf("l and right must swing back to 90, got %v", got)
		}
		m = press(m, left())
		if !strings.Contains(m.View().Content, "aim 105°") {
			t.Fatal("the status line must follow the aim")
		}
	})
	t.Run("happy: the blast flies where the aim points — a half-turn shot goes left", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		_ = m.View()
		for i := 0; i < 6; i++ { // 90° + 6×15° = 180°: leftward
			m = press(m, left())
		}
		if got := gunfire.ActiveBlast().AngleDeg; got != 180 {
			t.Fatalf("six left presses aim %v°, want 180", got)
		}
		m = frames(m, 1)
		m = press(m, space())
		m = frames(m, 2)
		if len(m.blast.Flame.Particles) == 0 {
			t.Fatal("the leftward shot must burn")
		}
		for i, p := range m.blast.Flame.Particles {
			if p.Vel.X >= 0 {
				t.Fatalf("flame speck %d flies at %+v — a 180° shot must head left", i, p.Vel)
			}
		}
	})
	t.Run("happy: the aim wraps past the half turn and a full turn comes home", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		_ = m.View()
		for i := 0; i < 7; i++ { // 90° + 105° = 195° -> wraps to -165°
			m = press(m, left())
		}
		if got := gunfire.ActiveBlast().AngleDeg; got != -165 {
			t.Fatalf("seven left presses aim %v°, want the wrap to -165", got)
		}
		for i := 0; i < 17; i++ { // 24 steps of 15° is one full turn
			m = press(m, left())
		}
		if got := gunfire.ActiveBlast().AngleDeg; got != 90 {
			t.Fatalf("a full turn must come home to 90, got %v", got)
		}
	})
	t.Run("unhappy: aiming alone never pulls the trigger", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		_ = m.View()
		for i := 0; i < 10; i++ {
			m = press(m, left())
			m = press(m, right())
		}
		if n := len(m.blast.Flame.Particles) + len(m.blast.Core.Particles); n != 0 {
			t.Fatalf("aim keys spawned %d particles — only the trigger fires", n)
		}
	})
}

func TestApplyBlast(t *testing.T) {
	t.Run("happy: a missing config keeps the stock blast without complaint", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		if err := applyBlast(filepath.Join(t.TempDir(), "nope.json")); err != nil {
			t.Fatalf("a missing file must be fine: %v", err)
		}
		if gunfire.ActiveBlast() != gunfire.DefaultBlast() {
			t.Fatal("a missing file must keep the stock blast")
		}
	})
	t.Run("happy: a real config becomes the active blast", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		path := filepath.Join(t.TempDir(), "blast.json")
		c := gunfire.DefaultBlast()
		c.AngleDeg = 10
		if err := c.Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		if err := applyBlast(path); err != nil {
			t.Fatalf("applyBlast: %v", err)
		}
		if gunfire.ActiveBlast() != c {
			t.Fatal("the file's blast must go active")
		}
	})
	t.Run("unhappy: a broken config is an error", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := applyBlast(bad); err == nil {
			t.Fatal("broken JSON must error")
		}
	})
}
