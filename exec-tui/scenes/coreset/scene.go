// Package coreset is the anatomy lesson: the Executive's memory unit
// torn open, every fact from Luminary099. Act one shows the core set
// panel and the VAC panel side by side, tops aligned, living jobs in
// the boxes — SERVICER holding a core set AND a VAC area, CHARIN a
// core set alone. Act two drains the pools: every box dissolves away
// one FadeBeat at a time, VACs first, until only CS1 stands. Act
// three relabels the survivor plain CORE SET — no number — and glides
// it to the top center. Act four builds the twelve-word bar under it,
// one word per WordBeat: MPAC through MPAC+6, MODE, LOC, BANKSET,
// PUSHLOC, PRIORITY — the exact ERASABLE_ASSIGNMENTS page-99 layout —
// each group in its own ink with its own sourced caption. Act five
// fades the rest, glides PRIORITY to center stage, and breaks the
// 15-bit word open: the top six bits are the job's priority, the low
// nine its VAC area address (EXECUTIVE.agc, VACFOUND: "STORE THE
// ADDRESS OF THE FIRST WORD OF IT IN THE LOW NINE BITS OF THE
// PRIORITY WORD"). The worked example is the real one: SERVICER at
// PRIO 20 working VAC1 at 400 — OCT 20400 in one word. The scene
// holds there until the cut.
package coreset

import (
	"fmt"

	"github.com/theprimeagen/apollo-11/exec-tui/components/pools"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	// UnitSeconds holds act one: the full memory unit.
	UnitSeconds = 4.0
	// FadeBeat is the stagger between one box's dissolve and the next.
	FadeBeat = 0.22
	// DissolveSeconds is one box's full shade-ramp burn to nothing.
	DissolveSeconds = 0.4
	// MoveSeconds is the survivor's glide to the top center.
	MoveSeconds = 1.5
	// WordBeat reveals one word of the twelve-word bar.
	WordBeat = 0.35
	// WordHold keeps the finished anatomy on stage before the zoom.
	WordHold = 3.0
	// ZoomSeconds covers the whole priority zoom: first FadeOutSeconds
	// of everything else burning away while PRIORITY holds its slot,
	// then the glide to center stage.
	ZoomSeconds = 1.4
	// FadeOutSeconds is the quarter second the rest gets to leave
	// before PRIORITY moves.
	FadeOutSeconds = 0.25

	// FadeSeconds covers the whole drain: fourteen dissolves — five
	// VAC boxes, the VAC title, seven core sets, the core title.
	FadeSeconds = 14*FadeBeat + DissolveSeconds

	// The act boundaries, cumulative.
	FadeStart    = UnitSeconds
	MoveStart    = FadeStart + FadeSeconds
	WordsStart   = MoveStart + MoveSeconds
	WordsSeconds = 12*WordBeat + WordHold
	ZoomStart    = WordsStart + WordsSeconds
	BitsStart    = ZoomStart + ZoomSeconds

	// The worked example, straight from the sources: SERVICER is
	// scheduled CA PRIO20 / TC FINDVAC, and VACFOUND packs VAC1's
	// address 400 into the low nine bits — OCT 20400 in one word.
	PrioOctal    = 0o20
	VACAddrOctal = 0o400
	PriorityWord = PrioOctal<<9 | VACAddrOctal
	PrioBitCount = 6
	VACBitCount  = 9

	// PrioInk dresses the priority field; VACAddrInk the VAC address.
	PrioInk    = 214
	VACAddrInk = 83

	// The act captions, one per beat of the lesson.
	CaptionUnit  = "every job holds a core set — the big vector jobs hold a VAC area too"
	CaptionFade  = "drain the pools — follow the one that stays"
	CaptionMove  = "one core set — twelve erasable words"
	CaptionWords = "DYNAMICALLY ALLOCATED CORE SETS FOR JOBS — ERASABLE_ASSIGNMENTS p.99"
	CaptionZoom  = "eleven words of workspace — one word runs the show"
	CaptionBits  = "one 15-bit word — OCT 20400: SERVICER at PRIO 20, working VAC1 at 400"

	dimInk        = 240
	panelGap      = 6
	shadeStep     = DissolveSeconds / 5
	zoomShadeStep = FadeOutSeconds / 5

	// The bit row's geometry: every bit bitPitch columns from its
	// neighbor and a fieldGapW gutter between the six priority bits
	// and the nine VAC bits, so both field labels seat on one line.
	bitPitch  = 4
	fieldGapW = 6
)

