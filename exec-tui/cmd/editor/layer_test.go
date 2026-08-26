package editor

// Tests written FIRST. The canvas is three layers cycled with Ctrl-H /
// Ctrl-L: ascii outline (white glyphs), foreground, background (fill
// clearly visible). hjkl walk the canvas cursor. Ctrl-E is the glyph
// selector. Ctrl-P thumbs are the full composite. Plain h/l never
// change layers. Labels under the art box name the three layers and
// highlight the current one. d and x on fg/bg strip that color only; the
// glyph stays, and uncolored ASCII reads as magenta.

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func keyArrow(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code}
}

func goldCell() sprite.Cell {
	return sprite.Cell{Ch: '█', FG: 178, BG: 94}
}

func stampGold(t *testing.T, m Model) Model {
	t.Helper()
	sp := cloneSprite(m.Current())
	sp.Set(0, 0, goldCell())
	m.setCurrent(sp)
	m.CursorR, m.CursorC = 0, 0
	return m
}

func TestLayersCycleHL(t *testing.T) {
	t.Run("happy: a new editor starts on the outline layer", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		if m.Layer != LayerOutline {
			t.Fatalf("boot layer %v, want outline", m.Layer)
		}
		v := strings.ToLower(m.View().Content)
		if !strings.Contains(v, "outline") && !strings.Contains(v, "ascii") {
			t.Fatal("title/status/labels must name the outline/ascii layer")
		}
	})
	t.Run("happy: ctrl-l walks outline → fg → bg → outline", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		m = send(m, keyCtrl('l'))
		if m.Layer != LayerFG {
			t.Fatalf("one ctrl-l must land on fg, got %v", m.Layer)
		}
		if !strings.Contains(strings.ToLower(m.View().Content), "fg") &&
			!strings.Contains(strings.ToLower(m.View().Content), "foreground") {
			t.Fatal("fg layer must be named in the view")
		}
		m = send(m, keyCtrl('l'))
		if m.Layer != LayerBG {
			t.Fatalf("two ctrl-l must land on bg, got %v", m.Layer)
		}
		if !strings.Contains(strings.ToLower(m.View().Content), "bg") &&
			!strings.Contains(strings.ToLower(m.View().Content), "background") {
			t.Fatal("bg layer must be named in the view")
		}
		m = send(m, keyCtrl('l'))
		if m.Layer != LayerOutline {
			t.Fatalf("three ctrl-l must wrap to outline, got %v", m.Layer)
		}
	})
	t.Run("happy: ctrl-h walks the other way", func(t *testing.T) {
		m := newEd(t)
		m = send(m, keyCtrl('h'))
		if m.Layer != LayerBG {
			t.Fatalf("ctrl-h from outline must wrap to bg, got %v", m.Layer)
		}
		m = send(m, keyCtrl('h'))
		if m.Layer != LayerFG {
			t.Fatalf("ctrl-h from bg must land on fg, got %v", m.Layer)
		}
	})
	t.Run("unhappy: ctrl-h/l do not move the canvas cursor", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 2, 4
		beforeR, beforeC := m.CursorR, m.CursorC
		before := sprite.Render(m.Current())
		m = send(m, keyCtrl('l'))
		m = send(m, keyCtrl('h'))
		if m.CursorR != beforeR || m.CursorC != beforeC {
			t.Fatalf("ctrl-h/l must not walk the cursor, got (%d,%d)", m.CursorR, m.CursorC)
		}
		if sprite.Render(m.Current()) != before {
			t.Fatal("switching layers must not paint the atlas")
		}
	})
	t.Run("unhappy: plain h/l do not change the canvas layer", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 2, 2
		m = send(m, key('l'))
		m = send(m, key('h'))
		if m.Layer != LayerOutline {
			t.Fatalf("plain h/l must leave the outline layer, got %v", m.Layer)
		}
	})
}

