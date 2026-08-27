// Package pools is the Executive's job memory as scene components:
// the eight core sets and the five VAC areas drawn as columns of
// bordered slot boxes. NewCoreSets is the core set view (CS1…CS8, two
// stacks of four, program alarm 1202 when the eighth fills) and
// NewVACs is the VAC view (VC1…VC5, one stack, alarm 1201). Add parks
// a job in the lowest free slot; Remove frees the lowest slot wearing
// that name. Every job carries an ink — the xterm-256 color its lanes
// wear on the other graphs — and its box wears it on the name·prio
// glyphs and the border, after an arrival flash of FlashSeconds. The
// title row counts busy/cap, turns red one slot from full, and a full
// pool raises its alarm chip white-on-red — the Executive has nowhere
// to put the next job.
package pools

import (
	"fmt"
	"unicode/utf8"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	// BoxW and BoxH are one slot box's full footprint, borders included.
	BoxW = 18
	BoxH = 3
	// FlashSeconds is how long a fresh arrival's border burns FlashInk.
	FlashSeconds = 0.45

	// DefaultInk is the sim's job green — the ink a job with none wears.
	DefaultInk = 83
	// DimInk dresses everything idle: free boxes, calm counts, the word free.
	DimInk = 240
	// LabelInk is the white the slot labels wear over a busy box.
	LabelInk = 255
	// TitleInk is the pool title's gray.
	TitleInk = 250
	// RedInk warns on the count from one-slot-from-full.
	RedInk = 196
	// FlashInk is the arrival border.
	FlashInk = 255
	// AlarmFG over AlarmBG is the full pool's program-alarm chip.
	AlarmFG = 255
	AlarmBG = 196
)

// Job is one Executive job occupying a slot. Ink is the xterm-256
// color that highlights its box — give related jobs the ink their
// lanes wear on the other graphs, or leave it zero for DefaultInk.
// A zero Prio prints the name alone.
type Job struct {
	Name string
	Prio int
	Ink  int
}

type slot struct {
	busy  bool
	job   Job
	flash float64
}

// View is one pool of job slots as a scene component. Jobs are the
// view's identity and survive Stop/Start (a resize); the stage does
// not.
type View struct {
	title   string
	alarm   string
	prefix  string
	columns int
	slots   []slot
	w, h    int
	staged  bool
}

// NewCoreSets is the core set view: the eight 12-word register blocks
// as two stacks of four, raising 1202 when the last one fills.
func NewCoreSets() *View {
	return &View{title: "CORE SETS", alarm: "1202", prefix: "CS", columns: 2, slots: make([]slot, 8)}
}

// NewVACs is the VAC view: the five 44-word vector accumulator areas
// as one stack, raising 1201 when the last one fills.
func NewVACs() *View {
	return &View{title: "VAC AREAS", alarm: "1201", prefix: "VC", columns: 1, slots: make([]slot, 5)}
}

// Add parks the job in the lowest free slot and reports which. A full
// pool refuses — that refusal is the moment the real Executive raised
// its alarm.
func (v *View) Add(j Job) (int, bool) {
	if v == nil {
		return 0, false
	}
	for i := range v.slots {
		if v.slots[i].busy {
			continue
		}
		v.slots[i] = slot{busy: true, job: j, flash: FlashSeconds}
		return i, true
	}
	return 0, false
}

// Remove frees the lowest slot wearing the name, leaving later copies
// of the same job in place. An unknown name is refused.
func (v *View) Remove(name string) bool {
	if v == nil {
		return false
	}
	for i := range v.slots {
		if v.slots[i].busy && v.slots[i].job.Name == name {
			v.slots[i] = slot{}
			return true
		}
	}
	return false
}

// Busy counts occupied slots.
func (v *View) Busy() int {
	if v == nil {
		return 0
	}
	n := 0
	for _, s := range v.slots {
		if s.busy {
			n++
		}
	}
	return n
}

// Cap is the pool size: 8 core sets, 5 VACs.
func (v *View) Cap() int {
	if v == nil {
		return 0
	}
	return len(v.slots)
}

// Full reports a pool with nowhere to put the next job.
func (v *View) Full() bool {
	return v != nil && len(v.slots) > 0 && v.Busy() == len(v.slots)
}

// JobAt is the job holding slot i; free slots and slots off the pool
// are refused.
func (v *View) JobAt(i int) (Job, bool) {
	if v == nil || i < 0 || i >= len(v.slots) || !v.slots[i].busy {
		return Job{}, false
	}
	return v.slots[i].job, true
}

// Start pins the stage. The slots carry across from any earlier run,
// so a resize never drops a job.
func (v *View) Start(w, h int) {
	if v == nil {
		return
	}
	v.w, v.h = w, h
	v.staged = true
}

