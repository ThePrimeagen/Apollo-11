// Package ie is the old Internet Explorer logo as a terminal still:
// the bold blue lowercase e with the golden swoosh orbiting it — the
// swoosh crossing in front of the e on the lower left, sweeping under
// the bowl, and flicking off the top right shoulder.
//
// The card is a fixed 14×7 cells, drawn in half-cell pixels: a
// terminal cell is about half as wide as it is tall, so each cell
// stacks two pixels with the half-block glyphs and the 14×14-pixel
// logo reads square on a real terminal. Art is the card alone; Logo
// is the screenplay component that centers it on any stage big enough
// to hold it and sits the show out on one that is not.
package ie

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

const (
	// Cols and Rows are the card's fixed size in terminal cells.
	Cols = 14
	Rows = 7
	// BlueInk is the e's IE blue, GoldInk the swoosh, GoldFade the
	// swoosh's tapering tips — all xterm-256.
	BlueInk  = 33
	GoldInk  = 220
	GoldFade = 178
)

// pixels is the logo, one rune per half-cell pixel, two rows per
// terminal cell: '.' empty, 'B' the blue e, 'G' the golden swoosh,
// 'g' the swoosh's faded taper. The swoosh's left limb runs down over
// the e's crossbar and lower stroke, rounds the bottom of the bowl,
// and the tail clears the top of the e to flick off the right.
var pixels = [2 * Rows]string{
	"........GG....",
	"....BBBBBBGG..",
	"..BBBBBBBBBBGg",
	"..BBB....BBBBg",
	"gBBB......BBB.",
	"GBBB......BBB.",
	"GGBBBBBBBBBBB.",
	"GGBBBBBBBBBBB.",
	"GGBB..........",
	".GGB..........",
	"..GGB.....BB..",
	"..GGGBBBBBBB..",
	"....GGGBBB....",
	".......GGg....",
}

// inkOf maps one pixel rune to its xterm ink; empty pixels are -1.
func inkOf(px byte) int {
	switch px {
	case 'B':
		return BlueInk
	case 'G':
		return GoldInk
	case 'g':
		return GoldFade
	}
	return -1
}

// Art paints the card fresh: 14×7 cells, each the half-block pairing
// of its two pixels — upper half, lower half, a full block when both
// pixels agree, or gold-over-blue where the swoosh crosses the e.
// Every call hands out its own cells, so no two cards alias.
func Art() sprite.Sprite {
	card := sprite.New(Cols, Rows)
	for r := 0; r < Rows; r++ {
		for c := 0; c < Cols; c++ {
			top, bot := inkOf(pixels[2*r][c]), inkOf(pixels[2*r+1][c])
			switch {
			case top < 0 && bot < 0:
				// empty sky — the stage shows through
			case bot < 0:
				card.Set(r, c, sprite.Cell{Ch: '▀', FG: top, BG: -1})
			case top < 0:
				card.Set(r, c, sprite.Cell{Ch: '▄', FG: bot, BG: -1})
			case top == bot:
				card.Set(r, c, sprite.Cell{Ch: '█', FG: top, BG: -1})
			default:
				card.Set(r, c, sprite.Cell{Ch: '▀', FG: top, BG: bot})
			}
		}
	}
	return card
}

// Logo is the card as a scene component. Start paints and caches the
// still for its stage; Update moves nothing — the logo holds still;
// Render lays the card centered on a stage-sized sprite; Stop drops
// the card so a stopped logo holds no allocation.
type Logo struct {
	card   sprite.Sprite
	w, h   int
	staged bool
}

// New binds nothing yet: the curtain owns the allocation.
func New() *Logo {
	return &Logo{}
}

// Start paints the card for a w×h stage.
func (l *Logo) Start(w, h int) {
	if l == nil {
		return
	}
	l.w, l.h = w, h
	l.card = Art()
	l.staged = true
}

// Update is a no-op: the logo is a still.
func (l *Logo) Update(dt float64) {}

// Render is the card centered on a stage-sized sprite. Before Start
// and after Stop the card is off, so the stage is empty; a stage too
// small for the card stays transparent.
func (l *Logo) Render() sprite.Sprite {
	if l == nil || !l.staged || l.w < 1 || l.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(l.w, l.h)
	if l.w < Cols || l.h < Rows {
		return stage
	}
	sprite.Blit(stage, (l.w-Cols)/2, (l.h-Rows)/2, l.card)
	return stage
}

// Stop drops the card for the collector; a fresh Start repaints it.
func (l *Logo) Stop() {
	if l == nil {
		return
	}
	l.card = sprite.Sprite{}
	l.staged = false
}
