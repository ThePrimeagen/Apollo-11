package lander

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/fire"
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// BodyCols/BodyRows is the size-4 frame: the full zoomed-in craft.
	BodyCols = 26
	BodyRows = 10
	// FlyInHoldSeconds is how long the craft waits offstage before
	// the fly-in starts.
	FlyInHoldSeconds = 3.0
	// FlyInSeconds is how long the slide from the right wing to center
	// stage takes, after the hold.
	FlyInSeconds = 4.0
	// BobPeriodSeconds is one full up-and-down of the parked bobble.
	BobPeriodSeconds = 10.0
	// BobAmplitudeCells is how far the bobble rides from center: one
	// full cell up and one down. (Half a cell would need half-shifted
	// art the atlas doesn't have yet.)
	BobAmplitudeCells = 1
	// FlameRow/FlameCol hang the 16×8 booster box off the tail,
	// relative to the hull's top-left, centered on the size-4 west
	// grey nozzle: the box covers the bell's four lip rows (3..6), so
	// the beam rides the two nozzle rows and the flare spills one row
	// above and one row below — never just off the bottom.
	FlameRow = 3
	FlameCol = 19
	// NorthFlameRow/NorthFlameCol hang the south-firing plume under
	// the size-4 north engine bell, matching the rocket card.
	NorthFlameRow = 8
	NorthFlameCol = 7
	// DropSeconds is how long the north-facing fall from off the top
	// of the stage to off the bottom takes.
	DropSeconds = 6.0
	// LandSeconds is how long the north-facing drop from off the top
	// onto the moon horizon pad takes.
	LandSeconds = 5.0
	// LandEasePower is the ease-out exponent on the landing path:
	// 1 is linear, 3 is the fly-in's cubic. 5 is a heavy settle —
	// fast off the top, then a long crawl that clinks onto the pad.
	LandEasePower = 5.0
	// ThrottleStageSeconds is how long each landing booster step
	// (¾, ½, ¼) lasts. Three stages, then off on the pad.
	ThrottleStageSeconds = 0.4
	// LandThrottleLead is the last three stages of a landing: the
	// booster stays full until then, then steps ¾, ½, ¼, and cuts
	// off on the pad.
	LandThrottleLead = 3 * ThrottleStageSeconds
	// dustDieSeconds is how fast the pad cloud counts down once a
	// DustAt run ends. Short so "how long it runs" is the emit
	// window, not a two-second linger after.
	dustDieSeconds = 0.15
	// landSurfaceRows is the moon horizon's center thickness — the
	// hull parks with its feet on that ridge.
	landSurfaceRows = 5
	// dustRowOffset lifts a stage-sized dust cloud so its shared
	// floor point rides the pad surface the feet land on: the cloud's
	// nozzles sit three rows above its own bottom edge, the surface
	// sits landSurfaceRows above the stage's.
	dustRowOffset = 3 - landSurfaceRows
)

// Ship is the Apollo craft as a scene component: the size-4 W-heading
// frame with its baked tilde plume stripped and, unless Dark, a live
// left-to-right booster fire trailing from the tail. It slides in from
// the right wing, parks at center stage, and bobbles on a slow sine.
// A landing ship also kicks dust off the pad — one continuous cloud
// on both sides of the booster from the first throttle step-down
// through booster-off, then counting down to nothing. Start builds
// the hull and arms the fire for its stage; Stop drops everything so
// a stopped ship holds no allocation, and a later Start rebuilds it.
type Ship struct {
	Body       sprite.Sprite
	Flame      *fire.Flame
	seed       int64
	clock      float64
	w, h       int
	dark       bool
	hold       float64
	heading    sprite.Heading
	dropSec    float64
	landSec    float64
	stageSec   float64
	dustSec    float64
	dustStart  float64
	dustRun    float64
	dustTimed  bool
	flameBase  particle.Config
	padDust    *dust.Cloud
	dustFading bool
}

// NewShip binds the craft to its fire seed. Nothing is built until
// Start — the curtain owns the allocation.
func NewShip(seed int64) *Ship {
	return &Ship{seed: seed}
}

