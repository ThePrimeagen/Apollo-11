package moonwalk

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// blockRows/blockCols is one crate's cell footprint.
	blockRows = astro.BlockPxH / 2
	blockCols = astro.BlockPxW
	// standBeat is the breath he takes on top of each crate.
	standBeat = 0.18
	// baseStand is the bow at the pole's base before the camera moves.
	baseStand = 1.4
	// endHold parks the camera on the rover before the loop restarts.
	endHold = 2.0
	// slideFPS is the fixed wobble rate of the locked-grip slide.
	slideFPS = 4.0
	// leapStretch makes the final leap longer than a crate hop.
	leapStretch = 1.5
	// clockEps soaks float noise off modulo-rebuilt clocks.
	clockEps = 1e-7
)

// groundRow is the ground line's stage row.
func groundRow(stageH int) int { return stageH - 2 }

// groundedY is the sprite top row that parks the boots on the floor.
func groundedY(stageH int) int { return groundRow(stageH) - astro.Rows }

// poleCol is the flagpole's world column.
func poleCol(stageW int) int { return stageW - 14 }

// poleTopRow is the pole tip's stage row for a config.
func poleTopRow(cfg Config, stageH int) int {
	top := groundRow(stageH) - cfg.PoleRows
	if top < 1 {
		top = 1
	}
	return top
}

// blockAX is the low crate's left world column; the tall stack stands
// a gap to its right.
func blockAX(stageW int) int { return stageW / 4 }
func blockBX(stageW int) int { return blockAX(stageW) + blockCols + 4 }

// roverX is the little rover prop's old parking spot; the module
// replaces it in the green pass.
func roverX(cfg Config, stageW int) int { return stageW + 1 }

// PoseGone is the frame after he boards the module: nothing to draw.
const PoseGone = sprite.Heading("")

// stackX is stack i's left world column (i = 0, 1, 2 — one, two,
// three crates high), marching toward the pole from BoxStart.
func stackX(cfg Config, stageW, i int) int { return 0 }

// landerX is the lunar module's left world column — beyond the
// viewport until the pan finds it.
func landerX(cfg Config, stageW int) int { return 0 }

// flagAt is the flag sprite's top row at t and whether it exists yet
// — it appears the moment the slide starts.
func flagAt(cfg Config, stageW, stageH int, t float64) (int, bool) { return 0, false }

// route is one loop of choreography, precomputed for a stage and a
// config: where every leg starts and lands, and when.
type route struct {
	grounded, yA, yB, yC, grabY              int
	x0, xJ1, xA, xB, xC, grabX, boardX       int
	hop1At, beat1At, hop2At, beat2At, leapAt float64
	topAt, slideAt, standAt, panAt, exitAt   float64
	boardAt, goneAt, cycle                   float64
	leapSec                                  float64
}

func routeFor(cfg Config, stageW, stageH int) route {
	r := route{grounded: groundedY(stageH)}
	r.yA = r.grounded - blockRows
	r.yB = r.grounded - 2*blockRows
	// One row above the tip: the fists close on the gold ball itself.
	r.grabY = poleTopRow(cfg, stageH) - 1
	r.x0 = -astro.Cols
	// He stands centered on a crate: crate center minus half a sprite.
	r.xA = blockAX(stageW) + blockCols/2 - astro.Cols/2
	r.xB = blockBX(stageW) + blockCols/2 - astro.Cols/2
	r.xJ1 = r.xA - 8
	r.grabX = poleCol(stageW) - astro.GripCol
	r.leapSec = cfg.JumpSeconds * leapStretch
	d1 := runFor(r.xJ1-r.x0, cfg.RunSpeed)
	r.hop1At = d1
	r.beat1At = r.hop1At + cfg.JumpSeconds
	r.hop2At = r.beat1At + standBeat
	r.beat2At = r.hop2At + cfg.JumpSeconds
	r.leapAt = r.beat2At + standBeat
	r.slideAt = r.leapAt + r.leapSec
	r.standAt = r.slideAt + cfg.SlideSeconds
	r.panAt = r.standAt + baseStand
	r.cycle = r.panAt + cfg.PanSeconds + endHold
	return r
}

// runFor is how long a ground sprint over cells takes; a degenerate
// stage still gets a beat so the piecewise clock never divides by zero.
func runFor(cells int, speed float64) float64 {
	if speed <= 0 {
		speed = 1
	}
	d := float64(cells) / speed
	if d < 0.05 {
		return 0.05
	}
	return d
}

// CycleSeconds is one full loop of the show.
func CycleSeconds(cfg Config, stageW, stageH int) float64 {
	return routeFor(cfg, stageW, stageH).cycle
}

func runPose(cfg Config, t float64) sprite.Heading {
	return astro.RunPoses[int(t*cfg.StrideFPS+clockEps)%len(astro.RunPoses)]
}

// arc is a parabolic lift: 0 at p=0 and p=1, peak in the middle.
func arc(peak int, p float64) int {
	return int(math.Round(float64(peak) * 4 * p * (1 - p)))
}

func lerp(a, b int, p float64) int {
	return a + int(math.Round(p*float64(b-a)))
}

// wrap folds t onto one cycle, clamping time before the curtain.
func wrap(t, cycle float64) float64 {
	if t < 0 {
		return 0
	}
	t = math.Mod(t, cycle)
	if t < 0 {
		t += cycle
	}
	return t
}

