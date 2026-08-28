package main

// Demo harness tests, written first: cmd/moon runs the moon screenplay
// — the composable two-scene bill from shows/moonshow. The house opens
// on "the moon": the bare disc alone under a parked sky, nothing
// moving. Waiting never conjures the lander — space cuts to "orbit",
// where the lander streaks in off the left wing, brakes onto the
// ring, and circles indefinitely. It must not already sit on the ring
// at the cut. Space on the last scene ends the show — there is
// nothing left, so the program quits. q and ctrl+c quit anywhere.
// The view is the rendered screen plus one status line, always
// exactly window-height lines.

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
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

func hasStar(v string) bool {
	for _, g := range stars.Glyphs {
		if strings.ContainsRune(v, g) {
			return true
		}
	}
	return false
}

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// markerCell locates the gold ship in a rendered view, ANSI stripped
// so escape bytes never shift the column count.
func markerCell(v string) (row, col int, ok bool) {
	for r, line := range strings.Split(ansiPat.ReplaceAllString(v, ""), "\n") {
		for c, ch := range []rune(line) {
			if ch == moon.MarkerGlyph {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

// skyColumns is the leftmost n columns of every view row, ANSI
// stripped — pure sky, west of the ring, so it watches the stars alone.
func skyColumns(v string, n int) string {
	var b strings.Builder
	for _, line := range strings.Split(ansiPat.ReplaceAllString(v, ""), "\n") {
		rs := []rune(line)
		if len(rs) > n {
			rs = rs[:n]
		}
		b.WriteString(string(rs))
		b.WriteString("\n")
	}
	return b.String()
}

func TestMoonScreenplay(t *testing.T) {
	t.Run("happy: the house opens on scene 1/2 — just the moon", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"1/2", "the moon", "space", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !strings.ContainsRune(v, '▓') {
			t.Fatal("the moon must fill the middle of the stage")
		}
		if strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("no ship yet — the opening scene is just the moon")
		}
		if !hasStar(v) {
			t.Fatal("the moon plays under the stars")
		}
	})
	t.Run("happy: scene one holds perfectly still — nothing moves at all", func(t *testing.T) {
		m := newModel(0)
		before := m.View().Content
		m = frames(m, 90)
		if m.View().Content != before {
			t.Fatal("the bare moon under a parked sky must not move a cell")
		}
	})
	t.Run("unhappy: waiting out the old arrival delay never brings the lander", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, int(moon.ArriveSeconds*30)+30)
		v := m.View().Content
		if !strings.Contains(v, "1/2") {
			t.Fatal("waiting must leave the house on scene one")
		}
		if strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("the lander must not appear until space cuts — there is no delay cue")
		}
	})
	t.Run("happy: space cuts to scene 2/2 — the lander streaks in and orbits indefinitely", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 30)
		m = press(m, space())
		opening := m.View().Content
		for _, want := range []string{"2/2", "orbit"} {
			if !strings.Contains(opening, want) {
				t.Fatalf("the orbit scene is missing %q", want)
			}
		}
		if strings.ContainsRune(opening, moon.MarkerGlyph) {
			t.Fatal("the lander opens off the left wing — not on stage yet")
		}
		m = frames(m, 30) // one second: mid-streak
		streak := m.View().Content
		r0, c0, ok := markerCell(streak)
		if !ok {
			t.Fatal("one second in, the fast lander must be streaking across")
		}
		m = frames(m, 60) // two more: settled into the orbit
		orbit1 := m.View().Content
		r1, c1, ok := markerCell(orbit1)
		if !ok {
			t.Fatal("the lander must settle into the orbit")
		}
		if r0 == r1 && c0 == c1 {
			t.Fatal("frames must carry the lander from the streak onto the ring")
		}
		m = frames(m, 45) // and on it goes — the orbit never parks
		r2, c2, ok := markerCell(m.View().Content)
		if !ok {
			t.Fatal("the lander must keep orbiting until the next cut")
		}
		if r1 == r2 && c1 == c2 {
			t.Fatal("the orbit loops indefinitely — the lander must keep moving")
		}
		if skyColumns(orbit1, 12) != skyColumns(m.View().Content, 12) {
			t.Fatal("the stars behind the orbit hold still")
		}
	})
	t.Run("unhappy: the first frame of scene two never parks the lander on the ring", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, space())
		v := m.View().Content
		row, col := moon.MarkerAt(defaultW, defaultH-1, 0)
		lines := strings.Split(ansiPat.ReplaceAllString(v, ""), "\n")
		if row < 0 || row >= len(lines) {
			t.Fatalf("MarkerAt row %d is off the %d-row view", row, len(lines))
		}
		rs := []rune(lines[row])
		if col < 0 || col >= len(rs) {
			t.Fatalf("MarkerAt col %d is off the %d-col view", col, len(rs))
		}
		if rs[col] == moon.MarkerGlyph {
			t.Fatal("the stock orbit start still holds the lander — it must fly in, not appear")
		}
		if _, _, ok := markerCell(v); ok {
			t.Fatal("no lander on the first frame of scene two — appearing anywhere is the same cheat")
		}
	})
	t.Run("happy: space on the last scene ends the show — nothing left", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, space())
		mm, cmd := m.Update(space())
		if cmd == nil {
			t.Fatal("space after the last scene must end the show")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("the end of the bill must issue tea.Quit")
		}
		_ = mm
	})
	t.Run("happy: the plain-rune spacebar cuts too", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = press(m, runeKey(' '))
		if !strings.Contains(m.View().Content, "2/2") {
			t.Fatal("a rune spacebar must cut to the next scene")
		}
		_, cmd := m.Update(runeKey(' '))
		if cmd == nil {
			t.Fatal("a rune spacebar must also end the show on the last scene")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("the end of the bill must issue tea.Quit")
		}
	})
	t.Run("happy: each frame schedules the next", func(t *testing.T) {
		m := newModel(0)
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: Init schedules the first frame", func(t *testing.T) {
		if newModel(0).Init() == nil {
			t.Fatal("Init must start the clock")
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
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
	t.Run("happy: the view fills the window even when the sky runs short", func(t *testing.T) {
		m := newModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 20 {
			t.Fatalf("view has %d lines for a 20-line window", got)
		}
		mm, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 32})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 32 {
			t.Fatalf("view has %d lines for a 32-line window", got)
		}
	})
	t.Run("unhappy: q and ctrl+c close the house from any scene", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			runeKey('q'),
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
}
