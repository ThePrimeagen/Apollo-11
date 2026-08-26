package main

// Demo harness tests, written first: dustoff is the landing kick-up
// demo — two mirrored swirl engines blowing dust out of the floor at
// 15° above horizontal, leftward and rightward, with a still 8-column
// gap between the nozzles. Heavy dust wears gray shades, the fringe
// wears computed braille. The demo reads the same JSON the dust-off
// editor saves.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiPat.ReplaceAllString(s, "") }

// dustAt reports dust glyphs left of and right of the split column.
func dustAt(view string, split int) (left, right bool) {
	for _, line := range strings.Split(stripAnsi(view), "\n") {
		for col, r := range []rune(line) {
			if (r >= '⠀' && r <= '⣿') || r == '░' || r == '▒' {
				if col < split {
					left = true
				}
				if col > split {
					right = true
				}
			}
		}
	}
	return left, right
}

func TestDustoffDemo(t *testing.T) {
	t.Run("happy: the curtain rises on dust blowing out both sides", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		m := newModel(0)
		v := m.View().Content
		left, right := dustAt(v, defaultW/2)
		if !left || !right {
			t.Fatalf("the demo must open dusty on both sides, left=%v right=%v", left, right)
		}
		if !strings.Contains(v, "dust off") {
			t.Fatal("the status line must name the demo")
		}
	})
	t.Run("happy: frames advance the kick and schedule the next tick", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		m := newModel(0)
		mm, cmd := m.Update(frameMsg{})
		m = mm.(model)
		if m.elapsed <= 0 {
			t.Fatal("a frame must advance the clock")
		}
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
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
		t.Cleanup(dust.ResetPuff)
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
		t.Cleanup(dust.ResetPuff)
		m := newModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 8, Height: 3})
		m = mm.(model)
		if m.View().Content == "" {
			t.Fatal("tiny terminals must still render")
		}
	})
}

func TestApplyPuff(t *testing.T) {
	t.Run("happy: a missing config keeps the stock puff without complaint", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		dust.ResetPuff()
		if err := applyPuff(filepath.Join(t.TempDir(), "nope.json")); err != nil {
			t.Fatalf("a missing file must be fine: %v", err)
		}
		if dust.ActivePuff() != dust.DefaultPuff() {
			t.Fatal("a missing file must keep the stock puff")
		}
	})
	t.Run("happy: a real config becomes the active puff", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		path := filepath.Join(t.TempDir(), "dust.json")
		c := dust.DefaultPuff()
		c.Gap = 16
		if err := c.Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		if err := applyPuff(path); err != nil {
			t.Fatalf("applyPuff: %v", err)
		}
		if dust.ActivePuff() != c {
			t.Fatal("the file's puff must go active")
		}
	})
	t.Run("unhappy: a broken config is an error", func(t *testing.T) {
		t.Cleanup(dust.ResetPuff)
		bad := filepath.Join(t.TempDir(), "bad.json")
		if err := os.WriteFile(bad, []byte("{nope"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := applyPuff(bad); err == nil {
			t.Fatal("broken JSON must error")
		}
	})
}
