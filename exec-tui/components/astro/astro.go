// Package astro is an original NES-styled astronaut built on the
// classic 16×16 side-scroller envelope: a big-helmet hero with a gold
// visor and a life-support pack, drawn as flat-color pixel grids and
// compiled to terminal half-blocks. The atlas ships as editable JSON
// in assets/astronaut.json, in the same format as the lunar module art.
package astro

import (
	"fmt"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// PxW/PxH is the pixel canvas every pose is drawn on.
	PxW = 16
	PxH = 16
	// Cols/Rows is the compiled terminal footprint: one cell per pixel
	// column, two pixel rows per half-block cell row.
	Cols = PxW
	Rows = PxH / 2
	// GripCol is the pixel column the pole hands reach for — the scene
	// parks the sprite so this column rides the flagpole.
	GripCol = 12
)

// Size is the atlas size slot the astronaut frames live in.
const Size = sprite.Size1

// The seven poses: standing, the three-frame run cycle, the jump, and
// the two alternating pole grips.
const (
	PoseStand = sprite.Heading("stand")
	PoseRun1  = sprite.Heading("run1")
	PoseRun2  = sprite.Heading("run2")
	PoseRun3  = sprite.Heading("run3")
	PoseJump  = sprite.Heading("jump")
	PosePole1 = sprite.Heading("pole1")
	PosePole2 = sprite.Heading("pole2")
)

// Poses is every frame the atlas must carry.
var Poses = []sprite.Heading{PoseStand, PoseRun1, PoseRun2, PoseRun3, PoseJump, PosePole1, PosePole2}

// Palette is the astronaut's outfit as named colors.
var Palette []sprite.PaletteEntry

// BuildAtlas compiles every pixel grid into one atlas.
func BuildAtlas() (*sprite.Atlas, error) {
	return nil, fmt.Errorf("astro: BuildAtlas not implemented")
}

// FindAtlas locates the shipped assets/astronaut.json path.
func FindAtlas() string { return "" }

// Load reads the shipped atlas.
func Load() (*sprite.Atlas, error) {
	return nil, fmt.Errorf("astro: Load not implemented")
}

// LoadPath reads an astronaut atlas from path.
func LoadPath(path string) (*sprite.Atlas, error) {
	return nil, fmt.Errorf("astro: LoadPath not implemented")
}
