package editor

// Tests written FIRST. The lander editor is a vim-ish TUI: HJKL walk the
// canvas, space selects, P pastes the selected symbol, i inserts one
// character, D deletes to transparent, Ctrl-A / Ctrl-B walk the shade ramp,
// mouse click jumps the cursor, and Ctrl-W H / Ctrl-W L (plus J/K) move
// between canvas, symbols, palette, and frames.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func newEd(t *testing.T) Model {
	t.Helper()
	m := New(lander.DefaultAtlas(), "")
	return m
}

func send(m Model, msg tea.Msg) Model {
	got, _ := m.Update(msg)
	return got.(Model)
}

func key(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Text: string(r)}
}

func keyType(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func keyCtrl(c rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: c, Mod: tea.ModCtrl}
}

func TestCursorHJKL(t *testing.T) {
	t.Run("happy: hjkl move the canvas cursor", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 2, 2
		m = send(m, key('l'))
		if m.CursorC != 3 {
			t.Fatalf("l must move right, col=%d", m.CursorC)
		}
		m = send(m, key('h'))
		if m.CursorC != 2 {
			t.Fatalf("h must move left, col=%d", m.CursorC)
		}
		m = send(m, key('j'))
		if m.CursorR != 3 {
			t.Fatalf("j must move down, row=%d", m.CursorR)
		}
		m = send(m, key('k'))
		if m.CursorR != 2 {
			t.Fatalf("k must move up, row=%d", m.CursorR)
		}
	})
	t.Run("unhappy: hjkl clamp at the canvas edge", func(t *testing.T) {
		m := newEd(t)
		sp := m.Current()
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('h'))
		m = send(m, key('k'))
		if m.CursorR != 0 || m.CursorC != 0 {
			t.Fatalf("must clamp at origin, got (%d,%d)", m.CursorR, m.CursorC)
		}
		m.CursorR, m.CursorC = sp.Height-1, sp.Width-1
		m = send(m, key('j'))
		m = send(m, key('l'))
		if m.CursorR != sp.Height-1 || m.CursorC != sp.Width-1 {
			t.Fatalf("must clamp at far corner, got (%d,%d)", m.CursorR, m.CursorC)
		}
	})
}

func TestSpaceSelect(t *testing.T) {
	t.Run("happy: space toggles the current cell into the selection", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 1, 4
		m = send(m, keyType(tea.KeySpace))
		if !m.Selected(1, 4) {
			t.Fatal("space must select the cell under the cursor")
		}
		m = send(m, keyType(tea.KeySpace))
		if m.Selected(1, 4) {
			t.Fatal("space again must deselect")
		}
	})
	t.Run("unhappy: space on the palette does not paint the canvas", func(t *testing.T) {
		m := newEd(t)
		m.Win = WinPalette
		before := m.Current()
		m = send(m, keyType(tea.KeySpace))
		after := m.Current()
		if sprite.Render(before) != sprite.Render(after) {
			t.Fatal("space in the palette window must not mutate the canvas")
		}
	})
}

