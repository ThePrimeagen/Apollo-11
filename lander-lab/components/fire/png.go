package fire

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

// RenderPNG paints a sprite as a bitmap. cellW is the pixel width of one
// terminal cell; height is 2× that so one unit is a square. Block glyphs
// are rasterized geometrically so the flame reads as fire, not font boxes.
func RenderPNG(sp sprite.Sprite, cellW int) (image.Image, error) {
	if cellW < 6 {
		cellW = 8
	}
	cellH := cellW * 2
	void := xterm256(-1)
	img := image.NewRGBA(image.Rect(0, 0, sp.Width*cellW, sp.Height*cellH))
	draw.Draw(img, img.Bounds(), &image.Uniform{void}, image.Point{}, draw.Src)
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			cell := sp.At(r, c)
			rect := image.Rect(c*cellW, r*cellH, (c+1)*cellW, (r+1)*cellH)
			bg := void
			if !cell.Transparent() && cell.BG >= 0 {
				bg = xterm256(cell.BG)
			}
			draw.Draw(img, rect, &image.Uniform{bg}, image.Point{}, draw.Src)
			if cell.Transparent() || cell.Ch == ' ' {
				continue
			}
			fg := xterm256(cell.FG)
			if cell.FG < 0 {
				fg = color.RGBA{0xd0, 0xd0, 0xd0, 255}
			}
			paintGlyph(img, rect, cell.Ch, fg)
		}
	}
	return img, nil
}

// WritePNG writes one sprite to path.
func WritePNG(path string, sp sprite.Sprite, cellW int) error {
	img, err := RenderPNG(sp, cellW)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, img); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// WriteTape writes n PNG frames of a running flame into dir at 20 fps.
// The canvas is View, so every frame is the same size.
func WriteTape(dir string, f *Flame, n, cellW int) ([]string, error) {
	if n <= 0 {
		return nil, fmt.Errorf("need at least one frame")
	}
	if f == nil {
		return nil, fmt.Errorf("nil flame")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	dt := 1.0 / float64(fps)
	paths := make([]string, 0, n)
	for i := 0; i < n; i++ {
		f.Update(dt)
		p := filepath.Join(dir, fmt.Sprintf("frame-%04d.png", i))
		if err := WritePNG(p, f.View(), cellW); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func xterm256(n int) color.RGBA {
	if n < 0 {
		return color.RGBA{0x05, 0x06, 0x08, 0xff}
	}
	if n < 16 {
		return ansi16[n]
	}
	if n < 232 {
		n -= 16
		return color.RGBA{cube(n / 36), cube((n % 36) / 6), cube(n % 6), 0xff}
	}
	v := uint8(8 + (n-232)*10)
	return color.RGBA{v, v, v, 0xff}
}

func cube(i int) uint8 {
	if i == 0 {
		return 0
	}
	return uint8(55 + i*40)
}

var ansi16 = [16]color.RGBA{
	{0, 0, 0, 255}, {205, 0, 0, 255}, {0, 205, 0, 255}, {205, 205, 0, 255},
	{0, 0, 238, 255}, {205, 0, 205, 255}, {0, 205, 205, 255}, {229, 229, 229, 255},
	{127, 127, 127, 255}, {255, 0, 0, 255}, {0, 255, 0, 255}, {255, 255, 0, 255},
	{92, 92, 255, 255}, {255, 0, 255, 255}, {0, 255, 255, 255}, {255, 255, 255, 255},
}

func paintGlyph(img *image.RGBA, r image.Rectangle, ch rune, fg color.RGBA) {
	w, h := r.Dx(), r.Dy()
	set := func(x, y int) {
		if x < 0 || y < 0 || x >= w || y >= h {
			return
		}
		img.SetRGBA(r.Min.X+x, r.Min.Y+y, fg)
	}
	fill := func(rr image.Rectangle) {
		draw.Draw(img, rr, &image.Uniform{fg}, image.Point{}, draw.Src)
	}
	shade := func(step int) {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if (x+y*3)%step == 0 {
					set(x, y)
				}
			}
		}
	}
	switch ch {
	case '█':
		fill(r)
	case '▓':
		shade(2)
	case '▒':
		shade(3)
	case '░':
		shade(5)
	case '▄':
		fill(image.Rect(r.Min.X, r.Min.Y+h/2, r.Max.X, r.Max.Y))
	case '·':
		cx, cy := w/2, h/2
		rad := max(2, min(w, h)/6)
		for y := cy - rad; y <= cy+rad; y++ {
			for x := cx - rad; x <= cx+rad; x++ {
				if (x-cx)*(x-cx)+(y-cy)*(y-cy) <= rad*rad {
					set(x, y)
				}
			}
		}
	default:
		fill(r)
	}
}
