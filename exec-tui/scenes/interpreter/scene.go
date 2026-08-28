// Package interpreter walks the REAL interpreter code the way DANZIG
// sees it — slimmed down so it reads at a glance. The scroll is
// MUNRVG, the average-G integration SERVICER ran every two seconds
// of the powered descent: the real opcodes and operands from
// Luminary099/SERVICER.agc, consecutive from TC INTPRET through RVQ,
// with the listing's dense inline comments stripped away. Every
// spotlit block reads the same simple way: ONE plain comment on top
// that just says what the block does, the bare instructions, and
// the whole DANZIG construction as one pseudo call —
//
//	check_for_higher_priority_jobs()    # DANZIG
//
// — the same line in every block, wearing the love mark. The
// function itself is the Check Priority scene's whole show; here it
// is just the question the interpreter asks between op pairs. The
// DOT altitude-rate block scrolls past between the last two stops
// and three stripped chunks (through MUNGRAV to RVQ) trail below the
// last stop so the fade still has code to sink through. The code
// component displays the cards, the scrollcode component moves them.
// Two live knobs (hold, glide) retune the walkthrough and save to
// scenes/interpreter/config.json.
package interpreter

import (
	"strings"

	"github.com/theprimeagen/apollo-11/exec-tui/components/code"
	"github.com/theprimeagen/apollo-11/exec-tui/components/scrollcode"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// The stock knobs — DefaultConfig is these two timings.
const (
	// HoldSeconds rests the spotlight on each block.
	HoldSeconds = 4.0
	// GlideSeconds carries the camera from one stop to the next.
	GlideSeconds = 0.9
)

// addrBase seats the scroll's gutter: fake octal locations, one per
// non-empty line, counted from here.
const addrBase = 0o4000

// CheckLine is the whole DANZIG construction, reduced to one pseudo
// call — the same line closing every block.
const CheckLine = "\t\tcheck_for_higher_priority_jobs()\t# DANZIG"

// Chunk is one spotlit block: one plain comment on top, the bare
// verbatim ops, and (assembled onto the card) the one-line check.
type Chunk struct {
	Name    string
	Comment string
	Source  []string
	Caption string
}

// PrologueLines is the hand-off: one plain comment over the real
// TC INTPRET, riding dark above the first block.
func PrologueLines() []string {
	return []string{
		"# HAND THE DESCENT MATH TO THE INTERPRETER",
		"\t\tTC\tINTPRET",
	}
}

// MidLines is the DOT altitude-rate block — the real ops between the
// velocity block and the VXV block, comments stripped. The camera
// scrolls through it without stopping, so the run stays consecutive.
func MidLines() []string {
	return []string{
		"\t\t\tUNIT/R/",
		"\t\tDOT\tSL1",
		"\t\t\tV1S",
		"\t\tSTOVL\tHDOTDISP",
		"\t\t\tR1S",
	}
}

// EpilogueBlocks is the rest of the routine, verbatim ops in order —
// the display terms, MUNGRAV, and the RVQ return — so the vignette
// below the last stop has real code to fade through.
func EpilogueBlocks() [][]string {
	return [][]string{
		{
			"\t\tDSU",
			"\t\t\t/LAND/",
			"\t\tSTCALL\tHCALC",
			"\t\t\tMUNRETRN",
		},
		{
			"MUNGRAV\t\tUNIT",
			"\t\tSTODL\tUNIT/R/",
			"\t\t\t34D",
			"\t\tSL\tBDDV",
			"\t\t\t6D",
			"\t\t\t-MUDTMUN",
		},
		{
			"\t\tDMP\tVXSC",
			"\t\t\tSHIFT11",
			"\t\t\tUNIT/R/",
			"\t\tSTORE\tGDT1/2",
			"\t\tRVQ",
		},
	}
}

// Chunks is the walkthrough: five consecutive blocks of MUNRVG, each
// one comment, its bare ops, and the same one-line check.
func Chunks() []Chunk {
	return []Chunk{
		{
			Name:    "VLOAD VXSC",
			Comment: "# LOAD THE VELOCITY CHANGE MEASURED THIS CYCLE",
			Source: []string{
				"MUNRVG\t\tVLOAD\tVXSC",
				"\t\t\tDELV",
				"\t\t\tKPIP2",
				"\t\tPUSH\tVAD",
				"\t\t\tGDT/2",
			},
			Caption: "1/5 VLOAD VXSC — the ΔV load",
		},
		{
			Name:    "PDDL DDV",
			Comment: "# AVERAGE THE VELOCITY AND DIVIDE BY THE GUIDANCE PERIOD",
			Source: []string{
				"\t\tPUSH\tVAD",
				"\t\t\tV",
				"\t\tPDDL\tDDV",
				"\t\t\tPGUIDE",
				"\t\t\tSHIFT11",
			},
			Caption: "2/5 PDDL DDV — the guidance push",
		},
		{
			Name:    "STCALL R1S",
			Comment: "# UPDATE THE POSITION AND CALL GRAVITY",
			Source: []string{
				"\t\tVXSC",
				"\t\tVAD",
				"\t\t\tR",
				"\t\tSTCALL\tR1S",
				"\t\t\tMUNGRAV",
			},
			Caption: "3/5 STCALL R1S — position out",
		},
		{
			Name:    "STORE V1S",
			Comment: "# UPDATE THE VELOCITY AND SAVE THE SPEED FOR DISPLAYS",
			Source: []string{
				"\t\tVAD\tVAD",
				"\t\tVAD",
				"\t\t\tV",
				"\t\tSTORE\tV1S",
				"\t\tABVAL",
				"\t\tSTOVL\tABVEL",
			},
			Caption: "4/5 STORE V1S — velocity out",
		},
		{
			Name:    "VXV VSL2",
			Comment: "# CORRECT FOR THE MOON TURNING BENEATH THE LANDER",
			Source: []string{
				"\t\tVXV\tVSL2",
				"\t\t\tWM",
				"\t\tSTODL\tDELVS",
				"\t\t\t36D",
			},
			Caption: "5/5 VXV VSL2 — the lunar rotation correction",
		},
	}
}

// Show is the Interpreter scene: one director composing the code
// cards and the scroll. Cfg is the two knobs Assemble reads on each
// Start, so a replay (Stop then Start) rebuilds the walkthrough from
// whatever they hold now.
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

// director owns the composition: the scroll of real code below, the
// caption above the house floor. The scroll's clock is its identity
// — a resize (Stop then Start) keeps it — while a fresh scene Start
// assembles a fresh director from the Show's current knobs.
type director struct {
	cfg    Config
	scroll *scrollcode.Scroll
	w, h   int
	staged bool
}

func newDirector(cfg Config) *director {
	return &director{
		cfg:    cfg,
		scroll: assemble().Tune(cfg.HoldSeconds, cfg.GlideSeconds),
	}
}

// assemble builds the roster: the prologue, the five spotlit blocks
// — comment, ops, blank, the love-marked check — with the DOT block
// scrolling past before the last one, and the three-chunk tail;
// every card on one continuous octal gutter.
func assemble() *scrollcode.Scroll {
	var blocks []scrollcode.Block
	addr := addrBase
	add := func(c *code.Code, lines []string, stop bool) {
		blocks = append(blocks, scrollcode.Block{Code: c.Gutter(addr), Stop: stop})
		addr += nonEmpty(lines)
	}

	add(code.New(code.LangAGC, PrologueLines()), PrologueLines(), false)
	chunks := Chunks()
	for i, ch := range chunks {
		if i == len(chunks)-1 {
			add(code.New(code.LangAGC, MidLines()), MidLines(), false)
		}
		lines := append([]string{ch.Comment}, ch.Source...)
		lines = append(lines, "", CheckLine)
		c := code.New(code.LangAGC, lines)
		markSpans(c, len(lines)-1, "check_for_higher_priority_jobs", code.Love)
		add(c, lines, true)
	}
	for _, ep := range EpilogueBlocks() {
		add(code.New(code.LangAGC, ep), ep, false)
	}
	return scrollcode.New(blocks...)
}

// markSpans marks every occurrence of needle on one expanded line.
func markSpans(c *code.Code, line int, needle string, ink int) {
	lines := c.Lines()
	if line < 0 || line >= len(lines) {
		return
	}
	rs := []rune(lines[line])
	ns := []rune(needle)
	for i := 0; i+len(ns) <= len(rs); i++ {
		if string(rs[i:i+len(ns)]) == needle {
			c.Mark(line, i, i+len(ns), ink)
		}
	}
}

func nonEmpty(lines []string) int {
	n := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}

func (d *director) Start(w, h int) {
	d.w, d.h = w, h
	d.staged = true
	sh := h - 2
	if sh < 1 {
		sh = 1
	}
	d.scroll.Start(w, sh)
}

func (d *director) Update(dt float64) {
	d.scroll.Update(dt)
}

func (d *director) Stop() {
	d.scroll.Stop()
	d.staged = false
}

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
	sprite.Blit(stage, 0, 0, d.scroll.Render())
	caption := Chunks()[d.scroll.FocusStop()].Caption
	col := 2
	for _, ch := range caption {
		stage.Set(d.h-1, col, sprite.Cell{Ch: ch, FG: code.Muted, BG: code.Base})
		col++
	}
	return stage
}
