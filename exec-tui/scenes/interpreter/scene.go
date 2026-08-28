// Package interpreter is the walkthrough of the virtual machine's
// code the way DANZIG sees it: a scrolling column of five fake
// interpretive instructions — VXV (vector cross vector) first — each
// block spelling the op, how its arguments arrive, what the op costs
// in milliseconds, and then the check to DANZIG that ends nearly
// every real instruction in INTERPRETER.agc: is a job of higher
// priority waiting? The check wears five different dresses, one per
// instruction — the real assembly (CCS NEWJOB / TCF CHANG2),
// pseudocode, a fork diagram, a weighing of PRIORITY words, and a
// rubber stamp — so the same question is asked five ways.
//
// The column wears Rose Pine over the Rose Pine base, behind a
// vertical vignette: the spotlit block is bright at its anchor, the
// blocks one step above and below sit equally dimmed, two steps away
// is barely visible, and past that the code cannot be seen at all.
// An INTPRET prologue rides above the first instruction and an EXIT
// epilogue below the last — seven blocks — so the vignette never
// runs out of code. The spotlight rests HoldSeconds on each
// instruction, glides GlideSeconds to the next on an eased camera
// that lands exactly on its anchor, and holds forever on the fifth.
// Both timings are live knobs on a Config the standalone runner
// nudges 50ms at a time and saves to scenes/interpreter/config.json.
package interpreter