func TestLayerCursorArrows(t *testing.T) {
	t.Run("happy: arrows and jk still walk the canvas", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 2, 2
		m = send(m, keyArrow(tea.KeyRight))
		if m.CursorC != 3 {
			t.Fatalf("right arrow must move right, col=%d", m.CursorC)
		}
		m = send(m, keyArrow(tea.KeyLeft))
		if m.CursorC != 2 {
			t.Fatalf("left arrow must move left, col=%d", m.CursorC)
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
	t.Run("unhappy: arrows clamp at the canvas edge", func(t *testing.T) {
		m := newEd(t)
		sp := m.Current()
		m.CursorR, m.CursorC = 0, 0
		m = send(m, keyArrow(tea.KeyLeft))
		m = send(m, keyArrow(tea.KeyUp))
		if m.CursorR != 0 || m.CursorC != 0 {
			t.Fatalf("must clamp at origin, got (%d,%d)", m.CursorR, m.CursorC)
		}
		m.CursorR, m.CursorC = sp.Height-1, sp.Width-1
		m = send(m, keyArrow(tea.KeyDown))
		m = send(m, keyArrow(tea.KeyRight))
		if m.CursorR != sp.Height-1 || m.CursorC != sp.Width-1 {
			t.Fatalf("must clamp at far corner, got (%d,%d)", m.CursorR, m.CursorC)
		}
	})
}

func TestLayerViews(t *testing.T) {
	t.Run("happy: outline renders glyphs in white without the cell background", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		m.TermW, m.TermH = 80, 24
		m.Layer = LayerOutline
		v := m.View().Content
		if !strings.Contains(v, "\x1b[38;5;231m") {
			t.Fatal("outline layer must paint glyphs white (xterm 231)")
		}
		if strings.Contains(v, "\x1b[38;5;226m") {
			t.Fatal("outline layer must not paint glyphs yellow")
		}
		if strings.Contains(v, "\x1b[48;5;94m") {
			t.Fatal("outline layer must not show the gold background fill")
		}
	})
	t.Run("happy: fg layer shows foreground color without the background fill", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		m.TermW, m.TermH = 80, 24
		m.Layer = LayerFG
		v := m.View().Content
		if !strings.Contains(v, "\x1b[38;5;178m") {
			t.Fatal("fg layer must show the cell foreground")
		}
		if strings.Contains(v, "\x1b[48;5;94m") {
			t.Fatal("fg layer must hide background so fg is readable")
		}
	})
	t.Run("happy: bg layer makes the background fill visible", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		m.TermW, m.TermH = 80, 24
		m.Layer = LayerBG
		v := m.View().Content
		if !strings.Contains(v, "\x1b[48;5;94m") {
			t.Fatal("bg layer must show the gold background (xterm 94)")
		}
	})
	t.Run("unhappy: an empty cell on bg does not invent a fill", func(t *testing.T) {
		a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
		a.SetFrame(sprite.Size4, sprite.N, sprite.New(26, 10))
		m := New(a, "")
		m.TermW, m.TermH = 80, 24
		m.Layer = LayerBG
		v := m.View().Content
		if strings.Contains(v, "\x1b[48;5;94m") {
			t.Fatal("bg layer must not paint gold onto a transparent cell")
		}
	})
}

