package cast

// Tests written FIRST: the title card sets banner text with
// terminal-fonts and holds it centered on whatever stage it paints.
// termfont's own failures — heights outside 1..5, runes off the charset
// — surface at construction, before the show starts.

import (
	"errors"
	"testing"

	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

func stageRow(st *screenplay.Stage, row, col, width int) string {
	out := make([]rune, width)
	for i := range out {
		ch := st.Board.At(row, col+i).Ch
		if ch == 0 {
			ch = ' '
		}
		out[i] = ch
	}
	return string(out)
}

func TestNewTitle(t *testing.T) {
	t.Run("happy: THE END at height 5 lands centered, glyph for glyph", func(t *testing.T) {
		title, err := NewTitle("THE END", 5)
		if err != nil {
			t.Fatalf("NewTitle: %v", err)
		}
		st := screenplay.NewStage(stageW, stageH)
		title.Paint(st)
		lines, err := termfont.Lines(5, "THE END")
		if err != nil {
			t.Fatalf("termfont: %v", err)
		}
		width := len(lines[0])
		top := (stageH - len(lines)) / 2
		left := (stageW - width) / 2
		for r, want := range lines {
			if got := stageRow(st, top+r, left, width); got != want {
				t.Fatalf("card row %d\n got %q\nwant %q", r, got, want)
			}
		}
		for r := range lines {
			for c := 0; c < width; c++ {
				cell := st.Board.At(top+r, left+c)
				if cell.Transparent() {
					continue
				}
				if cell.FG != TitleFG {
					t.Fatalf("ink at (%d,%d) is %d, want %d", r, c, cell.FG, TitleFG)
				}
			}
		}
	})
	t.Run("happy: height 1 is the plain terminal font, still centered", func(t *testing.T) {
		title, err := NewTitle("GO", 1)
		if err != nil {
			t.Fatalf("NewTitle: %v", err)
		}
		st := screenplay.NewStage(stageW, stageH)
		title.Paint(st)
		row, col := (stageH-1)/2, (stageW-2)/2
		if got := stageRow(st, row, col, 2); got != "GO" {
			t.Fatalf("plain card %q at (%d,%d), want GO", got, row, col)
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

func TestTitleOnStage(t *testing.T) {
	t.Run("unhappy: a stage smaller than the card clips instead of panicking", func(t *testing.T) {
		title, err := NewTitle("THE END", 5)
		if err != nil {
			t.Fatalf("NewTitle: %v", err)
		}
		st := screenplay.NewStage(10, 3)
		title.Paint(st)
		if n := litCells(st); n > 10*3 {
			t.Fatalf("tiny stage lit %d cells, has only %d", n, 10*3)
		}
	})
	t.Run("unhappy: a nil card skips its cue without a panic", func(t *testing.T) {
		var title *Title
		title.Advance(1)
		title.Paint(screenplay.NewStage(4, 2))
	})
}