// Group is one logical run of the twelve words, its ink, and its
// sourced caption.
type Group struct {
	Name    string
	Words   int
	Ink     int
	Caption string
}

// Groups is the page-99 core set: MPAC through MPAC+6, then MODE,
// LOC, BANKSET, PUSHLOC, PRIORITY — twelve registers.
func Groups() []Group {
	return []Group{
		{Name: "MPAC", Words: 7, Ink: 87, Caption: "the multi-purpose accumulator — the job's scratch math space"},
		{Name: "MODE", Words: 1, Ink: 213, Caption: "+1 for TP, +0 for DP, -1 for vector"},
		{Name: "LOC", Words: 1, Ink: 220, Caption: "the location associated with the job"},
		{Name: "BANKSET", Words: 1, Ink: 75, Caption: "usually the BBANK setting"},
		{Name: "PUSHLOC", Words: 1, Ink: 141, Caption: "word of packed interpretive parameters"},
		{Name: "PRIORITY", Words: 1, Ink: PrioInk, Caption: "priority of present job and work area"},
	}
}

// wordLabel is the bar tag for word i of twelve.
func wordLabel(i int) string {
	switch {
	case i == 0:
		return "MPAC"
	case i < 7:
		return fmt.Sprintf("+%d", i)
	case i == 7:
		return "MODE"
	case i == 8:
		return "LOC"
	case i == 9:
		return "BANK"
	case i == 10:
		return "PUSH"
	default:
		return "PRIO"
	}
}

// wordGroup is the group word i belongs to.
func wordGroup(i int) Group {
	gs := Groups()
	switch {
	case i < 7:
		return gs[0]
	case i == 7:
		return gs[1]
	case i == 8:
		return gs[2]
	case i == 9:
		return gs[3]
	case i == 10:
		return gs[4]
	default:
		return gs[5]
	}
}

// groupFirstWord is the bar index whose reveal brings the group's
// caption with it.
var groupFirstWord = []int{0, 7, 8, 9, 10, 11}

// Show is the Core Set scene: one director component running the five
// acts on its own clock.
type Show struct {
	screenplay.Ensemble
}

// New is the scene, ready for its curtain.
func New() *Show {
	s := &Show{}
	s.Assemble = func() []screenplay.Component {
		return []screenplay.Component{newDirector()}
	}
	return s
}

// Bill is the Core Set scene as a one-scene screenplay.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "Core Set", Scene: New()},
	}
}

// fadeRect is one dissolve target of the drain: a panel box, or a
// panel's title row.
type fadeRect struct {
	vac   bool
	slot  int
	title bool
}

// fadeOrder is the drain: the VAC stack from the bottom up, its
// title, then the core sets from CS8 down to CS2, then the core
// title. CS1 is never listed — it survives.
func fadeOrder() []fadeRect {
	var out []fadeRect
	for i := 4; i >= 0; i-- {
		out = append(out, fadeRect{vac: true, slot: i})
	}
	out = append(out, fadeRect{vac: true, title: true})
	for i := 7; i >= 1; i-- {
		out = append(out, fadeRect{slot: i})
	}
	out = append(out, fadeRect{title: true})
	return out
}

// director owns the whole lesson: the two panels, the lone box, and
// the clock. The clock is its identity — a resize (Stop then Start)
// keeps it — while a fresh scene Start assembles a fresh director.
type director struct {
	core, vac *pools.Panel
	lone      *pools.Box
	clock     float64
	w, h      int
	staged    bool
}

func newDirector() *director {
	d := &director{
		core: pools.NewCoreSetPanel(),
		vac:  pools.NewVACPanel(),
		lone: pools.NewCoreSet(),
	}
	servicer := pools.Job{Name: "SERVICER", Prio: 20, Ink: 83}
	monitor := pools.Job{Name: "MONITOR", Prio: 26, Ink: 220}
	d.core.Add(servicer)
	d.core.Add(pools.Job{Name: "CHARIN", Prio: 30, Ink: 213})
	d.core.Add(monitor)
	d.vac.Add(servicer)
	d.vac.Add(monitor)
	d.lone.Set(pools.Job{Ink: 83})
	return d
}

