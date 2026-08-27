package astro

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

// The scene props, drawn in the same pixel language as the astronaut
// and shipped in the same editable atlas: the supply crate he climbs,
// the flag that rides the pole, and the rover the camera finds at the
// end of the show.
const (
	PropBlock = sprite.Heading("block")
	PropFlag  = sprite.Heading("flag")
	PropRover = sprite.Heading("rover")
)

// Props is every prop frame the atlas carries beside the poses.
var Props = []sprite.Heading{PropBlock, PropFlag, PropRover}

// Prop pixel canvases. One pixel is one terminal column and half a
// terminal row, same as the character.
const (
	BlockPxW = 8
	BlockPxH = 8
	FlagPxW  = 10
	FlagPxH  = 6
	RoverPxW = 20
	RoverPxH = 10
)

// The crate is a dark-rimmed supply box with a bright face plate and
// gold corner bolts — chunky enough to read as a platform. The flag
// is the stars and stripes at ten pixels: a blue canton over
// alternating red and white bands. The rover is the buggy left parked
// in the dark: a gold-foil deck on a grey frame, two fat wheels with
// bright hubs, the dish up front and a whip antenna at the console.
func init() {
	grids[PropBlock] = []string{
		//  01234567
		"DDDDDDDD",
		"DWHHHHWD",
		"DHHWWHHD",
		"DHWWWWHD",
		"DHWWWWHD",
		"DHHWWHHD",
		"DWHHHHWD",
		"DDDDDDDD",
	}
	grids[PropFlag] = []string{
		//  0123456789
		"BBBBRRRRRR",
		"BWBBWWWWWW",
		"BBBBRRRRRR",
		"WWWWWWWWWW",
		"RRRRRRRRRR",
		"WWWWWWWWWW",
	}
	grids[PropRover] = []string{
		//  01234567890123456789
		"..HHH...............",
		"..HH........H.......",
		"...H...WW...H.......",
		"..HHHHWWWHHHH.......",
		".HVVVVVVVVVVVVVVVH..",
		".HHHHHHHHHHHHHHHHH..",
		"...DD..........DD...",
		"..DHHD........DHHD..",
		"..DHHD........DHHD..",
		"...DD..........DD...",
	}
}
