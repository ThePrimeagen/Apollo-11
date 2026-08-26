package sprite

import "fmt"

// DefaultAnimationFPS is the frame rate an Animation plays at when the
// caller does not pick one.
const DefaultAnimationFPS = 8.0

// Animation is a list of sprites played in order.
type Animation struct {
	Frames []Sprite
	FPS    float64
}

// Len is how many frames the animation holds.
func (a Animation) Len() int { return 0 }

// Frame returns the i-th frame, wrapping so the list loops.
func (a Animation) Frame(i int) Sprite { return Sprite{} }

// At returns the frame playing at t seconds.
func (a Animation) At(t float64) Sprite { return Sprite{} }

// AnimationFrom builds an animation from named atlas frames, in the
// caller's order.
func AnimationFrom(a *Atlas, sz Size, names []Heading, fps float64) (Animation, error) {
	return Animation{}, fmt.Errorf("sprite: AnimationFrom not implemented")
}
