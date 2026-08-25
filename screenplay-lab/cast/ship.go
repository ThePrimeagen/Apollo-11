// Package cast is the screenplay-lab troupe: the actors the demo puts on
// stage. Each one wraps an existing lab component — the LM sprite atlas
// and booster fire from lander-lab, the starfield from stars-lab, the
// banner font from terminal-fonts — behind the screenplay.Actor face.
package cast

import (
	"math"

	"github.com/theprimeagen/apollo-11/lander-lab/components/fire"
	"github.com/theprimeagen/apollo-11/lander-lab/particle"
	"github.com/theprimeagen/apollo-11/lander-lab/sprite"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

const (
	// BodyCols/BodyRows is the size-4 frame: the full zoomed-in craft.
	BodyCols = 26
	BodyRows = 10
	// FlyInSeconds is how long the slide from the right wing to center
	// stage takes.
	FlyInSeconds = 4.0
	// BobPeriodSeconds is one full up-and-down of the parked bobble.
	BobPeriodSeconds = 10.0
	// BobAmplitudeCells is how far the bobble rides from center: one
	// full cell up and one down. (Half a cell would need half-shifted
	// art the atlas doesn't have yet.)
	BobAmplitudeCells = 1
	// FlameRow/FlameCol hang the 12×6 booster box off the tail,
	// relative to the hull's top-left, so the nozzle cell sits just
	// right of the engine bell and the plume trails behind the craft.
	FlameRow = 3
	FlameCol = 19
)

// Ship is the Apollo craft flying west: the size-4 W-heading frame with
// its baked tilde plume stripped and a live left-to-right booster fire
// trailing from the tail. It slides in from the right wing, parks at
// center stage, and bobbles on a slow sine.
type Ship struct {
	Body  sprite.Sprite
	Flame *fire.Flame
	clock float64
}

// NewShip builds the westbound craft. The fire has not yet emitted.
func NewShip(seed int64) *Ship {
	return &Ship{
		Body:  stripPlume(sprite.Default().MustFrame(sprite.Size4, sprite.W)),
		Flame: &fire.Flame{Eng: particle.New(seed, shipFlameConfig())},
	}
}

// shipFlameConfig slims the stock left-to-right booster to a cruise
// plume: a 16×6-unit box (three rows) so the beam hugs the engine bell,
// a lighter emission, a tighter jet, and speeds that let the tail taper
// out inside the box instead of dying on a wall.
func shipFlameConfig() particle.Config {
	cfg := fire.BoosterConfig()
	cfg.Width = 16 - 0.01
	cfg.Height = 6 - 0.01
	cfg.Origin = particle.Vec2{X: 1.0, Y: 3.0}
	cfg.Direction = particle.Vec2{X: 1, Y: 0}
	cfg.Count = 2
	cfg.MinSpeed, cfg.MaxSpeed = 11, 22
	cfg.Spread = 0.18
	cfg.Nozzle = 1.2
	return cfg
}

// Update moves the ship's clock and burns the fire. dt <= 0 holds.
func (s *Ship) Update(dt float64) {
	if s == nil || dt <= 0 {
		return
	}
	s.clock += dt
	s.Flame.Update(dt)
}

// Clock is how many seconds of scene time the ship has played.
func (s *Ship) Clock() float64 {
	if s == nil {
		return 0
	}
	return s.clock
}

// Render composes fire first, hull second, so the hull always wins the
// overlap at the tail and the plume appears from behind the bell.
func (s *Ship) Render(scr *screenplay.Screen) {
	if s == nil || scr == nil {
		return
	}
	w, h := scr.Size()
	row, col := FlightPath(w, h, s.clock)
	if s.Flame != nil {
		BlitSprite(scr, col+FlameCol, row+FlameRow, s.Flame.Sprite())
	}
	BlitSprite(scr, col, row, s.Body)
}

// FlightPath is the hull's top-left at t seconds into the scene, on a
// stageW×stageH stage. The craft holds level at center height: fully
// off the right wing at t=0, an eased slide that parks at center stage
// by FlyInSeconds, then a ±BobAmplitudeCells sine bobble with a
// BobPeriodSeconds period.
func FlightPath(stageW, stageH int, t float64) (row, col int) {
	if t < 0 {
		t = 0
	}
	row = (stageH - BodyRows) / 2
	park := (stageW - BodyCols) / 2
	if t < FlyInSeconds {
		// ease-out cubic: fast off the wing, gentle into the park.
		p := t / FlyInSeconds
		eased := 1 - math.Pow(1-p, 3)
		return row, stageW + int(math.Round(eased*float64(park-stageW)))
	}
	phase := 2 * math.Pi * (t - FlyInSeconds) / BobPeriodSeconds
	bob := int(math.Round(BobAmplitudeCells * math.Sin(phase)))
	return row - bob, park
}

// stripPlume drops the art's baked '~'/'≈' exhaust; the live particle
// fire is the plume here.
func stripPlume(sp sprite.Sprite) sprite.Sprite {
	out := sprite.New(sp.Width, sp.Height)
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			cell := sp.At(r, c)
			if cell.Ch == '~' || cell.Ch == '≈' {
				continue
			}
			out.Set(r, c, cell)
		}
	}
	return out
}
