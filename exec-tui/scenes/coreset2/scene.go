// Package coreset2 is the scan lesson — Core Sets Two. It opens on
// exactly the frame the Core Set breakdown held at its cut: the
// PRIORITY word broken into six priority bits over nine VAC-address
// bits, OCT 20400, every label in place. Act two burns that word away
// while the roster lands: six real Executive jobs, one per JobBeat,
// each wearing its own ink and its own priority — six different
// claims on the CPU. Act three sweeps the stage clean. Act four
// creates the pseudocode by showing it: the EJSCAN loop reveals one
// line per CodeBeat — walk every core set, read the PRIORITY word,
// keep the highest — sourced from EXECUTIVE.agc (EJSCAN L437+, EJ1
// L492-L499, CHANJOB L251+). Act five redraws five core sets with the
// full word math beside each box — the priority plus the VAC address,
// 000 for the NOVAC jobs whose low nine bits stay empty — and walks
// the scan one comparison per CompareBeat, the cursor on the examined
// set, the arrow on the leader, until the third box down (RR READ at
// 32000) is SELECTED. Act six is the redo with a duplicated job:
// three SERVICER copies at the same PRIO 20 climbing the real VAC
// addresses 400, 454, 530 — every equal-priority compare falls to the
// VAC address, the newest copy is always selected, and the passed-over
// copies are tagged as the stubs they become (the engine of the 1202
// leak). The scene holds there until the cut.
package coreset2

import (
	"fmt"
	"strings"

	"github.com/theprimeagen/apollo-11/exec-tui/components/pools"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	// HoldSeconds keeps the pickup — scene one's final frame — on
	// stage before the lesson moves.
	HoldSeconds = 3.0
	// JobBeat lands one roster job.
	JobBeat = 0.55
	// JobsHold keeps the finished roster readable.
	JobsHold = 2.0
	// PickupFadeSeconds burns the held word away as the roster arrives.
	PickupFadeSeconds = 0.5
	// ClearSeconds dissolves the roster to an empty stage.
	ClearSeconds = 1.0
	// CodeBeat reveals one line of the scan loop.
	CodeBeat = 0.5
	// CodeHold keeps the finished loop on stage before the scan.
	CodeHold = 2.5
	// CodeFadeSeconds burns the code away as scan one opens.
	CodeFadeSeconds = 0.4
	// SlotBeat redraws one core set of a scan pass.
	SlotBeat = 0.5
	// CompareBeat speaks one step of the scan.
	CompareBeat = 1.4
	// WinnerHold keeps pass one's SELECTED arrow parked before the redo.
	WinnerHold = 3.5
	// SwapSeconds dissolves pass one's boxes ahead of the redo.
	SwapSeconds = 1.0

	// The act boundaries, cumulative.
	JobsStart      = HoldSeconds
	JobsSeconds    = 6*JobBeat + JobsHold
	ClearStart     = JobsStart + JobsSeconds
	CodeStart      = ClearStart + ClearSeconds
	CodeSeconds    = 6*CodeBeat + CodeHold
	ScanOneStart   = CodeStart + CodeSeconds
	BuildSeconds   = 5 * SlotBeat
	StepsSeconds   = 5 * CompareBeat
	SelectOneStart = ScanOneStart + BuildSeconds + StepsSeconds
	ScanTwoStart   = SelectOneStart + WinnerHold + SwapSeconds
	SelectTwoStart = ScanTwoStart + BuildSeconds + StepsSeconds

	// LeadMark rides the current best while the scan walks; the
	// SelectedMark replaces it when the walk is done; the StubTag
	// names what a passed-over copy becomes.
	LeadMark     = "◀ best"
	SelectedMark = "◀ SELECTED"
	StubTag      = "stub — never finishes"

	// The act captions, one per beat of the lesson.
	CaptionJobs      = "every job carries a priority — six jobs, six different claims on the CPU"
	CaptionClear     = "one CPU — every time a job ends or sleeps, the Executive must pick again"
	CaptionCode      = "EXAMINE EACH PRIORITY REGISTER TO FIND THE JOB OF HIGHEST ACTIVE PRIORITY — EXECUTIVE.agc, EJSCAN"
	CaptionScanOne   = "EJSCAN walks the core sets — the FULL word: priority bits + VAC address bits"
	CaptionWinnerOne = "the third core set down holds the highest word — RR READ·32 runs (CHANJOB)"
	CaptionScanTwo   = "one job, three copies — each fresh copy claims the next free VAC: a higher address"
	CaptionWinnerTwo = "equal PRIO 20 — the higher VAC address wins: the newest copy is selected, the old ones starve"

	dimInk    = 240
	codeInk   = 252
	wordInk   = 255
	titleInk  = 250
	cursorInk = 255
	stubInk   = 196

	pickupShadeStep = PickupFadeSeconds / 5
	clearShadeStep  = ClearSeconds / 5
	codeShadeStep   = CodeFadeSeconds / 5
	swapShadeStep   = SwapSeconds / 5

	// The scan column's geometry: the word math sits mathGap right of
	// a box, the lead marker markGap right of the math.
	mathGap = 2
	markGap = 2
	mathW   = 16 // "20 + 400 = 20400"
)

// Slot is one core set of a scan pass: the box label, the job holding
// it (Free marks an empty set), and the octal VAC-area address packed
// in the low nine bits of its PRIORITY word — 0 for the NOVAC jobs,
// whose low bits stay empty.
type Slot struct {
	Label   string
	Job     pools.Job
	VACAddr int
	Free    bool
}

// Word is the slot's full PRIORITY word: the job's priority — its
// octal spelling, PRIO 20 is OCT 20 — over the nine VAC-address bits
// (VACFOUND: "STORE THE ADDRESS OF THE FIRST WORD OF IT IN THE LOW
// NINE BITS OF THE PRIORITY WORD").
func (s Slot) Word() int {
	if s.Free {
		return 0
	}
	return octOf(s.Job.Prio)<<9 | s.VACAddr
}

// octOf reads a decimal-spelled AGC priority as the octal it is:
// PRIO 20 is OCT 20.
func octOf(p int) int {
	v := 0
	for _, d := range fmt.Sprintf("%d", p) {
		v = v*8 + int(d-'0')
	}
	return v
}

// Jobs is the roster act: six real Executive jobs, six different
// priorities, each in the ink its lanes wear on the other graphs.
func Jobs() []pools.Job {
	return []pools.Job{
		{Name: "SELFCHK", Prio: 1, Ink: 245},
		{Name: "SERVICER", Prio: 20, Ink: 83},
		{Name: "1/GYRO", Prio: 21, Ink: 141},
		{Name: "MONITOR", Prio: 26, Ink: 220},
		{Name: "CHARIN", Prio: 30, Ink: 213},
		{Name: "RR READ", Prio: 32, Ink: 87},
	}
}

// ScanOne is the first pass: five busy core sets, five different
// priorities. SERVICER works VAC1 at 400 and MONITOR VAC2 at 454; the
// rest are NOVAC — address 000. The highest word is the third one
// down: RR READ at 32000.
func ScanOne() []Slot {
	jobs := Jobs()
	return []Slot{
		{Label: "CS1", Job: jobs[1], VACAddr: 0o400},
		{Label: "CS2", Job: jobs[2]},
		{Label: "CS3", Job: jobs[5]},
		{Label: "CS4", Job: jobs[3], VACAddr: 0o454},
		{Label: "CS5", Job: jobs[4]},
	}
}

// ScanTwo is the redo with a duplicated job: three SERVICER copies,
// all at PRIO 20, climbing the real VAC-area addresses 400, 454, 530
// — each fresh copy claimed the next free VAC, so the newest always
// carries the highest word. The last set is free.
func ScanTwo() []Slot {
	jobs := Jobs()
	return []Slot{
		{Label: "CS1", Job: jobs[1], VACAddr: 0o400},
		{Label: "CS2", Job: jobs[0]},
		{Label: "CS3", Job: jobs[1], VACAddr: 0o454},
		{Label: "CS4", Job: jobs[1], VACAddr: 0o530},
		{Label: "CS5", Free: true},
	}
}

// CodeLines is the scan written as the code the audience watches run:
// loop over the core sets, read each PRIORITY word, keep the highest
// (EXECUTIVE.agc — EJSCAN L437+, EJ1 L492-L499, CHANJOB L251+).
func CodeLines() []string {
	return []string{
		"best ← -0                     # BUF+1: nothing found yet",
		"for each of the 8 core sets:  # EJSCAN",
		"  word ← PRIORITY             # priority bits + VAC address bits",
		"  if word > best:             # EJ1 — the FULL word decides",
		"    best ← word               # the new high priority",
		"run the job holding best      # CHANJOB",
	}
}

// Step is one beat of the scan: the slot examined, where the lead
// sits after it, and the line the stage speaks.
type Step struct {
	Slot int
	Best int
	Text string
}