// Start builds the hull from the atlas and arms a fresh booster fire
// for a w×h stage. The clock carries across restarts, so a resize
// never replays the fly-in.
func (s *Ship) Start(w, h int) {
	if s == nil {
		return
	}
	s.w, s.h = w, h
	heading := s.heading
	if heading == "" {
		heading = sprite.W
	}
	s.Body = stripPlume(DefaultAtlas().MustFrame(sprite.Size4, heading))
	if s.dark {
		s.Flame = nil
		return
	}
	if heading == sprite.N {
		s.Flame = fire.Toward(s.seed, particle.Vec2{X: 0, Y: 1})
	} else {
		s.Flame = &fire.Flame{Eng: particle.New(s.seed, shipFlameConfig())}
	}
	if s.landSec > 0 {
		s.armLandPlume()
	}
}

// shipFlameConfig slims the stock left-to-right booster to a cruise
// plume: a 16×8-unit box (four rows) whose origin sits on the boundary
// between the two nozzle rows, so the beam straddles them and flares
// evenly one row up and one row down; a lighter emission, a tighter
// jet, and speeds that let the tail taper out inside the box instead
// of dying on a wall.
func shipFlameConfig() particle.Config {
	cfg := fire.BoosterConfig()
	cfg.Width = 16 - 0.01
	cfg.Height = 8 - 0.01
	cfg.Origin = particle.Vec2{X: 1.0, Y: 4.0}
	cfg.Direction = particle.Vec2{X: 1, Y: 0}
	cfg.Count = 2
	cfg.MinSpeed, cfg.MaxSpeed = 11, 22
	cfg.Spread = 0.18
	cfg.Nozzle = 1.2
	return cfg
}

// armLandPlume snapshots the full-strength booster so later
// throttles can scale count, life, and distance from a stable base.
func (s *Ship) armLandPlume() {
	if s == nil || s.Flame == nil {
		return
	}
	cfg := s.Flame.Config()
	if cfg.MaxDistance <= 0 {
		cfg.MaxDistance = plumeReach(cfg)
	}
	_ = s.Flame.SetConfig(cfg)
	s.flameBase = s.Flame.Config()
}

// plumeReach is how far a particle can travel from the origin to the
// far wall of the box along the exhaust axis — the full-strength
// max distance of a landing plume.
func plumeReach(cfg particle.Config) float64 {
	dir := cfg.Direction.Normalize()
	if dir == (particle.Vec2{}) {
		return 0
	}
	t := math.Inf(1)
	if dir.X > 1e-12 {
		t = math.Min(t, (cfg.Width-cfg.Origin.X)/dir.X)
	} else if dir.X < -1e-12 {
		t = math.Min(t, -cfg.Origin.X/dir.X)
	}
	if dir.Y > 1e-12 {
		t = math.Min(t, (cfg.Height-cfg.Origin.Y)/dir.Y)
	} else if dir.Y < -1e-12 {
		t = math.Min(t, -cfg.Origin.Y/dir.Y)
	}
	if math.IsInf(t, 1) || t < 0 {
		return 0
	}
	return t
}

func (s *Ship) applyLandThrottle() {
	if s == nil || s.Flame == nil {
		return
	}
	th := landThrottle(s.clock-s.hold, s.landSec, s.stageOrDefault())
	cfg := s.flameBase
	if th <= 0 {
		cfg.Count = 0
		cfg.Period = 0
		_ = s.Flame.SetConfig(cfg)
		if s.Flame.Eng != nil {
			s.Flame.Eng.Particles = nil
		}
		return
	}
	cfg.Count = int(math.Round(float64(s.flameBase.Count) * th))
	cfg.MinLife = s.flameBase.MinLife * th
	cfg.MaxLife = s.flameBase.MaxLife * th
	cfg.MaxDistance = s.flameBase.MaxDistance * th
	_ = s.Flame.SetConfig(cfg)
}

// Update moves the ship's clock, burns the fire, and runs the landing
// dust kicks. dt <= 0 holds.
func (s *Ship) Update(dt float64) {
	if s == nil || dt <= 0 {
		return
	}
	s.clock += dt
	if s.landSec > 0 {
		s.applyLandThrottle()
		s.updateLandDust(dt)
	}
	if s.Flame != nil {
		s.Flame.Update(dt)
	}
}