func TestLayerColorPicks(t *testing.T) {
	t.Run("happy: ctrl-k on fg applies the named color to Brush.FG", func(t *testing.T) {
		m := newEd(t)
		m = send(m, keyCtrl('l')) // fg
		gold := m.Atlas.Palette[2]
		m.Brush = Swatch{FG: 252, BG: -1}
		m.PalIdx = 1
		m = send(m, keyCtrl('k'))
		m = send(m, key('j'))
		m = send(m, keyType(tea.KeyEnter))
		if m.Brush.FG != gold.FG {
			t.Fatalf("fg pick Brush.FG %d, want gold %d", m.Brush.FG, gold.FG)
		}
		if m.Brush.BG != -1 {
			t.Fatalf("fg pick must not clobber Brush.BG, got %d", m.Brush.BG)
		}
	})
	t.Run("happy: ctrl-k on bg applies the named color to Brush.BG", func(t *testing.T) {
		m := newEd(t)
		m = send(m, keyCtrl('l'))
		m = send(m, keyCtrl('l')) // bg
		gold := m.Atlas.Palette[2]
		m.Brush = Swatch{FG: 252, BG: -1}
		m = send(m, keyCtrl('k'))
		m = send(m, key('j'))
		m = send(m, keyType(tea.KeyEnter))
		if m.Brush.BG != gold.BG {
			t.Fatalf("bg pick Brush.BG %d, want gold %d", m.Brush.BG, gold.BG)
		}
	})
	t.Run("happy: p on fg paints only foreground; p on bg paints only background", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		m.PalIdx = 2              // gold
		m = send(m, keyCtrl('l')) // fg
		before := m.Current().At(0, 0)
		m = send(m, key('p'))
		got := m.Current().At(0, 0)
		if got.FG != m.Atlas.Palette[2].FG {
			t.Fatalf("fg-layer p fg %d, want %d", got.FG, m.Atlas.Palette[2].FG)
		}
		if got.BG != before.BG {
			t.Fatal("fg-layer p must not clobber bg")
		}
		if got.Ch != before.Ch {
			t.Fatal("fg-layer p must keep the glyph")
		}
		m = send(m, keyCtrl('l')) // bg
		m.PalIdx = 2
		m = send(m, key('p'))
		got = m.Current().At(0, 0)
		if got.BG != m.Atlas.Palette[2].BG {
			t.Fatalf("bg-layer p bg %d, want %d", got.BG, m.Atlas.Palette[2].BG)
		}
		if got.Ch != before.Ch {
			t.Fatal("bg-layer p must keep the glyph")
		}
	})
	t.Run("unhappy: a color pick does not stamp the canvas", func(t *testing.T) {
		m := newEd(t)
		m = send(m, keyCtrl('l'))
		before := sprite.Render(m.Current())
		m = send(m, keyCtrl('k'))
		m = send(m, keyType(tea.KeyEnter))
		if sprite.Render(m.Current()) != before {
			t.Fatal("picking a color must only change the brush")
		}
	})
	t.Run("unhappy: 8-bit picker on bg sets Brush.BG, not FG", func(t *testing.T) {
		m := newEd(t)
		m = send(m, keyCtrl('l'))
		m = send(m, keyCtrl('l')) // bg
		m.Brush = Swatch{FG: 252, BG: -1}
		m = send(m, key('c'))
		m.PickerIdx = len(Greys) - 1
		m = send(m, keyType(tea.KeySpace))
		if m.Brush.BG != 255 {
			t.Fatalf("bg-layer 8-bit pick Brush.BG %d, want 255", m.Brush.BG)
		}
		if m.Brush.FG != 252 {
			t.Fatalf("bg-layer 8-bit pick must keep FG, got %d", m.Brush.FG)
		}
	})
}

func TestCtrlEStillGlyphSelector(t *testing.T) {
	t.Run("happy: ctrl-e still opens the glyph grid on every layer", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 120, 40
		for i, name := range []string{"outline", "fg", "bg"} {
			if i > 0 {
				m = send(m, keyCtrl('l'))
			}
			m = send(m, keyCtrl('e'))
			if !m.GlyphGridOpen {
				t.Fatalf("ctrl-e must open the glyph grid on %s", name)
			}
			v := m.View().Content
			if !strings.Contains(v, "glyphs") {
				t.Fatalf("%s: ctrl-e view must still be the glyph selector", name)
			}
			m = send(m, keyType(tea.KeyEsc))
		}
	})
	t.Run("unhappy: glyph-grid h/l do not change the canvas layer", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 120, 40
		m = send(m, keyCtrl('e'))
		m = send(m, key('l'))
		if m.Layer != LayerOutline {
			t.Fatalf("glyph-grid l must not switch layers, got %v", m.Layer)
		}
		if !m.GlyphGridOpen {
			t.Fatal("glyph-grid must stay open")
		}
	})
}