// Steps replays EJSCAN over the slots: the first busy set seeds the
// lead, every later busy set compares FULL words (EJ1), an identical
// word keeps the earlier find (CCS on -0 proceeds with the search),
// equal priorities fall to the VAC address — the newer copy — and a
// free set (PRIORITY -0) is skipped.
func Steps(slots []Slot) []Step {
	var out []Step
	best := -1
	for i, sl := range slots {
		st := Step{Slot: i}
		switch {
		case sl.Free:
			st.Text = fmt.Sprintf("%s — PRIORITY -0 · free · skipped", sl.Label)
		case best < 0:
			st.Text = fmt.Sprintf("%s — word %05o · the first busy set leads", sl.Label, sl.Word())
			best = i
		default:
			bs := slots[best]
			switch {
			case sl.Word() == bs.Word():
				st.Text = fmt.Sprintf("%s — %05o = %05o · tie — the earlier find keeps it", sl.Label, sl.Word(), bs.Word())
			case octOf(sl.Job.Prio) == octOf(bs.Job.Prio) && sl.Word() > bs.Word():
				st.Text = fmt.Sprintf("%s — PRIO %d = %d · VAC %03o > %03o · the newer copy leads",
					sl.Label, sl.Job.Prio, bs.Job.Prio, sl.VACAddr, bs.VACAddr)
				best = i
			case octOf(sl.Job.Prio) == octOf(bs.Job.Prio):
				st.Text = fmt.Sprintf("%s — PRIO %d = %d · VAC %03o < %03o · %s keeps the lead",
					sl.Label, sl.Job.Prio, bs.Job.Prio, sl.VACAddr, bs.VACAddr, bs.Label)
			case sl.Word() > bs.Word():
				st.Text = fmt.Sprintf("%s — %05o > %05o · takes the lead", sl.Label, sl.Word(), bs.Word())
				best = i
			default:
				st.Text = fmt.Sprintf("%s — %05o < %05o · %s keeps the lead", sl.Label, sl.Word(), bs.Word(), bs.Label)
			}
		}
		st.Best = best
		out = append(out, st)
	}
	return out
}

// Winner is the slot the scan selects — the lead after the last step,
// -1 when every set is free (DUMMYJOB idles).
func Winner(slots []Slot) int {
	steps := Steps(slots)
	if len(steps) == 0 {
		return -1
	}
	return steps[len(steps)-1].Best
}

// Show is the Core Sets Two scene: one director component running the
// six acts on its own clock.
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

// Bill is the Core Sets Two scene as a one-scene screenplay.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "Core Sets Two", Scene: New()},
	}
}

// director owns the whole lesson and the clock. The clock is its
// identity — a resize (Stop then Start) keeps it — while a fresh
// scene Start assembles a fresh director.
type director struct {
	clock  float64
	w, h   int
	staged bool
}

func newDirector() *director { return &director{} }

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
	t := d.clock
	switch {
	case t < JobsStart:
		d.paintPickup(stage, 0)
		d.caption(stage, coreset.CaptionBits)
	case t < ClearStart:
		d.paintPickup(stage, stepsFor(t-JobsStart, pickupShadeStep))
		d.paintJobs(stage, t, 0)
		d.caption(stage, CaptionJobs)
	case t < CodeStart:
		d.paintJobs(stage, t, stepsFor(t-ClearStart, clearShadeStep))
		d.caption(stage, CaptionClear)
	case t < ScanOneStart:
		d.paintCode(stage, t, 0)
		d.caption(stage, CaptionCode)
	case t < ScanTwoStart:
		d.paintCode(stage, t, stepsFor(t-ScanOneStart, codeShadeStep))
		burn := stepsFor(t-(SelectOneStart+WinnerHold), swapShadeStep)
		d.paintScan(stage, t, ScanOne(), ScanOneStart, CaptionScanOne, CaptionWinnerOne, false, burn)
	default:
		d.paintScan(stage, t, ScanTwo(), ScanTwoStart, CaptionScanTwo, CaptionWinnerTwo, true, 0)
	}
	return stage
}

// paintPickup is scene one's held bits frame, burned down the shade
// ramp once the roster act begins.
func (d *director) paintPickup(stage sprite.Sprite, steps int) {
	if steps == 0 {
		coreset.PaintBits(stage, d.w, d.h)
		return
	}
	art := sprite.New(d.w, d.h)
	coreset.PaintBits(art, d.w, d.h)
	dissolve(art, 0, 0, d.w, d.h, steps)
	sprite.Blit(stage, 0, 0, art)
}

// paintJobs lands the roster one job per JobBeat — each row flashing
// white on arrival, then settling into the job's ink, the priority
// spelled beside the name. steps > 0 is the clear act's burn.
func (d *director) paintJobs(stage sprite.Sprite, t float64, steps int) {
	art := sprite.New(d.w, d.h)
	jobs := Jobs()
	top := (d.h - (2*len(jobs) - 1)) / 2
	if top < 1 {
		top = 1
	}
	nameX := (d.w - 21) / 2
	if nameX < 0 {
		nameX = 0
	}
	for i, j := range jobs {
		at := JobsStart + float64(i)*JobBeat
		if t < at {
			break
		}
		ink := j.Ink
		if t-at < pools.FlashSeconds {
			ink = pools.FlashInk
		}
		y := top + 2*i
		putText(art, y, nameX, j.Name, ink)
		putText(art, y, nameX+12, "PRIO", dimInk)
		putText(art, y, nameX+17, fmt.Sprintf("%2d", j.Prio), coreset.PrioInk)
	}
	dissolve(art, 0, 0, d.w, d.h, steps)
	sprite.Blit(stage, 0, 0, art)
}

// paintCode reveals the scan loop one line per CodeBeat, centered and
// double-spaced, the code bright over dim sourced comments — the two
// bit fields tinted in their scene-one inks. steps > 0 burns it away
// as scan one opens.
func (d *director) paintCode(stage sprite.Sprite, t float64, steps int) {
	art := sprite.New(d.w, d.h)
	lines := CodeLines()
	widest := 0
	for _, l := range lines {
		if n := runeLen(l); n > widest {
			widest = n
		}
	}
	x := (d.w - widest) / 2
	if x < 0 {
		x = 0
	}
	top := (d.h - (2*len(lines) - 1)) / 2
	if top < 1 {
		top = 1
	}
	for i, line := range lines {
		if t < CodeStart+float64(i)*CodeBeat {
			break
		}
		paintCodeLine(art, top+2*i, x, line)
	}
	dissolve(art, 0, 0, d.w, d.h, steps)
	sprite.Blit(stage, 0, 0, art)
}

// paintCodeLine writes one loop line: code bright, the # comment dim,
// and the priority/VAC field names tinted in their inks.
func paintCodeLine(sp sprite.Sprite, y, x int, line string) {
	hash := strings.Index(line, "#")
	for i, ch := range []rune(line) {
		ink := codeInk
		if hash >= 0 && i >= hash {
			ink = dimInk
		}
		sp.Set(y, x+i, sprite.Cell{Ch: ch, FG: ink, BG: -1})
	}
	tint(sp, y, x, line, "priority bits", coreset.PrioInk)
	tint(sp, y, x, line, "VAC address bits", coreset.VACAddrInk)
}

// tint recolors one substring of an already-painted line.
func tint(sp sprite.Sprite, y, x int, line, sub string, ink int) {
	idx := strings.Index(line, sub)
	if idx < 0 {
		return
	}
	start := runeLen(line[:idx])
	for i, ch := range []rune(sub) {
		sp.Set(y, x+start+i, sprite.Cell{Ch: ch, FG: ink, BG: -1})
	}
}

// scanLayout is the scan column's frame: the box column's left edge
// and its top row.
func (d *director) scanLayout() (boxX, top int) {
	total := pools.BoxW + mathGap + mathW + markGap + runeLen(StubTag)
	boxX = (d.w - total) / 2
	if boxX < 2 {
		boxX = 2
	}
	top = (d.h-5*pools.BoxH)/2 - 1
	if top < 3 {
		top = 3
	}
	return
}

