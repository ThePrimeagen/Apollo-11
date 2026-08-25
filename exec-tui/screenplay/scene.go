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

// Ensemble is the common scene shape: a cast of components playing
// over time. Assemble builds the cast when the scene starts, so a
// scene allocates nothing until its curtain. The ensemble owns the
// staging: a scene's Start knows no stage size, so the components
// start on the first render — the moment the stage is finally known —
// and a resize stops and restarts them at the new size. Stop stops
// every started component and drops the cast for the collector.
type Ensemble struct {
	Assemble func() []Component
	cast     []Component
	staged   bool
	w, h     int
}

// Start assembles the cast. No component starts here — the stage size
// arrives with the first render.
func (e *Ensemble) Start() {
	if e == nil || e.Assemble == nil {
		return
	}
	e.cast = e.Assemble()
	e.staged = false
}

// Update forwards dt seconds to every started component in cast order.
// Before the first render nothing has started, so nothing ticks; a
// component's own Start is always its first cue. dt <= 0 is a no-op:
// time never runs backwards mid-scene.
func (e *Ensemble) Update(dt float64) {
	if e == nil || dt <= 0 || !e.staged {
		return
	}
	for _, c := range e.cast {
		if c == nil {
			continue
		}
		c.Update(dt)
	}
}

// Render composites the cast onto the screen in order — every
// component hands back its stage-sized sprite and later sprites land
// on top, transparent cells sparing whatever is beneath. When the
// stage is new or has changed size, the cast is (re)started first.
// Without a screen there is nothing to compose onto, so the cast is
// never called.
func (e *Ensemble) Render(scr *Screen) {
	if e == nil || scr == nil {
		return
	}
	if w, h := scr.Size(); !e.staged || w != e.w || h != e.h {
		e.restage(w, h)
	}
	for _, c := range e.cast {
		if c == nil {
			continue
		}
		scr.Blit(0, 0, c.Render())
	}
}

// restage stops whatever was running and starts the cast for a w×h
// stage: a resize is a Stop followed by a Start at the new size.
func (e *Ensemble) restage(w, h int) {
	if e.staged {
		e.stopCast()
	}
	for _, c := range e.cast {
		if c == nil {
			continue
		}
		c.Start(w, h)
	}
	e.w, e.h = w, h
	e.staged = true
}

func (e *Ensemble) stopCast() {
	for _, c := range e.cast {
		if c == nil {
			continue
		}
		c.Stop()
	}
}

// Stop stops every started component and drops the cast.
func (e *Ensemble) Stop() {
	if e == nil {
		return
	}
	if e.staged {
		e.stopCast()
	}
	e.cast = nil
	e.staged = false
}