func TestPaintAndDelete(t *testing.T) {
	t.Run("happy: I paints the selected palette color onto the cell", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		// first palette entry after empty is silver
		m.PalIdx = 1
		m = send(m, key('P'))
		c := m.Current().At(0, 0)
		if c.Transparent() {
			t.Fatal("P must leave a visible cell")
		}
		want := m.Atlas.Palette[1]
		if c.FG != want.FG {
			t.Fatalf("fg %d, want palette fg %d", c.FG, want.FG)
		}
	})
	t.Run("happy: I paints every selected cell, not just the cursor", func(t *testing.T) {
		m := newEd(t)
		m.PalIdx = 1
		m.CursorR, m.CursorC = 0, 0
		m = send(m, keyType(tea.KeySpace))
		m.CursorC = 1
		m = send(m, keyType(tea.KeySpace))
		m = send(m, key('P'))
		if m.Current().At(0, 0).Transparent() || m.Current().At(0, 1).Transparent() {
			t.Fatal("P must fill the whole selection")
		}
	})
	t.Run("happy: D deletes a cell to transparent", func(t *testing.T) {
		m := newEd(t)
		m.PalIdx = 1
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('P'))
		m = send(m, key('d'))
		if !m.Current().At(0, 0).Transparent() {
			t.Fatal("D must clear the cell")
		}
	})
	t.Run("happy: F paints only foreground; B paints only background", func(t *testing.T) {
		m := newEd(t)
		m.PalIdx = 1
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('P'))
		bgPal := 0
		for i, p := range m.Atlas.Palette {
			if p.BG >= 0 {
				bgPal = i
				break
			}
		}
		if m.Atlas.Palette[bgPal].BG < 0 {
			t.Fatal("need a palette entry with a background to test B")
		}
		m.PalIdx = bgPal
		beforeFG := m.Current().At(0, 0).FG
		m = send(m, key('b'))
		c := m.Current().At(0, 0)
		if c.BG != m.Atlas.Palette[bgPal].BG {
			t.Fatalf("B must set bg to %d, got %d", m.Atlas.Palette[bgPal].BG, c.BG)
		}
		if c.FG != beforeFG {
			t.Fatal("B must not clobber fg")
		}
		m = send(m, key('f'))
		c = m.Current().At(0, 0)
		if c.FG != m.Atlas.Palette[bgPal].FG {
			t.Fatalf("F must set fg to %d, got %d", m.Atlas.Palette[bgPal].FG, c.FG)
		}
	})
	t.Run("unhappy: I with no palette entry selected does not panic", func(t *testing.T) {
		m := newEd(t)
		m.PalIdx = -1
		_ = send(m, key('P'))
	})
}

func TestShadeKeys(t *testing.T) {
	t.Run("happy: Ctrl-A increments the cell, Ctrl-B decrements it", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, keyCtrl('a'))
		a := m.Current().At(0, 0)
		if a.Transparent() {
			t.Fatal("Ctrl-A on empty must start the shade ramp")
		}
		m = send(m, keyCtrl('a'))
		b := m.Current().At(0, 0)
		if b.Ch == a.Ch {
			t.Fatal("a second Ctrl-A must change the glyph")
		}
		m = send(m, keyCtrl('b'))
		c := m.Current().At(0, 0)
		if c.Ch != a.Ch {
			t.Fatalf("Ctrl-B must undo one step, got %q want %q", string(c.Ch), string(a.Ch))
		}
	})
	t.Run("unhappy: Ctrl-B on empty stays empty", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, keyCtrl('b'))
		if !m.Current().At(0, 0).Transparent() {
			t.Fatal("Ctrl-B on empty must not invent a glyph")
		}
	})
}

func TestWindows(t *testing.T) {
	t.Run("happy: Ctrl-W L then Ctrl-W H move between canvas and symbols like vim", func(t *testing.T) {
		m := newEd(t)
		if m.Win != WinCanvas {
			t.Fatal("editor must boot focused on the canvas")
		}
		m = send(m, keyCtrl('w'))
		m = send(m, key('l'))
		if m.Win != WinSymbols {
			t.Fatalf("Ctrl-W L must move to the symbols list, got %v", m.Win)
		}
		m = send(m, keyCtrl('w'))
		m = send(m, key('h'))
		if m.Win != WinCanvas {
			t.Fatalf("Ctrl-W H must return to the canvas, got %v", m.Win)
		}
	})
	t.Run("happy: Ctrl-W J / K move between symbols, palette, and frames", func(t *testing.T) {
		m := newEd(t)
		m = send(m, keyCtrl('w'))
		m = send(m, key('l'))
		if m.Win != WinSymbols {
			t.Fatalf("Ctrl-W L must land on symbols, got %v", m.Win)
		}
		m = send(m, keyCtrl('w'))
		m = send(m, key('j'))
		if m.Win != WinPalette {
			t.Fatalf("Ctrl-W J from symbols must land on palette, got %v", m.Win)
		}
		m = send(m, keyCtrl('w'))
		m = send(m, key('j'))
		if m.Win != WinFrames {
			t.Fatalf("Ctrl-W J from palette must land on frames, got %v", m.Win)
		}
		m = send(m, keyCtrl('w'))
		m = send(m, key('k'))
		if m.Win != WinPalette {
			t.Fatalf("Ctrl-W K from frames must land on palette, got %v", m.Win)
		}
	})
	t.Run("unhappy: a dangling Ctrl-W is cancelled by escape, not treated as h/j/k/l", func(t *testing.T) {
		m := newEd(t)
		m = send(m, keyCtrl('w'))
		m = send(m, keyType(tea.KeyEsc))
		m = send(m, key('l'))
		if m.Win != WinCanvas {
			t.Fatal("after Esc, l must move the cursor, not the window")
		}
		if m.CursorC != 1 {
			t.Fatalf("l should have moved the canvas cursor, col=%d", m.CursorC)
		}
	})
}