func (d *director) Start(w, h int) {
	d.w, d.h = w, h
	d.staged = true
	cw, ch := d.core.Size()
	d.core.Start(cw, ch)
	vw, vh := d.vac.Size()
	d.vac.Start(vw, vh)
	d.lone.Start(pools.BoxW, pools.BoxH)
}

func (d *director) Update(dt float64) {
	if dt <= 0 {
		return
	}
	d.clock += dt
	d.core.Update(dt)
	d.vac.Update(dt)
	d.lone.Update(dt)
}

func (d *director) Stop() { d.staged = false }

func (d *director) Render() sprite.Sprite {
	if !d.staged || d.w < 1 || d.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(d.w, d.h)
	t := d.clock
	switch {
	case t < MoveStart:
		d.paintPanels(stage, t)
		if t < FadeStart {
			d.caption(stage, CaptionUnit)
		} else {
			d.caption(stage, CaptionFade)
		}
	case t < WordsStart:
		d.paintMove(stage, t)
		d.caption(stage, CaptionMove)
	case t < ZoomStart:
		d.paintAnatomy(stage, t)
		d.caption(stage, CaptionWords)
	case t < BitsStart:
		d.paintZoom(stage, t)
		d.caption(stage, CaptionZoom)
	default:
		d.paintBits(stage)
		d.caption(stage, CaptionBits)
	}
	return stage
}

// panelLayout is where the two panels sit: side by side, centered as
// a group, tops aligned.
func (d *director) panelLayout() (coreX, vacX, topY int) {
	cw, ch := d.core.Size()
	vw, vh := d.vac.Size()
	total := cw + panelGap + vw
	coreX = (d.w - total) / 2
	if coreX < 0 {
		coreX = 0
	}
	vacX = coreX + cw + panelGap
	maxH := ch
	if vh > maxH {
		maxH = vh
	}
	topY = (d.h - maxH) / 2
	if topY < 0 {
		topY = 0
	}
	return
}

// cs1Home is CS1's stage position — where the drain leaves off and
// the move begins.
func (d *director) cs1Home() (x, y int) {
	coreX, _, topY := d.panelLayout()
	ox, oy := d.core.Origin(0)
	return coreX + ox, topY + oy
}

// paintPanels draws the memory unit, dissolving the drained boxes
// once the fade act is running.
func (d *director) paintPanels(stage sprite.Sprite, t float64) {
	coreArt := d.core.Art()
	vacArt := d.vac.Art()
	if t >= FadeStart {
		for i, fr := range fadeOrder() {
			steps := stepsFor(t-FadeStart-float64(i)*FadeBeat, shadeStep)
			if steps == 0 {
				continue
			}
			art, panel := coreArt, d.core
			if fr.vac {
				art, panel = vacArt, d.vac
			}
			if fr.title {
				dissolve(art, 0, 0, art.Width, 1, steps)
				continue
			}
			x, y := panel.Origin(fr.slot)
			dissolve(art, x, y, pools.BoxW, pools.BoxH, steps)
		}
	}
	coreX, vacX, topY := d.panelLayout()
	sprite.Blit(stage, coreX, topY, coreArt)
	sprite.Blit(stage, vacX, topY, vacArt)
}

// paintMove glides the unnumbered survivor to the top center.
func (d *director) paintMove(stage sprite.Sprite, t float64) {
	p := ease((t - MoveStart) / MoveSeconds)
	fromX, fromY := d.cs1Home()
	toX, toY := d.loneHome()
	x := fromX + int(p*float64(toX-fromX))
	y := fromY + int(p*float64(toY-fromY))
	sprite.Blit(stage, x, y, d.lone.Render())
}

// loneHome is the parked box's top-center spot.
func (d *director) loneHome() (x, y int) {
	return (d.w - pools.BoxW) / 2, 1
}

