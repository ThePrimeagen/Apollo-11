// Package lander renders the Apollo 11 powered descent as a fixed 40×30
// terminal cell view, sized to fit the gap between exec-tui's timing graphs
// and its DSKY. The LM falls down an altitude scale (square-root scaled so
// the final thousand feet stay readable), rotating with the phase:
// horizontal through the P63 braking burn, pitched over at high gate,
// vertical for the P66 landing, engine off on the surface. A four-glyph
// starfield fills the sky and scrolls right-to-left while flying. Program
// alarms leave persistent markers at the altitude where they fired.
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
	AltFt     float64
	VelFps    float64
	TimeSec   float64
	LandInSec float64 // countdown to touchdown; 0 hides it
	Tick      int     // animation frame counter (plume flicker, starfield)
	Phase     string
	Attitude  Attitude
	Event     string
	Alarms    []Alarm
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
	colStar0 = 238 // far dust
	colStar1 = 244 // spark
	colStar2 = 252 // mid star
	colStar3 = 229 // near star (pale gold)
)

type cell struct {
	ch rune
	fg int
}

// sprites, 13 wide × 5 tall (Sol design reviews, rounds 1-2): the angular
// ascent cabin atop the wider gold-foil descent stage, four splayed legs
// with footpads, a discrete engine bell, an attitude-correct plume, and
// rigid rotation for the braking/pitched attitudes. '~' marks flame cells.
var sprites = map[Attitude][5]string{
	Horizontal: {
		"   ▁╲   ╱▁   ",
		"    ╲▟▓▓▙▛◣▖ ",
		"   ◢▟▓▓▙░██▜▙",
		"  ~~◥▜▓▛╲▝◤  ",
		"~~ ▁╱   ╲▁   ",
	},
	Tilted: {
		"        ▗▛◣▖ ",
		"▁ ╲   ▟░╲▜▙  ",
		"  ╲◢▟▓██▓▙   ",
		"   ◥▜▓▓▛  ╲  ",
		" ~~~▜▙    ╲▁ ",
	},
	Vertical: {
		"    ▗▛◣▖     ",
		"   ▟░◣╲▜▙    ",
		"  ▟▓████▓▙   ",
		"╱ ◢▔▔▟▄▙▔▔◣ ╲",
		"▁ ▁  ~~~  ▁ ▁",
	},
	Landed: {
		"    ▗▛◣▖     ",
		"   ▟░◣╲▜▙    ",
		"  ▟▓████▓▙   ",
		"╱ ◢▔▔▟▄▙▔▔◣ ╲",
		"▁ ▁       ▁ ▁",
	},
}

// colorMasks paint the craft's materials per cell (Sol round-1 checklist):
// S silver ascent hull/struts, G gold kapton foil, W dark windows, E steel
// engine/nozzle, P plume, '.' empty.
var colorMasks = map[Attitude][5]string{
	Horizontal: {
		"...SS...SS...",
		"....SGGGGSSS.",
		"...EGGGGWSSSS",
		"..PPEGGGWSS..",
		"PP.SS...SS...",
	},
	Tilted: {
		"........SSSS.",
		"S.S...SWWSS..",
		"..SGGGGGGG...",
		"...EEGGG..S..",
		".PPPEE....SS.",
	},
	Vertical: {
		"....SSSS.....",
		"...SWSWSS....",
		"..GGGGGGGG...",
		"S.SSSEEESSS.S",
		"S.S..PPP..S.S",
	},
	Landed: {
		"....SSSS.....",
		"...SWSWSS....",
		"..GGGGGGGG...",
		"S.SSSEEESSS.S",
		"S.S.......S.S",
	},
}

// materialColor maps mask characters to xterm-256 indices.
func materialColor(mask byte) int {
	switch mask {
	case 'S':
		return 252 // silver-gray hull and struts
	case 'G':
		return 178 // muted gold descent-stage foil
	case 'W':
		return 24 // dark blue-black windows
	case 'E':
		return 245 // steel engine bell
	case 'P':
		return colFlame
	default:
		return colBody
	}
}

// Four one-cell background stars, far/dim → near/bright. They are just
// dots — no per-glyph twinkle. The field flying past is the animation.
var starGlyphs = [4]rune{'·', '˚', '*', '✦'}
var starColors = [4]int{colStar0, colStar1, colStar2, colStar3}

// starDelay is ticks per cell of travel. Far stars crawl; near stars
// streak — parallax, as if the LM were flying through them.
var starDelay = [4]int{8, 4, 2, 1}

type bgStar struct{ row, col, kind int }

