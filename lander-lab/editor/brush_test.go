package editor

// Tests written FIRST. The paint kit: 1-0 clutch the last ten colors,
// !@#$%^&*() clutch the ten paint glyphs (shades, halves, quadrants),
// c opens an 8-bit picker with a greyscale ramp that has enough whites
// to shade, and i stamps the selected glyph in the selected color.

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
			m = send(m, key('i'))
			got := m.Current().At(0, 0).Ch
			if got != tc.ch {
				t.Fatalf("i after %q painted %q, want %q", string(tc.key), string(got), string(tc.ch))
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
		m = send(m, key('i'))
		if m.Current().At(0, 0).FG != want.FG {
			t.Fatalf("i must paint clutch fg %d, got %d", want.FG, m.Current().At(0, 0).FG)
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
		m = send(m, key('i'))
		if m.RecentGlyphs[0] != '▗' {
			t.Fatalf("painted glyph must become past-paint 1, got %q", string(m.RecentGlyphs[0]))
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
	t.Run("happy: p cycles onto the extra quadrants (▘ ▝ ▛ ▜ ▙ ▟ ▞ ▚)", func(t *testing.T) {
		m := newEd(t)
		seen := map[rune]bool{}
		for i := 0; i < len(DefaultGlyphs)+len(ExtraGlyphs)+2; i++ {
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
	t.Run("happy: i with a half-block selected paints ▀, not █", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('%'))
		m = send(m, key('i'))
		if m.Current().At(0, 0).Ch != '▀' {
			t.Fatalf("expected upper half, got %q", string(m.Current().At(0, 0).Ch))
		}
	})
	t.Run("unhappy: i still no-ops a transparent delete path via d, not the glyph keys", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('%'))
		m = send(m, key('i'))
		m = send(m, key('d'))
		if !m.Current().At(0, 0).Transparent() {
			t.Fatal("d must still clear, even after a partial-block paint")
		}
	})
}
