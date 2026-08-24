// Package rocket stands the size-4 LM from the sprite atlas on the live
// booster plume aimed straight down: the large rocket with fire on the
// bottom. The atlas art keeps its baked-in tilde plume for the small
// descent views; here those cells are stripped and the particle fire is
// the plume. The flame burns in its own window under the engine bell and
// never paints over the hull.
package rocket

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/theprimeagen/apollo-11/lander-lab/components/fire"
	"github.com/theprimeagen/apollo-11/lander-lab/particle"
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

const (
	// Cols is the size-4 canvas width; Rows adds the 6-row flame box
	// under the 8 hull rows above the nozzle exit.
	Cols = 26
	Rows = 14
	// FlameRow/FlameCol place the 12×6 flame box so its nozzle cell
	// (col 6) lands on the engine bell's centre column (13), one row
	// below the bell.
	FlameRow = 8
	FlameCol = 7
	fps      = 20
)

// Rocket is the composed craft: hull on top, booster fire below.
type Rocket struct {
	Body  sprite.Sprite
	Flame *fire.Flame
}

// New builds the size-4 north-facing rocket over a south-firing plume.
// The flame has not yet emitted.
func New(seed int64) *Rocket {
	return &Rocket{
		Body:  stripPlume(sprite.Default().MustFrame(sprite.Size4, sprite.N)),
		Flame: fire.Toward(seed, particle.Vec2{X: 0, Y: 1}),
	}
}

// Update advances the booster fire.
func (r *Rocket) Update(dt float64) {
	if r == nil {
		return
	}
	r.Flame.Update(dt)
}

// View is the fixed canvas: the hull drawn first, the flame composed
// into its window below the bell. Transparent flame cells let the
// footpads show through.
func (r *Rocket) View() sprite.Sprite {
	board := sprite.New(Cols, Rows)
	if r == nil {
		return board
	}
	for row := 0; row < r.Body.Height; row++ {
		for col := 0; col < r.Body.Width; col++ {
			cell := r.Body.At(row, col)
			if cell.Transparent() {
				continue
			}
			board.Set(row, col, cell)
		}
	}
	if r.Flame == nil {
		return board
	}
	flame := r.Flame.Sprite()
	for row := 0; row < flame.Height; row++ {
		for col := 0; col < flame.Width; col++ {
			cell := flame.At(row, col)
			if cell.Transparent() {
				continue
			}
			board.Set(FlameRow+row, FlameCol+col, cell)
		}
	}
	return board
}

// Render is the ANSI view of the rocket.
func (r *Rocket) Render() string { return sprite.Render(r.View()) }

// WriteTape writes n PNG frames of a burning rocket into dir at 20 fps.
func WriteTape(dir string, r *Rocket, n, cellW int) ([]string, error) {
	if n <= 0 {
		return nil, fmt.Errorf("need at least one frame")
	}
	if r == nil {
		return nil, fmt.Errorf("nil rocket")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	paths := make([]string, 0, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(dir, fmt.Sprintf("frame-%04d.png", i))
		r.Update(1.0 / float64(fps))
		if err := fire.WritePNG(p, r.View(), cellW); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// stripPlume drops the art's static '~'/'≈' exhaust cells; the live
// particle fire replaces them.
func stripPlume(sp sprite.Sprite) sprite.Sprite {
	out := sprite.New(sp.Width, sp.Height)
	for row := 0; row < sp.Height; row++ {
		for col := 0; col < sp.Width; col++ {
			cell := sp.At(row, col)
			if cell.Ch == '~' || cell.Ch == '≈' {
				continue
			}
			out.Set(row, col, cell)
		}
	}
	return out
}
