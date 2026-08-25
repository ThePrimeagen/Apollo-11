package cast

import (
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

// TitleFG is the card's xterm-256 ink: the mission gold.
const TitleFG = 178

// Title is a static card of banner text, centered on whatever stage it
// paints.
type Title struct {
	lines []string
}

// NewTitle sets text in the terminal-fonts banner at the given height
// (1..5). Unsupported heights and runes are termfont's errors, surfaced
// here — before the show starts, not on stage.
func NewTitle(text string, height int) (*Title, error) {
	lines, err := termfont.Lines(height, text)
	if err != nil {
		return nil, err
	}
	return &Title{lines: lines}, nil
}

// Advance is a no-op: a title card holds still.
func (t *Title) Advance(dt float64) {}

// Paint centers the card on the stage; edges clip on a stage too small
// for it.
func (t *Title) Paint(st *screenplay.Stage) {
	if t == nil || st == nil || len(t.lines) == 0 {
		return
	}
	w, h := st.Size()
	top := (h - len(t.lines)) / 2
	left := (w - len(t.lines[0])) / 2
	for r, line := range t.lines {
		for c, ch := range line {
			if ch == ' ' {
				continue
			}
			st.Put(top+r, left+c, sprite.Cell{Ch: ch, FG: TitleFG, BG: -1})
		}
	}
}