// timelineAt is the choreography: which pose plays and where its
// sprite's top-left sits in world coordinates at t.
func timelineAt(cfg Config, stageW, stageH int, t float64) (pose sprite.Heading, x, y int) {
	r := routeFor(cfg, stageW, stageH)
	t = wrap(t, r.cycle)
	switch {
	case t < r.hop1At: // run in from the left wing
		p := t / r.hop1At
		return runPose(cfg, t), lerp(r.x0, r.xJ1, p), r.grounded
	case t < r.beat1At: // hop onto the low crate
		p := (t - r.hop1At) / cfg.JumpSeconds
		return astro.PoseJump, lerp(r.xJ1, r.xA, p), lerp(r.grounded, r.yA, p) - arc(3, p)
	case t < r.hop2At: // a breath on top
		return astro.PoseStand, r.xA, r.yA
	case t < r.beat2At: // hop onto the tall stack
		p := (t - r.hop2At) / cfg.JumpSeconds
		return astro.PoseJump, lerp(r.xA, r.xB, p), lerp(r.yA, r.yB, p) - arc(3, p)
	case t < r.leapAt: // a breath, nice and high
		return astro.PoseStand, r.xB, r.yB
	case t < r.slideAt: // the leap onto the very tip of the pole
		p := (t - r.leapAt) / r.leapSec
		return astro.PoseJump, lerp(r.xB, r.grabX, p), lerp(r.yB, r.grabY, p) - arc(4, p)
	case t < r.standAt: // slide down while the flag goes up
		p := (t - r.slideAt) / cfg.SlideSeconds
		grip := astro.PolePoses[int((t-r.slideAt)*slideFPS+clockEps)%len(astro.PolePoses)]
		return grip, r.grabX, lerp(r.grabY, r.grounded, p)
	default: // stand at the base; the camera does the rest
		return astro.PoseStand, r.grabX, r.grounded
	}
}

// cameraAt is how far the viewport has panned right at t: zero through
// the action, an eased little move after the bow, parked on the rover
// until the loop restarts.
func cameraAt(cfg Config, stageW, stageH int, t float64) int {
	r := routeFor(cfg, stageW, stageH)
	t = wrap(t, r.cycle)
	if t < r.panAt {
		return 0
	}
	if cfg.PanSeconds <= 0 {
		return cfg.PanCols
	}
	p := (t - r.panAt) / cfg.PanSeconds
	if p > 1 {
		p = 1
	}
	eased := p * p * (3 - 2*p)
	return int(math.Round(eased * float64(cfg.PanCols)))
}

// flagRows is the flag sprite's cell height.
const flagRows = astro.FlagPxH / 2

// flagTopAt is the flag sprite's top row at t: parked by the base
// until the slide starts, then hoisted to just under the tip over
// FlagSeconds — up he goes down.
func flagTopAt(cfg Config, stageW, stageH int, t float64) int {
	r := routeFor(cfg, stageW, stageH)
	t = wrap(t, r.cycle)
	start := groundRow(stageH) - flagRows
	end := poleTopRow(cfg, stageH) + 1
	if t < r.slideAt {
		return start
	}
	p := (t - r.slideAt) / cfg.FlagSeconds
	if p > 1 {
		p = 1
	}
	return lerp(start, end, p)
}

// Frame renders the visible viewport of the show at t: the world —
// stars, ground, crates, pole, flag, rover, astronaut — drawn wide,
// then cropped to the camera.
func Frame(cfg Config, atlas *sprite.Atlas, stageW, stageH int, t float64) sprite.Sprite {
	if stageW < 1 {
		stageW = 1
	}
	if stageH < 1 {
		stageH = 1
	}
	worldW := stageW + cfg.PanCols
	world := sprite.New(worldW, stageH)
	paintSet(cfg, world, stageW)
	if atlas != nil {
		gr := groundRow(stageH)
		if block, ok := atlas.Frame(astro.Size, astro.PropBlock); ok {
			sprite.Blit(world, blockAX(stageW), gr-blockRows, block)
			sprite.Blit(world, blockBX(stageW), gr-blockRows, block)
			sprite.Blit(world, blockBX(stageW), gr-2*blockRows, block)
		}
		if flag, ok := atlas.Frame(astro.Size, astro.PropFlag); ok {
			sprite.Blit(world, poleCol(stageW)+1, flagTopAt(cfg, stageW, stageH, t), flag)
		}
		if rover, ok := atlas.Frame(astro.Size, astro.PropRover); ok {
			sprite.Blit(world, roverX(cfg, stageW), gr-astro.RoverPxH/2, rover)
		}
		pose, x, y := timelineAt(cfg, stageW, stageH, t)
		if sp, ok := atlas.Frame(astro.Size, pose); ok {
			sprite.Blit(world, x, y, sp)
		}
	}
	view := sprite.New(stageW, stageH)
	sprite.Blit(view, -cameraAt(cfg, stageW, stageH, t), 0, world)
	return view
}

// paintSet lays the still life across the whole world: a sparse
// starfield, the ground line dusted with regolith, and the flagpole
// with its gold ball tip.
func paintSet(cfg Config, world sprite.Sprite, stageW int) {
	gr := groundRow(world.Height)
	for i := 0; i < world.Width/6; i++ {
		col := (i*23 + 7) % world.Width
		row := (i*13 + 3) % maxInt(1, gr-3)
		world.Set(row, col, sprite.Cell{Ch: '·', FG: 240, BG: -1})
	}
	for c := 0; c < world.Width; c++ {
		world.Set(gr, c, sprite.Cell{Ch: '▀', FG: 245, BG: -1})
		if (c*7+gr)%13 == 0 {
			world.Set(gr+1, c, sprite.Cell{Ch: '·', FG: 240, BG: -1})
		}
	}
	pc := poleCol(stageW)
	top := poleTopRow(cfg, world.Height)
	for row := top; row < gr; row++ {
		world.Set(row, pc, sprite.Cell{Ch: '│', FG: 250, BG: -1})
	}
	world.Set(top-1, pc, sprite.Cell{Ch: '●', FG: 220, BG: -1})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
