package screenplay

// Screenplay is scenes in order with a cursor on the one now playing.
// Time only ever reaches the current scene, so the next scene's clocks
// start the moment the play cuts to it.
type Screenplay struct {
	scenes []*Scene
	idx    int
}

// New binds scenes into a screenplay opened on the first scene.
func New(scenes ...*Scene) *Screenplay {
	return &Screenplay{scenes: scenes}
}

// Len is the number of scenes on the bill.
func (p *Screenplay) Len() int {
	if p == nil {
		return 0
	}
	return len(p.scenes)
}

// SceneIndex is the 0-based index of the current scene.
func (p *Screenplay) SceneIndex() int {
	if p == nil {
		return 0
	}
	return p.idx
}

// Current is the scene now playing, or nil for an empty screenplay.
func (p *Screenplay) Current() *Scene {
	if p == nil || p.idx >= len(p.scenes) {
		return nil
	}
	return p.scenes[p.idx]
}

// Next cuts to the following scene and reports whether it moved. On the
// final scene — or an empty screenplay — it holds and reports false.
func (p *Screenplay) Next() bool {
	if p == nil || p.idx+1 >= len(p.scenes) {
		return false
	}
	p.idx++
	return true
}

// Advance forwards dt seconds to the current scene only.
func (p *Screenplay) Advance(dt float64) {
	p.Current().Advance(dt)
}
