// Package button renders tactile, 1960s-cockpit-style toggle controls as
// fixed-size terminal cell grids. Four styles:
//
//	Panel    — 6×3 face (≈ square on screen) inside a half-cursor bezel:
//	           ▄/▀ half-blocks top and bottom, barely-there ░ shade sides.
//	           Off: half-filled dark red (▒). On: ¾-filled orange ring (▓)
//	           around a hot orange-white center.
//	Half­Cell — the two-colors-in-one-cell trick: the top row is ▄ with the
//	           face color as foreground on the bezel gray as background.
//	Protrude — sticks up half a cell when off (the cap hides the bezel bar
//	           behind it); pressing it visibly sinks the cap half a cell,
//	           reveals the bezel, and lights the face.
//	Switch   — a cockpit toggle: lever down and dull red when off; flipped
//	           up and lit in two orange tones (bright tip, orange body) with
//	           a dim glow left in the vacated slot.
//
// Output is raw ANSI 256-color — no terminal profile detection, so captures
// and recordings always keep their color. Every style renders a constant
// footprint in every state: pressing a button never shifts the layout.
package button

import "fmt"

// Style selects the control's shape and press behavior.
type Style int

const (
	Panel Style = iota
	HalfCell
	Protrude
	Switch
)

// The shared cockpit palette (xterm-256 indices).
const (
	ColorBezel      = 238 // faint gray surround
	ColorBezelFocus = 251 // the surround "lifts" when highlighted
	ColorOffFace    = 88  // dark red, waiting
	ColorOffDim     = 52  // darker red (switch slot)
	ColorOn         = 208 // lit orange
	ColorOnHot      = 223 // orangey-white hot center
	ColorOnShade    = 166 // darker orange (pressed-in lower face)
	ColorOnTip      = 214 // bright lever tip
	ColorOnGlow     = 130 // dim glow in a vacated slot
	ColorLeverOff   = 88  // held-down lever: ~50% darker red (was dull pink 131)
)

// Button is one control. The zero value is unusable; use New.
type Button struct {
	Label   string
	Style   Style
	On      bool
	Focused bool
}

// New returns an off, unfocused button.
func New(label string, style Style) Button {
	return Button{Label: label, Style: style}
}

// Toggle presses the button.
func (b *Button) Toggle() { b.On = !b.On }

// Size reports the fixed footprint (cells wide, rows tall).
func (b Button) Size() (w, h int) {
	switch b.Style {
	case Panel:
		return 8, 5
	case HalfCell:
		return 6, 3
	case Protrude:
		return 6, 3
	default: // Switch
		return 5, 4
	}
}

// cell is one grid position: a rune with a foreground and optional
// background palette index (bg < 0 means the terminal default).
type cell struct {
	ch rune
	fg int
	bg int
}

func c(ch rune, fg int) cell      { return cell{ch, fg, -1} }
func c2(ch rune, fg, bg int) cell { return cell{ch, fg, bg} }
func rowOf(n int, x cell) []cell {
	out := make([]cell, n)
	for i := range out {
		out[i] = x
	}
	return out
}

// Render returns the ANSI grid, rows joined by newlines. Pure.
func (b Button) Render() string {
	bez := ColorBezel
	if b.Focused {
		bez = ColorBezelFocus
	}
	var grid [][]cell
	switch b.Style {
	case Panel:
		grid = b.panel(bez)
	case HalfCell:
		grid = b.halfCell(bez)
	case Protrude:
		grid = b.protrude(bez)
	default:
		grid = b.toggle(bez)
	}
	out := ""
	for i, row := range grid {
		if i > 0 {
			out += "\n"
		}
		for _, cl := range row {
			if cl.bg >= 0 {
				out += fmt.Sprintf("\x1b[38;5;%d;48;5;%dm%c", cl.fg, cl.bg, cl.ch)
			} else {
				out += fmt.Sprintf("\x1b[38;5;%dm%c", cl.fg, cl.ch)
			}
		}
		out += "\x1b[0m"
	}
	return out
}

func frame(inner []cell, bez int) []cell {
	return append(append([]cell{c('░', bez)}, inner...), c('░', bez))
}

func (b Button) panel(bez int) [][]cell {
	var face [3][]cell
	if b.On {
		ring := c('▓', ColorOn)
		face[0] = rowOf(6, ring)
		face[1] = []cell{ring, c('█', ColorOn), c('█', ColorOnHot), c('█', ColorOnHot), c('█', ColorOn), ring}
		face[2] = rowOf(6, ring)
	} else {
		for i := range face {
			face[i] = rowOf(6, c('▒', ColorOffFace))
		}
	}
	return [][]cell{
		frame(rowOf(6, c('▄', bez)), bez),
		frame(face[0], bez),
		frame(face[1], bez),
		frame(face[2], bez),
		frame(rowOf(6, c('▀', bez)), bez),
	}
}

func (b Button) halfCell(bez int) [][]cell {
	faceTop, body := ColorOffFace, []cell{c('█', ColorOffFace), c('█', ColorOffFace), c('█', ColorOffFace), c('█', ColorOffFace)}
	if b.On {
		faceTop = ColorOn
		body = []cell{c('█', ColorOn), c('█', ColorOnHot), c('█', ColorOnHot), c('█', ColorOn)}
	}
	return [][]cell{
		frame(rowOf(4, c2('▄', faceTop, bez)), bez), // two colors, one cell
		frame(body, bez),
		frame(rowOf(4, c('▀', bez)), bez),
	}
}

func (b Button) protrude(bez int) [][]cell {
	gap := cell{' ', 0, -1}
	if b.On {
		// pressed in: the cap sank half a cell — the bezel bar shows, the
		// face lights, the lower row shades darker for depth.
		return [][]cell{
			{gap, c('▄', bez), c('▄', bez), c('▄', bez), c('▄', bez), gap},
			frame(rowOf(4, c('█', ColorOn)), bez),
			frame(rowOf(4, c('▓', ColorOnShade)), bez),
		}
	}
	// raised: the dark red cap sticks up and hides the bezel bar behind it.
	return [][]cell{
		{gap, c('▄', ColorOffFace), c('▄', ColorOffFace), c('▄', ColorOffFace), c('▄', ColorOffFace), gap},
		frame(rowOf(4, c('█', ColorOffFace)), bez),
		frame(rowOf(4, c('█', ColorOffFace)), bez),
	}
}

func (b Button) toggle(bez int) [][]cell {
	if b.On {
		return [][]cell{
			frame(rowOf(3, c('█', ColorOnTip)), bez),
			frame(rowOf(3, c('█', ColorOn)), bez),
			frame(rowOf(3, c('▒', ColorOnGlow)), bez),
			frame(rowOf(3, c('▒', ColorOnGlow)), bez),
		}
	}
	return [][]cell{
		frame(rowOf(3, c('▒', ColorOffDim)), bez),
		frame(rowOf(3, c('▒', ColorOffDim)), bez),
		frame(rowOf(3, c('█', ColorLeverOff)), bez),
		frame(rowOf(3, c('█', ColorLeverOff)), bez),
	}
}
