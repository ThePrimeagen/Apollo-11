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
// The unexported inks shade the exported families: the near-black
// wing edges, the mottled underwing, the white's soft side, the
// beak's orange underside, the black pupil and gape, the dark claws.
const (
	BodyInk    = 94
	ShadowInk  = 52
	HeadInk    = 255
	BeakInk    = 220
	darkInk    = 58
	neutralInk = 236
	lightInk   = 137
	grayInk    = 250
	orangeInk  = 214
	rustInk    = 166
	torsoInk   = 235
	pupilInk   = 16
	clawInk    = 238
)

// SignatureInks are the plumage inks that belong to the eagle alone —
// its browns and golds. The flag's fade ramps never pass through any
// of them, so a scene can tell the bird from the field by ink alone.
// The deep shadow, the neutral dark and the whites stay off the list:
// the fading flag and the white stripes wear those too.
func SignatureInks() []int {
	return []int{BodyInk, BeakInk, darkInk, lightInk, orangeInk, rustInk, torsoInk, clawInk}
}

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

// art is the model, drawn facing left. It is half-block pixel art: a
// BodyCols×(2·BodyRows) pixel grid folded into terminal cells two
// pixels at a time.
var art = buildArt()

// BodyCols and BodyRows are the model's fixed canvas, in cells.
const (
	BodyCols = 62
	BodyRows = 16
)

// pixelInks maps one pixel letter of the grid to its ink. A '.' (or
// anything short of the row's end) is transparent sky.
var pixelInks = map[byte]int{
	'K': darkInk,
	'N': neutralInk,
	'B': BodyInk,
	'L': lightInk,
	'S': ShadowInk,
	'W': HeadInk,
	'G': grayInk,
	'Y': BeakInk,
	'O': orangeInk,
	'R': rustInk,
	'T': torsoInk,
	'E': pupilInk,
	'D': clawInk,
}

// pixelRows is the eagle, one letter per pixel, 2·BodyRows rows of
// BodyCols columns: the swooping bald eagle of the reference photo —
// both wings thrown high with fingered tips, the white head low on
// the leading edge with the gold hooked beak open, the gold legs
// thrust forward-down with spread talons, and the white tail fanned
// behind. Facing left.
var pixelRows = []string{
	`..............................................................`,
	`D...N....N..N......................................N.D.DD.....`,
	`.N...N....N..N....................................NN..N...D...`,
	`..NB..NN...NN....................................NB..NN..N...D`,
	`..KNBBNNNBBNNNK...............................KNNN..NN..NN..NN`,
	`...NNLLNBBNNNNNNN...........................NNBBNNN.BBN.NNN.N.`,
	`....KNBBNNNNBBNNNNK.......................KNNNNNNNKNNNNNNNNK..`,
	`.....KNNNNBBNNNNBBNNK...................NNNBBNNNBBBNNNBNN.....`,
	`.......NNNNNBBNNNNNNSS................KNNNNNNNBBNNNNKNNK......`,
	`.........KNNNNBBNNBBSSK..............NNNNNNNBBNNNNNBNNN.......`,
	`...........KKWWWNNNNNSSKNNNNNNNNNNNNKSNNNNBBNNNNKNNNNN........`,
	`.........WWWWWWWWWSSSSSKNSNNNNNNNNTTSSNNBBNNNNNBBNNN..........`,
	`.....YYWWWYEWWWWWWW.TTTTTTTTTTTTTTKSTTBBNNNNKNNNNN............`,
	`...YYYY.WWWWWWWWWW.TTTKTTTTNTTTTTTKSTTNNNBBBNNNN..............`,
	`..YEEEE.WWWWWWWWGSTTNTTTTKTTTKTT.NTT..NNKNNNNN................`,
	`..OY.R...WWWWWWGSTTKTTTNTTTKTT...TTNNNN..NNN..................`,
	`.....RR...WWWGGSTTNTTTKTTTNTTT.TTN..NNNNN.....................`,
	`............WGWGWBBEBBBBEBTT..................................`,
	`...........SBWGWBBBBBBBBBBBB..................................`,
	`...........TSTWTETTTETTTETTGWW................................`,
	`............SWBBBBBBBBBBBBTGWWWWW.............................`,
	`............TSTBBTTYBBKTTTT.GWWWWWW...........................`,
	`.............SNY.YY.BBBBBBT..GWWWWWWW.........................`,
	`.............TY..Y.TTN.TTTT....WWWGWWW........................`,
	`.............Y..Y..TT.T.TT......WWWGWWGWW.....................`,
	`............Y..Y.................WWWGWWGW.....................`,
	`........Y.Y.Y...Y.Y.Y.............WWGWWWG.W...................`,
	`.......D.D.D...D.D.D...............WW.W.W.WW..................`,
	`........D.D.D...D.D.D..................W...W..................`,
	`.........D.......D..D......................W..................`,
	`..............................................................`,
	`..............................................................`,
}

// buildArt folds the pixel grid into half-block cells: two pixels of
// one color are a plain field cell, two different colors are an
// upper-half block over the lower pixel's background, and a lone
// pixel keeps its other half transparent so the silhouette blends
// with whatever flies beneath. A malformed grid is a programmer
// error worth stopping for.
func buildArt() sprite.Sprite {
	if len(pixelRows) != 2*BodyRows {
		panic("eagle: pixel rows do not match 2×BodyRows")
	}
	sp := sprite.New(BodyCols, BodyRows)
	for r := 0; r < BodyRows; r++ {
		for c := 0; c < BodyCols; c++ {
			up, upOK := pixelAt(2*r, c)
			low, lowOK := pixelAt(2*r+1, c)
			switch {
			case !upOK && !lowOK:
			case upOK && lowOK && up == low:
				sp.Set(r, c, sprite.Cell{Ch: ' ', FG: -1, BG: up})
			case upOK && lowOK:
				sp.Set(r, c, sprite.Cell{Ch: '▀', FG: up, BG: low})
			case upOK:
				sp.Set(r, c, sprite.Cell{Ch: '▀', FG: up, BG: -1})
			default:
				sp.Set(r, c, sprite.Cell{Ch: '▄', FG: low, BG: -1})
			}
		}
	}
	return sp
}

// pixelAt reads one pixel of the grid: its ink and whether it paints.
func pixelAt(pr, c int) (int, bool) {
	row := pixelRows[pr]
	if len(row) > BodyCols {
		panic("eagle: pixel row wider than BodyCols")
	}
	if c >= len(row) {
		return 0, false
	}
	ch := row[c]
	if ch == '.' || ch == ' ' {
		return 0, false
	}
	ink, ok := pixelInks[ch]
	if !ok {
		panic("eagle: unknown pixel letter " + string(ch))
	}
	return ink, ok
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
