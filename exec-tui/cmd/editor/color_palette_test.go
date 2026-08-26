package editor

// Tests written FIRST. Ctrl-K opens a dedicated color-select palette
// overlay: named atlas colors, navigable, closable. Enter/space applies
// the highlighted color to Brush (and PalIdx for a named entry). Escape
// and Ctrl-K-during-another-modal leave the brush and canvas alone.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestColorPalette(t *testing.T) {
	t.Run("happy: ctrl-k opens a color-select palette overlay", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		m = send(m, keyCtrl('k'))
		if !m.ColorPaletteOpen {
			t.Fatal("ctrl-k must open the color palette")
		}
		if m.PickerOpen {
			t.Fatal("ctrl-k must not steal the existing c 8-bit picker")
		}
		v := strings.ToLower(m.View().Content)
		for _, want := range []string{"silver", "gold"} {
			if !strings.Contains(v, want) {
				t.Fatalf("color palette view missing %q", want)
			}
		}
	})
	t.Run("happy: selecting a named color updates Brush and PalIdx, then closes", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		before := m.Current()
		if len(m.Atlas.Palette) < 3 {
			t.Fatal("need at least empty/silver/gold to pick a different color")
		}
		gold := m.Atlas.Palette[2]
		m.PalIdx = 1
		m.Brush = Swatch{FG: m.Atlas.Palette[1].FG, BG: m.Atlas.Palette[1].BG}
		m = send(m, keyCtrl('k'))
		m = send(m, key('j'))
		m = send(m, keyType(tea.KeyEnter))
		if m.ColorPaletteOpen {
			t.Fatal("selecting a color must close the palette")
		}
		if m.PalIdx != 2 {
			t.Fatalf("PalIdx %d, want 2 (gold)", m.PalIdx)
		}
		if m.Brush.FG != gold.FG || m.Brush.BG != gold.BG {
			t.Fatalf("Brush %+v, want gold fg %d bg %d", m.Brush, gold.FG, gold.BG)
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("picking a color must not stamp the canvas — only the brush")
		}
	})
	t.Run("unhappy: escape closes without changing the brush", func(t *testing.T) {
		m := newEd(t)
		m.Brush = Swatch{FG: 252, BG: -1}
		m.PalIdx = 1
		before := m.Current()
		m = send(m, keyCtrl('k'))
		m = send(m, key('j'))
		m = send(m, keyType(tea.KeyEsc))
		if m.ColorPaletteOpen {
			t.Fatal("esc must close the color palette")
		}
		if m.Brush != (Swatch{FG: 252, BG: -1}) {
			t.Fatalf("esc must keep Brush, got %+v", m.Brush)
		}
		if m.PalIdx != 1 {
			t.Fatalf("esc must keep PalIdx, got %d", m.PalIdx)
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("esc must not paint the canvas")
		}
	})
	t.Run("unhappy: ctrl-k while another modal is open does not panic or paint", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		before := m.Current()
		m = send(m, key('c'))
		if !m.PickerOpen {
			t.Fatal("need the 8-bit picker open")
		}
		m = send(m, keyCtrl('k'))
		if m.ColorPaletteOpen {
			t.Fatal("ctrl-k must not open over another modal")
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("ctrl-k in another modal must not paint the canvas")
		}
	})
	t.Run("unhappy: an invalid key in the palette does not paint the canvas", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		before := m.Current()
		m.PaintCh = '█'
		m = send(m, keyCtrl('k'))
		m = send(m, key('P'))
		m = send(m, key('d'))
		if !m.ColorPaletteOpen {
			t.Fatal("invalid keys must leave the palette open")
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("invalid input must not paint the canvas")
		}
	})
}