// barGeometry is the twelve-word bar: word width, bar left, bar top.
func (d *director) barGeometry() (wordW, barX, barY int) {
	wordW = (d.w - 4) / 12
	if wordW < 6 {
		wordW = 6
	}
	if wordW > 9 {
		wordW = 9
	}
	barX = (d.w - 12*wordW) / 2
	if barX < 0 {
		barX = 0
	}
	_, y := d.loneHome()
	barY = y + pools.BoxH + 1
	return
}

// paintWord draws bar word i as a mini box in its group's ink.
func paintWord(stage sprite.Sprite, i, x, y, wordW int) {
	g := wordGroup(i)
	stage.Set(y, x, sprite.Cell{Ch: '╭', FG: g.Ink, BG: -1})
	stage.Set(y, x+wordW-1, sprite.Cell{Ch: '╮', FG: g.Ink, BG: -1})
	stage.Set(y+2, x, sprite.Cell{Ch: '╰', FG: g.Ink, BG: -1})
	stage.Set(y+2, x+wordW-1, sprite.Cell{Ch: '╯', FG: g.Ink, BG: -1})
	for c := 1; c < wordW-1; c++ {
		stage.Set(y, x+c, sprite.Cell{Ch: '─', FG: g.Ink, BG: -1})
		stage.Set(y+2, x+c, sprite.Cell{Ch: '─', FG: g.Ink, BG: -1})
	}
	stage.Set(y+1, x, sprite.Cell{Ch: '│', FG: g.Ink, BG: -1})
	stage.Set(y+1, x+wordW-1, sprite.Cell{Ch: '│', FG: g.Ink, BG: -1})
	putText(stage, y+1, x+1, clipTo(wordLabel(i), wordW-2), g.Ink)
}

// paintAnatomy parks the box and builds the bar and its captions,
// one word per beat.
func (d *director) paintAnatomy(stage sprite.Sprite, t float64) {
	lx, ly := d.loneHome()
	sprite.Blit(stage, lx, ly, d.lone.Render())
	wordW, barX, barY := d.barGeometry()
	shown := 0
	for i := 0; i < 12; i++ {
		if t < WordsStart+float64(i)*WordBeat {
			break
		}
		paintWord(stage, i, barX+i*wordW, barY, wordW)
		shown = i + 1
	}
	listY := barY + pools.BoxH + 1
	for g, first := range groupFirstWord {
		if shown <= first {
			continue
		}
		grp := Groups()[g]
		x := putText(stage, listY+g, barX, fmt.Sprintf("%-9s", grp.Name), grp.Ink)
		putText(stage, listY+g, x+1, grp.Caption, dimInk)
	}
}

// paintZoom burns everything but PRIORITY away over FadeOutSeconds —
// the word holding its bar slot the whole while — and only then
// glides it to center stage.
func (d *director) paintZoom(stage sprite.Sprite, t float64) {
	e := t - ZoomStart
	steps := stepsFor(e, zoomShadeStep)
	wordW, barX, barY := d.barGeometry()

	fading := sprite.New(d.w, d.h)
	lx, ly := d.loneHome()
	sprite.Blit(fading, lx, ly, d.lone.Render())
	for i := 0; i < 11; i++ {
		paintWord(fading, i, barX+i*wordW, barY, wordW)
	}
	listY := barY + pools.BoxH + 1
	for g := range groupFirstWord {
		grp := Groups()[g]
		x := putText(fading, listY+g, barX, fmt.Sprintf("%-9s", grp.Name), grp.Ink)
		putText(fading, listY+g, x+1, grp.Caption, dimInk)
	}
	dissolve(fading, 0, 0, d.w, d.h, steps)
	sprite.Blit(stage, 0, 0, fading)

	p := ease((e - FadeOutSeconds) / (ZoomSeconds - FadeOutSeconds))
	fromX, fromY := barX+11*wordW, barY
	toX, toY := d.prioHome(wordW)
	x := fromX + int(p*float64(toX-fromX))
	y := fromY + int(p*float64(toY-fromY))
	paintWord(stage, 11, x, y, wordW)
}