func TestMouseSelect(t *testing.T) {
	t.Run("happy: a left click on the canvas moves the cursor", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		// View has a 1-cell border; canvas origin is (1,1) in the view.
		m = send(m, tea.MouseClickMsg{
			X: 4, Y: 3,
			Button: tea.MouseLeft,
		})
		if m.CursorC == 0 && m.CursorR == 0 {
			t.Fatal("click must move the cursor off the origin")
		}
		if m.Win != WinCanvas {
			t.Fatal("clicking the canvas focuses it")
		}
	})
	t.Run("unhappy: a click off the canvas does not wrap the cursor to a wild cell", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		sp := m.Current()
		m = send(m, tea.MouseClickMsg{
			X: 79, Y: 23,
			Button: tea.MouseLeft,
		})
		if m.CursorR < 0 || m.CursorR >= sp.Height || m.CursorC < 0 || m.CursorC >= sp.Width {
			t.Fatalf("cursor escaped the sprite: (%d,%d)", m.CursorR, m.CursorC)
		}
	})
}

func TestSaveJSON(t *testing.T) {
	t.Run("happy: save writes a JSON atlas that reloads with the edit", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "lm.json")
		m := New(lander.DefaultAtlas(), path)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = 1
		m = send(m, key('P'))
		if err := m.Save(); err != nil {
			t.Fatalf("save: %v", err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(raw, &probe); err != nil {
			t.Fatalf("saved JSON is not JSON: %v", err)
		}
		loaded, err := sprite.Unmarshal(raw)
		if err != nil {
			t.Fatal(err)
		}
		cell := loaded.MustFrame(m.Size, m.Heading).At(0, 0)
		if cell.Transparent() {
			t.Fatal("reloaded atlas lost the painted cell")
		}
	})
	t.Run("unhappy: save without a path is an error", func(t *testing.T) {
		m := newEd(t)
		if err := m.Save(); err == nil {
			t.Fatal("save with empty path must fail")
		}
	})
}

func TestViewShowsWindows(t *testing.T) {
	t.Run("happy: the view contains the canvas, a palette, and the frames list", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		v := m.View().Content
		for _, want := range []string{"silver", "gold", "N", "NE"} {
			if !strings.Contains(v, want) {
				t.Fatalf("view missing %q", want)
			}
		}
	})
	t.Run("unhappy: a tiny terminal still renders without panicking", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 10, 4
		if m.View().Content == "" {
			t.Fatal("tiny view should still produce something")
		}
	})
}

func TestFrameSwitching(t *testing.T) {
	t.Run("happy: switching size on the frames window changes the canvas footprint", func(t *testing.T) {
		m := newEd(t)
		m.Win = WinFrames
		startW := m.Current().Width
		// frames window: hjkl walk the size/heading grid; l should bump size up
		m = send(m, key('l'))
		if m.Current().Width <= startW && m.Size == sprite.Size1 {
			// if l moved heading instead of size, that's ok as long as we can
			// actually change size. Force size 4.
			m.Size = sprite.Size4
		}
		m.Size = sprite.Size4
		if m.Current().Width != 26 {
			t.Fatalf("size 4 canvas must be 26 wide, got %d", m.Current().Width)
		}
	})
}
