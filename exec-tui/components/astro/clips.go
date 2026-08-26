package astro

import (
	"fmt"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// RunFPS is the stride rate of the three-frame run cycle.
	RunFPS = 10.0
	// PoleFPS is the shimmy rate of the two-grip pole slide.
	PoleFPS = 6.0
)

// Run is the three-frame stride, in order, as an animation.
func Run(a *sprite.Atlas) (sprite.Animation, error) {
	return sprite.Animation{}, fmt.Errorf("astro: Run not implemented")
}

// Jump is the single airborne pose as a one-frame animation.
func Jump(a *sprite.Atlas) (sprite.Animation, error) {
	return sprite.Animation{}, fmt.Errorf("astro: Jump not implemented")
}

// Pole is the two alternating slide grips as an animation.
func Pole(a *sprite.Atlas) (sprite.Animation, error) {
	return sprite.Animation{}, fmt.Errorf("astro: Pole not implemented")
}
