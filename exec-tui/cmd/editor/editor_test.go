package editor

// Tests written FIRST. The lander editor is a vim-ish TUI: HJKL walk the
// canvas, space selects, p/P pastes the selected symbol, i inserts one
// character, D deletes the glyph on outline (color-only on fg/bg),
// Ctrl-A / Ctrl-B walk the shade ramp, mouse click jumps the cursor, and
// Ctrl-W H / Ctrl-W L (plus J/K) open control popups around a centered
// canvas (no permanent sidebar).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
)

// blankTestAtlas is a neutral atlas: the default palette and a blank
// frame for every size and heading. The editor no longer knows any
// project's art, so its tests must not lean on one either.
func blankTestAtlas() *sprite.Atlas {
	a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
	for _, sz := range sprite.Sizes {
		w, h := sz.Dim()
		for _, hd := range sprite.Headings {
			a.SetFrame(sz, hd, sprite.New(w, h))
		}
	}
	return a
}

func newEd(t *testing.T) Model {
	t.Helper()
	m := New(blankTestAtlas(), "")
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
		m = send(m, key('j'))
		if m.CursorR != 3 {
			t.Fatalf("j must move down, row=%d", m.CursorR)
		}
		m = send(m, key('k'))
		if m.CursorR != 2 {
			t.Fatalf("k must move up, row=%d", m.CursorR)
		}
		m = send(m, key('l'))
		if m.CursorC != 3 {
			t.Fatalf("l must move right, col=%d", m.CursorC)
		}
		if m.Layer != LayerOutline {
			t.Fatal("plain l must not switch layers")
		}
		m = send(m, key('h'))
		if m.CursorC != 2 {
			t.Fatalf("h must move left, col=%d", m.CursorC)
		}
		if m.Layer != LayerOutline {
			t.Fatal("plain h must not switch layers")
		}
	})
	t.Run("unhappy: hjkl clamp at the canvas edge", func(t *testing.T) {
		m := newEd(t)
		sp := m.Current()
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('k'))
		m = send(m, key('h'))
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
			t.Fatal("P on outline must leave a visible glyph")
		}
		m = send(m, keyCtrl('l')) // fg layer
		m = send(m, key('p'))
		c = m.Current().At(0, 0)
		want := m.Atlas.Palette[1]
		if c.FG != want.FG {
			t.Fatalf("fg-layer p fg %d, want palette fg %d", c.FG, want.FG)
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
	t.Run("unhappy: Ctrl-L alone does not move windows — that stays on the Ctrl-W prefix", func(t *testing.T) {
		m := newEd(t)
		m = send(m, keyCtrl('l'))
		if m.Win != WinCanvas {
			t.Fatalf("Ctrl-L must switch layers, not windows, got win %v", m.Win)
		}
		if m.Layer != LayerFG {
			t.Fatalf("Ctrl-L must land on fg, got %v", m.Layer)
		}
		m = send(m, keyCtrl('h'))
		if m.Win != WinCanvas {
			t.Fatalf("Ctrl-H must stay on the canvas, got win %v", m.Win)
		}
		if m.Layer != LayerOutline {
			t.Fatalf("Ctrl-H must walk back to outline, got %v", m.Layer)
		}
	})
	t.Run("unhappy: a dangling Ctrl-W is cancelled by escape, not treated as h/j/k/l", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		m = send(m, keyCtrl('w'))
		m = send(m, keyType(tea.KeyEsc))
		if m.Win != WinCanvas {
			t.Fatal("Ctrl-W Esc must stay on the canvas")
		}
		if sidebarChrome(m.View().Content) {
			t.Fatal("Ctrl-W Esc must not open a control popup")
		}
		m = send(m, keyArrow(tea.KeyRight))
		if m.Win != WinCanvas {
			t.Fatal("after Esc, right arrow must move the cursor, not the window")
		}
		if m.CursorC != 1 {
			t.Fatalf("right arrow should have moved the canvas cursor, col=%d", m.CursorC)
		}
	})
	t.Run("happy: Esc from a control popup returns to canvas-only", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		m = send(m, keyCtrl('w'))
		m = send(m, key('l'))
		if m.Win != WinSymbols {
			t.Fatal("need the symbols popup open")
		}
		m = send(m, keyType(tea.KeyEsc))
		if m.Win != WinCanvas {
			t.Fatalf("Esc must return focus to the canvas, got %v", m.Win)
		}
		if sidebarChrome(m.View().Content) {
			t.Fatal("Esc must dismiss the sidebar chrome, not leave it hanging")
		}
	})
}

