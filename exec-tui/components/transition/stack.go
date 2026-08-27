package transition

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Layers is several components painted in order as one: later
// sprites land on earlier floors, transparent cells sparing
// whatever is beneath. Skies uses this to treat sky+clouds as a
// single From for the flag crossfade.
type Layers struct {
	layers []screenplay.Component
	w, h   int
	staged bool
}

// Stack binds the layers in blit order. Nil entries are skipped.
func Stack(layers ...screenplay.Component) *Layers {
	return &Layers{layers: layers}
}

func (s *Layers) Start(w, h int) {
	if s == nil {
		return
	}
	s.w, s.h = w, h
	s.staged = true
	for _, l := range s.layers {
		if l != nil {
			l.Start(w, h)
		}
	}
}

func (s *Layers) Update(dt float64) {
	if s == nil {
		return
	}
	for _, l := range s.layers {
		if l != nil {
			l.Update(dt)
		}
	}
}

func (s *Layers) Render() sprite.Sprite {
	if s == nil || !s.staged || s.w < 1 || s.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(s.w, s.h)
	for _, l := range s.layers {
		if l == nil {
			continue
		}
		sprite.Blit(stage, 0, 0, l.Render())
	}
	return stage
}

func (s *Layers) Stop() {
	if s == nil {
		return
	}
	for _, l := range s.layers {
		if l != nil {
			l.Stop()
		}
	}
	s.staged = false
}
