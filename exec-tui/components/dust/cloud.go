package dust

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// warmSeconds primes the kick so the first frame is already dusty.
	warmSeconds = 1.0
	warmFPS     = 20
)

// FadeSeconds is the stock countdown of a fading cloud: from however
// many specks the warm kick holds at its max, linearly down to zero,
// over two seconds.
const FadeSeconds = 2.0

// Cloud is the dust-off component: two mirrored swirl engines kicking
// dust out of a shared floor point, leftward and rightward. Start
// builds both for a w×h stage; each Update re-reads the active puff so
// an in-process editor can retune the kick live; Render paints both
// engines' dust onto one stage-sized sprite. A Fade cloud is a
// one-shot burst: it opens on the full warm kick and counts its
// particles down to zero over the fade window, then stays clear.
type Cloud struct {
	Left, Right *particle.Engine

	seed int64
	w, h int

	fadeSec           float64
	fadeClock         float64
	leftMax, rightMax int

	losing     bool
	lossPerMs  float64
	lossBudget float64
}

// NewCloud binds a kick to its particle seed. Nothing is built until Start.
func NewCloud(seed int64) *Cloud {
	return &Cloud{seed: seed}
}

// Fade makes the cloud a one-shot burst: from the cue it counts its
// particles down — linearly, from however many the warm kick (or the
// live cloud, if Fade is called after Start) holds to zero over
// seconds — emission tapering with the same line and the freshest
// specks dying first, so the blown fringe drifts out and thins
// instead of being erased. seconds <= 0 keeps the endless kick.
// Call before Start to fade from the curtain, or after Start to fade
// from this instant; a fresh Start rewinds the countdown. Nil-safe.
func (c *Cloud) Fade(seconds float64) *Cloud {
	if c == nil {
		return nil
	}
	c.fadeSec = seconds
	c.fadeClock = 0
	if c.Left != nil {
		c.leftMax, c.rightMax = len(c.Left.Particles), len(c.Right.Particles)
	}
	return c
}

// LossPerMs is the stock drain of a landing cloud: 0.05 specks per
// millisecond (50 a second) once the engines start cutting. Slow
// enough that the blown fringe stays in the air; fast enough that
// the pad clears a couple of seconds after booster-off.
const LossPerMs = 0.05

// Loss drains live specks at perMs particles per millisecond from
// the cue, emission tapering with the remaining budget, so the kick
// dies down instead of blinking out. perMs <= 0 stops feeding but
// does not trim — specks die of old age. Call before Start to drain
// from the curtain, or after Start to drain from this instant.
// Nil-safe.
func (c *Cloud) Loss(perMs float64) *Cloud {
	if c == nil {
		return nil
	}
	c.losing = true
	c.lossPerMs = perMs
	if c.lossPerMs < 0 {
		c.lossPerMs = 0
	}
	c.fadeSec = 0
	if c.Left != nil {
		c.leftMax, c.rightMax = len(c.Left.Particles), len(c.Right.Particles)
		c.lossBudget = float64(c.leftMax + c.rightMax)
	}
	return c
}

// Start builds both engines for a w×h stage and warms them so the
// curtain rises on dust already in the air. The warm kick is a fading
// cloud's max: its countdown starts here.
func (c *Cloud) Start(w, h int) {
	if c == nil {
		return
	}
	c.w, c.h = w, h
	uw, uh := c.units()
	left, right := ActivePuff().Engines(uw, uh)
	c.Left = particle.New(c.seed, left)
	c.Right = particle.New(c.seed+1, right)
	dt := 1.0 / float64(warmFPS)
	for t := 0.0; t < warmSeconds; t += dt {
		c.Left.Update(dt)
		c.Right.Update(dt)
	}
	c.fadeClock = 0
	c.leftMax, c.rightMax = len(c.Left.Particles), len(c.Right.Particles)
	if c.losing {
		c.lossBudget = float64(c.leftMax + c.rightMax)
	}
}

func (c *Cloud) units() (w, h float64) {
	return float64(c.w)*particle.CellWidthUnits - 0.01,
		float64(c.h)*particle.CellHeightUnits - 0.01
}

// Update pulls the active puff onto both engines and burns them. A
// fading cloud also runs its countdown: emission scales down the fade
// line and the live specks are capped to it, freshest first, so the
// count falls from the max to zero across the window. dt <= 0 holds.
func (c *Cloud) Update(dt float64) {
	if c == nil || dt <= 0 || c.Left == nil || c.Right == nil {
		return
	}
	uw, uh := c.units()
	left, right := ActivePuff().Engines(uw, uh)
	frac := c.countdown(dt)
	if c.losing {
		frac = c.drain(dt)
	}
	left.Count = scaled(left.Count, frac)
	right.Count = scaled(right.Count, frac)
	c.Left.Cfg = left
	c.Right.Cfg = right
	c.Left.Update(dt)
	c.Right.Update(dt)
	if c.fadeSec > 0 || (c.losing && c.lossPerMs > 0) {
		trim(c.Left, scaled(c.leftMax, frac))
		trim(c.Right, scaled(c.rightMax, frac))
	}
}

// countdown advances the fade clock and reports the fraction of the
// kick still allowed: 1 for an endless cloud, falling to 0 at the end
// of the fade window.
func (c *Cloud) countdown(dt float64) float64 {
	if c.fadeSec <= 0 {
		return 1
	}
	c.fadeClock += dt
	frac := 1 - c.fadeClock/c.fadeSec
	if frac < 0 {
		return 0
	}
	return frac
}

// drain spends the loss budget and reports the fraction of the
// opening kick still allowed. A zero rate stops emission (frac 0)
// without trimming.
func (c *Cloud) drain(dt float64) float64 {
	if c.lossPerMs <= 0 {
		return 0
	}
	c.lossBudget -= c.lossPerMs * dt * 1000
	if c.lossBudget < 0 {
		c.lossBudget = 0
	}
	max := float64(c.leftMax + c.rightMax)
	if max <= 0 {
		return 0
	}
	return c.lossBudget / max
}

// scaled is n specks allowed at frac of the countdown.
func scaled(n int, frac float64) int {
	if frac >= 1 {
		return n
	}
	return int(math.Round(float64(n) * frac))
}

// trim caps the live specks at allowed, freshest first: the kick
// stops feeding while the blown fringe drifts out, thins, and expires
// on its own — a dust-off dying down, not contracting into a churn
// at the floor.
func trim(e *particle.Engine, allowed int) {
	if allowed < 0 {
		allowed = 0
	}
	if len(e.Particles) > allowed {
		e.Particles = e.Particles[:allowed]
	}
}

// Render paints both engines onto one stage-sized sprite. Before Start
// and after Stop there is nothing built, so the stage is empty.
func (c *Cloud) Render() sprite.Sprite {
	if c == nil || c.Left == nil || c.Right == nil || c.w < 1 || c.h < 1 {
		return sprite.Sprite{}
	}
	return paint(ActivePuff(), c.w, c.h, c.Left, c.Right)
}

// Stop drops both engines; a fresh Start rebuilds them.
func (c *Cloud) Stop() {
	if c == nil {
		return
	}
	c.Left, c.Right = nil, nil
}
