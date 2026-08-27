package astro

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// WriteAtlasFile compiles the pixel grids and writes the editable
// atlas JSON to path.
func WriteAtlasFile(path string) error {
	a, err := BuildAtlas()
	if err != nil {
		return err
	}
	return a.WriteFile(path)
}

// xtermRGB is the handful of xterm-256 indexes the palette uses,
// mapped to their real screen colors for the PNG dumps.
func xtermRGB(n int) color.RGBA {
	switch {
	case n >= 232 && n <= 255: // grey ramp
		v := uint8(8 + 10*(n-232))
		return color.RGBA{v, v, v, 255}
	case n >= 16 && n <= 231: // 6×6×6 cube
		levels := []uint8{0, 95, 135, 175, 215, 255}
		i := n - 16
		return color.RGBA{levels[i/36], levels[(i/6)%6], levels[i%6], 255}
	default:
		return color.RGBA{0, 0, 0, 255}
	}
}

// pngBackground is a deep space blue so the white suit reads on the
// review sheet the way it does on a dark terminal.
var pngBackground = color.RGBA{12, 12, 28, 255}

// drawGrid paints one pixel grid into img with its top-left art pixel
// at (ox, oy) art coordinates, magnified by scale.
func drawGrid(img *image.RGBA, grid []string, ox, oy, scale int) error {
	for r, row := range grid {
		for c, px := range []rune(row) {
			n, err := pixelColor(px)
			if err != nil {
				return err
			}
			if n < 0 {
				continue
			}
			rgba := xtermRGB(n)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					img.Set((ox+c)*scale+dx, (oy+r)*scale+dy, rgba)
				}
			}
		}
	}
	return nil
}

func newSheet(wPx, hPx, scale int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, wPx*scale, hPx*scale))
	for y := 0; y < hPx*scale; y++ {
		for x := 0; x < wPx*scale; x++ {
			img.Set(x, y, pngBackground)
		}
	}
	return img
}

func savePNG(path string, img *image.RGBA) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}

// WritePNGs dumps a magnified PNG of every pose, a run-cycle strip,
// and a full sheet into dir for art review outside a terminal.
func WritePNGs(dir string, scale int) error {
	if scale < 1 {
		return fmt.Errorf("astro: scale %d draws nothing — need at least 1", scale)
	}
	st, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("astro: png dir: %w", err)
	}
	if !st.IsDir() {
		return fmt.Errorf("astro: png dir %s is not a directory", dir)
	}
	for _, pose := range Poses {
		img := newSheet(PxW, PxH, scale)
		if err := drawGrid(img, grids[pose], 0, 0, scale); err != nil {
			return fmt.Errorf("astro: pose %q: %w", pose, err)
		}
		if err := savePNG(filepath.Join(dir, "astronaut-"+string(pose)+".png"), img); err != nil {
			return err
		}
	}
	for _, prop := range Props {
		grid := grids[prop]
		img := newSheet(len([]rune(grid[0])), len(grid), scale)
		if err := drawGrid(img, grid, 0, 0, scale); err != nil {
			return fmt.Errorf("astro: prop %q: %w", prop, err)
		}
		if err := savePNG(filepath.Join(dir, "astronaut-"+string(prop)+".png"), img); err != nil {
			return err
		}
	}
	// Strips cap their magnification so a wide sheet stays under
	// preview width and never gets resampled off the pixel grid.
	stripScale := scale
	if stripScale > 8 {
		stripScale = 8
	}
	if err := writeStrip(filepath.Join(dir, "astronaut-run-strip.png"), RunPoses, stripScale); err != nil {
		return err
	}
	if err := writeStrip(filepath.Join(dir, "astronaut-sheet.png"), Poses, stripScale); err != nil {
		return err
	}
	return nil
}

// writeStrip lays poses side by side with a one-pixel gutter.
func writeStrip(path string, poses []sprite.Heading, scale int) error {
	wPx := len(poses)*(PxW+1) - 1
	img := newSheet(wPx, PxH, scale)
	for i, pose := range poses {
		if err := drawGrid(img, grids[pose], i*(PxW+1), 0, scale); err != nil {
			return fmt.Errorf("astro: strip pose %q: %w", pose, err)
		}
	}
	return savePNG(path, img)
}
