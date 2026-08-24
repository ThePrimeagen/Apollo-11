package editor

// Tests written FIRST. The paint kit: 1-0 clutch the last ten colors,
// !@#$%^&*() clutch the ten paint glyphs (shades, halves, quadrants),
// c opens an 8-bit picker with a greyscale ramp that has enough whites
// to shade, P pastes the selected symbol, and i inserts one typed character.

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestGlyphKeysPaintPartialBlocks(t *testing.T) {
	t.Run("happy: ! @ # $ select ░ ▒ ▓ █ and i stamps that glyph", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		cases := []struct {
			key rune
			ch  rune
		}{{'!', '░'}, {'@', '▒'}, {'#', '▓'}, {'$', '█'}}
		for _, tc := range cases {
			m = send(m, key(tc.key))
			if m.PaintCh != tc.ch {
				t.Fatalf("key %q: PaintCh %q, want %q", string(tc.key), string(m.PaintCh), string(tc.ch))
			}
			m = send(m, key('P'))
			got := m.Current().At(0, 0).Ch
			if got != tc.ch {
				t.Fatalf("P after %q pasted %q, want %q", string(tc.key), string(got), string(tc.ch))
			}
		}
	})
	t.Run("happy: % ^ & * ( ) are the half-blocks and quadrants", func(t *testing.T) {
		m := newEd(t)
		want := []rune{'▀', '▄', '▌', '▐', '▖', '▗'}
		keys := []rune{'%', '^', '&', '*', '(', ')'}
		for i, k := range keys {
			m = send(m, key(k))
			if m.PaintCh != want[i] {
				t.Fatalf("key %q: PaintCh %q, want %q", string(k), string(m.PaintCh), string(want[i]))
			}
		}
	})
	t.Run("unhappy: a symbol that is not a paint key is ignored", func(t *testing.T) {
		m := newEd(t)
		before := m.PaintCh
		m = send(m, key('~'))
		if m.PaintCh != before {
			t.Fatalf("~ must not change the paint glyph, got %q", string(m.PaintCh))
		}
	})
}

func TestColorClutch(t *testing.T) {
	t.Run("happy: 1-0 select the past-ten colors and i uses that fg", func(t *testing.T) {
		m := newEd(t)
		if len(m.RecentColors) < 10 {
			t.Fatalf("clutch must boot with 10 colors, got %d", len(m.RecentColors))
		}
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('3'))
		want := m.RecentColors[2]
		if m.Brush != want {
			t.Fatalf("3 must load clutch slot 3, got %+v want %+v", m.Brush, want)
		}
		m = send(m, key('P'))
		if m.Current().At(0, 0).FG != want.FG {
			t.Fatalf("P must paint clutch fg %d, got %d", want.FG, m.Current().At(0, 0).FG)
		}
	})
	t.Run("happy: 0 is the tenth slot, not a zero color", func(t *testing.T) {
		m := newEd(t)
		m = send(m, key('0'))
		if m.Brush != m.RecentColors[9] {
			t.Fatalf("0 must load slot 10, got %+v", m.Brush)
		}
	})
	t.Run("unhappy: clutch on a short list does not panic", func(t *testing.T) {
		m := newEd(t)
		m.RecentColors = m.RecentColors[:1]
		_ = send(m, key('9'))
	})
}

func TestRecentMemory(t *testing.T) {
	t.Run("happy: picking an 8-bit color moves it to clutch slot 1", func(t *testing.T) {
		m := newEd(t)
		m = send(m, key('c'))
		if !m.PickerOpen {
			t.Fatal("c must open the 8-bit picker")
		}
		// land on white 255 (last grey) and confirm
		m.PickerIdx = len(Greys) - 1
		m = send(m, keyType(tea.KeySpace))
		if m.Brush.FG != 255 {
			t.Fatalf("space in the picker must take grey %d, got fg %d", 255, m.Brush.FG)
		}
		if m.PickerOpen {
			t.Fatal("confirming a color must close the dropdown")
		}
		if m.RecentColors[0].FG != 255 {
			t.Fatalf("picked color must become clutch 1, got fg %d", m.RecentColors[0].FG)
		}
	})
	t.Run("happy: painting a glyph moves it to the front of the past-ten paints", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key(')')) // ▗
		m = send(m, key('P'))
		if m.RecentGlyphs[0] != '▗' {
			t.Fatalf("pasted glyph must become past-paint 1, got %q", string(m.RecentGlyphs[0]))
		}
	})
	t.Run("unhappy: remembering a color that is already first does not grow past 10", func(t *testing.T) {
		m := newEd(t)
		m = send(m, key('1'))
		m = send(m, key('1'))
		if len(m.RecentColors) > 10 {
			t.Fatalf("clutch must stay at 10, got %d", len(m.RecentColors))
		}
	})
}

