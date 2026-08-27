// Package eagle is the very large bald eagle as a scene component:
// white head low on the leading edge, gold hooked beak, gold talons
// reaching under the body, dark brown wings spread high and wide, and
// the white tail fanned out behind. It enters off the right wing of
// the stage and flies across to exit off the left over CrossSeconds,
// gliding on a shallow swoop with a gentle bob. Delay holds it off
// stage first, so a scene can finish its own beat — a fade, a title —
// before the flyover. The flight clock rides across restarts, so a
// resize never replays the crossing.
package eagle

import (
	"math"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// The eagle's xterm-256 inks: the dark brown wing, its darker shadow
// and outline, the white head and tail, and the gold beak and talons.
const (
	BodyInk   = 94
	ShadowInk = 52
	HeadInk   = 255
	BeakInk   = 220
	eyeInk    = 16
)

// DefaultCrossSeconds is the stock crossing: off one wing and off the
// other in twelve seconds.
const DefaultCrossSeconds = 12.0

// The glide: a shallow swoop that dips DipRows at center stage, plus
// a BobAmp-row bob at BobHz to keep the soar alive.
const (
	DipRows = 2.0
	BobHz   = 0.7
	BobAmp  = 1.0
)

// art is the model, drawn facing left. Transparent cells are spaces;
// inkFor assigns the materials by glyph and region.
var art = buildArt()

// BodyCols and BodyRows are the model's fixed canvas.
const (
	BodyCols = 62
	BodyRows = 16
)

// artRows is the eagle, row by row. Rows shorter than BodyCols are
// padded with transparent sky on the right.
var artRows = []string{
	`    ▄█▙▄█▖                                        ▗█▄▟█▄`,
	`   ▄██▛▟██▙▄                                    ▄▟██▙▛██▄▄`,
	`  ▄███▓▓▓▓██▄▄                                ▄▄██▓▓▓▓██████▄▄`,
	` ▄██▓▓▓▓▓▓▓▓██▄                              ▄██▓▓▓▓▓▓▓█████▄`,
	` ███▓▓▓▓▓▓▓▓▓▓██▄                          ▄██▓▓▓▓▓▓▓▓▓█████`,
	` ▀██▓▓▓▓▓▓▓▓▓▓▓▓██▄                      ▄██▓▓▓▓▓▓▓▓▓▓███▀`,
	`  ▀██▓▓▓▓▓▓▓▓▓▓▓▓▓██▄                  ▄██▓▓▓▓▓▓▓▓▓▓███▀`,
	`   ▀███▓▓▓▓▓▓▓▓▓▓▓▓▓██▄              ▄██▓▓▓▓▓▓▓▓▓▓███▀`,
	`  ▄███▄ ▀███▓▓▓▓▓▓▓▓▓▓▓██▄▄        ▄██▓▓▓▓▓▓▓▓▓███▀`,
	`▄▟██●██▙▄ ▀▀███▓▓▓▓▓▓▓▓▓▓▓███▄▄  ▄███▓▓▓▓▓▓▓▓███▀`,
	`◣█████████▄▄▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓████████▓▓▓▓████▀`,
	` ▀▀▀▀▀█████▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓▓▓▓▓▓▓▛▀▜███▙▄`,
	`       ▀▀█▓▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▓▓▓▓▓▓▀▀▀    ▝▜█████▙▖`,
	`         ▀▓▓▐█▌ ▐█▌ ▀▀▀▀▀▀              ▄▟██████▛▘`,
	`           ▐█▌ ▐█▌                     ▀▛▀▀▀▀▀`,
	`          ◢◤▘ ◢◤▘`,
}

// The material regions, one column span per row: the white head with
// its gold beak leading on the left, the white tail fanning behind on
// the right, and the gold legs and talons reaching below. Everything
// else is wing and body.
var (
	headSpans  = map[int][2]int{8: {2, 6}, 9: {1, 8}, 10: {0, 9}, 11: {1, 5}}
	tailSpans  = map[int][2]int{11: {40, 47}, 12: {40, 49}, 13: {40, 49}, 14: {39, 45}}
	talonSpans = map[int][2]int{13: {12, 18}, 14: {11, 17}, 15: {10, 16}}
	beakCells  = map[[2]int]bool{{9, 0}: true, {10, 0}: true}
)

func spanHas(spans map[int][2]int, r, c int) bool {
	s, ok := spans[r]
	return ok && c >= s[0] && c <= s[1]
}

// inkFor assigns one cell's material: gold beak, legs and talons, the
// black eye riding the white head, white head and tail, and brown
// wings with the dark shadow of their underbody and outline nubs.
func inkFor(r, c int, ch rune) sprite.Cell {
	switch {
	case beakCells[[2]int{r, c}], spanHas(talonSpans, r, c):
		return sprite.Cell{Ch: ch, FG: BeakInk, BG: -1}
	case ch == '●':
		return sprite.Cell{Ch: ch, FG: eyeInk, BG: HeadInk}
	case spanHas(headSpans, r, c), spanHas(tailSpans, r, c):
		return sprite.Cell{Ch: ch, FG: HeadInk, BG: -1}
	case ch == '▒':
		return sprite.Cell{Ch: ch, FG: ShadowInk, BG: -1}
	case ch == '▖' || ch == '▗' || ch == '▘' || ch == '▝':
		return sprite.Cell{Ch: ch, FG: ShadowInk, BG: -1}
	default:
		return sprite.Cell{Ch: ch, FG: BodyInk, BG: -1}
	}
}

// buildArt paints the rows onto the fixed canvas. A row longer than
// the canvas is a programmer error worth stopping for.
func buildArt() sprite.Sprite {
	sp := sprite.New(BodyCols, BodyRows)
	if len(artRows) != BodyRows {
		panic("eagle: art rows do not match BodyRows")
	}
	for r, row := range artRows {
		runes := []rune(row)
		if len(runes) > BodyCols {
			panic("eagle: art row wider than BodyCols")
		}
		for c, ch := range runes {
			if ch == ' ' {
				continue
			}
			sp.Set(r, c, inkFor(r, c, ch))
		}
	}
	return sp
}

// Art is the eagle model alone, on its own canvas — for tests,
// previews and anyone composing it by hand.
func Art() sprite.Sprite {
	out := sprite.New(BodyCols, BodyRows)
	sprite.Blit(out, 0, 0, art)
	return out
}

// Eagle is the component. Start pins the stage, Update runs the
// flight clock, Render lays the model at this instant of the
// crossing, Stop strikes the stage.
type Eagle struct {
	delay  float64
	cross  float64
	clock  float64
	body   sprite.Sprite
	w, h   int
	staged bool
}

// New binds an eagle that flies the default crossing with no delay.
// Nothing is built until Start.
func New() *Eagle {
	return &Eagle{cross: DefaultCrossSeconds}
}

// Delay holds the eagle off stage for the first seconds of the scene.
// Call before Start. Nil-safe.
func (e *Eagle) Delay(seconds float64) *Eagle {
	if e == nil {
		return nil
	}
	if seconds > 0 {
		e.delay = seconds
	}
	return e
}

// Cross sets how long the flyover takes, off one wing and off the
// other. Durations of zero or less keep the stock crossing. Nil-safe.
func (e *Eagle) Cross(seconds float64) *Eagle {
	if e == nil {
		return nil
	}
	if seconds > 0 {
		e.cross = seconds
	}
	return e
}

// Start builds the model for a w×h stage. The flight clock is not
// touched: a resize never replays the crossing.
func (e *Eagle) Start(w, h int) {
	if e == nil {
		return
	}
	e.w, e.h = w, h
	e.body = Art()
	e.staged = true
}

// Update advances the flight. dt <= 0 holds the eagle still.
func (e *Eagle) Update(dt float64) {
	if e == nil || dt <= 0 {
		return
	}
	e.clock += dt
}

// Render lays the eagle at its instant of the crossing on a
// stage-sized sprite. Before the delay, after the crossing, before
// Start and after Stop the sky is clear.
func (e *Eagle) Render() sprite.Sprite {
	if e == nil || !e.staged || e.w < 1 || e.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(e.w, e.h)
	t := e.clock - e.delay
	if t < 0 || e.cross <= 0 {
		return stage
	}
	p := t / e.cross
	if p >= 1 {
		return stage
	}
	span := float64(e.w + BodyCols)
	x := e.w - int(p*span)
	row := (e.h-BodyRows)/2 +
		int(math.Round(DipRows*math.Sin(p*math.Pi))) +
		int(math.Round(BobAmp*math.Sin(2*math.Pi*BobHz*t)))
	sprite.Blit(stage, x, row, e.body)
	return stage
}

// Stop strikes the stage. The clock stays, so the next Start picks
// the crossing up mid-flight.
func (e *Eagle) Stop() {
	if e == nil {
		return
	}
	e.body = sprite.Sprite{}
	e.staged = false
}