func TestCtrlPComposite(t *testing.T) {
	t.Run("happy: the ctrl-p preview renders outline+fg+bg together", func(t *testing.T) {
		dir := t.TempDir()
		writeMiniShip(t, dir, "gilded", 'G')
		m, err := Open(dir)
		if err != nil {
			t.Fatalf("Open(dir): %v", err)
		}
		m = stampGold(t, m)
		m.TermW, m.TermH = 160, 50
		m = send(m, keyCtrl('p'))
		if !m.FilePickerOpen {
			t.Fatal("ctrl-p must open the file picker")
		}
		v := m.View().Content
		if !strings.Contains(v, "\x1b[38;5;178m") {
			t.Fatal("picker preview must include the gold foreground")
		}
		if !strings.Contains(v, "\x1b[48;5;94m") {
			t.Fatal("picker preview must include the gold background, not a single layer")
		}
	})
	t.Run("unhappy: the preview still composites when the editor is on outline", func(t *testing.T) {
		dir := t.TempDir()
		writeMiniShip(t, dir, "gilded", 'G')
		m, err := Open(dir)
		if err != nil {
			t.Fatalf("Open(dir): %v", err)
		}
		m = stampGold(t, m)
		m.TermW, m.TermH = 160, 50
		m.Layer = LayerOutline // the preview must ignore the editor layer
		m = send(m, keyCtrl('p'))
		if !strings.Contains(m.View().Content, "\x1b[48;5;94m") {
			t.Fatal("ctrl-p must keep the real ship fill even on the outline layer")
		}
	})
}

func TestLayerTabsUnderCanvas(t *testing.T) {
	t.Run("happy: ascii · foreground · background sit under the art box", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		v := m.View().Content
		plain := strip(v)
		for _, name := range []string{"ascii", "foreground", "background"} {
			if !strings.Contains(plain, name) {
				t.Fatalf("view missing layer label %q", name)
			}
		}
		if !strings.Contains(plain, "ascii") || !strings.Contains(plain, "·") {
			t.Fatal("labels must be joined with a middle dot")
		}
		foundBox := false
		foundTabs := false
		for _, line := range strings.Split(v, "\n") {
			if strings.Contains(line, "╰") {
				foundBox = true
				continue
			}
			if foundBox {
				s := strip(line)
				if strings.Contains(s, "ascii") && strings.Contains(s, "foreground") && strings.Contains(s, "background") {
					foundTabs = true
					break
				}
			}
		}
		if !foundTabs {
			t.Fatal("layer labels must appear on a line under the art box, not only in the title")
		}
	})
	t.Run("happy: the current layer is reverse-video highlighted", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		v := m.View().Content
		if !strings.Contains(v, "\x1b[7mascii\x1b[0m") {
			t.Fatal("outline layer must highlight the ascii label")
		}
		m = send(m, keyCtrl('l'))
		v = m.View().Content
		if !strings.Contains(v, "\x1b[7mforeground\x1b[0m") {
			t.Fatal("fg layer must highlight the foreground label")
		}
		if strings.Contains(v, "\x1b[7mascii\x1b[0m") {
			t.Fatal("fg layer must not keep the ascii highlight")
		}
		m = send(m, keyCtrl('l'))
		v = m.View().Content
		if !strings.Contains(v, "\x1b[7mbackground\x1b[0m") {
			t.Fatal("bg layer must highlight the background label")
		}
	})
	t.Run("unhappy: every layer still shows all three labels", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		for i := 0; i < 3; i++ {
			plain := strip(m.View().Content)
			for _, name := range []string{"ascii", "foreground", "background"} {
				if !strings.Contains(plain, name) {
					t.Fatalf("layer %v missing label %q", m.Layer, name)
				}
			}
			m = send(m, keyCtrl('l'))
		}
	})
}

