// Package screenplay is the theater. A Screen is the shared cell grid a
// frame is composed on — a lip gloss canvas, one cell of content and
// style per terminal cell. A Scene is anything that can start, update,
// render onto that screen, and stop. A Screenplay runs scenes in order:
// every frame it updates then renders the one now playing, and a cut
// stops the old scene before the next one starts. The package knows
// nothing about landers, stars, or fonts.
package screenplay

import (
	lipgloss "charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// Screen is the render target every scene is handed a pointer to: the
// lip gloss cell canvas plus the frame bookkeeping around it. Cells
// carry everything lip gloss needs to draw them — content, foreground,
// background, attributes.
type Screen struct {
	canvas        *lipgloss.Canvas
	width, height int
	resized       bool
}

// NewScreen allocates a blank w×h screen. Negative dimensions clamp to
// zero: a screen that takes nothing and renders empty.
func NewScreen(w, h int) *Screen {
	w, h = max(w, 0), max(h, 0)
	return &Screen{
		canvas: lipgloss.NewCanvas(w, h),
		width:  w,
		height: h,
	}
}

// Size is the screen dimensions in terminal cells.
func (s *Screen) Size() (w, h int) {
	if s == nil {
		return 0, 0
	}
	return s.width, s.height
}

// Resize follows the terminal to w×h and flags the change so the next
// render knows everything must be repainted. Same-size calls are no-ops.
func (s *Screen) Resize(w, h int) {
	if s == nil {
		return
	}
	w, h = max(w, 0), max(h, 0)
	if w == s.width && h == s.height {
		return
	}
	s.canvas.Resize(w, h)
	s.width, s.height = w, h
	s.resized = true
}

// Resized reports whether this frame is the first after a size change.
// The screenplay clears the flag once the frame has rendered.
func (s *Screen) Resized() bool {
	return s != nil && s.resized
}

// Clear blanks every cell.
func (s *Screen) Clear() {
	if s == nil {
		return
	}
	s.canvas.Clear()
}

// Put writes one cell: a rune plus the lip gloss style to draw it with.
// Out of bounds is ignored.
func (s *Screen) Put(x, y int, ch rune, st uv.Style) {
	if s == nil || x < 0 || y < 0 || x >= s.width || y >= s.height {
		return
	}
	s.canvas.SetCell(x, y, &uv.Cell{Content: string(ch), Style: st, Width: 1})
}

// Cell reads a cell back, or nil out of bounds.
func (s *Screen) Cell(x, y int) *uv.Cell {
	if s == nil || x < 0 || y < 0 || x >= s.width || y >= s.height {
		return nil
	}
	return s.canvas.CellAt(x, y)
}

// Canvas is the underlying lip gloss canvas, for callers that want to
// compose layers or drive it directly.
func (s *Screen) Canvas() *lipgloss.Canvas {
	if s == nil {
		return nil
	}
	return s.canvas
}

// Render is the styled string of the whole grid.
func (s *Screen) Render() string {
	if s == nil || s.width < 1 || s.height < 1 {
		return ""
	}
	return s.canvas.Render()
}
