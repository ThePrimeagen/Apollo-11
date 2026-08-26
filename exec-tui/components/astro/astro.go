// Package astro is an original NES-styled astronaut built on the
// classic 16×16 side-scroller envelope: a big-helmet hero with a gold
// visor and a life-support pack, drawn as flat-color pixel grids and
// compiled to terminal half-blocks. The atlas ships as editable JSON
// in assets/astronaut.json, in the same format as the lunar module art.
package astro

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
	// GripCol is the pixel column the pole hands close around — the
	// scene parks the sprite so this column rides the flagpole.
	GripCol = 13
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

// Palette is the astronaut's outfit as named colors. Every entry keeps
// fg == bg so a pixel resolves to the same color from either mask of
// the atlas file format.
var Palette = []sprite.PaletteEntry{
	{ID: ".", Name: "empty", FG: -1, BG: -1},
	{ID: "W", Name: "suit", FG: 255, BG: 255},
	{ID: "H", Name: "shade", FG: 250, BG: 250},
	{ID: "V", Name: "visor", FG: 220, BG: 220},
	{ID: "D", Name: "dark", FG: 240, BG: 240},
	{ID: "R", Name: "accent", FG: 160, BG: 160},
}

// BuildAtlas compiles every pixel grid into one atlas sharing the
// astronaut palette.
func BuildAtlas() (*sprite.Atlas, error) {
	a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), Palette...)}
	for _, pose := range Poses {
		grid, ok := grids[pose]
		if !ok {
			return nil, fmt.Errorf("astro: pose %q has no pixel grid", pose)
		}
		sp, err := CompileGrid(grid)
		if err != nil {
			return nil, fmt.Errorf("astro: pose %q: %w", pose, err)
		}
		a.SetFrame(Size, pose, sp)
	}
	return a, nil
}

const atlasFile = "astronaut.json"

// FindAtlas locates the shipped assets/astronaut.json: a nearby
// assets/ already holding it, then any assets/ folder on the way up
// from here or the working directory — so the path also works before
// the file exists, for the generator's first write.
func FindAtlas() string {
	seen := map[string]bool{}
	var cands []string
	addFrom := func(start string) {
		dir := start
		for i := 0; i < 8; i++ {
			cand := filepath.Join(dir, "assets")
			if !seen[cand] {
				seen[cand] = true
				cands = append(cands, cand)
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				return
			}
			dir = parent
		}
	}
	if wd, err := os.Getwd(); err == nil {
		addFrom(wd)
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		addFrom(filepath.Dir(file))
	}
	var existingDir string
	for _, d := range cands {
		path := filepath.Join(d, atlasFile)
		if _, err := os.Stat(path); err == nil {
			return path
		}
		if existingDir == "" {
			if st, err := os.Stat(d); err == nil && st.IsDir() {
				existingDir = d
			}
		}
	}
	if existingDir != "" {
		return filepath.Join(existingDir, atlasFile)
	}
	return filepath.Join("assets", atlasFile)
}

// Load reads the shipped atlas.
func Load() (*sprite.Atlas, error) {
	return LoadPath(FindAtlas())
}

// LoadPath reads an astronaut atlas from path. A missing or corrupt
// file is an error, never a blank astronaut.
func LoadPath(path string) (*sprite.Atlas, error) {
	a, err := sprite.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("astro: %w", err)
	}
	return a, nil
}
