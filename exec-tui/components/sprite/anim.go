package sprite

import "fmt"

// DefaultAnimationFPS is the frame rate an Animation plays at when the
// caller does not pick one.
const DefaultAnimationFPS = 8.0

// clockEps soaks up float noise when a clock that was rebuilt from a
// modulo lands a hair under a frame boundary.
const clockEps = 1e-7

// Animation is the simplest thing that can play: a list of sprites
// played in order, at a frame rate.
type Animation struct {
	Frames []Sprite
	FPS    float64
}

// Len is how many frames the animation holds.
func (a Animation) Len() int { return len(a.Frames) }

// Frame returns the i-th frame, wrapping in both directions so the
// list loops. An empty animation renders an empty sprite, never panics.
func (a Animation) Frame(i int) Sprite {
	n := len(a.Frames)
	if n == 0 {
		return Sprite{}
	}
	i %= n
	if i < 0 {
		i += n
	}
	return a.Frames[i]
}

// At returns the frame playing at t seconds. Time before the curtain
// clamps to the first frame; FPS <= 0 plays at DefaultAnimationFPS.
func (a Animation) At(t float64) Sprite {
	if t < 0 {
		t = 0
	}
	fps := a.FPS
	if fps <= 0 {
		fps = DefaultAnimationFPS
	}
	return a.Frame(int(t*fps + clockEps))
}

// AnimationFrom builds an animation from named atlas frames, in the
// caller's order. A missing frame is an error naming the frame — an
// animation never invents art.
func AnimationFrom(a *Atlas, sz Size, names []Heading, fps float64) (Animation, error) {
	if a == nil {
		return Animation{}, fmt.Errorf("sprite: animation from a nil atlas")
	}
	out := Animation{FPS: fps, Frames: make([]Sprite, 0, len(names))}
	for _, name := range names {
		sp, ok := a.Frame(sz, name)
		if !ok {
			return Animation{}, fmt.Errorf("sprite: atlas has no frame %q at size %d", name, sz)
		}
		out.Frames = append(out.Frames, sp)
	}
	return out, nil
}
