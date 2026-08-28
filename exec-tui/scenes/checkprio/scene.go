// Package checkprio is the code scene behind the Interpreter's
// one-line call. The whole DANZIG check as one C-style function —
// check_for_higher_priority_jobs() — that walks every core set,
// pulls the twelfth word out of the data array
// (core_sets[i].data[11], the PRIORITY word), compares new against
// old, and whichever holds the highest priority wins the CPU. The
// scene reveals the function one line per RevealBeat over the Rose
// Pine floor, then walks it: a gold cursor steps the lines in
// execution order — old, the loop, the read, the compare, the win,
// the winner, the run — one caption per StepBeat, and rests forever
// on the run line. Core Sets Two imports Lines and the line marks so
// the very same code walks beside its scan.
package checkprio

import (
	"github.com/theprimeagen/apollo-11/exec-tui/components/code"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// The scene's clock.
const (
	// RevealBeat lands one line of the function.
	RevealBeat = 0.5
	// RevealHold keeps the finished card up before the walk.
	RevealHold = 2.0
	// StepBeat speaks one line of the walk.
	StepBeat = 3.0
)

// The captions around the walk.
const (
	CaptionReveal = "the DANZIG check as one function — what the interpreter asks between every op pair"
	CaptionHold   = "eight reads, one compare — the highest PRIORITY word owns the cpu"
)

// The line marks: where each beat of the function lives on the card.
const (
	LineName   = 0
	LineOld    = 2
	LineFor    = 3
	LineRead   = 5
	LineIf     = 6
	LineWin    = 8
	LineWinner = 9
	LineRun    = 12
)

// Lines is the function, C-style: for each core set read the twelfth
// word — data[11], the PRIORITY word — compare new against old, and
// the highest priority wins the CPU.
func Lines() []string {
	return []string{
		"check_for_higher_priority_jobs()",
		"{",
		"  old = -0",
		"  for (i = 0; i < 8; i++)",
		"  {",
		"    new = core_sets[i].data[11]",
		"    if (new > old)",
		"    {",
		"      old = new",
		"      winner = i",
		"    }",
		"  }",
		"  run(core_sets[winner])",
		"}",
	}
}

// WalkStep is one beat of the walk: the line the cursor rests on and
// the caption the stage speaks.
type WalkStep struct {
	Line int
	Text string
}

// WalkSteps is the walk in execution order.
func WalkSteps() []WalkStep {
	return []WalkStep{
		{LineOld, "old starts at -0 — no job has claimed the cpu yet"},
		{LineFor, "walk all eight core sets, one at a time"},
		{LineRead, "new is the set's data[11] — the twelfth word: the job's PRIORITY"},
		{LineIf, "compare the new word against the old"},
		{LineWin, "the higher priority wins — new becomes old"},
		{LineWinner, "and its core set becomes the winner"},
		{LineRun, "the walk done, the winner's job gets the cpu"},
	}
}

// WalkStart is when the cursor rises: the reveal plus its hold.
func WalkStart() float64 {
	return float64(len(Lines()))*RevealBeat + RevealHold
}

// StepAt is when the walk's step i speaks.
func StepAt(i int) float64 {
	return WalkStart() + float64(i)*StepBeat
}

// HoldStart is when the walk is done and the scene rests.
func HoldStart() float64 {
	return StepAt(len(WalkSteps()))
}

// Show is the Check Priority scene: one director on its own clock.
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

// Bill is the Check Priority scene as a one-scene screenplay.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "Check Priority", Scene: New()},
	}
}

// director owns the reveal, the walk, and the clock. The clock is
// its identity — a resize (Stop then Start) keeps it — while a fresh
// scene Start assembles a fresh director.
type director struct {
	clock  float64
	card   sprite.Sprite
	w, h   int
	staged bool
}

func newDirector() *director {
	return &director{card: code.New(code.LangPseudo, Lines()).Art()}
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

	cardX := (d.w - d.card.Width) / 2
	if cardX < 0 {
		cardX = 0
	}
	cardY := (d.h - 1 - d.card.Height) / 2
	if cardY < 1 {
		cardY = 1
	}
	revealed := int(d.clock/RevealBeat) + 1
	if revealed > d.card.Height {
		revealed = d.card.Height
	}
	for r := 0; r < revealed; r++ {
		for c := 0; c < d.card.Width; c++ {
			stage.Set(cardY+r, cardX+c, d.card.At(r, c))
		}
	}

	caption := CaptionReveal
	if d.clock >= WalkStart() {
		steps := WalkSteps()
		i := int((d.clock - WalkStart()) / StepBeat)
		if i >= len(steps) {
			i = len(steps) - 1
		}
		stage.Set(cardY+steps[i].Line, cardX-2, sprite.Cell{Ch: '▸', FG: code.Gold, BG: code.Base})
		caption = steps[i].Text
		if d.clock >= HoldStart() {
			caption = CaptionHold
		}
	}
	col := 2
	for _, ch := range caption {
		stage.Set(d.h-1, col, sprite.Cell{Ch: ch, FG: code.Muted, BG: code.Base})
		col++
	}
	return stage
}