func TestLayerDeleteKeepsGlyph(t *testing.T) {
	t.Run("happy: d on background removes the fill and keeps the glyph and fg", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		m.Layer = LayerBG
		before := m.Current().At(0, 0)
		m = send(m, key('d'))
		got := m.Current().At(0, 0)
		if got.Ch != before.Ch {
			t.Fatalf("bg delete must keep the glyph, got %q want %q", string(got.Ch), string(before.Ch))
		}
		if got.FG != before.FG {
			t.Fatalf("bg delete must keep fg %d, got %d", before.FG, got.FG)
		}
		if got.BG != -1 {
			t.Fatalf("bg delete must clear bg, got %d", got.BG)
		}
	})
	t.Run("happy: d on foreground removes the color and keeps the glyph and bg", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		m.Layer = LayerFG
		before := m.Current().At(0, 0)
		m = send(m, key('d'))
		got := m.Current().At(0, 0)
		if got.Ch != before.Ch {
			t.Fatalf("fg delete must keep the glyph, got %q want %q", string(got.Ch), string(before.Ch))
		}
		if got.BG != before.BG {
			t.Fatalf("fg delete must keep bg %d, got %d", before.BG, got.BG)
		}
		if got.FG != -1 {
			t.Fatalf("fg delete must clear fg, got %d", got.FG)
		}
	})
	t.Run("happy: leftover uncolored ASCII renders magenta on fg and bg layers", func(t *testing.T) {
		c := sprite.Cell{Ch: '█', FG: -1, BG: -1}
		fg := layerCell(LayerFG, c)
		if fg.Ch != '█' || fg.FG != asciiMagenta || fg.BG != -1 {
			t.Fatalf("fg layer uncolored glyph %+v, want ch █ fg %d bg -1", fg, asciiMagenta)
		}
		bg := layerCell(LayerBG, c)
		if bg.Ch != '█' || bg.FG != asciiMagenta || bg.BG != -1 {
			t.Fatalf("bg layer uncolored glyph %+v, want ch █ fg %d bg -1", bg, asciiMagenta)
		}

		blank := func() Model {
			a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
			a.SetFrame(sprite.Size4, sprite.N, sprite.New(26, 10))
			return stampGold(t, New(a, ""))
		}

		m := blank()
		m.TermW, m.TermH = 80, 24
		m.Layer = LayerBG
		m = send(m, key('d'))
		m.CursorC = 1 // off the leftover glyph so reverse-video does not hide the color
		v := m.View().Content
		if !strings.Contains(v, fmt.Sprintf("\x1b[38;5;%dm", asciiMagenta)) {
			t.Fatal("bg layer must show leftover ASCII in magenta after deleting the fill")
		}
		if strings.Contains(v, "\x1b[48;5;94m") {
			t.Fatal("bg layer must not keep the gold fill after d")
		}

		m = blank()
		m.TermW, m.TermH = 80, 24
		m.Layer = LayerFG
		m = send(m, key('d'))
		m.CursorC = 1
		v = m.View().Content
		if !strings.Contains(v, fmt.Sprintf("\x1b[38;5;%dm", asciiMagenta)) {
			t.Fatal("fg layer must show leftover ASCII in magenta after deleting the color")
		}
		if strings.Contains(v, "\x1b[38;5;178m") {
			t.Fatal("fg layer must not keep the gold foreground after d")
		}
	})
	t.Run("happy: d on a fg/bg selection strips that color from every selected cell", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		sp := cloneSprite(m.Current())
		sp.Set(0, 1, goldCell())
		m.setCurrent(sp)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, keyType(tea.KeySpace))
		m.CursorC = 1
		m = send(m, keyType(tea.KeySpace))
		m.Layer = LayerBG
		m = send(m, key('d'))
		for _, col := range []int{0, 1} {
			got := m.Current().At(0, col)
			if got.Ch != '█' {
				t.Fatalf("selected col %d must keep the glyph, got %q", col, string(got.Ch))
			}
			if got.FG != 178 {
				t.Fatalf("selected col %d must keep fg, got %d", col, got.FG)
			}
			if got.BG != -1 {
				t.Fatalf("selected col %d must clear bg, got %d", col, got.BG)
			}
		}
	})
	t.Run("unhappy: d on an empty fg/bg cell does not invent a glyph", func(t *testing.T) {
		a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
		a.SetFrame(sprite.Size4, sprite.N, sprite.New(26, 10))
		m := New(a, "")
		m.CursorR, m.CursorC = 0, 0
		for _, layer := range []EditLayer{LayerFG, LayerBG} {
			m.Layer = layer
			m = send(m, key('d'))
			got := m.Current().At(0, 0)
			if !got.Transparent() {
				t.Fatalf("d on empty %s must not invent a glyph, got %+v", layer, got)
			}
			if got.Ch != ' ' && got.Ch != 0 {
				t.Fatalf("d on empty %s must stay blank, got %q", layer, string(got.Ch))
			}
		}
	})
	t.Run("unhappy: d on outline still clears the whole cell", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		m.Layer = LayerOutline
		m = send(m, key('d'))
		if !m.Current().At(0, 0).Transparent() {
			t.Fatal("outline d must still delete the ASCII")
		}
	})
	t.Run("happy: x on background removes the fill, keeps the glyph, and picks up bg", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		m.Layer = LayerBG
		before := m.Current().At(0, 0)
		m.PaintCh = '░'
		m.Brush = Swatch{FG: 252, BG: -1}
		m.PalIdx = -1
		m = send(m, key('x'))
		got := m.Current().At(0, 0)
		if got.Ch != before.Ch {
			t.Fatalf("bg cut must keep the glyph, got %q want %q", string(got.Ch), string(before.Ch))
		}
		if got.FG != before.FG {
			t.Fatalf("bg cut must keep fg %d, got %d", before.FG, got.FG)
		}
		if got.BG != -1 {
			t.Fatalf("bg cut must clear bg, got %d", got.BG)
		}
		if m.PaintCh != '░' {
			t.Fatalf("bg cut must not steal the glyph brush, got %q", string(m.PaintCh))
		}
		if m.Brush.BG != before.BG {
			t.Fatalf("bg cut must pick up bg %d, got %d", before.BG, m.Brush.BG)
		}
		if m.Brush.FG != 252 {
			t.Fatalf("bg cut must not clobber Brush.FG, got %d", m.Brush.FG)
		}
	})
	t.Run("happy: x on foreground removes the color, keeps the glyph, and picks up fg", func(t *testing.T) {
		m := stampGold(t, newEd(t))
		m.Layer = LayerFG
		before := m.Current().At(0, 0)
		m.PaintCh = '░'
		m.Brush = Swatch{FG: 252, BG: -1}
		m.PalIdx = -1
		m = send(m, key('x'))
		got := m.Current().At(0, 0)
		if got.Ch != before.Ch {
			t.Fatalf("fg cut must keep the glyph, got %q want %q", string(got.Ch), string(before.Ch))
		}
		if got.BG != before.BG {
			t.Fatalf("fg cut must keep bg %d, got %d", before.BG, got.BG)
		}
		if got.FG != -1 {
			t.Fatalf("fg cut must clear fg, got %d", got.FG)
		}
		if m.PaintCh != '░' {
			t.Fatalf("fg cut must not steal the glyph brush, got %q", string(m.PaintCh))
		}
		if m.Brush.FG != before.FG {
			t.Fatalf("fg cut must pick up fg %d, got %d", before.FG, m.Brush.FG)
		}
		if m.Brush.BG != -1 {
			t.Fatalf("fg cut must not clobber Brush.BG, got %d", m.Brush.BG)
		}
	})
	t.Run("unhappy: x on an empty fg/bg cell does not invent a glyph or change the brush", func(t *testing.T) {
		a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
		a.SetFrame(sprite.Size4, sprite.N, sprite.New(26, 10))
		m := New(a, "")
		m.CursorR, m.CursorC = 0, 0
		m.PaintCh = '█'
		m.Brush = Swatch{FG: 252, BG: -1}
		m.PalIdx = -1
		for _, layer := range []EditLayer{LayerFG, LayerBG} {
			m.Layer = layer
			m = send(m, key('x'))
			got := m.Current().At(0, 0)
			if !got.Transparent() {
				t.Fatalf("x on empty %s must not invent a glyph, got %+v", layer, got)
			}
			if m.PaintCh != '█' {
				t.Fatalf("x on empty %s must keep PaintCh, got %q", layer, string(m.PaintCh))
			}
			if m.Brush != (Swatch{FG: 252, BG: -1}) {
				t.Fatalf("x on empty %s must keep the brush, got %+v", layer, m.Brush)
			}
		}
	})
}
