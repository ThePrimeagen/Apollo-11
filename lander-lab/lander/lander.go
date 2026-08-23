// Package lander renders the Apollo 11 powered descent as a fixed 40×30
// terminal cell view, sized to fit the gap between exec-tui's timing graphs
// and its DSKY. The LM falls down an altitude scale (square-root scaled so
// the final thousand feet stay readable), rotating with the phase:
// horizontal through the P63 braking burn, pitched over at high gate,
// vertical for the P66 landing, engine off on the surface. Program alarms
// leave persistent markers at the altitude where they fired.
//
// Output is raw ANSI 256-color; Render is pure and the footprint constant.
package lander

import (
	"fmt"
	"math"
	"strings"
)

// Fixed footprint.
const (
	Width  = 40
	Height = 30
)

// The scale ceiling: PDI altitude.
const maxAltFt = 49971

// Attitude selects the LM silhouette.
type Attitude int

const (
	Horizontal Attitude = iota // P63: feet first, engine braking
	Tilted                     // P64: pitched over, surface in view
	Vertical                   // P66: upright on the plume
	Landed                     // engine off, on the surface
)

// Alarm is one executive-overflow marker.
type Alarm struct {
	Code  string
	AltFt float64
}

// State is one instant of the descent.
type State struct {
	AltFt    float64
	VelFps   float64
	TimeSec  float64
	Phase    string
	Attitude Attitude
	Event    string
	Alarms   []Alarm
}

// Palette (xterm-256).
const (
	colPhase = 214 // amber phase banner
	colDim   = 240
	colGreen = 48  // telemetry
	colBody  = 252 // LM hull
	colFlame = 208 // engine plume
	colMoon  = 245 // regolith
	colAlarm = 196 // alarm markers
	colScale = 238 // altitude axis
)

type cell struct {
	ch rune
	fg int
}

// sprites, 7 wide × 3 tall; '~' marks flame cells (colored separately).
var sprites = map[Attitude][3]string{
	Horizontal: {"  ▗██▖ ", "~~████▌", "  ▝██▘ "},
	Tilted:     {"  ▗██▖ ", " ▟██▛  ", "~▞▘    "},
	Vertical:   {"  ▄██▄ ", " ▐██▌  ", " ▞ ~ ▚ "},
	Landed:     {"  ▄██▄ ", " ▐██▌  ", " ▞▔▔▔▚ "},
}

// surface is the lunar terrain strip, tiled across the width.
const surface = "▂▁▃▂▁▄▃▂▁▂▃▁▂▄▂▁▃▂▄▁"

// descent rows span [topRow, surfaceRow).
const (
	topRow     = 2
	surfaceRow = Height - 2
)

// rowFor maps an altitude to the sprite's base row on a square-root scale:
// PDI sits just under the header (leaving room for the 3-row sprite) and
// zero altitude sits directly on the surface.
func rowFor(alt float64) int {
	if alt < 0 {
		alt = 0
	}
	if alt > maxAltFt {
		alt = maxAltFt
	}
	frac := 1 - math.Sqrt(alt/maxAltFt)
	top := topRow + 2
	span := float64(surfaceRow - 1 - top)
	return top + int(frac*span+0.5)
}

func fg(n int) string { return fmt.Sprintf("\x1b[38;5;%dm", n) }

const reset = "\x1b[0m"

// Render draws one instant of the descent.
func Render(s State) string {
	grid := make([][]cell, Height)
	for i := range grid {
		grid[i] = make([]cell, Width)
		for j := range grid[i] {
			grid[i][j] = cell{' ', -1}
		}
	}
	put := func(row, col int, text string, color int) {
		if row < 0 || row >= Height {
			return
		}
		for i, r := range []rune(text) {
			c := col + i
			if c < 0 || c >= Width {
				continue
			}
			grid[row][c] = cell{r, color}
		}
	}

	// header: phase left, time right; telemetry below
	put(0, 0, s.Phase, colPhase)
	tstr := fmt.Sprintf("T+%.0fs", s.TimeSec)
	put(0, Width-len(tstr), tstr, colDim)
	put(1, 0, fmt.Sprintf("ALT %6.0fft", s.AltFt), colGreen)
	vstr := fmt.Sprintf("VEL %5dft/s", int(s.VelFps))
	put(1, Width-len(vstr), vstr, colGreen)

	// altitude axis
	for r := topRow; r < surfaceRow; r++ {
		put(r, 0, "│", colScale)
	}
	put(topRow, 0, "┬", colScale)

	// alarm markers, at their own altitudes, hugging the right edge
	for _, a := range s.Alarms {
		put(rowFor(a.AltFt), Width-7, "◄ "+a.Code, colAlarm)
	}

	// the LM: sprite bottom row sits at the altitude row
	sp := sprites[s.Attitude]
	base := rowFor(s.AltFt)
	if base > surfaceRow-1 {
		base = surfaceRow - 1
	}
	for i := 0; i < 3; i++ {
		row := base - 2 + i
		for j, r := range []rune(sp[i]) {
			if r == ' ' {
				continue
			}
			col := 13 + j
			if row < 0 || row >= Height || col >= Width {
				continue
			}
			color := colBody
			ch := r
			if r == '~' {
				ch = '≈'
				color = colFlame
			}
			grid[row][col] = cell{ch, color}
		}
	}

	// the moon
	tiled := strings.Repeat(surface, Width/len([]rune(surface))+1)
	put(surfaceRow, 0, string([]rune(tiled)[:Width]), colMoon)

	// event caption
	ev := s.Event
	if len(ev) > Width {
		ev = ev[:Width]
	}
	put(Height-1, 0, ev, colDim)

	// serialize
	var b strings.Builder
	for i, row := range grid {
		if i > 0 {
			b.WriteString("\n")
		}
		cur := -2
		for _, c := range row {
			if c.fg != cur {
				if c.fg < 0 {
					b.WriteString(reset)
				} else {
					b.WriteString(fg(c.fg))
				}
				cur = c.fg
			}
			b.WriteRune(c.ch)
		}
		b.WriteString(reset)
	}
	return b.String()
}
