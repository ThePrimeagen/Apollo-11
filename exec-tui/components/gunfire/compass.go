package gunfire

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	PanelCols   = 12
	PanelRows   = 7
	CompassCols = 36
	CompassRows = 21
	panelBox    = 12.0
	panelInset  = 1.2
)

// Course is one named muzzle direction.
type Course struct {
	Name    string
	Heading sprite.Heading
	Dir     particle.Vec2
}

// Courses is clockwise from north: four cardinals and four 45° offsets.
func Courses() []Course {
	out := make([]Course, len(sprite.Headings))
	for i, h := range sprite.Headings {
		out[i] = Course{Name: string(h), Heading: h, Dir: dirOf(h)}
	}
	return out
}

// ConfigToward is one shot aimed along dir inside a compass panel.
// Origin sits on the incoming wall so the flame has the whole box to travel.
func ConfigToward(dir particle.Vec2, l Layer) particle.Config {
	cfg := particle.Config{
		Width:       panelBox - 0.01,
		Height:      panelBox - 0.01,
		Direction:   dir,
		Count:       l.Count,
		MinLife:     l.MinLife,
		MaxLife:     l.MaxLife,
		MinSpeed:    l.MinSpeed,
		MaxSpeed:    l.MaxSpeed,
		Spread:      l.Spread,
		Nozzle:      l.Nozzle,
		MaxDistance: l.MaxDistance,
		Lift:        l.Lift,
		Drag:        l.Drag,
	}
	n := dir.Normalize()
	if n == (particle.Vec2{}) {
		cfg.Origin = particle.Vec2{X: panelBox / 2, Y: panelBox / 2}
		return cfg
	}
	cfg.Origin = particle.Vec2{
		X: panelBox/2 - n.X*(panelBox/2-panelInset),
		Y: panelBox/2 - n.Y*(panelBox/2-panelInset),
	}
	return cfg
}

// Compass is all eight headings on one fixed rose, each a one-shot
// muzzle flame aimed its own way. The tuner plays the whole rose at
// once, the way the flame config does.
type Compass struct {
	Slots []slot

	armed bool
	fuse  float64
}

type slot struct {
	Name    string
	Heading sprite.Heading
	Core    *particle.Engine
	Flame   *particle.Engine
	Col     int
	Row     int
}

// NewCompass builds the eight-direction rose. It has not yet fired.
func NewCompass(seed int64) *Compass {
	place := map[string][2]int{
		"NW": {0, 0}, "N": {1, 0}, "NE": {2, 0},
		"W": {0, 1}, "E": {2, 1},
		"SW": {0, 2}, "S": {1, 2}, "SE": {2, 2},
	}
	c := &Compass{}
	blast := ActiveBlast()
	for i, course := range Courses() {
		at := place[course.Name]
		c.Slots = append(c.Slots, slot{
			Name:    course.Name,
			Heading: course.Heading,
			Core:    particle.New(seed+int64(i)*17, ConfigToward(course.Dir, blast.Core)),
			Flame:   particle.New(seed+1+int64(i)*17, ConfigToward(course.Dir, blast.ShotAt(course.Heading).Layer)),
			Col:     at[0],
			Row:     at[1],
		})
	}
	return c
}

// Fire is the trigger: every heading's core and flame burst now, and
// Doom's second flash frame is fused against the whole rose.
func (c *Compass) Fire() {
	if c == nil {
		return
	}
	c.burst(1)
	cfg := ActiveBlast()
	if cfg.PulseDelay > 0 && cfg.PulseFrac > 0 {
		c.armed, c.fuse = true, cfg.PulseDelay
	}
}

func (c *Compass) burst(frac float64) {
	for i := range c.Slots {
		for _, e := range []*particle.Engine{c.Slots[i].Core, c.Slots[i].Flame} {
			if e == nil {
				continue
			}
			full := e.Cfg.Count
			e.Cfg.Count = int(float64(full)*frac + 0.5)
			e.Burst()
			e.Cfg.Count = full
		}
	}
}

// Update retunes every panel from the active blast, flies the rose
// dt seconds, and burns the fuse. dt <= 0 holds everything.
func (c *Compass) Update(dt float64) {
	if c == nil || dt <= 0 {
		return
	}
	blast := ActiveBlast()
	for i := range c.Slots {
		s := &c.Slots[i]
		dir := dirOf(s.Heading)
		if s.Core != nil {
			s.Core.Cfg = ConfigToward(dir, blast.Core)
			s.Core.Update(dt)
		}
		if s.Flame != nil {
			s.Flame.Cfg = ConfigToward(dir, blast.ShotAt(s.Heading).Layer)
			s.Flame.Update(dt)
		}
	}
	if c.armed {
		c.fuse -= dt
		if c.fuse <= 0 {
			c.armed = false
			c.burst(ActiveBlast().PulseFrac)
		}
	}
}

// Live is how many specks are still burning anywhere on the rose.
func (c *Compass) Live() int {
	if c == nil {
		return 0
	}
	n := 0
	for _, s := range c.Slots {
		if s.Core != nil {
			n += len(s.Core.Particles)
		}
		if s.Flame != nil {
			n += len(s.Flame.Particles)
		}
	}
	return n
}

// View is the 3×3 rose: eight muzzle flames, empty centre, labels on each panel.
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
		flame := s.panel()
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

func (s slot) panel() sprite.Sprite {
	flames := make([]*particle.Engine, len(sprite.Headings))
	if i := headingIndex(s.Heading); i >= 0 {
		flames[i] = s.Flame
	}
	return paint(ActiveBlast(), PanelCols, PanelRows-1, s.Core, flames)
}
