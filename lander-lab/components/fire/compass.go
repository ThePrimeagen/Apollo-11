package fire

import (
	"fmt"
	"path/filepath"

	"github.com/theprimeagen/apollo-11/lander-lab/particle"
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"
)

const (
	PanelCols   = 12
	PanelRows   = 7
	CompassCols = 36
	CompassRows = 21
)

// Course is one named exhaust direction.
type Course struct {
	Name string
	Dir  particle.Vec2
}

// Courses is clockwise from north: four cardinals and four 45° offsets.
func Courses() []Course {
	return []Course{
		{Name: "N", Dir: particle.Vec2{X: 0, Y: -1}},
		{Name: "NE", Dir: particle.Vec2{X: 1, Y: -1}},
		{Name: "E", Dir: particle.Vec2{X: 1, Y: 0}},
		{Name: "SE", Dir: particle.Vec2{X: 1, Y: 1}},
		{Name: "S", Dir: particle.Vec2{X: 0, Y: 1}},
		{Name: "SW", Dir: particle.Vec2{X: -1, Y: 1}},
		{Name: "W", Dir: particle.Vec2{X: -1, Y: 0}},
		{Name: "NW", Dir: particle.Vec2{X: -1, Y: -1}},
	}
}

// ConfigToward is the booster plume aimed along dir. Origin sits on the
// incoming wall so the fire has the whole box to travel.
func ConfigToward(dir particle.Vec2) particle.Config {
	cfg := BoosterConfig()
	const box, inset = 12.0, 1.2
	cfg.Width = box - 0.01
	cfg.Height = box - 0.01
	cfg.Direction = dir
	n := dir.Normalize()
	if n == (particle.Vec2{}) {
		cfg.Origin = particle.Vec2{X: box / 2, Y: box / 2}
		return cfg
	}
	cfg.Origin = particle.Vec2{
		X: box/2 - n.X*(box/2-inset),
		Y: box/2 - n.Y*(box/2-inset),
	}
	return cfg
}

// Toward starts a plume along dir.
func Toward(seed int64, dir particle.Vec2) *Flame {
	return &Flame{Eng: particle.New(seed, ConfigToward(dir))}
}

// Compass is all eight headings on one fixed rose.
type Compass struct {
	Slots []slot
}

type slot struct {
	Name  string
	Flame *Flame
	Col   int
	Row   int
}

// NewCompass builds the eight-direction rose. It has not yet emitted.
func NewCompass(seed int64) *Compass {
	place := map[string][2]int{
		"NW": {0, 0}, "N": {1, 0}, "NE": {2, 0},
		"W": {0, 1}, "E": {2, 1},
		"SW": {0, 2}, "S": {1, 2}, "SE": {2, 2},
	}
	c := &Compass{}
	for i, course := range Courses() {
		at := place[course.Name]
		c.Slots = append(c.Slots, slot{
			Name:  course.Name,
			Flame: Toward(seed+int64(i)*17, course.Dir),
			Col:   at[0],
			Row:   at[1],
		})
	}
	return c
}

// Update advances every heading.
func (c *Compass) Update(dt float64) {
	if c == nil {
		return
	}
	for i := range c.Slots {
		c.Slots[i].Flame.Update(dt)
	}
}

// View is the 3×3 rose: eight plumes, empty centre, labels on each panel.
func (c *Compass) View() sprite.Sprite {
	board := sprite.New(CompassCols, CompassRows)
	if c == nil {
		return board
	}
	label := sprite.Cell{Ch: ' ', FG: 250, BG: -1}
	for _, s := range c.Slots {
		px, py := s.Col*PanelCols, s.Row*PanelRows
		for i, r := range []rune(s.Name) {
			lab := label
			lab.Ch = r
			board.Set(py, px+i, lab)
		}
		flame := s.Flame.Sprite()
		for r := 0; r < flame.Height; r++ {
			for col := 0; col < flame.Width; col++ {
				cell := flame.At(r, col)
				if cell.Transparent() {
					continue
				}
				board.Set(py+1+r, px+col, cell)
			}
		}
	}
	return board
}

// WriteCompassTape writes n PNG frames of the rose at 20 fps.
func WriteCompassTape(dir string, c *Compass, n, cellW int) ([]string, error) {
	if n <= 0 {
		return nil, fmt.Errorf("need at least one frame")
	}
	if c == nil {
		return nil, fmt.Errorf("nil compass")
	}
	return writeFrames(dir, n, func() sprite.Sprite {
		c.Update(1.0 / float64(fps))
		return c.View()
	}, cellW)
}

func writeFrames(dir string, n int, frame func() sprite.Sprite, cellW int) ([]string, error) {
	if err := mkdir(dir); err != nil {
		return nil, err
	}
	paths := make([]string, 0, n)
	for i := 0; i < n; i++ {
		p := filepath.Join(dir, fmt.Sprintf("frame-%04d.png", i))
		if err := WritePNG(p, frame(), cellW); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}
