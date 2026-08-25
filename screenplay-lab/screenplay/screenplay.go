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
	return &Screenplay{}
}

// Len is the number of scenes on the bill.
func (p *Screenplay) Len() int {
	return 0
}

// SceneIndex is the 0-based index of the current scene.
func (p *Screenplay) SceneIndex() int {
	return 0
}

// CurrentName is the marquee name of the current scene, or "" for an
// empty bill.
func (p *Screenplay) CurrentName() string {
	return ""
}

// Start raises the first curtain. It is a no-op on an empty bill or a
// screenplay that is already running.
func (p *Screenplay) Start() {
}

// Update forwards dt seconds to the scene now playing. Before Start,
// after Stop, and for dt <= 0 it is a no-op.
func (p *Screenplay) Update(dt float64) {
}

// Render clears the screen, has the current scene paint it, and
// consumes the screen's resized flag. Before Start or without a screen
// it is a no-op.
func (p *Screenplay) Render(scr *Screen) {
}

// Next cuts to the following scene — the old scene stops, then the new
// one starts — and reports whether it moved. On the final scene, before
// Start, or on an empty bill it holds and reports false.
func (p *Screenplay) Next() bool {
	return false
}

// Stop stops the scene now playing and ends the run. Further calls are
// no-ops.
func (p *Screenplay) Stop() {
}
