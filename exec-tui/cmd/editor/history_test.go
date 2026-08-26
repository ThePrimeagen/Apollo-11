package editor

// Tests written FIRST. The editor keeps the last 10 unique glyphs and the
// last 10 unique colors the user actually used, most-recent-first, and
// shows both rows in the TUI so they can be seen and re-selected.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestUnusedEditorStartsEmpty(t *testing.T) {
	t.Run("happy: a fresh editor has no recent glyphs or colors", func(t *testing.T) {
		m := newEd(t)
		if len(m.RecentGlyphs) != 0 {
			t.Fatalf("unused editor RecentGlyphs must be empty, got %d %q", len(m.RecentGlyphs), string(m.RecentGlyphs))
		}
		if len(m.RecentColors) != 0 {
			t.Fatalf("unused editor RecentColors must be empty, got %d", len(m.RecentColors))
		}
	})
}

func TestUsingSymbolRecordsHistory(t *testing.T) {
	t.Run("happy: painting a glyph records it most-recent-first", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('!')) // ░
		m = send(m, key('P'))
		m = send(m, key('@')) // ▒
		m = send(m, key('P'))
		if len(m.RecentGlyphs) != 2 {
			t.Fatalf("two painted glyphs, got %d", len(m.RecentGlyphs))
		}
		if m.RecentGlyphs[0] != '▒' || m.RecentGlyphs[1] != '░' {
			t.Fatalf("most-recent-first, got %q", string(m.RecentGlyphs))
		}
	})
	t.Run("happy: inserting a character records that glyph", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('i'))
		m = send(m, key('X'))
		if len(m.RecentGlyphs) != 1 || m.RecentGlyphs[0] != 'X' {
			t.Fatalf("insert must record X, got %q", string(m.RecentGlyphs))
		}
	})
	t.Run("happy: picking a glyph from the grid records it without painting", func(t *testing.T) {
		m := newEd(t)
		want, ok := GlyphAt('3', 'c')
		if !ok {
			t.Fatal("3c must address a glyph")
		}
		before := m.Current()
		m = send(m, keyCtrl('e'))
		m = send(m, key('3'))
		m = send(m, key('c'))
		if len(m.RecentGlyphs) == 0 || m.RecentGlyphs[0] != want {
			t.Fatalf("grid pick must record %q, got %q", string(want), string(m.RecentGlyphs))
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("grid pick must not paint the canvas")
		}
	})
	t.Run("unhappy: deleting a cell does not invent a recent glyph", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('d'))
		if len(m.RecentGlyphs) != 0 {
			t.Fatalf("delete must not record a glyph, got %q", string(m.RecentGlyphs))
		}
	})
}

func TestUsingColorRecordsHistory(t *testing.T) {
	t.Run("happy: painting records the brush color most-recent-first", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		m.Brush = Swatch{FG: 200, BG: -1}
		m = send(m, key('P'))
		m.Brush = Swatch{FG: 201, BG: -1}
		m = send(m, key('P'))
		if len(m.RecentColors) != 2 {
			t.Fatalf("two painted colors, got %d", len(m.RecentColors))
		}
		if m.RecentColors[0].FG != 201 || m.RecentColors[1].FG != 200 {
			t.Fatalf("most-recent-first, got fg %d then %d", m.RecentColors[0].FG, m.RecentColors[1].FG)
		}
	})
	t.Run("happy: confirming the 8-bit picker records that color", func(t *testing.T) {
		m := newEd(t)
		m = send(m, key('c'))
		m.PickerIdx = len(Greys) - 1
		m = send(m, keyType(tea.KeySpace))
		if len(m.RecentColors) != 1 || m.RecentColors[0].FG != 255 {
			t.Fatalf("picker confirm must record grey 255, got %+v", m.RecentColors)
		}
	})
	t.Run("happy: confirming the named color palette records that color", func(t *testing.T) {
		m := newEd(t)
		gold := m.Atlas.Palette[2]
		m.PalIdx = 1
		m = send(m, keyCtrl('k'))
		m = send(m, key('j'))
		m = send(m, keyType(tea.KeyEnter))
		if len(m.RecentColors) != 1 {
			t.Fatalf("named pick must record one color, got %d", len(m.RecentColors))
		}
		if m.RecentColors[0].FG != gold.FG || m.RecentColors[0].BG != gold.BG {
			t.Fatalf("recorded %+v, want gold fg %d bg %d", m.RecentColors[0], gold.FG, gold.BG)
		}
	})
	t.Run("unhappy: escaping the picker does not record a color", func(t *testing.T) {
		m := newEd(t)
		m = send(m, key('c'))
		m.PickerIdx = 0
		m = send(m, keyType(tea.KeyEsc))
		if len(m.RecentColors) != 0 {
			t.Fatalf("esc must not record a color, got %d", len(m.RecentColors))
		}
	})
}