// rest positions at tick 0. col is a sky column in [1, Width).
var starfield = []bgStar{
	{2, 5, 0}, {2, 28, 0}, {4, 17, 0}, {5, 36, 0},
	{8, 9, 0}, {9, 31, 0}, {11, 3, 0}, {13, 22, 0},
	{16, 6, 0}, {16, 28, 0}, {18, 13, 0}, {20, 34, 0},
	{22, 8, 0}, {24, 19, 0}, {26, 37, 0},
	{3, 12, 1}, {6, 24, 1}, {7, 4, 1}, {10, 18, 1},
	{14, 32, 1}, {16, 14, 1}, {19, 7, 1}, {21, 26, 1},
	{25, 11, 1},
	{3, 33, 2}, {5, 8, 2}, {8, 20, 2}, {12, 15, 2},
	{16, 21, 2}, {17, 38, 2}, {20, 4, 2}, {23, 29, 2},
	{26, 16, 2},
	{4, 30, 3}, {7, 16, 3}, {11, 35, 3}, {16, 10, 3},
	{16, 33, 3}, {18, 24, 3}, {22, 17, 3}, {25, 5, 3},
}

// wrapSky keeps a star in the sky columns [1, Width), wrapping 1 ← Width-1
// so a fly-by off the left re-enters from the right. Column 0 is the axis.
func wrapSky(col int) int {
	const skyW = Width - 1
	x := col - 1
	x = ((x % skyW) + skyW) % skyW
	return x + 1
}

func paintStars(grid [][]cell, s State) {
	tick := s.Tick
	if s.Attitude == Landed {
		tick = 0
	}
	for _, st := range starfield {
		if st.row < topRow || st.row >= surfaceRow {
			continue
		}
		delay := starDelay[st.kind]
		if delay < 1 {
			delay = 1
		}
		col := wrapSky(st.col - tick/delay)
		if col <= 0 || col >= Width || st.row < 0 || st.row >= Height {
			continue
		}
		if grid[st.row][col].ch != ' ' {
			continue
		}
		grid[st.row][col] = cell{starGlyphs[st.kind], starColors[st.kind]}
	}
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
	top := topRow + 4 // room for the 5-row sprite under the header
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

	// header: phase left, time right; telemetry + touchdown countdown below
	put(0, 0, s.Phase, colPhase)
	tstr := fmt.Sprintf("T+%.0fs", s.TimeSec)
	put(0, Width-len(tstr), tstr, colDim)
	put(1, 0, fmt.Sprintf("ALT %6.0fft", s.AltFt), colGreen)
	if s.LandInSec > 0 {
		cd := fmt.Sprintf("▼ %.0fs", s.LandInSec)
		put(1, (Width-len([]rune(cd)))/2, cd, colPhase)
	}
	vstr := fmt.Sprintf("VEL %5dft/s", int(s.VelFps))
	put(1, Width-len(vstr), vstr, colGreen)

	// altitude axis
	for r := topRow; r < surfaceRow; r++ {
		put(r, 0, "│", colScale)
	}
	put(topRow, 0, "┬", colScale)

	// alarm markers, at their own altitudes, hugging the right edge; when
	// two altitudes round to the same row, nudge downward so every alarm
	// stays visible
	usedRows := map[int]bool{}
	for _, a := range s.Alarms {
		row := rowFor(a.AltFt)
		for usedRows[row] && row < surfaceRow-1 {
			row++
		}
		usedRows[row] = true
		put(row, Width-7, "◄ "+a.Code, colAlarm)
	}

	// the LM: the sprite's bottom row sits at the altitude row
	sp := sprites[s.Attitude]
	base := rowFor(s.AltFt)
	if base > surfaceRow-1 {
		base = surfaceRow - 1
	}
	mask := colorMasks[s.Attitude]
	for i := range sp {
		row := base - (len(sp) - 1) + i
		maskRow := []rune(mask[i])
		for j, r := range []rune(sp[i]) {
			if r == ' ' {
				continue
			}
			col := 12 + j
			if row < 0 || row >= Height || col >= Width {
				continue
			}
			color := materialColor(byte(maskRow[j]))
			ch := r
			if r == '~' {
				// flicker the plume with the animation tick so the burn
				// reads alive between the cell-quantized row steps
				if (s.Tick+i+j)%2 == 0 {
					ch = '≈'
				} else {
					ch = '~'
				}
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

	// starfield fills leftover sky cells last so it never covers the craft,
	// the axis, the alarms, or the ground
	paintStars(grid, s)

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
