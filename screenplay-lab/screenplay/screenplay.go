package screenplay

// Entry is one scene on the bill, with the name the marquee shows.
type Entry struct {
	Name  string
	Scene Scene
}

// Screenplay is scenes in order with a cursor on the one now playing.
// Nothing runs until Start; after that, every frame is Update then
// Render on the current scene only, Next cuts (stop the old, start the
// new), and Stop brings the house lights up.
type Screenplay struct {
	bill    []Entry
	idx     int
	running bool
}

// New binds the bill. No scene starts until the screenplay does.
func New(entries ...Entry) *Screenplay {
	return &Screenplay{bill: entries}
}

// Len is the number of scenes on the bill.
func (p *Screenplay) Len() int {
	if p == nil {
		return 0
	}
	return len(p.bill)
}

// SceneIndex is the 0-based index of the current scene.
func (p *Screenplay) SceneIndex() int {
	if p == nil {
		return 0
	}
	return p.idx
}

// CurrentName is the marquee name of the current scene, or "" for an
// empty bill.
func (p *Screenplay) CurrentName() string {
	if p == nil || p.idx >= len(p.bill) {
		return ""
	}
	return p.bill[p.idx].Name
}

// current is the scene now playing, or nil off the bill (including the
// tolerated hole of an Entry with no Scene).
func (p *Screenplay) current() Scene {
	if p == nil || p.idx >= len(p.bill) {
		return nil
	}
	return p.bill[p.idx].Scene
}

// Start raises the first curtain. It is a no-op on an empty bill or a
// screenplay that is already running.
func (p *Screenplay) Start() {
	if p == nil || p.running || len(p.bill) == 0 {
		return
	}
	p.running = true
	if sc := p.current(); sc != nil {
		sc.Start()
	}
}

// Update forwards dt seconds to the scene now playing. Before Start,
// after Stop, and for dt <= 0 it is a no-op.
func (p *Screenplay) Update(dt float64) {
	if p == nil || !p.running || dt <= 0 {
		return
	}
	if sc := p.current(); sc != nil {
		sc.Update(dt)
	}
}

// Render clears the screen, has the current scene paint this frame,
// and consumes the screen's resized flag on the way out. Before Start
// or without a screen it is a no-op.
func (p *Screenplay) Render(scr *Screen) {
	if p == nil || !p.running || scr == nil {
		return
	}
	scr.Clear()
	if sc := p.current(); sc != nil {
		sc.Render(scr)
	}
	scr.resized = false
}

// Next cuts to the following scene — the old scene stops, then the new
// one starts — and reports whether it moved. On the final scene, before
// Start, or on an empty bill it holds and reports false.
func (p *Screenplay) Next() bool {
	if p == nil || !p.running || p.idx+1 >= len(p.bill) {
		return false
	}
	if sc := p.current(); sc != nil {
		sc.Stop()
	}
	p.idx++
	if sc := p.current(); sc != nil {
		sc.Start()
	}
	return true
}

// Stop stops the scene now playing and ends the run. Further calls are
// no-ops.
func (p *Screenplay) Stop() {
	if p == nil || !p.running {
		return
	}
	if sc := p.current(); sc != nil {
		sc.Stop()
	}
	p.running = false
}
