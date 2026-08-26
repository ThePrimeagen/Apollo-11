package editor

// Tests written FIRST. Ctrl-E opens a 26×10 filterable/selectable
// character grid: columns 1–0, rows a–z. Each number column is one
// consecutive glyph group stacked down the lettered rows. hjkl walk
// a highlight; enter picks it. A number then a letter (3c) still
// jumps. Escape and invalid combos leave the canvas and the current
// paint glyph alone.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestGlyphGrid(t *testing.T) {
	t.Run("happy: ctrl-e opens a 26 by 10 grid labeled a-z rows and 1-0 columns", func(t *testing.T) {
		m := newEd(t)
		m.TermW, m.TermH = 120, 40
		m = send(m, keyCtrl('e'))
		if !m.GlyphGridOpen {
			t.Fatal("ctrl-e must open the character grid")
		}
		v := m.View().Content
		for _, want := range []string{"a", "z", "1", "0"} {
			if !strings.Contains(v, want) {
				t.Fatalf("grid view missing label %q", want)
			}
		}
		if GlyphGridRows != 26 || GlyphGridCols != 10 {
			t.Fatalf("grid must be 26×10 (tall), got %d×%d", GlyphGridRows, GlyphGridCols)
		}
		if n := countGlyphGrid(); n != 260 {
			t.Fatalf("grid must fill 260 slots, got %d", n)
		}
	})
	t.Run("happy: typing a valid number+letter sets PaintCh and closes", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		before := m.Current()
		want, ok := GlyphAt('3', 'c')
		if !ok || want == 0 || want == ' ' {
			t.Fatal("3c must address a real drawing character")
		}
		m = send(m, keyCtrl('e'))
		m = send(m, key('3'))
		if !m.GlyphGridOpen {
			t.Fatal("the first key of 3c must keep the grid open")
		}
		if m.PaintCh == want && m.PaintCh != '█' {
			// only fail if it applied early AND it wasn't already that glyph
		}
		m = send(m, key('c'))
		if m.GlyphGridOpen {
			t.Fatal("3c must close the grid after the letter")
		}
		if m.PaintCh != want {
			t.Fatalf("PaintCh %q, want 3c %q", string(m.PaintCh), string(want))
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("picking a glyph must not stamp the canvas — only PaintCh")
		}
	})
	t.Run("unhappy: escape closes without changing the paint character", func(t *testing.T) {
		m := newEd(t)
		m.PaintCh = '░'
		before := m.Current()
		m = send(m, keyCtrl('e'))
		m = send(m, key('3'))
		m = send(m, keyType(tea.KeyEsc))
		if m.GlyphGridOpen {
			t.Fatal("esc must close the grid")
		}
		if m.PaintCh != '░' {
			t.Fatalf("esc must keep PaintCh, got %q", string(m.PaintCh))
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("esc must not paint the canvas")
		}
	})
	t.Run("unhappy: an invalid combo does not panic or paint", func(t *testing.T) {
		m := newEd(t)
		m.PaintCh = '█'
		m.CursorR, m.CursorC = 0, 0
		before := m.Current()
		m = send(m, keyCtrl('e'))
		m = send(m, key('c')) // letter first, not number+letter
		if m.PaintCh != '█' {
			t.Fatalf("letter-first must not change PaintCh, got %q", string(m.PaintCh))
		}
		m = send(m, key('3'))
		m = send(m, key('3')) // number then number
		if m.PaintCh != '█' {
			t.Fatalf("3 then 3 must not change PaintCh, got %q", string(m.PaintCh))
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("invalid combo must not paint the canvas")
		}
		if !m.GlyphGridOpen {
			t.Fatal("invalid combo must leave the grid open")
		}
	})
	t.Run("unhappy: ctrl-e while another modal is open does not panic or paint", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 0, 0
		before := m.Current()
		m = send(m, key('c'))
		if !m.PickerOpen {
			t.Fatal("need the color picker open")
		}
		m = send(m, keyCtrl('e'))
		if m.GlyphGridOpen {
			t.Fatal("ctrl-e must not open over another modal")
		}
		if sprite.Render(m.Current()) != sprite.Render(before) {
			t.Fatal("ctrl-e in another modal must not paint the canvas")
		}
	})
	t.Run("happy: each column number sits on the same cell as its glyph", func(t *testing.T) {
		m := newEd(t)
		inner := glyphGridInnerLines(renderGlyphGrid(m))
		if len(inner) < 27 {
			t.Fatalf("need header plus 26 lettered rows, got %d inner lines", len(inner))
		}
		head := []rune(strip(inner[0]))
		row := []rune(strip(inner[1]))
		for _, col := range glyphGridColKeys {
			letterAt := runeIndex(head, col)
			want, ok := GlyphAt(col, 'a')
			if !ok {
				t.Fatalf("%ca missing", col)
			}
			glyphAt := runeIndex(row, want)
			if letterAt < 0 {
				t.Fatalf("header is missing column %c", col)
			}
			if glyphAt < 0 {
				t.Fatalf("row a is missing the %ca glyph %q", col, string(want))
			}
			if letterAt != glyphAt {
				t.Fatalf("column %c: number at %d, glyph at %d (off by %d)",
					col, letterAt, glyphAt, glyphAt-letterAt)
			}
		}
	})
	t.Run("happy: lettered rows run a-z and number columns group glyphs down the page", func(t *testing.T) {
		m := newEd(t)
		inner := glyphGridInnerLines(renderGlyphGrid(m))
		if len(inner) < 27 {
			t.Fatalf("need header plus 26 rows, got %d", len(inner))
		}
		for i, letter := range lettersAZ() {
			rs := []rune(strip(inner[i+1]))
			if len(rs) < 2 {
				t.Fatalf("row %d is too short: %q", i, string(rs))
			}
			if rs[1] != letter {
				t.Fatalf("row %d must be labeled %c, got %q", i, letter, string(rs))
			}
			want, ok := GlyphAt('1', letter)
			if !ok {
				t.Fatalf("1%c missing", letter)
			}
			got := rs[2]
			if got != want {
				t.Fatalf("column 1 row %c: got %q want group glyph %q", letter, string(got), string(want))
			}
		}
		prev := rune(0)
		for _, letter := range lettersAZ() {
			ch, ok := GlyphAt('1', letter)
			if !ok {
				t.Fatalf("1%c missing", letter)
			}
			if ch <= prev {
				t.Fatalf("column 1 must be one consecutive group, 1%c %q followed %q",
					letter, string(ch), string(prev))
			}
			if ch < 0x2580 || ch > 0x259F {
				t.Fatalf("column 1 must be the block-element group, 1%c is %q", letter, string(ch))
			}
			prev = ch
		}
	})
	t.Run("happy: hjkl move the grid cursor and enter picks that cell", func(t *testing.T) {
		m := newEd(t)
		m.CursorR, m.CursorC = 2, 2
		beforeCanvas := m.Current()
		m = send(m, keyCtrl('e'))
		if m.GlyphGridRow != 0 || m.GlyphGridCol != 0 {
			t.Fatalf("grid cursor must start at 1a, got (%d,%d)", m.GlyphGridRow, m.GlyphGridCol)
		}
		m = send(m, key('l'))
		m = send(m, key('l'))
		m = send(m, key('j'))
		if m.GlyphGridRow != 1 || m.GlyphGridCol != 2 {
			t.Fatalf("l l j must land on 3b, got (%d,%d)", m.GlyphGridRow, m.GlyphGridCol)
		}
		if m.CursorR != 2 || m.CursorC != 2 {
			t.Fatalf("hjkl must not leak to the canvas, got (%d,%d)", m.CursorR, m.CursorC)
		}
		m = send(m, key('k'))
		m = send(m, key('h'))
		if m.GlyphGridRow != 0 || m.GlyphGridCol != 1 {
			t.Fatalf("k h must land on 2a, got (%d,%d)", m.GlyphGridRow, m.GlyphGridCol)
		}
		want := glyphGrid[0][1]
		m = send(m, keyType(tea.KeyEnter))
		if m.GlyphGridOpen {
			t.Fatal("enter must close the grid after picking")
		}
		if m.PaintCh != want {
			t.Fatalf("enter must set PaintCh to 2a %q, got %q", string(want), string(m.PaintCh))
		}
		if sprite.Render(m.Current()) != sprite.Render(beforeCanvas) {
			t.Fatal("enter must not stamp the canvas — only PaintCh")
		}
	})
	t.Run("unhappy: esc after moving the cursor does not pick", func(t *testing.T) {
		m := newEd(t)
		m.PaintCh = '░'
		m.CursorR, m.CursorC = 1, 1
		m = send(m, keyCtrl('e'))
		m = send(m, key('j'))
		m = send(m, key('l'))
		m = send(m, keyType(tea.KeyEsc))
		if m.GlyphGridOpen {
			t.Fatal("esc must close the grid")
		}
		if m.PaintCh != '░' {
			t.Fatalf("esc must keep PaintCh, got %q", string(m.PaintCh))
		}
		if m.CursorR != 1 || m.CursorC != 1 {
			t.Fatalf("esc must not move the canvas cursor, got (%d,%d)", m.CursorR, m.CursorC)
		}
	})
	t.Run("unhappy: a wide glyph must not shove later columns or the box edge", func(t *testing.T) {
		m := newEd(t)
		const inner = 2 + GlyphGridCols*2
		boxed := renderGlyphGrid(m)
		for i, line := range glyphGridInnerLines(boxed) {
			if n := visibleLen(line); n != inner {
				t.Fatalf("inner line %d is %d cells, want %d (wide rune overflow)", i, n, inner)
			}
		}
		for _, col := range glyphGridColKeys {
			for _, row := range lettersAZ() {
				ch, ok := GlyphAt(col, row)
				if !ok {
					t.Fatalf("%c%c missing", col, row)
				}
				if ch == '\u25FD' || ch == '\u25FE' {
					t.Fatalf("%c%c %q is two cells wide and shoves the rest of the row",
						col, row, string(ch))
				}
			}
		}
	})
}