func TestMouseSelect(t *testing.T) {
	t.Run("happy: a left click on the centered canvas moves the cursor", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		sp := m.Current()
		ox, oy := wantCanvasOrigin(m.TermW, m.TermH, sp.Width, sp.Height)
		m = send(m, tea.MouseClickMsg{
			X: ox + 3, Y: oy + 2,
			Button: tea.MouseLeft,
		})
		if m.CursorC == 0 && m.CursorR == 0 {
			t.Fatal("click must move the cursor off the origin")
		}
		if m.CursorC != 3 || m.CursorR != 2 {
			t.Fatalf("click on centered canvas landed on (%d,%d), want (2,3) row/col", m.CursorR, m.CursorC)
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
		if m.CursorR != 0 || m.CursorC != 0 {
			t.Fatalf("off-canvas click must leave the cursor put, got (%d,%d)", m.CursorR, m.CursorC)
		}
	})
}

func TestSaveJSON(t *testing.T) {
	t.Run("happy: save writes a JSON atlas that reloads with the edit", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "craft.json")
		m := New(blankTestAtlas(), path)
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

func TestSavePersistsCurrentCanvas(t *testing.T) {
	t.Run("happy: unique painted cell survives Save and LoadFile", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "craft.json")
		m := New(blankTestAtlas(), path)
		m.Size = sprite.Size4
		m.Heading = sprite.N
		sp := cloneSprite(m.Current())
		want := sprite.Cell{Ch: 'Ω', FG: 123, BG: 45}
		sp.Set(1, 2, want)
		m.setCurrent(sp)
		if m.Path != path {
			t.Fatalf("title-bar path %q, want %q", m.Path, path)
		}
		if err := m.Save(); err != nil {
			t.Fatalf("save: %v", err)
		}
		loaded, err := sprite.LoadFile(m.Path)
		if err != nil {
			t.Fatalf("LoadFile(%s): %v", m.Path, err)
		}
		got := loaded.MustFrame(m.Size, m.Heading).At(1, 2)
		if got.Ch != want.Ch || got.FG != want.FG || got.BG != want.BG {
			t.Fatalf("disk cell %+v, want %+v", got, want)
		}
	})
	t.Run("unhappy: Save does not write an empty atlas over the title-bar file", func(t *testing.T) {
		dir := t.TempDir()
		path := writeMiniShip(t, dir, "quiet", 'Q')
		m, err := Open(path)
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		before, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		m.Atlas = nil
		if err := m.Save(); err == nil {
			t.Fatal("save with no atlas must fail")
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(before) {
			t.Fatal("failed save must not clobber the file")
		}
	})
}