import (
	"fmt"
	"math"
	"strings"

	"github.com/theprimeagen/apollo-11/exec-tui/components/danzig"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// The stock knobs — DefaultConfig is these two timings.
const (
	// HoldSeconds rests the spotlight on each instruction.
	HoldSeconds = 4.0
	// GlideSeconds carries the camera from one stop to the next.
	GlideSeconds = 0.9
)

// addrBase seats the fake column in fixed memory: the gutter counts
// octal locations from here, one per content line.
const addrBase = 0o4000

// gutterW is the octal address plus the two spaces after it.
const gutterW = 7

// Instr is one fake virtual instruction on the scroll: the op line,
// how the arguments arrive, what it costs, and its own look for the
// DANZIG check.
type Instr struct {
	Mnemonic string
	Op       string
	Args     []string
	Time     string
	Check    []string
	Style    string
	Caption  string
}

// Instructions is the walkthrough: five fake interpretive
// instructions, VXV first, each ending in a different dress for the
// same DANZIG question.
func Instructions() []Instr {
	stampTop := "    ╭─ DANZIG " + strings.Repeat("─", 15) + "╮"
	stampBody := "    │ NEWJOB = 0 ✓ carry on  │"
	stampBot := "    ╰" + strings.Repeat("─", 24) + "╯"
	return []Instr{
		{
			Mnemonic: "VXV",
			Op:       "VXV — vector cross vector         # MPAC ← MPAC × X",
			Args:     []string{"    args: V from MPAC, X ← fetch(ADDRWD)"},
			Time:     "    time: ≈ 5.0 ms                # ≈ 425 machine cycles",
			Check: []string{
				"    CCS NEWJOB                    # higher priority waiting?",
				"    TCF CHANG2                    # yes — swap to that job",
			},
			Style:   "assembly",
			Caption: "1/5 VXV — the check as the real assembly: CCS NEWJOB, TCF CHANG2 — INTERPRETER.agc",
		},
		{
			Mnemonic: "DOT",
			Op:       "DOT — vector dot product          # MPAC ← MPAC · X",
			Args:     []string{"    args: V from MPAC, X ← fetch(ADDRWD)"},
			Time:     "    time: ≈ 3.4 ms                # three DP multiplies",
			Check: []string{
				"    if NEWJOB != 0:               # DANZIG, spelled out",
				"        swap cores[0], cores[NEWJOB]",
			},
			Style:   "pseudocode",
			Caption: "2/5 DOT — the check as pseudocode: a non-zero NEWJOB swaps core sets before the next op",
		},
		{
			Mnemonic: "MXV",
			Op:       "MXV — matrix times vector         # MPAC ← M(X) × MPAC",
			Args:     []string{"    args: M ← fetch 6 words at ADDRWD"},
			Time:     "    time: ≈ 9.8 ms                # nine multiplies deep",
			Check: []string{
				"    DANZIG ─┬─ NEWJOB = 0 ──▶ next op",
				"            ╰─ NEWJOB > 0 ──▶ CHANG2",
			},
			Style:   "fork",
			Caption: "3/5 MXV — the check as a fork: zero rides on, non-zero exits to CHANG2",
		},
		{
			Mnemonic: "VXSC",
			Op:       "VXSC — vector times scalar        # MPAC ← MPAC × K",
			Args:     []string{"    args: K ← fetch(ADDRWD)       # the scalar rides along"},
			Time:     "    time: ≈ 2.6 ms                # one multiply per part",
			Check: []string{
				"    this job  ▓▓▓▓▓░░░░░ 20       # DANZIG weighs the words",
				"    NEWJOB    ▓▓▓▓▓▓▓░░░ 26 → CHANG2",
			},
			Style:   "weighing",
			Caption: "4/5 VXSC — the check as a weighing: the bigger PRIORITY word takes the CPU",
		},
		{
			Mnemonic: "DAD",
			Op:       "DAD — double precision add        # MPAC ← MPAC + X",
			Args:     []string{"    args: X ← fetch(ADDRWD)       # one erasable pair"},
			Time:     "    time: ≈ 0.9 ms                # the cheap one",
			Check:    []string{stampTop, stampBody, stampBot},
			Style:    "stamp",
			Caption:  "5/5 DAD — the check as a stamp: cleared through DANZIG, the job keeps the core",
		},
	}
}

// Block is one run of lines on the scroll, separated from the next
// by one blank row.
type Block struct {
	Name  string
	Lines []string
}

// prologue is the interpreter's own entry, riding above the first
// instruction — never spotlit, always half-seen.
func prologue() []string {
	return []string{
		"INTPRET:                          # enter the virtual machine",
		"    LOC ← the word after the TC   # op pairs live at LOC",
		"    unpack pair → op, EDOP        # two ops packed per word",
	}
}

// epilogue rides below the last instruction so the final stop still
// has code fading under it.
func epilogue() []string {
	return []string{
		"EXIT:                             # leave the interpreter",
		"    resume native code at LOC     # machine words again",
	}
}

// Blocks is the whole scroll: the INTPRET prologue, the five
// instructions (op, args, time, check), and the EXIT epilogue.
func Blocks() []Block {
	ins := Instructions()
	bs := make([]Block, 0, len(ins)+2)
	bs = append(bs, Block{Name: "INTPRET", Lines: prologue()})
	for _, in := range ins {
		lines := append([]string{in.Op}, in.Args...)
		lines = append(lines, in.Time)
		lines = append(lines, in.Check...)
		bs = append(bs, Block{Name: in.Mnemonic, Lines: lines})
	}
	bs = append(bs, Block{Name: "EXIT", Lines: epilogue()})
	return bs
}

// ink is one Rose Pine color the scene paints with, at whatever
// vignette level a block sits.
type ink int

const (
	inkText ink = iota
	inkMuted
	inkGold
	inkFoam
	inkIris
	inkRose
	inkCount
)

// shades is the vignette: level 0 is the danzig card's Rose Pine,
// level 1 keeps each hue but sinks it toward the base, level 2 is
// the same near-gone ink for everything — barely visible is past
// caring about hue.
var shades = [inkCount][3]int{
	inkText:  {danzig.Text256, 103, 237},
	inkMuted: {danzig.Muted256, 60, 237},
	inkGold:  {danzig.Gold256, 137, 237},
	inkFoam:  {danzig.Foam256, 66, 237},
	inkIris:  {danzig.Iris256, 97, 237},
	inkRose:  {danzig.Rose256, 138, 237},
}

// shade is ink i at a vignette level: negative levels clamp to the
// spotlight, level 3 and past do not paint at all, and a ghost ink
// falls back to text so a bad kind is still readable.
func shade(i ink, level int) int {
	if level < 0 {
		level = 0
	}
	if level > 2 {
		return -1
	}
	if i < 0 || i >= inkCount {
		i = inkText
	}
	return shades[i][level]
}

// vigLevel rounds a block's distance from the spotlight to its shade
// level. Broken distances are past seeing, never a panic.
func vigLevel(d float64) int {
	d = math.Abs(d)
	if math.IsNaN(d) || d >= 2.5 {
		return 3
	}
	return int(math.Round(d))
}

// kindInk maps the danzig tokenizer's syntax classes onto the inks.
func kindInk(k danzig.Kind) ink {
	switch k {
	case danzig.KindComment:
		return inkMuted
	case danzig.KindKeyword:
		return inkIris
	case danzig.KindLabel:
		return inkFoam
	case danzig.KindNumber:
		return inkGold
	case danzig.KindOp:
		return inkRose
	default:
		return inkText
	}
}

// anchorY is the screen row the spotlit block's op line parks on —
// high enough that two blocks fit below it, the way the vignette is
// framed.
func anchorY(h int) int {
	y := h / 4
	if y < 1 {
		y = 1
	}
	return y
}

// Show is the Interpreter scene: one director component running the
// scroll on its own clock. Cfg is the two knobs Assemble reads on
// each Start, so a replay (Stop then Start) rebuilds the walkthrough
// from whatever they hold now.
type Show struct {
	Cfg Config
	screenplay.Ensemble
}

// New is the scene, playing the Active knobs, ready for its curtain.
func New() *Show {
	s := &Show{Cfg: Active()}
	s.Assemble = func() []screenplay.Component {
		return []screenplay.Component{newDirector(s.Cfg)}
	}
	return s
}

// Bill is the Interpreter scene as a one-scene screenplay.
func Bill() screenplay.Bill {
	return screenplay.Bill{
		screenplay.Entry{Name: "Interpreter", Scene: New()},
	}
}

// director owns the walkthrough: the knobs it was cast with and the
// clock. The clock is its identity — a resize (Stop then Start)
// keeps it — while a fresh scene Start assembles a fresh director
// from the Show's current knobs.
type director struct {
	cfg    Config
	clock  float64
	w, h   int
	staged bool
}

func newDirector(cfg Config) *director {
	return &director{cfg: cfg}
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

// focusPos is the continuous spotlight position in [0, 4]: whole at
// a hold, fractional through a glide, capped forever on the last
// instruction.
func (c Config) focusPos(t float64) float64 {
	last := float64(len(Instructions()) - 1)
	if t <= 0 {
		return 0
	}
	period := c.HoldSeconds + c.GlideSeconds
	if period <= 0 {
		return last
	}
	i := math.Floor(t / period)
	if i >= last {
		return last
	}
	e := t - i*period
	if e <= c.HoldSeconds {
		return i
	}
	return i + ease((e-c.HoldSeconds)/c.GlideSeconds)
}

// camRow is the scroll row the camera holds at the anchor: the
// spotlit block's own row at a hold, the eased in-between on a
// glide, rounded — not truncated — so the camera lands exactly on
// its destination while its own glide still runs.
func camRow(rows []int, p float64) int {
	i := int(math.Floor(p))
	f := p - float64(i)
	from := rows[1+i]
	if f <= 0 {
		return from
	}
	return from + int(math.Round(f*float64(rows[2+i]-rows[1+i])))
}

// captionIdx is which instruction's caption the frame wears: the
// nearer stop, so a glide hands the words over at its midpoint.
func captionIdx(p float64) int {
	i := int(math.Round(p))
	if i < 0 {
		i = 0
	}
	if last := len(Instructions()) - 1; i > last {
		i = last
	}
	return i
}

// columnWidth is the widest content line plus the gutter.
func columnWidth(bs []Block) int {
	w := 0
	for _, b := range bs {
		for _, line := range b.Lines {
			if n := len([]rune(line)); n > w {
				w = n
			}
		}
	}
	return w + gutterW
}

func (d *director) Render() sprite.Sprite {
	if !d.staged || d.w < 1 || d.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(d.w, d.h)
	for r := 0; r < d.h; r++ {
		for c := 0; c < d.w; c++ {
			stage.Set(r, c, sprite.Cell{Ch: ' ', FG: -1, BG: danzig.Base256})
		}
	}

	bs := Blocks()
	rows := make([]int, len(bs)+1)
	for i, b := range bs {
		rows[i+1] = rows[i] + len(b.Lines) + 1
	}
	p := d.cfg.focusPos(d.clock)
	cam := camRow(rows, p)
	anchor := anchorY(d.h)
	left := (d.w - columnWidth(bs)) / 2
	if left < 0 {
		left = 0
	}

	addr := addrBase
	for b, blk := range bs {
		level := vigLevel(float64(b) - (1 + p))
		top := anchor + rows[b] - cam
		if level <= 2 {
			for li, line := range blk.Lines {
				d.paintLine(stage, top+li, left, addr+li, line, level)
			}
		}
		addr += len(blk.Lines)
	}

	putText(stage, d.h-1, 2, Instructions()[captionIdx(p)].Caption, shade(inkMuted, 0))
	return stage
}

// paintLine writes one gutter-led source line at its vignette level.
// Rows off the stage — or under the caption's breathing room — are
// skipped whole.
func (d *director) paintLine(stage sprite.Sprite, y, x, addr int, line string, level int) {
	if y < 0 || y > d.h-3 {
		return
	}
	col := x
	for _, r := range fmt.Sprintf("%05o  ", addr) {
		stage.Set(y, col, sprite.Cell{Ch: r, FG: shade(inkMuted, level), BG: danzig.Base256})
		col++
	}
	for _, tok := range danzig.TokenizeLine(line) {
		fg := shade(kindInk(tok.Kind), level)
		for _, r := range tok.Text {
			stage.Set(y, col, sprite.Cell{Ch: r, FG: fg, BG: danzig.Base256})
			col++
		}
	}
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

func putText(sp sprite.Sprite, r, c int, text string, fg int) {
	for _, ch := range text {
		sp.Set(r, c, sprite.Cell{Ch: ch, FG: fg, BG: danzig.Base256})
		c++
	}
}