// prioHome is where the PRIORITY word parks for the bit breakdown.
func (d *director) prioHome(wordW int) (x, y int) {
	y = d.h/2 - 8
	if y < 4 {
		y = 4
	}
	return (d.w - wordW) / 2, y
}

// paintBits parks PRIORITY and breaks the 15-bit word open: six
// priority bits over nine VAC-address bits, octal digits under each
// group of three, and both field labels seated on one shared line —
// the row is paced wide enough (bitPitch per bit, fieldGapW between
// the fields) that neither label crowds the other.
func (d *director) paintBits(stage sprite.Sprite) {
	wordW, _, _ := d.barGeometry()
	px, py := d.prioHome(wordW)
	paintWord(stage, 11, px, py, wordW)

	bitRow := py + pools.BoxH + 2
	prioW := bitPitch*(PrioBitCount-1) + 1
	vacW := bitPitch*(VACBitCount-1) + 1
	vacX := prioW + fieldGapW
	bx := (d.w - (vacX + vacW)) / 2
	if bx < 0 {
		bx = 0
	}
	for i := 0; i < 15; i++ {
		bit := (PriorityWord >> (14 - i)) & 1
		x := bx + bitPitch*i
		ink := PrioInk
		if i >= PrioBitCount {
			x += fieldGapW - bitPitch + 1
			ink = VACAddrInk
		}
		stage.Set(bitRow, x, sprite.Cell{Ch: rune('0' + bit), FG: ink, BG: -1})
	}
	digits := fmt.Sprintf("%05o", PriorityWord)
	for g := 0; g < 5; g++ {
		mid := 3*g + 1
		x := bx + bitPitch*mid
		ink := PrioInk
		if 3*g >= PrioBitCount {
			x += fieldGapW - bitPitch + 1
			ink = VACAddrInk
		}
		stage.Set(bitRow+2, x, sprite.Cell{Ch: rune(digits[g]), FG: ink, BG: -1})
	}
	prioLabel := fmt.Sprintf("PRIORITY — OCT %o", PrioOctal)
	vacLabel := fmt.Sprintf("VAC ADDRESS — OCT %o", VACAddrOctal)
	putText(stage, bitRow+4, bx+(prioW-runeLen(prioLabel))/2, prioLabel, PrioInk)
	putText(stage, bitRow+4, bx+vacX+(vacW-runeLen(vacLabel))/2, vacLabel, VACAddrInk)
}

func (d *director) caption(stage sprite.Sprite, text string) {
	putText(stage, d.h-1, 2, text, dimInk)
}

// dissolve walks every painted cell of the rect steps rungs down the
// shade ramp; four rungs is gone.
func dissolve(sp sprite.Sprite, x, y, w, h, steps int) {
	if steps <= 0 {
		return
	}
	for r := y; r < y+h && r < sp.Height; r++ {
		if r < 0 {
			continue
		}
		for c := x; c < x+w && c < sp.Width; c++ {
			if c < 0 {
				continue
			}
			cell := sp.At(r, c)
			if cell.Transparent() {
				continue
			}
			for s := 0; s < steps; s++ {
				cell = sprite.DecrementShade(cell)
			}
			sp.Set(r, c, cell)
		}
	}
}

// stepsFor is how many ramp rungs a dissolve that started e seconds
// ago has burned, at one rung per step seconds. Before its beat, none.
func stepsFor(e, step float64) int {
	if e <= 0 {
		return 0
	}
	s := int(e/step) + 1
	if s > 6 {
		s = 6
	}
	return s
}

// ease is the repo's ease-out cubic, clamped to the glide.
func ease(p float64) float64 {
	if p <= 0 {
		return 0
	}
	if p >= 1 {
		return 1
	}
	q := 1 - p
	return 1 - q*q*q
}

func runeLen(s string) int { return len([]rune(s)) }

func clipTo(s string, room int) string {
	if room < 0 {
		room = 0
	}
	rs := []rune(s)
	if len(rs) > room {
		return string(rs[:room])
	}
	return s
}

func putText(sp sprite.Sprite, r, c int, text string, fg int) int {
	for _, ch := range text {
		sp.Set(r, c, sprite.Cell{Ch: ch, FG: fg, BG: -1})
		c++
	}
	return c
}