func TestSaveKeyToast(t *testing.T) {
	t.Run("happy: s writes the atlas and a 3-height toast that clears", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "craft.json")
		m := New(blankTestAtlas(), path)
		m.TermW, m.TermH = 80, 24

		got, cmd := m.Update(key('s'))
		m = got.(Model)
		if cmd == nil {
			t.Fatal("s must start a 5s dismiss tick")
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("s must save: %v", err)
		}
		banner, err := termfont.Lines(3, "SAVED")
		if err != nil {
			t.Fatalf("termfont: %v", err)
		}
		v := m.View().Content
		if strings.Contains(v, "wrote ") {
			t.Fatal("must not keep the old wrote-path status")
		}
		for _, line := range banner {
			if !strings.Contains(v, strings.TrimSpace(line)) {
				t.Fatalf("view missing 3-height toast line %q\n%s", line, v)
			}
		}
		art := strings.Index(v, "canvas")
		toast := strings.Index(v, strings.TrimSpace(banner[0]))
		if toast < 0 || art < 0 || toast > art {
			t.Fatal("toast must sit above the art")
		}

		got, cmd = m.Update(keyCtrl('s'))
		m = got.(Model)
		if cmd == nil {
			t.Fatal("ctrl-s must keep saving with a dismiss tick")
		}

		m = send(m, saveToastClearMsg{id: m.toastID})
		if m.toast != "" {
			t.Fatal("toast must vanish after the tick")
		}
		v = m.View().Content
		if strings.Contains(v, strings.TrimSpace(banner[0])) {
			t.Fatal("dismissed toast must leave the view")
		}
	})
	t.Run("unhappy: failed save still toasts then clears; insert still types s", func(t *testing.T) {
		m := newEd(t)
		got, cmd := m.Update(key('s'))
		m = got.(Model)
		if cmd == nil {
			t.Fatal("failed save must still schedule dismiss")
		}
		if m.toast != "ERR" {
			t.Fatalf("failed save must toast ERR, got %q", m.toast)
		}
		if m.err != "" {
			t.Fatal("error must not stick in the status line")
		}
		m = send(m, saveToastClearMsg{id: m.toastID})
		if m.toast != "" {
			t.Fatal("error toast must clear")
		}

		dir := t.TempDir()
		path := filepath.Join(dir, "craft.json")
		m = New(blankTestAtlas(), path)
		m.Inserting = true
		m = send(m, key('s'))
		if _, err := os.Stat(path); err == nil {
			t.Fatal("insert mode must type s, not save")
		}
		cell := m.Current().At(m.CursorR, m.CursorC)
		if cell.Ch != 's' {
			t.Fatalf("insert s must land on the cell, got %q", string(cell.Ch))
		}
	})
}

func TestViewShowsWindows(t *testing.T) {
	t.Run("happy: the default view is centered art with no sidebar chrome", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		v := m.View().Content
		if !strings.Contains(v, "canvas") {
			t.Fatal("default view must still show the canvas")
		}
		if sidebarChrome(v) {
			t.Fatal("default view must not keep symbols/palette/frames on the side")
		}
		if !canvasLineIndented(v) {
			t.Fatal("the art box must be centered, not jammed against the left edge")
		}
		sp := m.Current()
		ox, _ := wantCanvasOrigin(m.TermW, m.TermH, sp.Width, sp.Height)
		if ox <= 1 {
			t.Fatalf("centered canvas origin x=%d, want > 1 on an 80-col terminal", ox)
		}
	})
	t.Run("happy: Ctrl-W L opens the symbols popup over the centered art", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		m = send(m, keyCtrl('w'))
		m = send(m, key('l'))
		v := strings.ToLower(m.View().Content)
		if !strings.Contains(v, "symbols") {
			t.Fatal("Ctrl-W L must show the symbols popup")
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

func sidebarChrome(view string) bool {
	v := strings.ToLower(view)
	for _, mark := range []string{"symbols", "silver", "gold", " frames "} {
		if strings.Contains(v, mark) {
			return true
		}
	}
	return false
}

func canvasLineIndented(view string) bool {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "canvas") && strings.Contains(line, "╭") {
			return strings.HasPrefix(line, " ")
		}
	}
	return false
}

// wantCanvasOrigin is the inner (x,y) of a box centered in the terminal
// under a 1-line title and 2-line footer (meta + status).
func wantCanvasOrigin(termW, termH, spriteW, spriteH int) (x, y int) {
	boxW, boxH := spriteW+2, spriteH+2
	padX := (termW - boxW) / 2
	if padX < 0 {
		padX = 0
	}
	avail := termH - 3
	if avail < 1 {
		avail = 1
	}
	padY := (avail - boxH) / 2
	if padY < 0 {
		padY = 0
	}
	return padX + 1, 1 + padY + 1
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
	t.Run("happy: frames h/l walk every heading the atlas ships", func(t *testing.T) {
		m := newEd(t)
		m.Win = WinFrames
		m.Size = sprite.Size4
		m.Heading = sprite.N
		want := sprite.Headings
		seen := map[sprite.Heading]bool{m.Heading: true}
		for i := 0; i < len(want); i++ {
			m = send(m, key('l'))
			seen[m.Heading] = true
		}
		if m.Heading != sprite.N {
			t.Fatalf("%d steps of l must return to N, got %s", len(want), m.Heading)
		}
		if len(seen) != len(want) {
			t.Fatalf("l must visit all %d size-4 headings, got %d", len(want), len(seen))
		}
	})
	t.Run("happy: [ and ] cycle headings on the canvas", func(t *testing.T) {
		m := newEd(t)
		m.Win = WinCanvas
		m.Size = sprite.Size1
		m.Heading = sprite.N
		m = send(m, key(']'))
		if m.Heading != sprite.NE {
			t.Fatalf("] must advance N → NE, got %s", m.Heading)
		}
		m = send(m, key('['))
		if m.Heading != sprite.N {
			t.Fatalf("[ must step back to N, got %s", m.Heading)
		}
		seen := map[sprite.Heading]bool{m.Heading: true}
		for i := 0; i < len(sprite.Headings); i++ {
			m = send(m, key(']'))
			seen[m.Heading] = true
		}
		if len(seen) != len(sprite.Headings) {
			t.Fatalf("] must visit all %d headings, got %d", len(sprite.Headings), len(seen))
		}
	})
	t.Run("unhappy: heading keys do not stamp the canvas", func(t *testing.T) {
		m := newEd(t)
		m.Win = WinCanvas
		before := sprite.Render(m.Current())
		m = send(m, key(']'))
		m = send(m, key('['))
		if sprite.Render(m.Atlas.MustFrame(m.Size, sprite.N)) != before {
			t.Fatal("[ ] must change heading only, not paint N")
		}
	})
	t.Run("unhappy: heading keys skip a missing frame instead of panicking", func(t *testing.T) {
		m := newEd(t)
		m.Win = WinCanvas
		m.Heading = sprite.N
		n := m.Current()
		a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), m.Atlas.Palette...)}
		a.SetFrame(sprite.Size4, sprite.N, n)
		a.SetFrame(sprite.Size4, sprite.E, n)
		m.Atlas = a
		m = send(m, key(']'))
		if m.Heading != sprite.E {
			t.Fatalf("] must skip the missing NE and land on E, got %s", m.Heading)
		}
	})
}

func TestWidePasteDoesNotShredFooter(t *testing.T) {
	const wide = '◽' // two terminal cells; pasting this used to wrap the footer
	t.Run("happy: hovering a wide pasted glyph keeps recent/cell/cut labels intact", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		sp := cloneSprite(m.Current())
		sp.Set(9, 8, sprite.Cell{Ch: wide, FG: 208, BG: 52})
		m.setCurrent(sp)
		m.CursorR, m.CursorC = 9, 8
		m.RecentGlyphs = []rune{wide}
		m.status = "cut " + string(wide) + " fg 208 bg 52"
		plain := strip(m.View().Content)
		if !strings.Contains(plain, "recent glyphs") {
			t.Fatalf("footer lost 'recent glyphs': %q", plain)
		}
		if !strings.Contains(plain, "cell (9,8)") {
			t.Fatalf("footer lost 'cell (9,8)': %q", plain)
		}
		if !strings.Contains(plain, "cut ") {
			t.Fatalf("footer lost cut status: %q", plain)
		}
		for i, line := range strings.Split(m.View().Content, "\n") {
			if n := visibleLen(line); n > m.TermW {
				t.Fatalf("line %d is %d cells, wider than term %d: %q", i, n, m.TermW, strip(line))
			}
		}
	})
	t.Run("unhappy: a row of wide runes still fits the 26-col canvas box", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		sp := cloneSprite(m.Current())
		for c := 0; c < sp.Width; c++ {
			sp.Set(0, c, sprite.Cell{Ch: wide, FG: 208, BG: 52})
		}
		m.setCurrent(sp)
		raw := renderCanvas(m.Current(), m)
		for i, line := range strings.Split(raw, "\n") {
			if n := visibleLen(line); n != sp.Width+2 {
				t.Fatalf("box line %d is %d cells, want %d: %q", i, n, sp.Width+2, strip(line))
			}
		}
	})
}
