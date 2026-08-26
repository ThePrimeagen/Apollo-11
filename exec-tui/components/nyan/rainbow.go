package nyan

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/fire"
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Classic nyan rainbow, red at the top through violet at the bottom.
var rainbowPalette = []struct{ fg, bg int }{
	{196, 88},  // red
	{208, 130}, // orange
	{226, 178}, // yellow
	{46, 22},   // green
	{39, 19},   // blue
	{129, 53},  // violet
}

const rainbowBands = 6

// paintTrail maps the engine's occupancy onto rainbow flame cells:
// fire.Heat picks the glyph (wispy braille → solid), band width and
// the cell's height pick the color. clock slides the classic 2-column
// nyan wave along the trail.
func paintTrail(eng *particle.Engine, bandWidth, clock float64) sprite.Sprite {
	cols, rows := trailBox(eng)
	sp := sprite.New(cols, rows)
	if eng == nil {
		return sp
	}
	occ := eng.Occupancy()
	dir := eng.Cfg.Direction
	origin := eng.Cfg.Origin
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			h := fire.Heat(occ, particle.Cell{Col: c, Row: r}, dir)
			if h <= 0 {
				continue
			}
			band := rainbowBand(r, c, origin, bandWidth, clock)
			sp.Set(r, c, rainbowCell(h, band))
		}
	}
	return sp
}

func trailBox(eng *particle.Engine) (cols, rows int) {
	if eng == nil {
		return 1, 1
	}
	cols = int(math.Ceil(eng.Cfg.Width - 1e-9))
	rows = int(math.Ceil((eng.Cfg.Height - 1e-9) / particle.CellHeightUnits))
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	return cols, rows
}

func rainbowBand(row, col int, origin particle.Vec2, bandWidth, clock float64) int {
	if bandWidth <= 0 {
		bandWidth = 2
	}
	y := float64(row)*particle.CellHeightUnits + particle.CellHeightUnits/2
	wave := 0.0
	if ((col+int(clock*8))/2)%2 == 1 {
		wave = particle.CellHeightUnits
	}
	rel := y - origin.Y + wave
	half := float64(rainbowBands) * bandWidth / 2
	idx := int(math.Floor((rel + half) / bandWidth))
	if idx < 0 {
		return 0
	}
	if idx >= rainbowBands {
		return rainbowBands - 1
	}
	return idx
}

func rainbowCell(heat, band int) sprite.Cell {
	if band < 0 {
		band = 0
	}
	if band >= len(rainbowPalette) {
		band = len(rainbowPalette) - 1
	}
	base := fire.Style(heat)
	pal := rainbowPalette[band]
	cell := sprite.Cell{Ch: base.Ch, FG: pal.fg, BG: -1}
	if base.BG >= 0 {
		cell.BG = pal.bg
	}
	return cell
}
