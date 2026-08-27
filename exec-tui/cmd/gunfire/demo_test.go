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
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
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
		if len(m.blast.FlameAt(sprite.N).Particles) == 0 {
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

func demoLive(m model) int {
	n := len(m.blast.Core.Particles)
	for _, e := range m.blast.Flames {
		n += len(e.Particles)
	}
	return n
}

func TestAim(t *testing.T) {
	t.Run("happy: left/h step the compass counterclockwise, right/l step it back, and the status names the heading", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		_ = m.View()
		if !strings.Contains(m.View().Content, "aim N ") {
			t.Fatal("the status line must read out the stock heading N")
		}
		m = press(m, left())
		if got := gunfire.ActiveBlast().Heading; got != sprite.NW {
			t.Fatalf("one left press aims %s, want NW", got)
		}
		m = press(m, runeKey('h'))
		if got := gunfire.ActiveBlast().Heading; got != sprite.W {
			t.Fatalf("h aims %s, want W", got)
		}
		m = press(m, runeKey('l'))
		m = press(m, right())
		if got := gunfire.ActiveBlast().Heading; got != sprite.N {
			t.Fatalf("l and right must step back to N, got %s", got)
		}
		m = press(m, right())
		if got := gunfire.ActiveBlast().Heading; got != sprite.NE {
			t.Fatalf("right must step clockwise to NE, got %s", got)
		}
		if !strings.Contains(m.View().Content, "aim NE ") {
			t.Fatal("the status line must follow the heading")
		}
	})
	t.Run("happy: the blast flies where the compass points — a W shot goes left", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		_ = m.View()
		m = press(m, left())
		m = press(m, left()) // N -> NW -> W
		if got := gunfire.ActiveBlast().Heading; got != sprite.W {
			t.Fatalf("two left presses aim %s, want W", got)
		}
		m = frames(m, 1)
		m = press(m, space())
		m = frames(m, 2)
		west := m.blast.FlameAt(sprite.W)
		if len(west.Particles) == 0 {
			t.Fatal("the W shot must burn")
		}
		for i, p := range west.Particles {
			if p.Vel.X >= 0 {
				t.Fatalf("flame speck %d flies at %+v — a W shot must head left", i, p.Vel)
			}
		}
		if got := len(m.blast.FlameAt(sprite.N).Particles); got != 0 {
			t.Fatalf("the unfired N flame holds %d specks — only the aimed direction fires", got)
		}
	})
	t.Run("happy: eight steps around the compass come home", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		_ = m.View()
		for i := 0; i < 8; i++ {
			m = press(m, right())
		}
		if got := gunfire.ActiveBlast().Heading; got != sprite.N {
			t.Fatalf("a full turn must come home to N, got %s", got)
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
		if n := demoLive(m); n != 0 {
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
		c.Heading = sprite.SW
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
