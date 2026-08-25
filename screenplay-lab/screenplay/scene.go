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

// Advance forwards dt seconds to every actor. dt <= 0 is a no-op.
func (s *Scene) Advance(dt float64) {
}

// Paint draws the cast in order onto the stage.
func (s *Scene) Paint(st *Stage) {
}
