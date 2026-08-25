package cast

// Tests written FIRST: the title card sets banner text with
// terminal-fonts and holds it centered on whatever screen it renders
// to, inked in the mission gold. termfont's own failures — heights
// outside 1..5, runes off the charset — surface at construction,
// before the show starts.

import (
	"errors"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

func screenRow(scr *screenplay.Screen, y, x, width int) string {
	out := make([]rune, width)
	for i := range out {
		s := contentAt(scr, x+i, y)
		if s == "" || s == " " {
			out[i] = ' '
			continue
		}
		out[i] = []rune(s)[0]
	}
	return string(out)
}

func TestNewTitle(t *testing.T) {
	t.Run("happy: THE END at height 5 lands centered, glyph for glyph", func(t *testing.T) {
		title, err := NewTitle("THE END", 5)
		if err != nil {
			t.Fatalf("NewTitle: %v", err)
		}
		scr := screenplay.NewScreen(screenW, screenH)
		title.Render(scr)
		lines, err := termfont.Lines(5, "THE END")
		if err != nil {
			t.Fatalf("termfont: %v", err)
		}
		width := len(lines[0])
		top := (screenH - len(lines)) / 2
		left := (screenW - width) / 2
		for r, want := range lines {
			if got := screenRow(scr, top+r, left, width); got != want {
				t.Fatalf("card row %d\n got %q\nwant %q", r, got, want)
			}
		}
		for r := range lines {
			for c := 0; c < width; c++ {
				cell := scr.Cell(left+c, top+r)
				if cell == nil || cell.Content == " " || cell.Content == "" {
					continue
				}
				if cell.Style.Fg != ansi.IndexedColor(TitleFG) {
					t.Fatalf("ink at (%d,%d) is %v, want indexed %d", r, c, cell.Style.Fg, TitleFG)
				}
			}
		}
	})
	t.Run("happy: height 1 is the plain terminal font, still centered", func(t *testing.T) {
		title, err := NewTitle("GO", 1)
		if err != nil {
			t.Fatalf("NewTitle: %v", err)
		}
		scr := screenplay.NewScreen(screenW, screenH)
		title.Render(scr)
		y, x := (screenH-1)/2, (screenW-2)/2
		if got := screenRow(scr, y, x, 2); got != "GO" {
			t.Fatalf("plain card %q at (%d,%d), want GO", got, x, y)
		}
	})
	t.Run("unhappy: a height outside 1..5 is termfont's error", func(t *testing.T) {
		title, err := NewTitle("THE END", 9)
		if title != nil || !errors.Is(err, termfont.ErrInvalidHeight) {
			t.Fatalf("want ErrInvalidHeight and no card, got %v / %v", title, err)
		}
	})
	t.Run("unhappy: a rune off the charset is termfont's error", func(t *testing.T) {
		title, err := NewTitle("thé end", 5)
		if title != nil || !errors.Is(err, termfont.ErrUnsupportedRune) {
			t.Fatalf("want ErrUnsupportedRune and no card, got %v / %v", title, err)
		}
	})
}

func TestTitleOnScreen(t *testing.T) {
	t.Run("unhappy: a screen smaller than the card clips instead of panicking", func(t *testing.T) {
		title, err := NewTitle("THE END", 5)
		if err != nil {
			t.Fatalf("NewTitle: %v", err)
		}
		scr := screenplay.NewScreen(10, 3)
		title.Render(scr)
		if n := litCount(scr); n > 10*3 {
			t.Fatalf("tiny screen lit %d cells, has only %d", n, 10*3)
		}
	})
	t.Run("unhappy: a nil card and a nil screen both skip the cue", func(t *testing.T) {
		var title *Title
		title.Update(1)
		title.Render(screenplay.NewScreen(4, 2))
		card, err := NewTitle("OK", 1)
		if err != nil {
			t.Fatalf("NewTitle: %v", err)
		}
		card.Render(nil)
	})
}