func TestRecentHistoryUniqueMRU(t *testing.T) {
	t.Run("happy: reusing a glyph moves it to the front and does not double-count", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		for _, k := range []rune{'!', '@', '#'} {
			m = send(m, key(k))
			m = send(m, key('P'))
		}
		m = send(m, key('!'))
		m = send(m, key('P'))
		if len(m.RecentGlyphs) != 3 {
			t.Fatalf("unique list must stay at 3, got %d %q", len(m.RecentGlyphs), string(m.RecentGlyphs))
		}
		if m.RecentGlyphs[0] != '░' {
			t.Fatalf("reused ░ must move to front, got %q", string(m.RecentGlyphs))
		}
	})
	t.Run("happy: reusing a color moves it to the front and does not double-count", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		for _, fg := range []int{10, 20, 30} {
			m.Brush = Swatch{FG: fg, BG: -1}
			m = send(m, key('P'))
		}
		m.Brush = Swatch{FG: 10, BG: -1}
		m = send(m, key('P'))
		if len(m.RecentColors) != 3 {
			t.Fatalf("unique list must stay at 3, got %d", len(m.RecentColors))
		}
		if m.RecentColors[0].FG != 10 {
			t.Fatalf("reused fg 10 must move to front, got fg %d", m.RecentColors[0].FG)
		}
	})
	t.Run("unhappy: remembering a color that is already first does not grow the list", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		m.Brush = Swatch{FG: 99, BG: -1}
		m = send(m, key('P'))
		m = send(m, key('P'))
		if len(m.RecentColors) != 1 {
			t.Fatalf("already-first color must stay a single slot, got %d", len(m.RecentColors))
		}
	})
}

func TestRecentHistoryCapsAtTen(t *testing.T) {
	t.Run("happy: an 11th unique glyph drops the oldest", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		for _, ch := range []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K'} {
			m = send(m, key('i'))
			m = send(m, key(ch))
		}
		if len(m.RecentGlyphs) != 10 {
			t.Fatalf("cap is 10, got %d %q", len(m.RecentGlyphs), string(m.RecentGlyphs))
		}
		if m.RecentGlyphs[0] != 'K' {
			t.Fatalf("newest must be K, got %q", string(m.RecentGlyphs[0]))
		}
		for _, ch := range m.RecentGlyphs {
			if ch == 'A' {
				t.Fatal("oldest glyph A must drop off")
			}
		}
	})
	t.Run("happy: an 11th unique color drops the oldest", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		for i := 0; i < 11; i++ {
			m.Brush = Swatch{FG: 100 + i, BG: -1}
			m = send(m, key('P'))
		}
		if len(m.RecentColors) != 10 {
			t.Fatalf("cap is 10, got %d", len(m.RecentColors))
		}
		if m.RecentColors[0].FG != 110 {
			t.Fatalf("newest must be fg 110, got %d", m.RecentColors[0].FG)
		}
		for _, s := range m.RecentColors {
			if s.FG == 100 {
				t.Fatal("oldest color fg 100 must drop off")
			}
		}
	})
}

func TestRecentHistoryEmptyEdges(t *testing.T) {
	t.Run("unhappy: rememberGlyph ignores space and zero", func(t *testing.T) {
		got := rememberGlyph(nil, ' ', 10)
		if len(got) != 0 {
			t.Fatalf("space must not enter history, got %q", string(got))
		}
		got = rememberGlyph(nil, 0, 10)
		if len(got) != 0 {
			t.Fatalf("zero rune must not enter history, got %q", string(got))
		}
	})
	t.Run("unhappy: 1-0 on an empty clutch does not panic", func(t *testing.T) {
		m := newEd(t)
		_ = send(m, key('1'))
		_ = send(m, key('0'))
		if len(m.RecentColors) != 0 {
			t.Fatalf("empty clutch must stay empty, got %d", len(m.RecentColors))
		}
	})
	t.Run("unhappy: empty recents still render without panicking", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		v := m.View().Content
		if !strings.Contains(v, "recent glyphs") {
			t.Fatal("empty editor must still show the recent glyphs row")
		}
		if !strings.Contains(v, "recent colors") {
			t.Fatal("empty editor must still show the recent colors row")
		}
	})
}

