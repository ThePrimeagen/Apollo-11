package shotgun

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

// Size is the atlas slot the eight shotgun frames live in.
const Size = sprite.Size1

// Palette is the gun's materials. fg == bg so a pixel compiles to a
// solid block that FlipH / FlipV can mirror without splitting halves.
var Palette = []sprite.PaletteEntry{
	{ID: ".", Name: "empty", FG: -1, BG: -1},
	{ID: "W", Name: "wood", FG: 94, BG: 94},
	{ID: "S", Name: "steel", FG: 250, BG: 250},
	{ID: "D", Name: "dark", FG: 238, BG: 238},
	{ID: "B", Name: "barrel", FG: 245, BG: 245},
	{ID: "G", Name: "gold", FG: 178, BG: 178},
	{ID: "P", Name: "pump", FG: 137, BG: 137},
}

// grids are the three hand-drawn headings. Each unique row is duplicated
// so CompileGrid emits solid █ cells — the other five headings are
// FlipH / FlipV of these, and a split half-block would not survive a
// vertical mirror. Every row is 32 pixels.
var grids = map[sprite.Heading][]string{
	sprite.E: dup(
		"................................",
		"........SSSSBBBBBBBBBBBBBBBBBB..",
		"...WWWWWSSSSBBBBBBBBBBBBBBBBBBB.",
		"..WWWWWWSSGGPP..................",
		"..WWWWWWDSS.....................",
		"...WWWWWG.......................",
		"....WWW.........................",
		"................................",
	),
	sprite.N: dup(
		"...............BB...............",
		"...............BB...............",
		"...............BB...............",
		"..............SSSS..............",
		"..............SGGS..............",
		".............WWDDWW.............",
		".............WWWWWW.............",
		"..............WWWW..............",
	),
	sprite.NE: dup(
		".........................BB.....",
		".......................BB.......",
		".....................SS.........",
		"..................WWDSG.........",
		"................WWWWW...........",
		"..............WWWWW.............",
		"............WWWW................",
		"..........WWW...................",
	),
}

func dup(rows ...string) []string {
	out := make([]string, 0, len(rows)*2)
	for _, row := range rows {
		out = append(out, row, row)
	}
	return out
}