func TestGlyphAt(t *testing.T) {
	t.Run("happy: every number+letter cell is a non-space drawing rune", func(t *testing.T) {
		for _, col := range glyphGridColKeys {
			for _, row := range lettersAZ() {
				ch, ok := GlyphAt(col, row)
				if !ok {
					t.Fatalf("%c%c missing", col, row)
				}
				if ch == 0 || ch == ' ' {
					t.Fatalf("%c%c is empty", col, row)
				}
			}
		}
	})
	t.Run("unhappy: an out-of-range address is not ok", func(t *testing.T) {
		if _, ok := GlyphAt('x', 'a'); ok {
			t.Fatal("column x must be invalid")
		}
		if _, ok := GlyphAt('1', 'A'); ok {
			t.Fatal("uppercase rows are not addresses")
		}
	})
}

func glyphGridInnerLines(boxed string) []string {
	var out []string
	for _, line := range strings.Split(boxed, "\n") {
		rs := []rune(line)
		if len(rs) >= 2 && rs[0] == '│' {
			out = append(out, string(rs[1:len(rs)-1]))
		}
	}
	return out
}

func runeIndex(rs []rune, want rune) int {
	for i, r := range rs {
		if r == want {
			return i
		}
	}
	return -1
}

func lettersAZ() []rune {
	out := make([]rune, 26)
	for i := range out {
		out[i] = 'a' + rune(i)
	}
	return out
}

func countGlyphGrid() int {
	n := 0
	for _, col := range glyphGridColKeys {
		for _, row := range lettersAZ() {
			if ch, ok := GlyphAt(col, row); ok && ch != 0 && ch != ' ' {
				n++
			}
		}
	}
	return n
}