func TestEightBitPicker(t *testing.T) {
	t.Run("happy: the greyscale ramp has enough whites to shade", func(t *testing.T) {
		if len(Greys) != 24 {
			t.Fatalf("xterm greyscale is 24 steps (232-255), got %d", len(Greys))
		}
		if Greys[0] != 232 || Greys[len(Greys)-1] != 255 {
			t.Fatalf("ramp must run 232→255, got %d→%d", Greys[0], Greys[len(Greys)-1])
		}
		nWhite := 0
		for _, g := range Greys {
			if g >= 250 {
				nWhite++
			}
		}
		if nWhite < 6 {
			t.Fatalf("need several near-whites for shading, got %d", nWhite)
		}
	})
	t.Run("happy: picker view shows the 8-bit dropdown and some greys", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 100, 30
		m = send(m, key('c'))
		v := m.View()
		if !strings.Contains(v, "8-bit") && !strings.Contains(v, "8BIT") && !strings.Contains(strings.ToLower(v), "8-bit") {
			if !strings.Contains(v, "grey") && !strings.Contains(v, "gray") && !strings.Contains(v, "picker") {
				t.Fatalf("open picker must be visible in the view, got %q", v[:min(200, len(v))])
			}
		}
	})
	t.Run("unhappy: escape closes the picker and keeps the old brush", func(t *testing.T) {
		m := newEd(t)
		before := m.Brush
		m = send(m, key('c'))
		m.PickerIdx = 0
		m = send(m, keyType(tea.KeyEsc))
		if m.PickerOpen {
			t.Fatal("esc must close the dropdown")
		}
		if m.Brush != before {
			t.Fatalf("esc must not apply the picker, got %+v", m.Brush)
		}
	})
}

func TestViewShowsPaintKit(t *testing.T) {
	t.Run("happy: the view lists clutch digits and paint symbols", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 100, 30
		v := m.View()
		for _, want := range []string{"1", "0", "!", "@", "#", "░", "█"} {
			if !strings.Contains(v, want) {
				t.Fatalf("view missing paint-kit mark %q", want)
			}
		}
	})
	t.Run("unhappy: a tiny terminal still includes the kit without panicking", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 10, 4
		_ = m.View()
	})
}

func TestPaintDropdownExtras(t *testing.T) {
	t.Run("happy: p cycles the symbol list onto extra quadrants (▘ ▝ ▛ ▜ ▙ ▟ ▞ ▚)", func(t *testing.T) {
		m := newEd(t)
		seen := map[rune]bool{}
		for i := 0; i < len(SymbolList)+2; i++ {
			m = send(m, key('p'))
			seen[m.PaintCh] = true
		}
		for _, ch := range ExtraGlyphs {
			if !seen[ch] {
				t.Fatalf("p never landed on extra glyph %q", string(ch))
			}
		}
	})
	t.Run("unhappy: p never selects a space or zero rune", func(t *testing.T) {
		m := newEd(t)
		m = send(m, key('p'))
		if m.PaintCh == 0 || m.PaintCh == ' ' {
			t.Fatal("p must pick a real block element")
		}
	})
}

func TestIUsesBrushGlyphNotAlwaysFullBlock(t *testing.T) {
	t.Run("happy: P with a half-block selected pastes ▀, not █", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('%'))
		m = send(m, key('P'))
		if m.Current().At(0, 0).Ch != '▀' {
			t.Fatalf("expected upper half, got %q", string(m.Current().At(0, 0).Ch))
		}
	})
	t.Run("unhappy: d still clears after a partial-block paste", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('%'))
		m = send(m, key('P'))
		m = send(m, key('d'))
		if !m.Current().At(0, 0).Transparent() {
			t.Fatal("d must still clear, even after a partial-block paste")
		}
	})
}

