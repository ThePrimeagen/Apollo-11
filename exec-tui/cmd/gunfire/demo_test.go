package main

// Demo harness tests, written first: gunfire is the one-shot shotgun
// demo — a bare stage, a stub of a barrel at the muzzle, and the
// trigger on the space bar (f works too). One squeeze blooms the
// white-hot flash, throws the Doom seven pellets and a fan of sparks,
// and curls gunsmoke out on a short fuse — then the stage goes quiet
// again, because a gunshot is a trigger, not a clock. The demo
// auto-fires once shortly after boot so tapes show the shot, and it
// reads the same JSON the gunfire tuner saves.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
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

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestGunfireDemo(t *testing.T) {
	t.Run("happy: the curtain rises on a quiet stage — a barrel, a status line, no blast", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		v := m.View().Content
		if !strings.Contains(v, "gunfire") || !strings.Contains(v, "space fire") {
			t.Fatal("the status line must name the demo and the trigger")
		}
		if !strings.Contains(v, "━") {
			t.Fatal("the stage must show the barrel at the muzzle")
		}
		for _, glyph := range []string{"█", "▓", "•", "░"} {
			if strings.Contains(v, glyph) {
				t.Fatalf("an untriggered stage shows %q — the one-shot must hold fire", glyph)
			}
		}
	})
	t.Run("happy: space pulls the trigger and the flash blooms white-hot at the muzzle", func(t *testing.T) {
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
	t.Run("happy: the demo auto-fires once for tapes, then the stage goes quiet again", func(t *testing.T) {
		t.Cleanup(gunfire.ResetBlast)
		gunfire.ResetBlast()
		m := newModel(0)
		idle := m.View().Content
		m = frames(m, 15) // 0.5s: past the 0.4s auto-fire
		if len(m.blast.Pellets.Particles) == 0 {
			t.Fatal("0.5s in, the auto-shot's pellets must be flying")
		}
		m = frames(m, 165) // 6s total: every life and the smoke fuse long spent
		if got := m.View().Content; got != idle {
			t.Fatal("a played-out one-shot must leave the stage exactly as it found it")
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