// Update burns down the arrival flashes. dt <= 0 holds.
func (v *View) Update(dt float64) {
	if v == nil || dt <= 0 {
		return
	}
	for i := range v.slots {
		if v.slots[i].flash > 0 {
			v.slots[i].flash -= dt
			if v.slots[i].flash < 0 {
				v.slots[i].flash = 0
			}
		}
	}
}

// Render paints the pool centered on a stage-sized sprite. Before
// Start and after Stop the stage is empty; a stage smaller than the
// pool clips at the edges.
func (v *View) Render() sprite.Sprite {
	if v == nil || !v.staged || v.w < 1 || v.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(v.w, v.h)
	art := v.art()
	x := (v.w - art.Width) / 2
	y := (v.h - art.Height) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	sprite.Blit(stage, x, y, art)
	return stage
}

// Stop clears the staging; the jobs are the view's identity and stay.
func (v *View) Stop() {
	if v == nil {
		return
	}
	v.staged = false
}

// art is the title row over the box grid, exactly as big as it needs
// to be. The width never moves when the alarm appears: the title row
// is budgeted for its longest form up front.
func (v *View) art() sprite.Sprite {
	rows := (len(v.slots) + v.columns - 1) / v.columns
	gridW := v.columns * BoxW
	titleW := runeLen(v.title) + 1 + runeLen(fmt.Sprintf("%d/%d", len(v.slots), len(v.slots))) + 1 + runeLen("→ ") + runeLen(v.alarm)
	artW := gridW
	if titleW > artW {
		artW = titleW
	}
	sp := sprite.New(artW, 1+rows*BoxH)
	v.paintTitle(sp)
	gx := (artW - gridW) / 2
	for i := range v.slots {
		v.paintBox(sp, i, gx+(i/rows)*BoxW, 1+(i%rows)*BoxH)
	}
	return sp
}

func (v *View) paintTitle(sp sprite.Sprite) {
	busy := v.Busy()
	countInk := DimInk
	if busy >= len(v.slots)-1 {
		countInk = RedInk
	}
	x := putText(sp, 0, 0, v.title, TitleInk, -1)
	x = putText(sp, 0, x+1, fmt.Sprintf("%d/%d", busy, len(v.slots)), countInk, -1)
	if v.Full() {
		putText(sp, 0, x+1, "→ "+v.alarm, AlarmFG, AlarmBG)
	}
}

func (v *View) paintBox(sp sprite.Sprite, i, x, y int) {
	s := v.slots[i]
	border := DimInk
	if s.busy {
		border = inkOf(s.job)
		if s.flash > 0 {
			border = FlashInk
		}
	}
	sp.Set(y, x, sprite.Cell{Ch: '╭', FG: border, BG: -1})
	sp.Set(y, x+BoxW-1, sprite.Cell{Ch: '╮', FG: border, BG: -1})
	sp.Set(y+2, x, sprite.Cell{Ch: '╰', FG: border, BG: -1})
	sp.Set(y+2, x+BoxW-1, sprite.Cell{Ch: '╯', FG: border, BG: -1})
	for c := 1; c < BoxW-1; c++ {
		sp.Set(y, x+c, sprite.Cell{Ch: '─', FG: border, BG: -1})
		sp.Set(y+2, x+c, sprite.Cell{Ch: '─', FG: border, BG: -1})
	}
	sp.Set(y+1, x, sprite.Cell{Ch: '│', FG: border, BG: -1})
	sp.Set(y+1, x+BoxW-1, sprite.Cell{Ch: '│', FG: border, BG: -1})

	label := fmt.Sprintf("%-4s", fmt.Sprintf("%s%d", v.prefix, i+1))
	if !s.busy {
		putText(sp, y+1, x+1, label, DimInk, -1)
		putText(sp, y+1, x+5, "free", DimInk, -1)
		return
	}
	putText(sp, y+1, x+1, label, LabelInk, -1)
	putText(sp, y+1, x+5, jobText(s.job), inkOf(s.job), -1)
}

// jobText is name·prio squeezed into the box: the name gives way so
// the priority always survives.
func jobText(j Job) string {
	const room = BoxW - 2 - 4
	suffix := ""
	if j.Prio > 0 {
		suffix = fmt.Sprintf("·%d", j.Prio)
	}
	name := j.Name
	if max := room - runeLen(suffix); runeLen(name) > max {
		if max < 0 {
			max = 0
		}
		name = string([]rune(name)[:max])
	}
	return name + suffix
}

func inkOf(j Job) int {
	if j.Ink > 0 {
		return j.Ink
	}
	return DefaultInk
}

func putText(sp sprite.Sprite, r, c int, text string, fg, bg int) int {
	for _, ch := range text {
		sp.Set(r, c, sprite.Cell{Ch: ch, FG: fg, BG: bg})
		c++
	}
	return c
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }
