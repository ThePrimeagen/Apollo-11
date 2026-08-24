package main

// Demo harness tests, written first: stars-lab is a standalone UI for the
// stars component. It does not import lander. n/p cycle fly strategies,
// space pauses, q quits, -strategy picks the opening style.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/stars-lab/stars"
)

func key(m demoModel, r rune) demoModel {
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	return mm.(demoModel)
}

func TestStarLab(t *testing.T) {
	t.Run("happy: boots on dust-rush with a flying field", func(t *testing.T) {
		m := newDemo(stars.DustRush)
		if m.strategy().Name != "dust-rush" {
			t.Fatalf("default strategy %q", m.strategy().Name)
		}
		v := m.View()
		if !strings.Contains(v, "dust-rush") {
			t.Fatal("the UI must name the strategy")
		}
		for _, g := range stars.Glyphs {
			if !strings.ContainsRune(v, g) {
				t.Fatalf("standalone UI missing star %q", string(g))
			}
		}
	})
	t.Run("happy: n/p cycle every named strategy", func(t *testing.T) {
		m := newDemo(stars.FarFast)
		seen := map[string]bool{m.strategy().Name: true}
		for i := 0; i < len(stars.Strategies())+2; i++ {
			m = key(m, 'n')
			seen[m.strategy().Name] = true
		}
		if len(seen) != len(stars.Strategies()) {
			t.Fatalf("n must visit every strategy, saw %v", seen)
		}
		m = key(m, 'p')
		if m.strategy().Name == "" {
			t.Fatal("p must land on a named strategy")
		}
	})
	t.Run("happy: frames advance the tick so the field flies", func(t *testing.T) {
		m := newDemo(stars.Hyperspace)
		t0 := m.tick
		mm, _ := m.Update(frameMsg{})
		m = mm.(demoModel)
		if m.tick <= t0 {
			t.Fatal("each frame must advance the animation tick")
		}
	})
	t.Run("happy: space pauses the fly-by", func(t *testing.T) {
		m := newDemo(stars.FarFast)
		m = key(m, ' ')
		before := m.tick
		m.advance()
		if m.tick != before {
			t.Fatal("a paused field must hold")
		}
		m = key(m, ' ')
		m.advance()
		if m.tick == before {
			t.Fatal("resuming must let the stars fly")
		}
	})
	t.Run("unhappy: q quits", func(t *testing.T) {
		m := newDemo(stars.FarFast)
		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if cmd == nil {
			t.Fatal("q must quit")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("q's command must be tea.Quit")
		}
	})
	t.Run("unhappy: the demo does not pull in the lander", func(t *testing.T) {
		v := newDemo(stars.FarFast).View()
		for _, gone := range []string{"P63", "ALT ", "VEL ", "ft/s"} {
			if strings.Contains(v, gone) {
				t.Fatalf("star lab must be stars only, found %q", gone)
			}
		}
	})
}

func TestLookupFlag(t *testing.T) {
	t.Run("happy: -strategy selects the opening style", func(t *testing.T) {
		m := newDemo(mustStrategy("dust-rush"))
		if m.strategy().Name != "dust-rush" {
			t.Fatalf("got %q", m.strategy().Name)
		}
	})
	t.Run("unhappy: unknown -strategy falls back to dust-rush", func(t *testing.T) {
		s := strategyOrDefault("not-a-style")
		if s.Name != "dust-rush" {
			t.Fatalf("fallback %q", s.Name)
		}
	})
}
