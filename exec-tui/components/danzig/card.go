package danzig

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

// Card is the picker as a screenplay component: Start pins the stage,
// Render paints the Rose Pine card centered (clipping if the stage is
// smaller than the card), Stop clears the staging.
type Card struct {
	w, h   int
	staged bool
}

// New returns an unstaged card.
func New() *Card { return &Card{} }

// Start pins the stage the card centers on.
func (c *Card) Start(w, h int) {
	if c == nil {
		return
	}
	c.w, c.h = w, h
	c.staged = true
}

// Update is a no-op: the picker is a still card.
func (c *Card) Update(dt float64) {}

// Render paints the card onto a stage-sized sprite. Before Start and
// after Stop the stage is empty.
func (c *Card) Render() sprite.Sprite {
	if c == nil || !c.staged || c.w < 1 || c.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(c.w, c.h)
	rows := frame(Source)
	top := (c.h - len(rows)) / 2
	left := 0
	if n := 0; len(rows) > 0 {
		n = len(rows[0])
		left = (c.w - n) / 2
	}
	for r, row := range rows {
		for col, cell := range row {
			stage.Set(top+r, left+col, sprite.Cell{
				Ch: cell.ch,
				FG: cell.fg.xterm(),
				BG: Base256,
			})
		}
	}
	return stage
}

// Stop clears the staging.
func (c *Card) Stop() {
	if c == nil {
		return
	}
	c.staged = false
}
