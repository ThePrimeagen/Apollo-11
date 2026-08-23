// Package button renders a tactile, 1960s-cockpit-style toggle switch as a
// fixed-size terminal cell grid — the control chosen for the exec-tui panel.
//
// Off: the lever is held down in a ~50%-darker red under a dark slot.
// Flicked on: the lever jumps to the top, lit in two orange tones (bright
// tip, orange body), leaving a dim glow in the slot it vacated. Focusing a
// switch lifts its ░ frame brightness.
//
// Output is raw ANSI 256-color — no terminal profile detection, so captures
// and recordings always keep their color. The footprint is constant in every
// state: flicking a switch never shifts the layout around it.
package button

import "fmt"

// The cockpit palette (xterm-256 indices).
const (
	ColorBezel      = 238 // faint gray frame
	ColorBezelFocus = 251 // the frame "lifts" when highlighted
	ColorOffDim     = 52  // dark red slot above the resting lever
	ColorOn         = 208 // lit orange lever body
	ColorOnTip      = 214 // bright lever tip
	ColorOnGlow     = 130 // dim glow left in the vacated slot
	ColorLeverOff   = 88  // held-down lever: ~50% darker red (was dull pink 131)
)

// Switch is one cockpit toggle. The zero value is unusable; use NewSwitch.
type Switch struct {
	Label   string
	On      bool
	Focused bool
}

// NewSwitch returns an off, unfocused switch.
func NewSwitch(label string) Switch { return Switch{Label: label} }

// Toggle flicks the switch.
func (b *Switch) Toggle() { b.On = !b.On }

// Size reports the fixed footprint (cells wide, rows tall).
func (b Switch) Size() (w, h int) { return 5, 4 }

// cell is one grid position: a rune with a foreground palette index.
type cell struct {
	ch rune
	fg int
}

func rowOf(n int, x cell) []cell {
	out := make([]cell, n)
	for i := range out {
		out[i] = x
	}
	return out
}

// Render returns the ANSI grid, rows joined by newlines. Pure.
func (b Switch) Render() string {
	bez := ColorBezel
	if b.Focused {
		bez = ColorBezelFocus
	}
	frame := func(inner []cell) []cell {
		return append(append([]cell{{'░', bez}}, inner...), cell{'░', bez})
	}
	var grid [][]cell
	if b.On {
		grid = [][]cell{
			frame(rowOf(3, cell{'█', ColorOnTip})),
			frame(rowOf(3, cell{'█', ColorOn})),
			frame(rowOf(3, cell{'▒', ColorOnGlow})),
			frame(rowOf(3, cell{'▒', ColorOnGlow})),
		}
	} else {
		grid = [][]cell{
			frame(rowOf(3, cell{'▒', ColorOffDim})),
			frame(rowOf(3, cell{'▒', ColorOffDim})),
			frame(rowOf(3, cell{'█', ColorLeverOff})),
			frame(rowOf(3, cell{'█', ColorLeverOff})),
		}
	}
	out := ""
	for i, row := range grid {
		if i > 0 {
			out += "\n"
		}
		for _, cl := range row {
			out += fmt.Sprintf("\x1b[38;5;%dm%c", cl.fg, cl.ch)
		}
		out += "\x1b[0m"
	}
	return out
}
