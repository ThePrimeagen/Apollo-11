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
	// NorthFlameRows is the south-firing plume box in cells: the
	// Toward(S) world is 12 units tall, two units per cell.
	NorthFlameRows = 6
	// LiftGoneRow is the hull top-left that puts the hull and the
	// down-firing booster fully above the stage. The plume hangs
	// from NorthFlameRow, so -14 puts the last fire cell at row -1.
	LiftGoneRow = -(NorthFlameRow + NorthFlameRows)
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
	Body        sprite.Sprite
	Flame       *fire.Flame
	seed        int64
	clock       float64
	w, h        int
	dark        bool
	hold        float64
	heading     sprite.Heading
	dropSec     float64
	beats       []DropBeat
	climbSec    float64
	landSec     float64
	liftAt      float64
	liftSec     float64
	stageSec    float64
	dustSec     float64
	dustStart   float64
	dustRun     float64
	dustTimed   bool
	thTimed     bool
	th75        float64
	th50        float64
	th25        float64
	thOff       float64
	igTimed     bool
	ig25        float64
	ig50        float64
	ig75        float64
	igFull      float64
	bobSet      bool
	bobPeriod   float64
	bobCells    int
	flySet      bool
	flySec      float64
	dustLoss    float64
	dustLossSet bool
	flameBase   particle.Config
	padDust     *dust.Cloud
	dustFading  bool
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
		s.applyLandThrottle()
	}
	if s.liftSec > 0 {
		s.armLandPlume()
		s.applyLiftThrottle()
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
	if s.thTimed {
		th = throttleAt(s.clock-s.hold, s.th75, s.th50, s.th25, s.thOff)
	}
	s.throttleTo(th)
}

// applyLiftThrottle runs the liftoff ignition: cold on the pad until
// the first offset, then ¼, ½, ¾, and full power — the landing
// throttle played backwards. A lift with no ignition schedule burns
// full from the opening.
func (s *Ship) applyLiftThrottle() {
	if s == nil || s.Flame == nil {
		return
	}
	th := 1.0
	if s.igTimed {
		th = igniteAt(s.clock-s.hold, s.ig25, s.ig50, s.ig75, s.igFull)
	}
	s.throttleTo(th)
}

