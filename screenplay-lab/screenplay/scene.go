package screenplay

// Actor is one performer. Advance moves its private clock forward dt
// seconds; Paint draws its current instant onto the stage.
type Actor interface {
	Advance(dt float64)
	Paint(st *Stage)
}

// Scene is one beat of the screenplay: a named cast playing over time.
// Cast order is paint order — later actors draw on top.
type Scene struct {
	Name string
	Cast []Actor
}

// Advance forwards dt seconds to every actor in cast order. dt <= 0 is
// a no-op: time never runs backwards mid-scene.
func (s *Scene) Advance(dt float64) {
	if s == nil || dt <= 0 {
		return
	}
	for _, a := range s.Cast {
		if a == nil {
			continue
		}
		a.Advance(dt)
	}
}

// Paint draws the cast in order onto the stage. Without a stage there
// is nothing to paint on, so the cast is never called.
func (s *Scene) Paint(st *Stage) {
	if s == nil || st == nil {
		return
	}
	for _, a := range s.Cast {
		if a == nil {
			continue
		}
		a.Paint(st)
	}
}
