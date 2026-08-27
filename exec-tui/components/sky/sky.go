package sky

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// The stock sky: a pale horizon (xterm 153) under a deep zenith
// (xterm 17), with the dark coming straight down from the top.
const (
	DefaultLight = 153
	DefaultDark  = 17
	DefaultAngle = 0.0
)

// WorldScale is how many stage-heights the moveable sky is tall.
// The camera opens on the bottom third (almost-pure light blue) and
// pans up over Rise seconds until the top third (darker blue, and
// the clouds that live there) fills the view.
const WorldScale = 3

// Sky is the blue field as a scene component. Start paints the
// gradient for a w×h stage from the active knobs; Update runs the
// rise clock; Render returns this instant's viewport; Stop strikes
// the stage. The rise clock rides across restarts, so a resize
// never falls back to the horizon.
type Sky struct {
	rise   float64
	pan0   float64
	clock  float64
	angle  float64
	light  int
	dark   int
	w, h   int
	staged bool
}

// New binds a sky that paints the Active knobs. Nothing is built
// until Start. The stock sky sits at pan 0 — the horizon shot.
func New() *Sky {
	c := Active()
	return &Sky{light: c.LightInk, dark: c.DarkInk, angle: c.AngleDeg}
}

// Rise makes the sky moveable: it opens on pan 0 and climbs to pan 1
// over seconds. seconds <= 0 keeps the sky still at its current pan.
// Call before Start. Nil-safe.
func (s *Sky) Rise(seconds float64) *Sky {
	if s == nil {
		return nil
	}
	if seconds > 0 {
		s.rise = seconds
	}
	return s
}

// At pins the sky at pan frac (0 = horizon, 1 = zenith) and holds
// it there. Out-of-range values clamp. Call before Start. Nil-safe.
func (s *Sky) At(frac float64) *Sky {
	if s == nil {
		return nil
	}
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	s.pan0 = frac
	s.rise = 0
	s.clock = 0
	return s
}

// Pan is how far the camera has tilted, 0 at the horizon to 1 at
// the zenith. Nil and unstarted skies sit at their pin.
func (s *Sky) Pan() float64 {
	if s == nil {
		return 0
	}
	if s.rise <= 0 {
		return s.pan0
	}
	p := s.pan0 + s.clock/s.rise
	if p > 1 {
		return 1
	}
	if p < 0 {
		return 0
	}
	return p
}

// Start paints the gradient for a w×h stage. The rise clock is not
// touched: a resize never falls back to the horizon.
func (s *Sky) Start(w, h int) {
	if s == nil {
		return
	}
	s.w, s.h = w, h
	s.staged = true
}

// Update advances the rise. dt <= 0 holds the camera still.
func (s *Sky) Update(dt float64) {
	if s == nil || dt <= 0 {
		return
	}
	s.clock += dt
}

// Render lays the visible slice of the taller sky onto a stage-sized
// sprite. Before Start and after Stop the stage is empty.
func (s *Sky) Render() sprite.Sprite {
	if s == nil || !s.staged || s.w < 1 || s.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(s.w, s.h)
	worldH := float64(s.h * WorldScale)
	viewTop := (1 - s.Pan()) * (worldH - float64(s.h))
	for r := 0; r < s.h; r++ {
		for c := 0; c < s.w; c++ {
			ink := s.inkAt(float64(c)+0.5, viewTop+float64(r)+0.5, float64(s.w), worldH)
			stage.Set(r, c, sprite.Cell{Ch: ' ', FG: -1, BG: ink})
		}
	}
	return stage
}

// inkAt is the gradient color of a world-space point: the dark ink
// comes from angle degrees clockwise of straight up, and the field
// reaches the light ink by two thirds of the way down the world so
// the opening horizon shot is almost-pure light blue.
func (s *Sky) inkAt(x, y, w, worldH float64) int {
	rad := s.angle * math.Pi / 180
	sin, cos := math.Sincos(rad)
	// Dark-source direction: 0° is up (0, -1), 90° is right (1, 0).
	nx, ny := sin, -cos
	t := x*nx + y*ny
	// Project the world corners to normalise t onto 0..1.
	minT, maxT := math.Inf(1), math.Inf(-1)
	for _, cx := range []float64{0, w} {
		for _, cy := range []float64{0, worldH} {
			p := cx*nx + cy*ny
			if p < minT {
				minT = p
			}
			if p > maxT {
				maxT = p
			}
		}
	}
	u := 0.0
	if maxT > minT {
		// u=0 at the dark source (max projection), u=1 at the light side.
		u = (maxT - t) / (maxT - minT)
	}
	// u=0 is the dark source, u=1 is the opposite (light) side.
	// Compress the light so the bottom third of a top-down sky is
	// solid light: remap u through a bias that hits 1 at 2/3.
	if u >= 2.0/3.0 {
		u = 1
	} else {
		u = u / (2.0 / 3.0)
	}
	return lerpInk(s.dark, s.light, u)
}

// Stop strikes the stage. The clock stays, so the next Start picks
// the rise up mid-tilt.
func (s *Sky) Stop() {
	if s == nil {
		return
	}
	s.staged = false
}
