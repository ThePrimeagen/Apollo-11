// Package title is the banner card component: text set in the
// terminal-fonts face, centered on whatever stage it starts on, inked
// in the mission gold.
package title

import (
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Ink is the card's xterm-256 color: the mission gold.
const Ink = 178

// Title is a static card of banner text. The banner itself is set at
// construction — a bad card fails before the show starts — and Start
// only pins the stage the card centers on.
type Title struct {
	lines  []string
	w, h   int
	staged bool
}

// New sets text in the terminal-fonts banner at the given height
// (1..5). Unsupported heights and runes are termfont's errors,
// surfaced here — before the show starts, not on stage.
func New(text string, height int) (*Title, error) {
	lines, err := termfont.Lines(height, text)
	if err != nil {
		return nil, err
	}
	return &Title{lines: lines}, nil
}

// Start pins the stage the card centers on.
func (t *Title) Start(w, h int) {
	if t == nil {
		return
	}
	t.w, t.h = w, h
	t.staged = true
}

// Update is a no-op: a title card holds still.
func (t *Title) Update(dt float64) {}

// Render centers the card on a stage-sized sprite; edges clip on a
// stage too small for it. Before Start and after Stop the card is off,
// so the stage is empty.
func (t *Title) Render() sprite.Sprite {
	if t == nil || !t.staged || t.w < 1 || t.h < 1 || len(t.lines) == 0 {
		return sprite.Sprite{}
	}
	stage := sprite.New(t.w, t.h)
	top := (t.h - len(t.lines)) / 2
	left := (t.w - len(t.lines[0])) / 2
	for r, line := range t.lines {
		for c, ch := range line {
			if ch == ' ' {
				continue
			}
			stage.Set(top+r, left+c, sprite.Cell{Ch: ch, FG: Ink, BG: -1})
		}
	}
	return stage
}

// Stop clears the staging; the banner itself is the card's identity
// and stays, so a fresh Start shows it again.
func (t *Title) Stop() {
	if t == nil {
		return
	}
	t.staged = false
}
