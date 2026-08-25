package screenplay

// Scene is one beat of the screenplay. The screenplay starts it when
// the bill reaches it, then every frame updates it and renders it onto
// the shared screen, and stops it on the cut away.
//
//	Start   allocate: the curtain rises
//	Update  advance internal clocks by dt seconds; no render data
//	Render  write this instant's cells into the screen
//	Stop    deallocate: the curtain falls
type Scene interface {
	Start()
	Update(dt float64)
	Render(scr *Screen)
	Stop()
}

// Actor is one performer inside an Ensemble: the same update/render
// pair, minus the lifecycle — the ensemble owns that.
type Actor interface {
	Update(dt float64)
	Render(scr *Screen)
}

// Ensemble is the common scene shape: a cast of sprites playing over
// time. Assemble builds the cast when the scene starts, so a scene
// allocates nothing until its curtain, and Stop drops the cast for the
// collector.
type Ensemble struct {
	Assemble func() []Actor
	cast     []Actor
}

// Start assembles the cast.
func (e *Ensemble) Start() {
	if e == nil || e.Assemble == nil {
		return
	}
	e.cast = e.Assemble()
}

// Update forwards dt seconds to every actor in cast order. dt <= 0 is
// a no-op: time never runs backwards mid-scene.
func (e *Ensemble) Update(dt float64) {
	if e == nil || dt <= 0 {
		return
	}
	for _, a := range e.cast {
		if a == nil {
			continue
		}
		a.Update(dt)
	}
}

// Render draws the cast in order — later actors land on top. Without a
// screen there is nothing to draw on, so the cast is never called.
func (e *Ensemble) Render(scr *Screen) {
	if e == nil || scr == nil {
		return
	}
	for _, a := range e.cast {
		if a == nil {
			continue
		}
		a.Render(scr)
	}
}

// Stop drops the cast.
func (e *Ensemble) Stop() {
	if e == nil {
		return
	}
	e.cast = nil
}