// paintScan plays one scan pass: the core sets redraw one per
// SlotBeat with the word math beside each, the steps speak one per
// CompareBeat — the cursor on the examined set, the arrow riding the
// lead — and the walk ends with the winner SELECTED. Pass two tags
// the passed-over copies of the winner's job as stubs and holds; a
// positive burn dissolves the whole pass (pass one leaving the stage).
func (d *director) paintScan(stage sprite.Sprite, t float64, slots []Slot, start float64, scanCap, winCap string, dupes bool, burn int) {
	e := t - start
	if e < 0 {
		return
	}
	art := sprite.New(d.w, d.h)
	boxX, top := d.scanLayout()
	mathX := boxX + pools.BoxW + mathGap
	markX := mathX + mathW + markGap
	putText(art, top-2, boxX, "CORE SETS", titleInk)

	for i, sl := range slots {
		at := float64(i) * SlotBeat
		if e < at {
			break
		}
		paintSlotBox(art, boxX, top+i*pools.BoxH, sl, e-at < pools.FlashSeconds)
		paintWordMath(art, top+i*pools.BoxH+1, mathX, sl)
	}

	steps := Steps(slots)
	selected := e >= BuildSeconds+StepsSeconds
	if !selected && e >= BuildSeconds {
		st := steps[int((e-BuildSeconds)/CompareBeat)]
		putText(art, top+st.Slot*pools.BoxH+1, boxX-2, "▸", cursorInk)
		textX := (d.w - runeLen(st.Text)) / 2
		if textX < 0 {
			textX = 0
		}
		putText(art, top+5*pools.BoxH+1, textX, st.Text, codeInk)
		if st.Best >= 0 {
			putText(art, top+st.Best*pools.BoxH+1, markX, LeadMark, pools.LabelInk)
		}
	}
	if selected {
		if winner := Winner(slots); winner >= 0 {
			putText(art, top+winner*pools.BoxH+1, markX, SelectedMark, wordInk)
			if dupes {
				for i, sl := range slots {
					if i == winner || sl.Free || sl.Job.Name != slots[winner].Job.Name {
						continue
					}
					putText(art, top+i*pools.BoxH+1, markX, StubTag, stubInk)
				}
			}
		}
	}

	dissolve(art, 0, 0, d.w, d.h, burn)
	sprite.Blit(stage, 0, 0, art)
	if selected {
		d.caption(stage, winCap)
	} else {
		d.caption(stage, scanCap)
	}
}

// paintSlotBox draws one core set pill at (x, y): the slot label over
// the job's name·prio in the job's ink — or free, dim — the border
// flashing white on arrival.
func paintSlotBox(sp sprite.Sprite, x, y int, sl Slot, flashing bool) {
	border := pools.DimInk
	if !sl.Free {
		border = sl.Job.Ink
	}
	if flashing {
		border = pools.FlashInk
	}
	sp.Set(y, x, sprite.Cell{Ch: '╭', FG: border, BG: -1})
	sp.Set(y, x+pools.BoxW-1, sprite.Cell{Ch: '╮', FG: border, BG: -1})
	sp.Set(y+2, x, sprite.Cell{Ch: '╰', FG: border, BG: -1})
	sp.Set(y+2, x+pools.BoxW-1, sprite.Cell{Ch: '╯', FG: border, BG: -1})
	for c := 1; c < pools.BoxW-1; c++ {
		sp.Set(y, x+c, sprite.Cell{Ch: '─', FG: border, BG: -1})
		sp.Set(y+2, x+c, sprite.Cell{Ch: '─', FG: border, BG: -1})
	}
	sp.Set(y+1, x, sprite.Cell{Ch: '│', FG: border, BG: -1})
	sp.Set(y+1, x+pools.BoxW-1, sprite.Cell{Ch: '│', FG: border, BG: -1})

	label := fmt.Sprintf("%-4s", sl.Label)
	if sl.Free {
		putText(sp, y+1, x+1, label, pools.DimInk)
		putText(sp, y+1, x+1+runeLen(label), "free", pools.DimInk)
		return
	}
	putText(sp, y+1, x+1, label, pools.LabelInk)
	putText(sp, y+1, x+1+runeLen(label), fmt.Sprintf("%s·%d", sl.Job.Name, sl.Job.Prio), sl.Job.Ink)
}

// paintWordMath writes the slot's full word beside its box: the octal
// priority, plus the VAC address — 000 dim for the NOVAC jobs, whose
// low nine bits stay empty — equals the packed PRIORITY word. A free
// set carries only -0.
func paintWordMath(sp sprite.Sprite, y, x int, sl Slot) {
	if sl.Free {
		putText(sp, y, x, "-0", pools.DimInk)
		return
	}
	c := putText(sp, y, x, fmt.Sprintf("%2d", sl.Job.Prio), coreset.PrioInk)
	c = putText(sp, y, c, " + ", dimInk)
	vacInk := coreset.VACAddrInk
	if sl.VACAddr == 0 {
		vacInk = pools.DimInk
	}
	c = putText(sp, y, c, fmt.Sprintf("%03o", sl.VACAddr), vacInk)
	c = putText(sp, y, c, " = ", dimInk)
	putText(sp, y, c, fmt.Sprintf("%05o", sl.Word()), wordInk)
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

func runeLen(s string) int { return len([]rune(s)) }

func putText(sp sprite.Sprite, r, c int, text string, fg int) int {
	for _, ch := range text {
		sp.Set(r, c, sprite.Cell{Ch: ch, FG: fg, BG: -1})
		c++
	}
	return c
}