func TestViewShowsRecentHistory(t *testing.T) {
	t.Run("happy: the default canvas view lists the last used glyphs and colors", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 80, 24
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		m.Brush = Swatch{FG: 200, BG: -1}
		m = send(m, key('i'))
		m = send(m, key('Q'))
		m.Brush = Swatch{FG: 201, BG: -1}
		m = send(m, key('i'))
		m = send(m, key('Z'))
		v := strip(m.View().Content)
		if !strings.Contains(v, "recent glyphs") {
			t.Fatal("view missing recent glyphs label")
		}
		if !strings.Contains(v, "recent colors") {
			t.Fatal("view missing recent colors label")
		}
		if !strings.Contains(v, "Z") || !strings.Contains(v, "Q") {
			t.Fatalf("view must show used glyphs Q and Z, got %q", v)
		}
	})
	t.Run("happy: all ten recent glyphs and ten clutch digits are visible", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 100, 30
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		glyphs := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J'}
		for i, ch := range glyphs {
			m.Brush = Swatch{FG: 210 + i, BG: -1}
			m = send(m, key('i'))
			m = send(m, key(ch))
		}
		v := strip(m.View().Content)
		for _, ch := range glyphs {
			if !strings.ContainsRune(v, ch) {
				t.Fatalf("view missing recent glyph %q", string(ch))
			}
		}
		for _, k := range ColorKeys {
			if !strings.ContainsRune(v, k) {
				t.Fatalf("view missing clutch digit %q", string(k))
			}
		}
	})
	t.Run("happy: recents sit in the color palette and glyph grid overlays", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 120, 40
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('i'))
		m = send(m, key('W'))
		m = send(m, key('c'))
		m.PickerIdx = len(Greys) - 1
		m = send(m, keyType(tea.KeySpace))

		m = send(m, keyCtrl('k'))
		cv := strip(m.View().Content)
		if !strings.Contains(cv, "recent colors") {
			t.Fatal("color palette overlay must show recent colors")
		}
		m = send(m, keyType(tea.KeyEsc))

		m = send(m, keyCtrl('e'))
		gv := strip(m.View().Content)
		if !strings.Contains(gv, "recent glyphs") {
			t.Fatal("glyph grid overlay must show recent glyphs")
		}
		if !strings.ContainsRune(gv, 'W') {
			t.Fatal("glyph grid overlay must include the used glyph W")
		}
	})
}

func TestReselectRecentHistory(t *testing.T) {
	t.Run("happy: 1-0 reload a used color without shuffling the clutch", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m.PalIdx = -1
		for _, fg := range []int{11, 22, 33} {
			m.Brush = Swatch{FG: fg, BG: -1}
			m = send(m, key('P'))
		}
		// most-recent-first: 33, 22, 11
		before := append([]Swatch(nil), m.RecentColors...)
		m = send(m, key('2'))
		if m.Brush.FG != 22 {
			t.Fatalf("2 must load clutch slot 2 (fg 22), got %+v", m.Brush)
		}
		if len(m.RecentColors) != len(before) || m.RecentColors[0].FG != 33 || m.RecentColors[1].FG != 22 {
			t.Fatalf("re-select must not shuffle, got %+v", m.RecentColors)
		}
	})
	t.Run("happy: picking a recent glyph reloads PaintCh without shuffling", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		m = send(m, key('!'))
		m = send(m, key('P'))
		m = send(m, key('@'))
		m = send(m, key('P'))
		m = send(m, key('#'))
		m = send(m, key('P'))
		// most-recent-first: ▓ ▒ ░
		before := string(m.RecentGlyphs)
		if !m.pickRecentGlyph(1) {
			t.Fatal("slot 2 must be pickable")
		}
		if m.PaintCh != '▒' {
			t.Fatalf("recent glyph 2 must be ▒, got %q", string(m.PaintCh))
		}
		if string(m.RecentGlyphs) != before {
			t.Fatalf("re-select must not shuffle, got %q want %q", string(m.RecentGlyphs), before)
		}
	})
	t.Run("unhappy: picking a recent glyph past the list is a no-op", func(t *testing.T) {
		m := newEd(t)
		before := m.PaintCh
		if m.pickRecentGlyph(0) {
			t.Fatal("empty recents must not pick")
		}
		if m.PaintCh != before {
			t.Fatalf("empty pick must keep PaintCh, got %q", string(m.PaintCh))
		}
	})
}
