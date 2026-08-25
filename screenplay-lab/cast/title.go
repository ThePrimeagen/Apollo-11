package cast

import (
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
// here before the show starts.
func NewTitle(text string, height int) (*Title, error) {
	return nil, nil
}

// Advance is a no-op: a title card holds still.
func (t *Title) Advance(dt float64) {
}

// Paint centers the card on the stage.
func (t *Title) Paint(st *screenplay.Stage) {
}
