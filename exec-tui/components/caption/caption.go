// Package caption is a timed side banner: Cue texts painted in the
// terminal-fonts face on the right of the stage. Alarm codes wear the
// PROG red; LAND wears the mission gold. At most one card is up, and
// a later cue wins an overlap. This is the 1202 / 1202 / 1201 (and
// 1202 / 1202 / LAND) talk sitting beside the spacelander.
package caption

import (
	"math"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
)

const (
	// Height is the terminal-fonts face the board paints in.
	Height = 3
	// AlarmInk is the PROG red — 1202 and 1201.
	AlarmInk = 196
	// LandInk is the mission gold — LAND.
	LandInk = 178
)

// Cue is one timed card: Text paints from At until At+Hold. A zero
// hold never paints. Blink > 0 is the on/off half-period in seconds
// — the card flashes and the rest of the stage can freeze around it.
type Cue struct {
	Text  string
	At    float64
	Hold  float64
	Blink float64
}

// Board is the timed side banner as a scene component. The cues are
// set at construction — a bad rune fails before the show starts —
// and Start only pins the stage the card sits on.
type Board struct {
	cues   []Cue
	lines  [][]string
	clock  float64
	w, h   int
	staged bool
}

// New sets the board's cues in the terminal-fonts banner. Unsupported
// runes refuse the board entirely — before the show starts, not on
// stage. An empty cue list is a blank board.
func New(cues ...Cue) *Board {
	lines := make([][]string, len(cues))
	for i, c := range cues {
		ls, err := termfont.Lines(Height, c.Text)
		if err != nil {
			return nil
		}
		lines[i] = ls
	}
	out := make([]Cue, len(cues))
	copy(out, cues)
	return &Board{cues: out, lines: lines}
}

// Start pins the stage the card sits on and rewinds the clock.
func (b *Board) Start(w, h int) {
	if b == nil {
		return
	}
	b.w, b.h = w, h
	b.clock = 0
	b.staged = true
}

// Update walks the board's clock. dt <= 0 holds.
func (b *Board) Update(dt float64) {
	if b == nil || !b.staged || dt <= 0 {
		return
	}
	b.clock += dt
}

// Render paints the active card on the right of a stage-sized sprite.
// Later cues win an overlap. Before Start and after Stop the board is
// off.
func (b *Board) Render() sprite.Sprite {
	if b == nil || !b.staged || b.w < 1 || b.h < 1 {
		return sprite.Sprite{}
	}
	idx := -1
	for i, c := range b.cues {
		if c.Hold <= 0 {
			continue
		}
		if b.clock >= c.At && b.clock < c.At+c.Hold {
			if c.Blink > 0 && blinkOff(b.clock-c.At, c.Blink) {
				continue
			}
			idx = i
		}
	}
	if idx < 0 {
		return sprite.New(b.w, b.h)
	}
	lines := b.lines[idx]
	if len(lines) == 0 {
		return sprite.New(b.w, b.h)
	}
	ink := AlarmInk
	if b.cues[idx].Text == "LAND" {
		ink = LandInk
	}
	stage := sprite.New(b.w, b.h)
	width := len(lines[0])
	left := b.w - width - 1
	if left < 0 {
		left = 0
	}
	top := (b.h - len(lines)) / 2
	for r, line := range lines {
		for c, ch := range line {
			if ch == ' ' {
				continue
			}
			stage.Set(top+r, left+c, sprite.Cell{Ch: ch, FG: ink, BG: -1})
		}
	}
	return stage
}

// Painted reports whether Text's height-3 glyphs sit on scr. Only
// the board's ink (PROG red / mission gold) counts, so stars, the
// meteor, and hull gaps in the font do not hide the card.
func Painted(scr *screenplay.Screen, text string) bool {
	if scr == nil || text == "" {
		return false
	}
	want, err := termfont.Lines(Height, text)
	if err != nil {
		return false
	}
	w, h := scr.Size()
	for _, line := range want {
		line = strings.TrimRight(line, " ")
		if strings.TrimSpace(line) == "" {
			continue
		}
		found := false
		for y := 0; y < h; y++ {
			var b strings.Builder
			for x := 0; x < w; x++ {
				c := scr.Cell(x, y)
				if captionInk(c) {
					b.WriteRune(cellRune(c))
					continue
				}
				b.WriteByte(' ')
			}
			if strings.Contains(b.String(), line) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func blinkOff(elapsed, period float64) bool {
	if period <= 0 || math.IsNaN(period) || math.IsInf(period, 0) {
		return false
	}
	phase := math.Mod(elapsed, period*2)
	if phase < 0 {
		phase += period * 2
	}
	return phase >= period
}

func captionInk(c *uv.Cell) bool {
	if c == nil || c.Style.Fg == nil {
		return false
	}
	ic, ok := c.Style.Fg.(ansi.IndexedColor)
	if !ok {
		return false
	}
	n := int(ic)
	return n == AlarmInk || n == LandInk
}

func cellRune(c *uv.Cell) rune {
	if c == nil || c.Content == "" {
		return ' '
	}
	for _, r := range c.Content {
		return r
	}
	return ' '
}

// Stop clears the staging; the cues stay, so a fresh Start shows the
// opening card again.
func (b *Board) Stop() {
	if b == nil {
		return
	}
	b.staged = false
}
