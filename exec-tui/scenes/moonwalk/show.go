package moonwalk

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Beat is one act of the Mario screenplay.
type Beat int

const (
	// BeatRun is the crate climb: he sprints in and hops one, two,
	// three high, then holds on the top stack.
	BeatRun Beat = iota
	// BeatPole is the flagpole: the leap onto the gold ball, the
	// hold, the slide while the flag flies up, then the bow.
	BeatPole
	// BeatBoard is the exit: the camera pans to the lunar module,
	// he runs over, jumps the hatch, and stays gone.
	BeatBoard
	BeatCount
)

// Show is one moonwalk beat as a live scene. Cfg is the knobs New
// copies from Active; the clock starts at the beat's first frame and
// freezes on its last so a cut never walks into the next act.
type Show struct {
	Cfg   Config
	beat  Beat
	atlas *sprite.Atlas
	clock float64
	ready bool
	w, h  int
}

// New is one beat of the moonwalk. The curtain copies Active so 03.
// Mario plays the saved knobs on the first frame.
func New(beat Beat) *Show {
	return &Show{Cfg: Active(), beat: beat}
}

// Beat is which act this show plays.
func (s *Show) Beat() Beat {
	if s == nil {
		return BeatRun
	}
	return s.beat
}

func (s *Show) window(r route) (start, hold float64) {
	switch s.beat {
	case BeatRun:
		return 0, r.leapAt - clockEps
	case BeatPole:
		return r.leapAt, r.panAt - clockEps
	case BeatBoard:
		return r.panAt, r.cycle - clockEps
	default:
		return 0, r.cycle - clockEps
	}
}

// Start loads the atlas. The clock waits for the first render — that
// is when the stage size, and so the route, is known.
func (s *Show) Start() {
	if s == nil {
		return
	}
	atlas, err := astro.Load()
	if err != nil {
		atlas, err = astro.BuildAtlas()
	}
	if err != nil {
		atlas = nil
	}
	s.atlas = atlas
	s.ready = false
	s.clock = 0
}

// Update advances the beat clock. Before the first render nothing
// has staged, so nothing ticks.
func (s *Show) Update(dt float64) {
	if s == nil || dt <= 0 || !s.ready {
		return
	}
	s.clock += dt
}

func (s *Show) displayT(w, h int) float64 {
	r := routeFor(s.Cfg, w, h)
	start, hold := s.window(r)
	t := s.clock
	if t < start {
		t = start
	}
	if t > hold {
		t = hold
	}
	return t
}

// Render paints this instant of the beat. The first call (or a
// resize) stages the clock on the beat's opening frame.
func (s *Show) Render(scr *screenplay.Screen) {
	if s == nil || scr == nil {
		return
	}
	w, h := scr.Size()
	if !s.ready || w != s.w || h != s.h {
		r := routeFor(s.Cfg, w, h)
		start, _ := s.window(r)
		if !s.ready {
			s.clock = start
		}
		s.w, s.h = w, h
		s.ready = true
	}
	scr.Blit(0, 0, Frame(s.Cfg, s.atlas, w, h, s.displayT(w, h)))
}

// Stop drops the atlas. Start may come again.
func (s *Show) Stop() {
	if s == nil {
		return
	}
	s.atlas = nil
	s.ready = false
}
