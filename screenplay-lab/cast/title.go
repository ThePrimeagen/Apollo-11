package cast

import (
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

// Update is a no-op: a title card holds still.
func (t *Title) Update(dt float64) {}

// Render centers the card on the screen; edges clip on a screen too
// small for it.
func (t *Title) Render(scr *screenplay.Screen) {
}
