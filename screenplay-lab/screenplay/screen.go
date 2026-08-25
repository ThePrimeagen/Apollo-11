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
	return &Screen{}
}

// Size is the screen dimensions in terminal cells.
func (s *Screen) Size() (w, h int) {
	return 0, 0
}

// Resize follows the terminal to w×h and flags the change so the next
// render knows everything must be repainted. Same-size calls are no-ops.
func (s *Screen) Resize(w, h int) {
}

// Resized reports whether this frame is the first after a size change.
// The screenplay clears the flag once the frame has rendered.
func (s *Screen) Resized() bool {
	return false
}

// Clear blanks every cell.
func (s *Screen) Clear() {
}

// Put writes one cell: a rune plus the lip gloss style to draw it with.
// Out of bounds is ignored.
func (s *Screen) Put(x, y int, ch rune, st uv.Style) {
}

// Cell reads a cell back, or nil out of bounds.
func (s *Screen) Cell(x, y int) *uv.Cell {
	return nil
}

// Canvas is the underlying lip gloss canvas, for callers that want to
// compose layers or drive it directly.
func (s *Screen) Canvas() *lipgloss.Canvas {
	return nil
}

// Render is the styled string of the whole grid.
func (s *Screen) Render() string {
	return ""
}