// throttleTo scales the booster to strength th from the armed base —
// count, life, and reach all follow — and zero cuts the fire outright.
func (s *Ship) throttleTo(th float64) {
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
// or liftoff dust kicks. dt <= 0 holds.
func (s *Ship) Update(dt float64) {
	if s == nil || dt <= 0 {
		return
	}
	s.clock += dt
	if s.landSec > 0 {
		s.applyLandThrottle()
		s.updateLandDust(dt)
	}
	if s.liftSec > 0 {
		s.applyLiftThrottle()
		s.updateLiftDust(dt)
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

// updateTimedDust is the DustAt path: emit on [start, start+run).
// The cloud starts draining at LossPerMs (or DustLoss) when the
// engines first cut (Fire75) or when the run ends, whichever is
// sooner — a taper, not the old 0.15s blink. A non-positive run
// never arms a cloud.
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
	if !s.dustFading && s.shouldDrain(t, end) {
		s.padDust.Loss(s.lossOrDefault())
		s.dustFading = true
	}
	s.padDust.Update(dt)
}

// updateLiftDust runs the liftoff pad dust. A liftoff has no stock
// choreography to inherit from the landing: the cloud only kicks when
// DustAt schedules it, and it drains at DustLoss when the run ends. A
// dark or unstarted ship kicks nothing.
func (s *Ship) updateLiftDust(dt float64) {
	if s.dark || s.Body.Width < 1 || !s.dustTimed {
		return
	}
	s.updateTimedDust(dt)
}

func (s *Ship) shouldDrain(t, end float64) bool {
	if t >= end {
		return true
	}
	return s.thTimed && t >= s.th75
}

func (s *Ship) lossOrDefault() float64 {
	if s != nil && s.dustLossSet {
		return s.dustLoss
	}
	return dust.LossPerMs
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

// Climb rises from fully off the bottom of the stage to fully off
// the top over seconds — Drop run the other way, linear, no pad,
// no ease. Call before Start. Nil-safe.
func (s *Ship) Climb(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.climbSec = seconds
	return s
}

// DropBeat is one segment of a pausing fall: Drop seconds of motion
// then Hold seconds parked. A zero hold skips the pause.
type DropBeat struct {
	Drop, Hold float64
}

// DropBeats falls the north-facing craft with pauses: each beat
// drops, then holds. Drop distances share the top-to-bottom span
// in proportion to each Drop duration. Call before Start. Nil-safe.
func (s *Ship) DropBeats(beats []DropBeat) *Ship {
	if s == nil {
		return nil
	}
	s.beats = append([]DropBeat(nil), beats...)
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

// Lift flies the liftoff: the hull opens parked on the moon-horizon
// pad, holds it until `at` seconds into the scene, then rises fully
// off the top over `seconds` — the landing played backwards, and
// then a little further so the down-firing booster leaves with the
// hull. Pad dust comes only from DustAt: a liftoff has no stock dust
// choreography. Call before Start. Nil-safe.
func (s *Ship) Lift(at, seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.liftAt = at
	s.liftSec = seconds
	return s
}

// IgniteAt times the liftoff ignition from t=0: cold until at25, then
// ¼ until at50, ½ until at75, ¾ until full, then full power — the
// landing throttle run backwards. Full at 0 burns from the opening.
// Ignition only speaks in lift mode. Call before Start. Nil-safe.
func (s *Ship) IgniteAt(at25, at50, at75, full float64) *Ship {
	if s == nil {
		return nil
	}
	s.ig25, s.ig50, s.ig75, s.igFull = at25, at50, at75, full
	s.igTimed = true
	return s
}

// Bob retunes the parked ride: one full up-and-down every period
// seconds, riding cells rows from center. A non-positive period or
// amplitude holds the park level. Call before Start. Nil-safe.
func (s *Ship) Bob(period float64, cells int) *Ship {
	if s == nil {
		return nil
	}
	s.bobPeriod = period
	s.bobCells = cells
	s.bobSet = true
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

// ThrottleAt times the landing booster steps from t=0: full until
// at75, then ¾ until at50, ½ until at25, ¼ until off, then cut.
// Off at 0 keeps the booster dark. Call before Start. Nil-safe.
// ThrottleAt, if set, wins over ThrottleStage.
func (s *Ship) ThrottleAt(at75, at50, at25, off float64) *Ship {
	if s == nil {
		return nil
	}
	s.th75, s.th50, s.th25, s.thOff = at75, at50, at25, off
	s.thTimed = true
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

// DustLoss is how many pad specks leave per millisecond once the
// engines start cutting (or the DustAt run ends). perMs <= 0 stops
// new emission but does not blink the live cloud out. Call before
// Start. Nil-safe.
func (s *Ship) DustLoss(perMs float64) *Ship {
	if s == nil {
		return nil
	}
	s.dustLoss = perMs
	s.dustLossSet = true
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

// position is this frame's hull top-left: a landing path, a pausing
// drop, a drop, a liftoff, a climb, or the westbound fly-in, depending
// on how the ship was asked to fly.
func (s *Ship) position() (row, col int) {
	t := s.clock - s.hold
	if s.landSec > 0 {
		return LandPath(s.w, s.h, t, s.landSec)
	}
	if len(s.beats) > 0 {
		return DropBeatPath(s.w, s.h, t, s.beats)
	}
	if s.dropSec > 0 {
		return DropPath(s.w, s.h, t, s.dropSec)
	}
	if s.liftSec > 0 {
		return LiftPath(s.w, s.h, t, s.liftAt, s.liftSec)
	}
	if s.climbSec > 0 {
		return ClimbPath(s.w, s.h, t, s.climbSec)
	}
	if s.bobSet {
		return flightPathIn(s.w, s.h, t, s.flyInOrDefault(), s.bobPeriod, s.bobCells)
	}
	return flightPathIn(s.w, s.h, t, s.flyInOrDefault(), BobPeriodSeconds, BobAmplitudeCells)
}

// FlyIn retunes this one ship's westbound slide: the wing-to-park
// glide takes seconds. The number is the caller's, verbatim — zero
// parks the craft instantly. Unset, the slide takes the stock
// FlyInSeconds. Call before Start (and before Parked). Nil-safe.
func (s *Ship) FlyIn(seconds float64) *Ship {
	if s == nil {
		return nil
	}
	s.flySet = true
	s.flySec = seconds
	return s
}

// flyInOrDefault is the slide this ship flies: its own number, or the
// stock FlyInSeconds when unset.
func (s *Ship) flyInOrDefault() float64 {
	if s.flySet {
		return s.flySec
	}
	return FlyInSeconds
}

// Parked starts the clock at the fly-in park so the first frame is
// already center-stage, skipping any Hold. Nil-safe.
func (s *Ship) Parked() *Ship {
	if s == nil {
		return nil
	}
	s.clock = s.hold + s.flyInOrDefault()
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
	return flightPath(stageW, stageH, t, BobPeriodSeconds, BobAmplitudeCells)
}

// flightPath is FlightPath with the parked ride retuned: the same
// fly-in, then a ±cells sine with the given period.
func flightPath(stageW, stageH int, t, period float64, cells int) (row, col int) {
	return flightPathIn(stageW, stageH, t, FlyInSeconds, period, cells)
}

// flightPathIn is flightPath on any slide: the wing-to-park glide
// takes flyIn seconds — the caller's number, verbatim; at zero the
// craft opens parked.
func flightPathIn(stageW, stageH int, t, flyIn, period float64, cells int) (row, col int) {
	if t < 0 {
		t = 0
	}
	row = (stageH - BodyRows) / 2
	park := (stageW - BodyCols) / 2
	if t < flyIn {
		// ease-out cubic: fast off the wing, gentle into the park.
		p := t / flyIn
		eased := 1 - math.Pow(1-p, 3)
		return row, stageW + int(math.Round(eased*float64(park-stageW)))
	}
	return row - ParkBob(t-flyIn, period, cells), park
}

// ParkBob is how many cells above center the parked bobble rides at t
// seconds after the park: a ±cells sine with the given period. A
// non-positive period or amplitude — or time before the park — holds
// the ride level.
func ParkBob(t, period float64, cells int) int {
	if t < 0 || period <= 0 || cells <= 0 {
		return 0
	}
	return int(math.Round(float64(cells) * math.Sin(2*math.Pi*t/period)))
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

// ClimbPath is DropPath run the other way: fully off the bottom at
// t=0, fully off the top at t=seconds, centered horizontally. Time
// before the curtain clamps to the start. seconds<=0 snaps off the
// top.
func ClimbPath(stageW, stageH int, t, seconds float64) (row, col int) {
	if t < 0 {
		t = 0
	}
	col = (stageW - BodyCols) / 2
	start, end := stageH, -BodyRows
	if seconds <= 0 || t >= seconds {
		return end, col
	}
	p := t / seconds
	return start + int(math.Round(p*float64(end-start))), col
}

// DropBeatPath is the hull's top-left at t seconds of a pausing
// fall: fully off the top at t=0, fully off the bottom after the
// last beat, centered. Each beat drops, then holds. Drop distances
// share the span in proportion to each Drop duration, so equal
// drops travel equal rows. Empty beats or all-zero drops snap off
// the bottom. Time before the curtain clamps to the start.
func DropBeatPath(stageW, stageH int, t float64, beats []DropBeat) (row, col int) {
	if t < 0 {
		t = 0
	}
	col = (stageW - BodyCols) / 2
	start, finish := -BodyRows, stageH
	var totalDrop float64
	for _, b := range beats {
		if b.Drop > 0 {
			totalDrop += b.Drop
		}
	}
	if totalDrop <= 0 {
		return finish, col
	}
	span := float64(finish - start)
	elapsed := t
	traveled := 0.0
	for _, b := range beats {
		drop := b.Drop
		if drop < 0 {
			drop = 0
		}
		hold := b.Hold
		if hold < 0 {
			hold = 0
		}
		if drop > 0 && elapsed < drop {
			frac := elapsed / drop
			return start + int(math.Round((traveled+frac*drop)/totalDrop*span)), col
		}
		elapsed -= drop
		traveled += drop
		if elapsed < hold {
			return start + int(math.Round(traveled/totalDrop*span)), col
		}
		elapsed -= hold
	}
	return finish, col
}

// DropBeatHold is the index of the beat currently parked at t, or
// -1 while falling (and after the last beat).
func DropBeatHold(t float64, beats []DropBeat) int {
	if t < 0 {
		t = 0
	}
	elapsed := t
	for i, b := range beats {
		drop := b.Drop
		if drop < 0 {
			drop = 0
		}
		hold := b.Hold
		if hold < 0 {
			hold = 0
		}
		if elapsed < drop {
			return -1
		}
		elapsed -= drop
		if elapsed < hold {
			return i
		}
		elapsed -= hold
	}
	return -1
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

// throttleAt is booster strength at t seconds when the four stage
// offsets are set from t=0.
func throttleAt(t, at75, at50, at25, off float64) float64 {
	if t < 0 {
		t = 0
	}
	if t >= off {
		return 0
	}
	if t >= at25 {
		return 0.25
	}
	if t >= at50 {
		return 0.5
	}
	if t >= at75 {
		return 0.75
	}
	return 1
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

// LiftPath is the hull's top-left at t seconds of a liftoff that
// leaves the pad at `at` and takes `seconds` to clear the stage: the
// landing path played backwards, then a little further so the
// down-firing booster is gone too. The craft holds the horizon pad
// until lift-at, then rises on the mirrored ease (p^LandEasePower) —
// a slow, heavy crawl off the pad that rockets off the top — and is
// fully gone (hull and plume) by at+seconds. Time before the curtain
// clamps to the pad.
func LiftPath(stageW, stageH int, t, at, seconds float64) (row, col int) {
	if t < 0 {
		t = 0
	}
	col = (stageW - BodyCols) / 2
	start, end := LandPadRow(stageH), LiftGoneRow
	if t < at {
		return start, col
	}
	if seconds <= 0 || t >= at+seconds {
		return end, col
	}
	p := (t - at) / seconds
	eased := math.Pow(p, LandEasePower)
	return start + int(math.Round(eased*float64(end-start))), col
}

// igniteAt is booster strength at t seconds when the four ignition
// offsets are set from t=0 — throttleAt run backwards: cold, then ¼,
// ½, ¾, then full power from `full` on.
func igniteAt(t, at25, at50, at75, full float64) float64 {
	if t < 0 {
		t = 0
	}
	if t >= full {
		return 1
	}
	if t >= at75 {
		return 0.75
	}
	if t >= at50 {
		return 0.5
	}
	if t >= at25 {
		return 0.25
	}
	return 0
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