// updateLandDust runs the landing dust: one mirrored cloud blown out
// of the pad on both sides of the booster — leftward and rightward,
// climbing away from the bell. DustAt times the cloud from an offset
// for a run, independent of the booster. Otherwise the cloud starts
// when the booster first steps down and fades after booster-off over
// Dust seconds (dust.FadeSeconds when unset). A dark or unstarted
// ship kicks nothing.
func (s *Ship) updateLandDust(dt float64) {
	if s.dark || s.Body.Width < 1 {
		return
	}
	if s.dustTimed {
		s.updateTimedDust(dt)
		return
	}
	t := s.clock - s.hold
	lead := 3 * s.stageOrDefault()
	fade := s.dustOrDefault()
	if s.padDust == nil {
		if t < s.landSec-lead {
			return
		}
		if t > s.landSec+fade {
			return
		}
		s.padDust = dust.NewCloud(s.seed + 2)
		s.padDust.Start(s.w, s.h)
		s.dustFading = false
	}
	if t >= s.landSec && !s.dustFading {
		s.padDust.Fade(fade)
		s.dustFading = true
	}
	s.padDust.Update(dt)
}

// updateTimedDust is the DustAt path: emit on [start, start+run),
// then a short fade. A non-positive run never arms a cloud.
func (s *Ship) updateTimedDust(dt float64) {
	if s.dustRun <= 0 {
		return
	}
	t := s.clock - s.hold
	start := s.dustStart
	if start < 0 {
		start = 0
	}
	end := start + s.dustRun
	if s.padDust == nil {
		if t < start || t >= end {
			return
		}
		s.padDust = dust.NewCloud(s.seed + 2)
		s.padDust.Start(s.w, s.h)
		s.dustFading = false
	}
	if t >= end && !s.dustFading {
		s.padDust.Fade(dustDieSeconds)
		s.dustFading = true
	}
	s.padDust.Update(dt)
}

// Clock is how many seconds of scene time the ship has played.
func (s *Ship) Clock() float64 {
	if s == nil {
		return 0
	}
	return s.clock
}

// Render composes dust first, fire second, hull last, into a
// stage-sized sprite, so the hull always wins the overlap at the tail
// and the plume appears from behind the bell. The dust clouds are
// stage-sized and ride dustRowOffset up, so their shared floor point
// sits on the pad surface. Before Start and after Stop there is
// nothing built, so the stage is empty.
func (s *Ship) Render() sprite.Sprite {
	if s == nil || s.w < 1 || s.h < 1 || s.Body.Width < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(s.w, s.h)
	row, col := s.position()
	sprite.Blit(stage, 0, dustRowOffset, s.padDust.Render())
	if s.Flame != nil {
		fr, fc := FlameRow, FlameCol
		if s.heading == sprite.N {
			fr, fc = NorthFlameRow, NorthFlameCol
		}
		sprite.Blit(stage, col+fc, row+fr, s.Flame.Sprite())
	}
	sprite.Blit(stage, col, row, s.Body)
	return stage
}

// Dark is a hull-only ship: Start still builds the body, but the
// booster stays cold. Nil-safe.
func (s *Ship) Dark() *Ship {
	if s == nil {
		return nil
	}
	s.dark = true
	s.Flame = nil
	return s
}

// Hold waits seconds offstage before the fly-in starts. Call before
// Start. Nil-safe.
func (s *Ship) Hold(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.hold = seconds
	return s
}

// North flies the size-4 north-facing frame with a down-firing
// booster. Call before Start. Nil-safe.
func (s *Ship) North() *Ship {
	if s == nil {
		return nil
	}
	s.heading = sprite.N
	return s
}

// Drop falls from fully off the top of the stage to fully off the
// bottom over seconds. Call before Start. Nil-safe.
func (s *Ship) Drop(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.dropSec = seconds
	return s
}

// Land falls from fully off the top onto the moon-horizon pad over
// seconds, then stays put. Call before Start. Nil-safe.
func (s *Ship) Land(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.landSec = seconds
	return s
}

// ThrottleStage sets how long each landing booster step (¾, ½, ¼)
// lasts. seconds <= 0 keeps the stock ThrottleStageSeconds. Call
// before Start. Nil-safe.
func (s *Ship) ThrottleStage(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.stageSec = seconds
	return s
}

// Dust sets how long the pad cloud lingers after the booster cuts
// off. seconds <= 0 keeps dust.FadeSeconds. Call before Start.
// Nil-safe. DustAt, if also set, wins.
func (s *Ship) Dust(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.dustSec = seconds
	return s
}

