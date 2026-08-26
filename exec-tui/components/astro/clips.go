package astro

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// RunFPS is the stride rate of the three-frame run cycle.
	RunFPS = 10.0
	// PoleFPS is the shimmy rate of the two-grip pole slide.
	PoleFPS = 6.0
)

// RunPoses is the stride order the run cycle plays.
var RunPoses = []sprite.Heading{PoseRun1, PoseRun2, PoseRun3}

// PolePoses is the grip order the slide alternates.
var PolePoses = []sprite.Heading{PosePole1, PosePole2}

// Run is the three-frame stride, in order, as an animation.
func Run(a *sprite.Atlas) (sprite.Animation, error) {
	return sprite.AnimationFrom(a, Size, RunPoses, RunFPS)
}

// Jump is the single airborne pose as a one-frame animation.
func Jump(a *sprite.Atlas) (sprite.Animation, error) {
	return sprite.AnimationFrom(a, Size, []sprite.Heading{PoseJump}, 1)
}

// Pole is the two alternating slide grips as an animation.
func Pole(a *sprite.Atlas) (sprite.Animation, error) {
	return sprite.AnimationFrom(a, Size, PolePoses, PoleFPS)
}
