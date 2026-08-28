package shootingstar

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/bigstar"
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/components/startrail"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Show is the shooting-star scene: a still night sky, a larger star,
// and a persist-particle trail. New always falls right-to-left, high
// on the right to low on the left. NewPreview follows Cfg.Path: fall
// is that same meteor; circle and square are optional tail loops.
type Show struct {
	Cfg     Config
	Seed    int64
	preview bool
	sky     *stars.Continuity
	flyer   *Flyer
	cross   Crossing
	screenplay.Ensemble
}

// New is the scene: a right-to-left fall, whatever Path the knobs say.
func New(sky *stars.Continuity) *Show {
	return newShow(sky, false)
}

// NewPreview is the tuner: fall (the scene), or a circle/square loop.
func NewPreview(sky *stars.Continuity) *Show {
	return newShow(sky, true)
}

// closedLoop is true only for the tuner's circle and square. The
// scene, and the tuner on fall, always use a right-to-left crossing.
func (s *Show) closedLoop() bool {
	return s != nil && s.preview && (s.Cfg.Path == PathCircle || s.Cfg.Path == PathSquare)
}

func newShow(sky *stars.Continuity, preview bool) *Show {
	s := &Show{Cfg: Active(), Seed: 11, preview: preview, sky: sky}
	s.Assemble = s.assemble
	return s
}

func (s *Show) assemble() []screenplay.Component {
	field := stars.NewTunedStarfield().Still()
	if s.sky != nil {
		field = field.Seed(s.sky)
	}
	s.flyer = newFlyer(s)
	return []screenplay.Component{
		field,
		s.flyer,
	}
}

// Bill is the shooting star as a one-scene screenplay.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "shootingstar", Scene: New(nil)},
	}
}

// Flyer is the composite performer: larger star plus persist trail,
// walked along the path.
type Flyer struct {
	star  *bigstar.Star
	trail *startrail.Trail
	show  *Show
	clock float64
	lap   float64
	uw    float64
	uh    float64
	w, h  int
}

func newFlyer(s *Show) *Flyer {
	return &Flyer{show: s}
}

func (f *Flyer) Start(w, h int) {
	if f == nil || f.show == nil {
		return
	}
	f.w, f.h = w, h
	f.uw = float64(w)*particle.CellWidthUnits - 0.01
	f.uh = float64(h)*particle.CellHeightUnits - 0.01
	if f.uw < 1 {
		f.uw = 1
	}
	if f.uh < 1 {
		f.uh = 1
	}
	seed := f.show.Seed
	if seed == 0 {
		seed = 11
	}
	size := f.show.Cfg.Size
	f.star = bigstar.New(seed)
	f.star.Size = size
	f.star.RandomSize = f.show.Cfg.RandomSize
	f.trail = startrail.New(seed)
	if !f.show.closedLoop() {
		f.show.cross = RandomCrossing(seed, f.uw, f.uh)
		f.lap = f.show.cross.length()
		if f.lap < 1 {
			f.lap = 1
		}
	}
	f.clock = 0
	pos, head := f.at(0)
	f.star.Heading = head
	f.star.Place(cellOf(pos))
	f.trail.Follow(pos, head)
	f.star.Start(w, h)
	f.trail.Start(w, h)
	// Place again after Start so a parked default center cannot win
	f.star.Place(cellOf(pos))
}

func (f *Flyer) Update(dt float64) {
	if f == nil || dt <= 0 || f.show == nil {
		return
	}
	_ = startrail.Use(f.show.Cfg.Trail())
	f.clock += dt
	if !f.show.closedLoop() {
		speed := f.show.Cfg.Speed
		if f.clock*speed >= f.lap {
			f.clock = 0
			f.show.Seed++
			f.show.cross = RandomCrossing(f.show.Seed, f.uw, f.uh)
			f.lap = f.show.cross.length()
			if f.lap < 1 {
				f.lap = 1
			}
			if f.show.Cfg.RandomSize && f.star != nil {
				f.star.RandomSize = true
				f.star.Seed = f.show.Seed
				f.star.Start(f.w, f.h)
			}
		}
	}
	pos, head := f.at(f.clock)
	if f.star != nil {
		if !f.show.Cfg.RandomSize {
			f.star.Size = f.show.Cfg.Size
			f.star.RandomSize = false
		}
		f.star.Heading = head
		f.star.Place(cellOf(pos))
		f.star.Update(dt)
	}
	if f.trail != nil {
		f.trail.Follow(pos, head)
		f.trail.Update(dt)
	}
}

func (f *Flyer) at(clock float64) (pos, heading particle.Vec2) {
	if f.show.closedLoop() {
		return f.previewAt(clock)
	}
	speed := f.show.Cfg.Speed
	t := 0.0
	if f.lap > 0 {
		t = clock * speed / f.lap
	}
	if t > 1 {
		t = 1
	}
	return f.show.cross.At(t)
}

func (f *Flyer) previewAt(clock float64) (pos, heading particle.Vec2) {
	cx, cy := f.uw/2, f.uh/2
	speed := f.show.Cfg.Speed
	switch f.show.Cfg.Path {
	case PathSquare:
		m := math.Min(f.uw, f.uh) * 0.18
		if m < 4 {
			m = 4
		}
		perim := 2 * ((f.uw - 2*m) + (f.uh - 2*m))
		if perim < 1 {
			perim = 1
		}
		t := math.Mod(clock*speed/perim, 1)
		if t < 0 {
			t += 1
		}
		return SquareAt(m, m, f.uw-m, f.uh-m, t)
	default:
		r := math.Min(cx, cy) * 0.42
		if r < 4 {
			r = 4
		}
		circ := 2 * math.Pi * r
		ang := clock * speed / circ * 2 * math.Pi
		return CircleAt(cx, cy, r, ang)
	}
}

func (f *Flyer) Render() sprite.Sprite {
	if f == nil || f.w < 1 || f.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(f.w, f.h)
	if f.trail != nil {
		sprite.Blit(stage, 0, 0, f.trail.Render())
	}
	if f.star != nil {
		sprite.Blit(stage, 0, 0, f.star.Render())
	}
	return stage
}

func (f *Flyer) Stop() {
	if f == nil {
		return
	}
	if f.star != nil {
		f.star.Stop()
	}
	if f.trail != nil {
		f.trail.Stop()
	}
}

func cellOf(p particle.Vec2) (col, row int) {
	return int(math.Floor(p.X)), int(math.Floor(p.Y / particle.CellHeightUnits))
}
