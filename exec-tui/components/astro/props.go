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