// DustAt times the pad cloud from start seconds (from t=0) for a run
// of that many seconds, independent of the booster. A non-positive
// run never kicks. Call before Start. Nil-safe.
func (s *Ship) DustAt(start, run float64) *Ship {
	if s == nil {
		return nil
	}
	s.dustStart = start
	s.dustRun = run
	s.dustTimed = true
	return s
}

func (s *Ship) stageOrDefault() float64 {
	if s != nil && s.stageSec > 0 {
		return s.stageSec
	}
	return ThrottleStageSeconds
}

func (s *Ship) dustOrDefault() float64 {
	if s != nil && s.dustSec > 0 {
		return s.dustSec
	}
	return dust.FadeSeconds
}

// position is this frame's hull top-left: a landing path, a drop, or
// the westbound fly-in, depending on how the ship was asked to fly.
func (s *Ship) position() (row, col int) {
	t := s.clock - s.hold
	if s.landSec > 0 {
		return LandPath(s.w, s.h, t, s.landSec)
	}
	if s.dropSec > 0 {
		return DropPath(s.w, s.h, t, s.dropSec)
	}
	return FlightPath(s.w, s.h, t)
}

// Parked starts the clock at the fly-in park so the first frame is
// already center-stage, skipping any Hold. Nil-safe.
func (s *Ship) Parked() *Ship {
	if s == nil {
		return nil
	}
	s.clock = s.hold + FlyInSeconds
	return s
}

// Stop drops the hull, the fire, and the dust for the collector; a
// fresh Start rebuilds what its moment still calls for.
func (s *Ship) Stop() {
	if s == nil {
		return
	}
	s.Body = sprite.Sprite{}
	s.Flame = nil
	s.padDust = nil
	s.dustFading = false
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

// DropPath is the hull's top-left at t seconds of a seconds-long fall
// on a stageW×stageH stage: fully off the top at t=0, fully off the
// bottom at t=seconds, centered horizontally. Time before the curtain
// clamps to the start.
func DropPath(stageW, stageH int, t, seconds float64) (row, col int) {
	if t < 0 {
		t = 0
	}
	col = (stageW - BodyCols) / 2
	start, end := -BodyRows, stageH
	if seconds <= 0 || t >= seconds {
		return end, col
	}
	p := t / seconds
	return start + int(math.Round(p*float64(end-start))), col
}

// LandPadRow is where the hull's top-left parks on a landing: the
// moon horizon's center ridge minus the hull height, so the feet sit
// on the surface.
func LandPadRow(stageH int) int {
	// +1 puts the feet on the ridge instead of one cell above it.
	return stageH - landSurfaceRows - BodyRows + 1
}

// LandThrottle is the booster strength at t seconds of a seconds-long
// landing: full until LandThrottleLead remains, then three equal
// intervals of ¾, ½, ¼, then off on the pad. A ship that called
// ThrottleStage uses that stage length instead; this helper always
// speaks the stock stages.
func LandThrottle(t, seconds float64) float64 {
	return landThrottle(t, seconds, ThrottleStageSeconds)
}

func landThrottle(t, seconds, stageSec float64) float64 {
	if seconds <= 0 {
		return 0
	}
	if t < 0 {
		t = 0
	}
	if t >= seconds {
		return 0
	}
	if stageSec <= 0 {
		stageSec = ThrottleStageSeconds
	}
	lead := 3 * stageSec
	remaining := seconds - t
	if remaining > lead {
		return 1
	}
	switch {
	case remaining > 2*stageSec:
		return 0.75
	case remaining > stageSec:
		return 0.5
	default:
		return 0.25
	}
}

// LandPath is the hull's top-left at t seconds of a seconds-long
// landing: fully off the top at t=0, parked on the horizon pad by
// t=seconds, then held there. The fall eases out (1-(1-p)^LandEasePower)
// so it comes in fast and clinks on. Time before the curtain clamps
// to the start.
func LandPath(stageW, stageH int, t, seconds float64) (row, col int) {
	if t < 0 {
		t = 0
	}
	col = (stageW - BodyCols) / 2
	start, end := -BodyRows, LandPadRow(stageH)
	if seconds <= 0 || t >= seconds {
		return end, col
	}
	p := t / seconds
	eased := 1 - math.Pow(1-p, LandEasePower)
	return start + int(math.Round(eased*float64(end-start))), col
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