func TestInsertOneCharacter(t *testing.T) {
	t.Run("happy: i then a character inserts that character in the current color", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = 1
		m = send(m, key('i'))
		if !m.Inserting {
			t.Fatal("i must enter one-shot insert")
		}
		m = send(m, key('▀'))
		if m.Inserting {
			t.Fatal("insert consumes one character and leaves")
		}
		c := m.Current().At(0, 0)
		if c.Ch != '▀' {
			t.Fatalf("inserted %q, want ▀", string(c.Ch))
		}
		if c.FG != m.Atlas.Palette[1].FG {
			t.Fatalf("insert must keep the current color, fg %d", c.FG)
		}
	})
	t.Run("happy: i then a typed ASCII letter inserts that letter", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('i'))
		m = send(m, key('A'))
		if m.Current().At(0, 0).Ch != 'A' {
			t.Fatalf("got %q", string(m.Current().At(0, 0).Ch))
		}
	})
	t.Run("unhappy: i then esc cancels and does not touch the cell", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		before := m.Current().At(0, 0)
		m = send(m, key('i'))
		m = send(m, keyType(tea.KeyEsc))
		if m.Inserting {
			t.Fatal("esc must leave insert")
		}
		if m.Current().At(0, 0) != before {
			t.Fatal("esc must not stamp a character")
		}
	})
	t.Run("unhappy: i then h inserts h, it does not move the cursor", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 2, 2
		m = send(m, key('i'))
		m = send(m, key('h'))
		if m.CursorR != 2 || m.CursorC != 2 {
			t.Fatalf("insert must consume h, cursor moved to (%d,%d)", m.CursorR, m.CursorC)
		}
		if m.Current().At(2, 2).Ch != 'h' {
			t.Fatalf("inserted %q, want h", string(m.Current().At(2, 2).Ch))
		}
		if m.Inserting {
			t.Fatal("one-shot insert must end after the character")
		}
	})
}

func TestPasteFromSymbolList(t *testing.T) {
	t.Run("happy: the list has halves and quarters, P pastes the selected one", func(t *testing.T) {
		m := newEd(t)
		kinds := map[string]bool{}
		for _, s := range SymbolList {
			kinds[s.Kind] = true
		}
		for _, want := range []string{"full", "half", "quarter", "shade"} {
			if !kinds[want] {
				t.Fatalf("symbol list missing %s blocks", want)
			}
		}
		m.CursorR, m.CursorC = 0, 0
		m.SymIdx = 1 // ▀ up half
		m.PaintCh = SymbolList[1].Ch
		m = send(m, key('P'))
		if m.Current().At(0, 0).Ch != SymbolList[1].Ch {
			t.Fatalf("P must paste list[%d]=%q", 1, string(SymbolList[1].Ch))
		}
	})
	t.Run("happy: the view shows a symbols list with halves and quarters", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 100, 36
		v := m.View()
		for _, want := range []string{"symbols", "▀", "▖", "half", "quarter"} {
			if !strings.Contains(strings.ToLower(v), strings.ToLower(want)) && !strings.Contains(v, want) {
				t.Fatalf("view missing %q in symbol list", want)
			}
		}
	})
	t.Run("unhappy: P on an empty selection still pastes onto the cursor, not panic", func(t *testing.T) {
		m := newEd(t)
		m.sel = map[cellKey]bool{}
		m.CursorR, m.CursorC = 0, 0
		_ = send(m, key('P'))
	})
}

func TestSymbolListNavigation(t *testing.T) {
	t.Run("happy: j/k in the symbols window walk the list", func(t *testing.T) {
		m := newEd(t)
		m.Win = WinSymbols
		m.SymIdx = 1
		m = send(m, key('j'))
		if m.SymIdx != 2 {
			t.Fatalf("j must move to the next symbol, idx=%d", m.SymIdx)
		}
		m = send(m, key('k'))
		if m.SymIdx != 1 {
			t.Fatalf("k must move to the previous symbol, idx=%d", m.SymIdx)
		}
		if m.PaintCh != SymbolList[1].Ch {
			t.Fatalf("walking the list must select %q, got %q", string(SymbolList[1].Ch), string(m.PaintCh))
		}
	})
	t.Run("unhappy: j past the last symbol wraps instead of panicking", func(t *testing.T) {
		m := newEd(t)
		m.Win = WinSymbols
		m.SymIdx = len(SymbolList) - 1
		m = send(m, key('j'))
		if m.SymIdx != 0 {
			t.Fatalf("j at the end must wrap to 0, idx=%d", m.SymIdx)
		}
	})
}
