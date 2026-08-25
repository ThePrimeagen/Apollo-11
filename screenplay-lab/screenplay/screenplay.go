package screenplay

// Screenplay is scenes in order with a cursor on the one now playing.
// Time only ever reaches the current scene; the next scene's clocks
// start when the play cuts to it.
type Screenplay struct {
	scenes []*Scene
	idx    int
}

// New binds scenes into a screenplay opened on the first scene.
func New(scenes ...*Scene) *Screenplay {
	return &Screenplay{}
}

// Len is the number of scenes.
func (p *Screenplay) Len() int {
	return 0
}

// SceneIndex is the 0-based index of the current scene.
func (p *Screenplay) SceneIndex() int {
	return 0
}

// Current is the scene now playing, or nil for an empty screenplay.
func (p *Screenplay) Current() *Scene {
	return nil
}

// Next cuts to the following scene. On the final scene (or empty
// screenplay) it stays put and reports false.
func (p *Screenplay) Next() bool {
	return false
}

// Advance forwards dt seconds to the current scene only.
func (p *Screenplay) Advance(dt float64) {
}
