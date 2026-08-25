package title

// Tests written FIRST: the title card sets banner text with
// terminal-fonts and holds it centered on whatever stage it starts on,
// inked in the mission gold. termfont's own failures — heights outside
// 1..5, runes off the charset — surface at construction, before the
// show starts. As a component: Start pins the stage, Render hands back
// the centered card as a sprite, Stop clears the staging.

import (
	"errors"
	"testing"

	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 72
	stageH = 28
)

// The compile-time pin: a Title plays as a screenplay component.
var _ screenplay.Component = (*Title)(nil)

func stageRow(sp sprite.Sprite, r, c, width int) string {
	out := make([]rune, width)
	for i := range out {
		cell := sp.At(r, c+i)
		if cell.Transparent() {
			out[i] = ' '
			continue
		}
		out[i] = cell.Ch
	}
	return string(out)
}

func opaqueCells(sp sprite.Sprite) int {
	n := 0
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			if !sp.At(r, c).Transparent() {
				n++
			}
		}
	}
	return n
}

func TestNewTitle(t *testing.T) {
	t.Run("happy: THE END at height 5 lands centered, glyph for glyph", func(t *testing.T) {
		card, err := New("THE END", 5)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		card.Start(stageW, stageH)
		stage := card.Render()
		if stage.Width != stageW || stage.Height != stageH {
			t.Fatalf("stage %dx%d, want %dx%d", stage.Width, stage.Height, stageW, stageH)
		}
		lines, err := termfont.Lines(5, "THE END")
		if err != nil {
			t.Fatalf("termfont: %v", err)
		}
		width := len(lines[0])
		top := (stageH - len(lines)) / 2
		left := (stageW - width) / 2
		for r, want := range lines {
			if got := stageRow(stage, top+r, left, width); got != want {
				t.Fatalf("card row %d\n got %q\nwant %q", r, got, want)
			}
		}
		for r := range lines {
			for c := 0; c < width; c++ {
				cell := stage.At(top+r, left+c)
				if cell.Transparent() {
					continue
				}
				if cell.FG != Ink {
					t.Fatalf("ink at (%d,%d) is %d, want %d", r, c, cell.FG, Ink)
				}
			}
		}
	})
	t.Run("happy: height 1 is the plain terminal font, still centered", func(t *testing.T) {
		card, err := New("GO", 1)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		card.Start(stageW, stageH)
		stage := card.Render()
		r, c := (stageH-1)/2, (stageW-2)/2
		if got := stageRow(stage, r, c, 2); got != "GO" {
			t.Fatalf("plain card %q at (%d,%d), want GO", got, r, c)
		}
	})
	t.Run("unhappy: a height outside 1..5 is termfont's error", func(t *testing.T) {
		card, err := New("THE END", 9)
		if card != nil || !errors.Is(err, termfont.ErrInvalidHeight) {
			t.Fatalf("want ErrInvalidHeight and no card, got %v / %v", card, err)
		}
	})
	t.Run("unhappy: a rune off the charset is termfont's error", func(t *testing.T) {
		card, err := New("thé end", 5)
		if card != nil || !errors.Is(err, termfont.ErrUnsupportedRune) {
			t.Fatalf("want ErrUnsupportedRune and no card, got %v / %v", card, err)
		}
	})
}

func TestTitleOnStage(t *testing.T) {
	t.Run("happy: stop clears the staging; a fresh start centers again", func(t *testing.T) {
		card, err := New("OK", 1)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		card.Start(stageW, stageH)
		if opaqueCells(card.Render()) == 0 {
			t.Fatal("test premise: a started card must show")
		}
		card.Stop()
		if sp := card.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped card rendered %dx%d", sp.Width, sp.Height)
		}
		card.Start(10, 3)
		if sp := card.Render(); sp.Width != 10 || sp.Height != 3 {
			t.Fatalf("a restaged card rendered %dx%d, want 10x3", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: rendering before the first start is an empty stage", func(t *testing.T) {
		card, err := New("OK", 1)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if sp := card.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted card rendered %dx%d", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a stage smaller than the card clips instead of panicking", func(t *testing.T) {
		card, err := New("THE END", 5)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		card.Start(10, 3)
		if n := opaqueCells(card.Render()); n > 10*3 {
			t.Fatalf("tiny stage lit %d cells, has only %d", n, 10*3)
		}
	})
	t.Run("unhappy: a nil card skips every cue", func(t *testing.T) {
		var ghost *Title
		ghost.Start(4, 2)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}
