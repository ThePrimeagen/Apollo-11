// Package scrollcode is the moving part of a code walkthrough. You
// hand it many code cards in reading order and flag which of them
// are stops; the cards themselves never move — this component moves
// them. It stacks the cards into one column with a blank row between
// them, parks the spotlit card's first row on the anchor, and hangs
// the vignette on both sides: the focused card bright, one card out
// equally dimmed above and below, two out faint, three out barely
// there, and past that nothing paints. On its own clock it rests
// HoldSeconds on each stop, glides GlideSeconds to the next on an
// eased camera that lands exactly before the hold begins, and holds
// forever on the last stop. Next cuts the current rest short for
// callers that want to drive the walkthrough by hand.
package scrollcode

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/code"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// The stock clock.
const (
	// HoldSeconds rests the spotlight on each stop.
	HoldSeconds = 4.0
	// GlideSeconds carries the camera between stops.
	GlideSeconds = 0.9
)

// Block is one card on the scroll and whether the spotlight stops
// on it.
type Block struct {
	Code *code.Code
	Stop bool
}

// Scroll runs the column. The clock is its identity — a resize
// (Stop then Start) keeps it — so a fresh walkthrough is a fresh
// Scroll.
type Scroll struct {
	blocks      []Block
	stops       []int
	hold, glide float64
	clock       float64
	w, h        int
	staged      bool
}

// New is a scroll over the blocks, on the stock clock.
func New(blocks ...Block) *Scroll {
	s := &Scroll{blocks: blocks, hold: HoldSeconds, glide: GlideSeconds}
	for i, b := range blocks {
		if b.Stop {
			s.stops = append(s.stops, i)
		}
	}
	return s
}

// Tune sets the rest on each stop and the glide between them.
func (s *Scroll) Tune(hold, glide float64) *Scroll {
	if s == nil {
		return s
	}
	s.hold, s.glide = hold, glide
	return s
}

// Stops is how many stops the spotlight visits.
func (s *Scroll) Stops() int {
	if s == nil {
		return 0
	}
	return len(s.stops)
}

// AnchorY is the screen row a spotlit card's first row parks on —
// high enough that the fade below has room to run out.
func AnchorY(h int) int {
	y := h / 4
	if y < 1 {
		y = 1
	}
	return y
}

// Start pins the stage.
func (s *Scroll) Start(w, h int) {
	if s == nil {
		return
	}
	s.w, s.h = w, h
	s.staged = true
}

// Update advances the clock. Time never runs backwards.
func (s *Scroll) Update(dt float64) {
	if s == nil || dt <= 0 {
		return
	}
	s.clock += dt
}

// Stop clears the staging and keeps the clock, so a resize resumes
// where it was.
func (s *Scroll) Stop() {
	if s == nil {
		return
	}
	s.staged = false
}

// FocusPos is the continuous spotlight position in stop ordinals:
// whole at a rest, fractional through a glide, capped forever on the
// last stop.
func (s *Scroll) FocusPos() float64 {
	if s == nil || len(s.stops) == 0 {
		return 0
	}
	last := float64(len(s.stops) - 1)
	if s.clock <= 0 || last == 0 {
		return 0
	}
	period := s.hold + s.glide
	if period <= 0 || math.IsNaN(period) {
		return last
	}
	i := math.Floor(s.clock / period)
	if i >= last {
		return last
	}
	e := s.clock - i*period
	if e <= s.hold {
		return i
	}
	if s.glide <= 0 {
		return i + 1
	}
	return i + ease((e-s.hold)/s.glide)
}

// FocusStop is the nearest stop ordinal — a glide hands it over at
// its midpoint.
func (s *Scroll) FocusStop() int {
	if s == nil || len(s.stops) == 0 {
		return 0
	}
	i := int(math.Round(s.FocusPos()))
	if i < 0 {
		i = 0
	}
	if last := len(s.stops) - 1; i > last {
		i = last
	}
	return i
}

// Next cuts the current rest short: the glide to the next stop
// begins now. Mid-glide, or on the last stop, there is nothing to
// cut and Next refuses.
func (s *Scroll) Next() bool {
	if s == nil || len(s.stops) < 2 {
		return false
	}
	pos := s.FocusPos()
	i := int(pos)
	if pos != float64(i) || i >= len(s.stops)-1 {
		return false
	}
	s.clock = float64(i)*(s.hold+s.glide) + s.hold
	return true
}

// Render paints the column with the vignette hung around the
// spotlight. Without stops there is nothing to spotlight, so nothing
// paints.
func (s *Scroll) Render() sprite.Sprite {
	if s == nil || !s.staged || s.w < 1 || s.h < 1 || len(s.stops) == 0 {
		return sprite.Sprite{}
	}
	arts := make([]sprite.Sprite, len(s.blocks))
	rows := make([]int, len(s.blocks)+1)
	width := 0
	for i, b := range s.blocks {
		arts[i] = b.Code.Art()
		rows[i+1] = rows[i] + arts[i].Height + 1
		if arts[i].Width > width {
			width = arts[i].Width
		}
	}

	pos := s.FocusPos()
	i := int(math.Floor(pos))
	f := pos - float64(i)
	from := s.stops[i]
	to := from
	if f > 0 && i+1 < len(s.stops) {
		to = s.stops[i+1]
	}
	// The camera rounds — never truncates — so it lands exactly on
	// its stop while its own glide still runs, and the hold never
	// hops the column one more cell.
	cam := rows[from] + int(math.Round(f*float64(rows[to]-rows[from])))
	blockPos := float64(from) + f*float64(to-from)

	anchor := AnchorY(s.h)
	left := (s.w - width) / 2
	if left < 0 {
		left = 0
	}
	stage := sprite.New(s.w, s.h)
	for b := range s.blocks {
		level := vigLevel(float64(b) - blockPos)
		if level > 3 {
			continue
		}
		top := anchor + rows[b] - cam
		art := arts[b]
		for r := 0; r < art.Height; r++ {
			y := top + r
			if y < 0 || y >= s.h {
				continue
			}
			for c := 0; c < art.Width; c++ {
				cell := art.At(r, c)
				cell.FG = code.Dim(cell.FG, level)
				stage.Set(y, left+c, cell)
			}
		}
	}
	return stage
}

// vigLevel rounds a block's distance from the spotlight to its shade
// level. Broken distances are past seeing, never a panic.
func vigLevel(d float64) int {
	d = math.Abs(d)
	if math.IsNaN(d) || d >= 3.5 {
		return 4
	}
	return int(math.Round(d))
}

// ease is the repo's ease-out cubic, clamped to the glide.
func ease(p float64) float64 {
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		return 1
	}
	q := 1 - p
	return 1 - q*q*q
}
