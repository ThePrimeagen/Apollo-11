// Package pools is the Executive's job memory as scene components,
// split into two layers. The Box is one memory slot on its own — the
// bordered pill that turns on and off, wears a job's ink on its text
// and border after a FlashInk arrival flash, and carries a little
// label ("CS1", "VC3", or the unnumbered "CORE SET"). The Panel is
// the composite: NewCoreSetPanel is eight core set boxes (CS1…CS8,
// two stacks of four, program alarm 1202 when the eighth fills) and
// NewVACPanel is five VAC boxes (VC1…VC5, one stack, alarm 1201),
// each a real Box component the panel starts at BoxW×BoxH and blits
// onto its grid under a title-and-count row. Add parks a job in the
// lowest free box; Remove frees the lowest box wearing that name.
// Box(i), Origin(i) and Size() hand scenes the live components and
// the grid geometry, so a screenplay can choreograph the boxes
// itself.
package pools

import (
	"fmt"
	"strings"
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
	// TitleInk is the panel title's gray.
	TitleInk = 250
	// RedInk warns on the count from one-slot-from-full.
	RedInk = 196
	// FlashInk is the arrival border.
	FlashInk = 255
	// AlarmFG over AlarmBG is the full panel's program-alarm chip.
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

// Box is one memory slot as its own component: label, an on/off job,
// and the arrival flash. The job is the box's identity and survives
// Stop/Start; the stage does not.
type Box struct {
	label  string
	busy   bool
	job    Job
	flash  float64
	w, h   int
	staged bool
}

// NewBox is a slot wearing the given label.
func NewBox(label string) *Box { return &Box{label: label} }

// NewCoreSet is the unnumbered core set box — the one the breakdown
// scene parks at the top of the stage.
func NewCoreSet() *Box { return NewBox("CORE SET") }

// NewVAC is the unnumbered VAC box.
func NewVAC() *Box { return NewBox("VAC") }

// Label is the box's fixed label.
func (b *Box) Label() string {
	if b == nil {
		return ""
	}
	return b.label
}

// Set turns the box on with the job and lights the arrival flash. A
// busy box swaps the job in place.
func (b *Box) Set(j Job) {
	if b == nil {
		return
	}
	b.busy, b.job, b.flash = true, j, FlashSeconds
}

// Clear turns the box off.
func (b *Box) Clear() {
	if b == nil {
		return
	}
	b.busy, b.job, b.flash = false, Job{}, 0
}

// Busy reports an occupied box.
func (b *Box) Busy() bool { return b != nil && b.busy }

// Job is the occupant; a free box is refused.
func (b *Box) Job() (Job, bool) {
	if b == nil || !b.busy {
		return Job{}, false
	}
	return b.job, true
}

// Start pins the stage.
func (b *Box) Start(w, h int) {
	if b == nil {
		return
	}
	b.w, b.h = w, h
	b.staged = true
}

// Update burns down the arrival flash. dt <= 0 holds.
func (b *Box) Update(dt float64) {
	if b == nil || dt <= 0 {
		return
	}
	if b.flash > 0 {
		b.flash -= dt
		if b.flash < 0 {
			b.flash = 0
		}
	}
}

// Render paints the pill centered on a stage-sized sprite. Before
// Start and after Stop the stage is empty.
func (b *Box) Render() sprite.Sprite {
	if b == nil || !b.staged || b.w < 1 || b.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(b.w, b.h)
	x := (b.w - BoxW) / 2
	y := (b.h - BoxH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	b.paint(stage, x, y)
	return stage
}

// Stop clears the staging; the job stays.
func (b *Box) Stop() {
	if b == nil {
		return
	}
	b.staged = false
}

// paint draws the pill with its top-left at (x, y) on sp.
func (b *Box) paint(sp sprite.Sprite, x, y int) {
	border := DimInk
	if b.busy {
		border = inkOf(b.job)
		if b.flash > 0 {
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

	label := fmt.Sprintf("%-4s", b.label)
	if !strings.HasSuffix(label, " ") {
		label += " "
	}
	room := BoxW - 2 - runeLen(label)
	if !b.busy {
		putText(sp, y+1, x+1, label, DimInk, -1)
		putText(sp, y+1, x+1+runeLen(label), clip("free", room), DimInk, -1)
		return
	}
	putText(sp, y+1, x+1, label, LabelInk, -1)
	putText(sp, y+1, x+1+runeLen(label), jobText(b.job, room), inkOf(b.job), -1)
}

// jobText is name·prio squeezed into room cells: the name gives way
// so the priority always survives.
func jobText(j Job, room int) string {
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

func clip(s string, room int) string {
	if room < 0 {
		room = 0
	}
	if runeLen(s) > room {
		return string([]rune(s)[:room])
	}
	return s
}

func inkOf(j Job) int {
	if j.Ink > 0 {
		return j.Ink
	}
	return DefaultInk
}

// Panel is one pool of Box components under a title-and-count row:
// the composite view. The boxes are the panel's identity and survive
// Stop/Start (a resize); the stage does not.
type Panel struct {
	title   string
	alarm   string
	columns int
	boxes   []*Box
	w, h    int
	staged  bool
}

// NewCoreSetPanel is the eight core sets: CS1…CS8 as two stacks of
// four, raising 1202 when the last one fills.
func NewCoreSetPanel() *Panel {
	return newPanel("CORE SETS", "1202", "CS", 8, 2)
}

// NewVACPanel is the five vector accumulator areas: VC1…VC5 as one
// stack, raising 1201 when the last one fills.
func NewVACPanel() *Panel {
	return newPanel("VAC AREAS", "1201", "VC", 5, 1)
}

func newPanel(title, alarm, prefix string, n, columns int) *Panel {
	p := &Panel{title: title, alarm: alarm, columns: columns}
	for i := 0; i < n; i++ {
		p.boxes = append(p.boxes, NewBox(fmt.Sprintf("%s%d", prefix, i+1)))
	}
	return p
}

// Add parks the job in the lowest free box and reports which. A full
// panel refuses — the moment the real Executive raised its alarm.
func (p *Panel) Add(j Job) (int, bool) {
	if p == nil {
		return 0, false
	}
	for i, b := range p.boxes {
		if b.Busy() {
			continue
		}
		b.Set(j)
		return i, true
	}
	return 0, false
}

// Remove frees the lowest box wearing the name, leaving later copies
// of the same job in place. An unknown name is refused.
func (p *Panel) Remove(name string) bool {
	if p == nil {
		return false
	}
	for _, b := range p.boxes {
		if j, ok := b.Job(); ok && j.Name == name {
			b.Clear()
			return true
		}
	}
	return false
}

// Busy counts occupied boxes.
func (p *Panel) Busy() int {
	if p == nil {
		return 0
	}
	n := 0
	for _, b := range p.boxes {
		if b.Busy() {
			n++
		}
	}
	return n
}

// Cap is the pool size: 8 core sets, 5 VACs.
func (p *Panel) Cap() int {
	if p == nil {
		return 0
	}
	return len(p.boxes)
}

// Full reports a panel with nowhere to put the next job.
func (p *Panel) Full() bool {
	return p != nil && len(p.boxes) > 0 && p.Busy() == len(p.boxes)
}

// JobAt is the job holding slot i; free slots and slots off the grid
// are refused.
func (p *Panel) JobAt(i int) (Job, bool) {
	if b := p.Box(i); b != nil {
		return b.Job()
	}
	return Job{}, false
}

// Box hands out the live component at slot i, nil off the grid.
func (p *Panel) Box(i int) *Box {
	if p == nil || i < 0 || i >= len(p.boxes) {
		return nil
	}
	return p.boxes[i]
}

// rows is how tall each stack stands.
func (p *Panel) rows() int {
	return (len(p.boxes) + p.columns - 1) / p.columns
}

// Size is the panel's art: the title row over the box grid, budgeted
// up front for the title's longest (alarmed) form so the art never
// jumps when the chip appears.
func (p *Panel) Size() (w, h int) {
	if p == nil {
		return 0, 0
	}
	gridW := p.columns * BoxW
	titleW := runeLen(p.title) + 1 + runeLen(fmt.Sprintf("%d/%d", len(p.boxes), len(p.boxes))) + 1 + runeLen("→ ") + runeLen(p.alarm)
	if titleW > gridW {
		gridW = titleW
	}
	return gridW, 1 + p.rows()*BoxH
}

// Origin is box i's top-left within the panel art. Slots off the grid
// are refused with the zero corner.
func (p *Panel) Origin(i int) (x, y int) {
	if p == nil || i < 0 || i >= len(p.boxes) {
		return 0, 0
	}
	artW, _ := p.Size()
	gx := (artW - p.columns*BoxW) / 2
	rows := p.rows()
	return gx + (i/rows)*BoxW, 1 + (i%rows)*BoxH
}

// Start pins the stage and raises each box on its own BoxW×BoxH
// stage. The boxes carry across from any earlier run, so a resize
// never drops a job.
func (p *Panel) Start(w, h int) {
	if p == nil {
		return
	}
	p.w, p.h = w, h
	p.staged = true
	for _, b := range p.boxes {
		b.Start(BoxW, BoxH)
	}
}

// Update forwards the clock to every box. dt <= 0 holds.
func (p *Panel) Update(dt float64) {
	if p == nil {
		return
	}
	for _, b := range p.boxes {
		b.Update(dt)
	}
}

// Render paints the title and every box centered on a stage-sized
// sprite. Before Start and after Stop the stage is empty; a stage
// smaller than the panel clips at the edges.
func (p *Panel) Render() sprite.Sprite {
	if p == nil || !p.staged || p.w < 1 || p.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(p.w, p.h)
	art := p.Art()
	x := (p.w - art.Width) / 2
	y := (p.h - art.Height) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	sprite.Blit(stage, x, y, art)
	return stage
}

// Art is the panel at its exact Size: the title row over the grid,
// each box blitted at its Origin. Scenes that place the panel
// themselves start it at Size() and take this straight.
func (p *Panel) Art() sprite.Sprite {
	if p == nil {
		return sprite.Sprite{}
	}
	w, h := p.Size()
	sp := sprite.New(w, h)
	p.paintTitle(sp)
	for i, b := range p.boxes {
		x, y := p.Origin(i)
		sprite.Blit(sp, x, y, b.Render())
	}
	return sp
}

// Stop clears the staging on the panel and its boxes; the jobs stay.
func (p *Panel) Stop() {
	if p == nil {
		return
	}
	p.staged = false
	for _, b := range p.boxes {
		b.Stop()
	}
}

func (p *Panel) paintTitle(sp sprite.Sprite) {
	busy := p.Busy()
	countInk := DimInk
	if busy >= len(p.boxes)-1 {
		countInk = RedInk
	}
	x := putText(sp, 0, 0, p.title, TitleInk, -1)
	x = putText(sp, 0, x+1, fmt.Sprintf("%d/%d", busy, len(p.boxes)), countInk, -1)
	if p.Full() {
		putText(sp, 0, x+1, "→ "+p.alarm, AlarmFG, AlarmBG)
	}
}

func putText(sp sprite.Sprite, r, c int, text string, fg, bg int) int {
	for _, ch := range text {
		sp.Set(r, c, sprite.Cell{Ch: ch, FG: fg, BG: bg})
		c++
	}
	return c
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }
