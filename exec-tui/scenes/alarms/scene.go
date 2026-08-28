// Package alarms is the allocation lesson as two C-style pseudo
// functions, walked to both of the landing's famous codes.
// find_free_core_set() loops the eight core sets, pulls the in-use
// word out of the data array — core_sets[i].data[11], the PRIORITY
// word, -0 when the set is free — continues past busy sets, returns
// the first free index, and at the bottom of the loop throws
// error 1202. find_free_vac_area() is the same walk over the five
// VAC areas — vac_areas[i].data[0] is the use word, 0 when a job
// claimed it, its own address when free — and throws 1201. Each
// function reveals one line per RevealBeat, walks a happy pass (a
// free slot found, its index returned), rests, then walks the full
// pool — every set busy, the loop falls off its end, the throw line
// burns alarm red under a PROG ALARM chip. The scene ends holding
// the VAC card and naming both codes.
package alarms

import (
	"fmt"

	"github.com/theprimeagen/apollo-11/exec-tui/components/code"
	"github.com/theprimeagen/apollo-11/exec-tui/components/pools"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// The scene's clock.
const (
	// RevealBeat lands one line of a function.
	RevealBeat = 0.5
	// RevealHold keeps a finished card up before its happy walk.
	RevealHold = 1.5
	// HappyBeat speaks one step of a happy walk.
	HappyBeat = 1.6
	// HappyHold rests on the returned index before the full pool.
	HappyHold = 2.5
	// FullBeat hammers one step of the everything-busy walk.
	FullBeat = 0.55
	// AlarmHold keeps a thrown alarm burning.
	AlarmHold = 3.5
	// FadeSeconds is the dark beat between the two functions.
	FadeSeconds = 1.0
)

// The alarm dress: the thrown line's ink and the chip's colors.
const (
	AlarmInk = 196
	AlarmFG  = pools.AlarmFG
	AlarmBG  = pools.AlarmBG
)

// The captions around the walks.
const (
	CaptionCore  = "a new job needs a core set — eight of them, the first free one wins"
	CaptionVAC   = "an interpretive job needs a vac area too — five of them"
	CaptionFinal = "no core sets → 1202 · no vac areas → 1201 — the landing's two alarms"
)

// The line marks: both functions share one shape.
const (
	LineName     = 0
	LineFor      = 2
	LineRead     = 4
	LineIf       = 5
	LineContinue = 6
	LineReturn   = 7
	LineThrow    = 9
)

// CoreLines is the core-set allocation: PRIORITY doubles as the
// in-use word, -0 means free, and an exhausted loop is alarm 1202.
func CoreLines() []string {
	return []string{
		"find_free_core_set()",
		"{",
		"  for (i = 0; i < 8; i++)",
		"  {",
		"    in_use = core_sets[i].data[11]",
		"    if (in_use != -0)",
		"      continue",
		"    return i",
		"  }",
		"  throw new error(1202)",
		"}",
	}
}

// VACLines is the vac-area allocation: the area's first word is the
// use word, 0 when claimed, its own address when free, and an
// exhausted loop is alarm 1201.
func VACLines() []string {
	return []string{
		"find_free_vac_area()",
		"{",
		"  for (i = 0; i < 5; i++)",
		"  {",
		"    in_use = vac_areas[i].data[0]",
		"    if (in_use == 0)",
		"      continue",
		"    return i",
		"  }",
		"  throw new error(1201)",
		"}",
	}
}

// Probe is one slot as the walk reads it: the word the read pulls
// out, and whether that word means free.
type Probe struct {
	Value string
	Free  bool
}

// Step is one beat of a walk: the line the cursor rests on and the
// caption the stage speaks.
type Step struct {
	Line int
	Text string
}

// CoreHappy is the happy pool: two busy sets, then a free one.
func CoreHappy() []Probe {
	return []Probe{
		{Value: "20400"},
		{Value: "32000"},
		{Value: "-0", Free: true},
	}
}

// CoreFull is the unhappy pool: all eight sets busy.
func CoreFull() []Probe {
	return []Probe{
		{Value: "20400"}, {Value: "21000"}, {Value: "26454"}, {Value: "30000"},
		{Value: "32000"}, {Value: "20454"}, {Value: "24000"}, {Value: "21400"},
	}
}

// VACHappy is the happy pool: one claimed area, then a free one
// holding its own address.
func VACHappy() []Probe {
	return []Probe{
		{Value: "0"},
		{Value: "454", Free: true},
	}
}

// VACFull is the unhappy pool: all five areas claimed.
func VACFull() []Probe {
	return []Probe{{Value: "0"}, {Value: "0"}, {Value: "0"}, {Value: "0"}, {Value: "0"}}
}

// scan replays one allocation walk over a pool: a read per slot, a
// continue past every busy one, a return on the first free one, and
// — when the loop runs out — the throw.
func scan(pool []Probe, readFmt, busyText string, ret func(Probe, int) string, throwText string) []Step {
	var out []Step
	for i, p := range pool {
		out = append(out, Step{LineRead, fmt.Sprintf(readFmt, i, p.Value)})
		if p.Free {
			return append(out, Step{LineReturn, ret(p, i)})
		}
		out = append(out, Step{LineContinue, busyText})
	}
	return append(out, Step{LineThrow, throwText})
}

// CoreScan walks a core-set pool as find_free_core_set would.
func CoreScan(pool []Probe) []Step {
	return scan(pool,
		"core_sets[%d].data[11] = %s",
		"in_use != -0 — a job holds this set · continue",
		func(p Probe, i int) string {
			return fmt.Sprintf("%s — free · return %d: the new job moves in", p.Value, i)
		},
		"every core set busy — 1202: NO CORE SETS AVAILABLE")
}

// VACScan walks a vac-area pool as find_free_vac_area would.
func VACScan(pool []Probe) []Step {
	return scan(pool,
		"vac_areas[%d].data[0] = %s",
		"in_use == 0 — a job claimed this area · continue",
		func(p Probe, i int) string {
			return fmt.Sprintf("%s — its own address: free · return %d", p.Value, i)
		},
		"every vac area claimed — 1201: NO VAC AREAS AVAILABLE")
}

// The act boundaries, derived from the walks.

// revealSpan is a card's reveal plus its rest.
func revealSpan(lines []string) float64 {
	return float64(len(lines))*RevealBeat + RevealHold
}

// CoreHappyStart is when the core happy walk's first step speaks.
func CoreHappyStart() float64 { return revealSpan(CoreLines()) }

// CoreFullStart is when the everything-busy core walk begins.
func CoreFullStart() float64 {
	return CoreHappyStart() + float64(len(CoreScan(CoreHappy())))*HappyBeat + HappyHold
}

// CoreAlarmAt is when the core loop falls off its end — 1202.
func CoreAlarmAt() float64 {
	return CoreFullStart() + float64(len(CoreScan(CoreFull()))-1)*FullBeat
}

// CoreEnd is when the core act leaves the stage.
func CoreEnd() float64 { return CoreAlarmAt() + FullBeat + AlarmHold }

// VACStart is when the vac card begins its reveal.
func VACStart() float64 { return CoreEnd() + FadeSeconds }

// VACHappyStart is when the vac happy walk's first step speaks.
func VACHappyStart() float64 { return VACStart() + revealSpan(VACLines()) }

// VACFullStart is when the everything-claimed vac walk begins.
func VACFullStart() float64 {
	return VACHappyStart() + float64(len(VACScan(VACHappy())))*HappyBeat + HappyHold
}

// VACAlarmAt is when the vac loop falls off its end — 1201.
func VACAlarmAt() float64 {
	return VACFullStart() + float64(len(VACScan(VACFull()))-1)*FullBeat
}

// VACEnd is when the scene settles into its final hold.
func VACEnd() float64 { return VACAlarmAt() + FullBeat + AlarmHold }

// Show is the Alarms scene: one director on its own clock.
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

// Bill is the Alarms scene as a one-scene screenplay.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "Alarms", Scene: New()},
	}
}

// act is one function's whole performance: its card (calm and
// alarmed), its walks, its chip, and its caption.
type act struct {
	card    sprite.Sprite
	alarmed sprite.Sprite
	happy   []Step
	full    []Step
	chip    string
	caption string
}

func newAct(lines []string, happy, full []Step, chip, caption string) act {
	alarmed := code.New(code.LangPseudo, lines)
	alarmed.Mark(LineThrow, 0, len([]rune(lines[LineThrow])), AlarmInk)
	return act{
		card:    code.New(code.LangPseudo, lines).Art(),
		alarmed: alarmed.Art(),
		happy:   happy,
		full:    full,
		chip:    chip,
		caption: caption,
	}
}

// director owns the two acts and the clock. The clock is its
// identity — a resize (Stop then Start) keeps it — while a fresh
// scene Start assembles a fresh director.
type director struct {
	clock  float64
	core   act
	vac    act
	w, h   int
	staged bool
}

func newDirector() *director {
	return &director{
		core: newAct(CoreLines(), CoreScan(CoreHappy()), CoreScan(CoreFull()), "PROG ALARM 1202", CaptionCore),
		vac:  newAct(VACLines(), VACScan(VACHappy()), VACScan(VACFull()), "PROG ALARM 1201", CaptionVAC),
	}
}

func (d *director) Start(w, h int) {
	d.w, d.h = w, h
	d.staged = true
}

func (d *director) Update(dt float64) {
	if dt <= 0 {
		return
	}
	d.clock += dt
}

func (d *director) Stop() { d.staged = false }

func (d *director) Render() sprite.Sprite {
	if !d.staged || d.w < 1 || d.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(d.w, d.h)
	for r := 0; r < d.h; r++ {
		for c := 0; c < d.w; c++ {
			stage.Set(r, c, sprite.Cell{Ch: ' ', FG: -1, BG: code.Base})
		}
	}

	caption := ""
	switch {
	case d.clock < CoreEnd():
		caption = d.paintAct(stage, d.core, d.clock)
	case d.clock < VACStart():
		caption = CaptionVAC
	default:
		caption = d.paintAct(stage, d.vac, d.clock-VACStart())
		if d.clock >= VACEnd() {
			caption = CaptionFinal
		}
	}

	col := 2
	for _, ch := range caption {
		stage.Set(d.h-1, col, sprite.Cell{Ch: ch, FG: code.Muted, BG: code.Base})
		col++
	}
	return stage
}

// paintAct plays one function's performance at its local clock e and
// hands back the caption the stage speaks.
func (d *director) paintAct(stage sprite.Sprite, a act, e float64) string {
	cardX := (d.w - a.card.Width) / 2
	if cardX < 0 {
		cardX = 0
	}
	cardY := (d.h - 1 - a.card.Height) / 2
	if cardY < 1 {
		cardY = 1
	}

	happyStart := float64(a.card.Height)*RevealBeat + RevealHold
	happyEnd := happyStart + float64(len(a.happy))*HappyBeat
	fullStart := happyEnd + HappyHold

	cursor := -1
	caption := a.caption
	card := a.card
	alarmed := false
	switch {
	case e < happyStart:
	case e < fullStart:
		i := int((e - happyStart) / HappyBeat)
		if i >= len(a.happy) {
			i = len(a.happy) - 1
		}
		cursor = a.happy[i].Line
		caption = a.happy[i].Text
	default:
		i := int((e - fullStart) / FullBeat)
		if i >= len(a.full) {
			i = len(a.full) - 1
		}
		cursor = a.full[i].Line
		caption = a.full[i].Text
		alarmed = i == len(a.full)-1
	}
	if alarmed {
		card = a.alarmed
	}

	revealed := int(e/RevealBeat) + 1
	if revealed > card.Height {
		revealed = card.Height
	}
	for r := 0; r < revealed; r++ {
		for c := 0; c < card.Width; c++ {
			stage.Set(cardY+r, cardX+c, card.At(r, c))
		}
	}
	if cursor >= 0 {
		stage.Set(cardY+cursor, cardX-2, sprite.Cell{Ch: '▸', FG: code.Gold, BG: code.Base})
	}
	if alarmed {
		for i, ch := range " " + a.chip + " " {
			stage.Set(cardY+card.Height+1, cardX+i, sprite.Cell{Ch: ch, FG: AlarmFG, BG: AlarmBG})
		}
	}
	return caption
}
