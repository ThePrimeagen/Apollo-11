// Package interpreter walks the REAL interpreter code the way DANZIG
// sees it. The scroll is MUNRVG — the average-G integration SERVICER
// ran every two seconds of the powered descent — verbatim and
// consecutive from Luminary099/SERVICER.agc, TC INTPRET through RVQ.
// The code component displays the cards, the scrollcode component
// moves them; the scene only chooses what to show: a prologue (the
// routine's own header comments and the TC INTPRET hand-off), five
// spotlit chunks — the ΔV load, the guidance push, the position out,
// the velocity out, and the VXV cross product the scene is named
// for — the DOT altitude-rate block scrolling past between the last
// two stops, and three trailing chunks (through MUNGRAV to RVQ) so
// the fade below the last stop still has code to sink through.
//
// Each spotlit chunk ends in an annotated check to DANZIG — the
// question the interpreter asks between op pairs: is a job of higher
// priority waiting? — in its own dress: the real assembly (verbatim
// INTERPRETER.agc), pseudocode, a fork, a weighing, a stamp. Every
// NEWJOB in the checks wears a love mark and the VXV op itself wears
// gold. Two live knobs (hold, glide) retune the walkthrough and save
// to scenes/interpreter/config.json.
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
	// HoldSeconds rests the spotlight on each chunk.
	HoldSeconds = 4.0
	// GlideSeconds carries the camera from one stop to the next.
	GlideSeconds = 0.9
)

// addrBase seats the scroll's gutter: fake octal locations, one per
// non-empty line, counted from here.
const addrBase = 0o4000

// Chunk is one spotlit stretch of the real listing: the verbatim
// SERVICER source, then the annotated check to DANZIG in this
// chunk's own dress.
type Chunk struct {
	Name    string
	Source  []string
	Intro   string
	Check   []string
	Style   string
	Caption string
}

// PrologueLines is the routine's own header and the TC INTPRET
// hand-off — verbatim SERVICER.agc, never spotlit, riding dark above
// the first chunk.
func PrologueLines() []string {
	return []string{
		"# MUNRVG IS A SPECIAL AVERAGE G INTEGRATION ROUTINE USED BY THRUSTING",
		"# PROGRAMS WHICH FUNCTION IN THE VICINITY OF AN ASSUMED SPHERICAL MOON.",
		"\t\tTC\tINTPRET",
	}
}

// MidLines is the DOT altitude-rate block — the real code between
// the velocity chunk and the VXV chunk. The camera scrolls through
// it without stopping, so the run stays consecutive.
func MidLines() []string {
	return []string{
		"\t\t\tUNIT/R/",
		"\t\tDOT\tSL1",
		"\t\t\tV1S",
		"\t\tSTOVL\tHDOTDISP\t# HDOT = V. UNIT(R)*2(7) M/CS.",
		"\t\t\tR1S",
	}
}

// EpilogueBlocks is the rest of the routine, verbatim and in order —
// the lunar-landing display terms, MUNGRAV, and the RVQ return — so
// the vignette below the last stop has real code to fade through.
func EpilogueBlocks() [][]string {
	return [][]string{
		{
			"\t\tDSU",
			"\t\t\t/LAND/",
			"\t\tSTCALL\tHCALC\t\t# FOR NOW, DISPLAY WHETHER POS OR NEG",
			"\t\t\tMUNRETRN",
		},
		{
			"MUNGRAV\t\tUNIT\t\t\t# AT 36D HAVE ABVAL(R), AT 34D R.R",
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
			"\t\tSTORE\tGDT1/2\t\t# 1/2GDT SCALED AT 2(7) M/CS.",
			"\t\tRVQ",
		},
	}
}

// Chunks is the walkthrough: five consecutive stretches of MUNRVG,
// each ending in the same check to DANZIG in a different dress.
func Chunks() []Chunk {
	stampTop := "\t\t╭─ DANZIG " + strings.Repeat("─", 15) + "╮"
	stampBody := "\t\t│ NEWJOB = 0 ✓ CARRY ON  │"
	stampBot := "\t\t╰" + strings.Repeat("─", 24) + "╯"
	return []Chunk{
		{
			Name: "VLOAD VXSC",
			Source: []string{
				"MUNRVG\t\tVLOAD\tVXSC",
				"\t\t\tDELV",
				"\t\t\tKPIP2",
				"\t\tPUSH\tVAD\t\t# 1ST PUSH:  DELV IN UNITS OF 2(8) M/CS",
				"\t\t\tGDT/2",
			},
			Intro: "# ...THEN THE DISPATCH — THE CHECK, AS THE REAL ASSEMBLY (INTERPRETER.AGC):",
			Check: []string{
				"\t\tCCS\tNEWJOB\t\t\t# SEE IF A JOB OF HIGHER PRIORITY IS",
				"\t\tTCF\tCHANG2\t\t\t# PRESENT, AND IF SO, CHANGE JOBS.",
			},
			Style:   "assembly",
			Caption: "1/5 VLOAD VXSC — the ΔV load — the check as the real assembly: CCS NEWJOB / TCF CHANG2",
		},
		{
			Name: "PDDL DDV",
			Source: []string{
				"\t\tPUSH\tVAD\t\t# 2ND PUSH:  (DELV + GDT)/2, UNITS OF 2(7)",
				"\t\t\tV\t\t#\t\t\t\t(12)",
				"\t\tPDDL\tDDV",
				"\t\t\tPGUIDE",
				"\t\t\tSHIFT11",
			},
			Intro: "# ...THEN THE SAME CHECK, SPELLED OUT AS PSEUDOCODE:",
			Check: []string{
				"\t\tIF\tNEWJOB != 0:\t\t# DANZIG, SPELLED OUT",
				"\t\tSWAP\tCORES[0], CORES[NEWJOB]",
			},
			Style:   "pseudocode",
			Caption: "2/5 PDDL DDV — the guidance push — the check as pseudocode: a non-zero NEWJOB swaps cores",
		},
		{
			Name: "STCALL R1S",
			Source: []string{
				"\t\tVXSC",
				"\t\tVAD",
				"\t\t\tR",
				"\t\tSTCALL\tR1S\t\t# STORE R SCALED AT 2(+24) M",
				"\t\t\tMUNGRAV",
			},
			Intro: "# ...THEN THE SAME CHECK, AS A FORK:",
			Check: []string{
				"\t\tDANZIG ─┬─ NEWJOB = 0 ──▶ NEXT OP",
				"\t\t        ╰─ NEWJOB > 0 ──▶ CHANG2",
			},
			Style:   "fork",
			Caption: "3/5 STCALL R1S — position out, call gravity — the check as a fork: zero rides on",
		},
		{
			Name: "STORE V1S",
			Source: []string{
				"# Page 883",
				"\t\tVAD\tVAD",
				"\t\tVAD",
				"\t\t\tV",
				"\t\tSTORE\tV1S\t\t# STORE V SCALED AT 2(+7) M/CS.",
				"\t\tABVAL",
				"\t\tSTOVL\tABVEL\t\t# STORE SPEED FOR LR AND DISPLAYS.",
			},
			Intro: "# ...THEN THE SAME CHECK, AS A WEIGHING OF PRIORITY WORDS:",
			Check: []string{
				"\t\tTHIS JOB  ▓▓▓▓▓░░░░░ 20\t\t# DANZIG WEIGHS THE WORDS",
				"\t\tNEWJOB    ▓▓▓▓▓▓▓░░░ 26 ──▶ CHANG2",
			},
			Style:   "weighing",
			Caption: "4/5 STORE V1S — velocity out, speed for the LR — the check as a weighing of PRIORITY words",
		},
		{
			Name: "VXV VSL2",
			Source: []string{
				"\t\tVXV\tVSL2",
				"\t\t\tWM",
				"\t\tSTODL\tDELVS\t\t# LUNAR ROTATION CORRECTION TERM*2(5) M/CS.",
				"\t\t\t36D",
			},
			Intro:   "# ...THEN THE SAME CHECK, AS A STAMP:",
			Check:   []string{stampTop, stampBody, stampBot},
			Style:   "stamp",
			Caption: "5/5 VXV VSL2 — the V cross V itself — the check as a stamp: carry on",
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

// assemble builds the roster: the prologue, the five spotlit chunks
// with the DOT block scrolling past before the last one, and the
// three-chunk tail — every card on one continuous octal gutter.
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
		lines := append([]string{}, ch.Source...)
		lines = append(lines, "", ch.Intro)
		lines = append(lines, ch.Check...)
		c := code.New(code.LangAGC, lines)
		markChecks(c, len(ch.Source))
		if ch.Style == "stamp" {
			markSpans(c, 0, "VXV", code.Gold)
		}
		add(c, lines, true)
	}
	for _, ep := range EpilogueBlocks() {
		add(code.New(code.LangAGC, ep), ep, false)
	}
	return scrollcode.New(blocks...)
}

// markChecks highlights every NEWJOB of the annotation lines in love
// ink — the word the whole check turns on.
func markChecks(c *code.Code, srcLen int) {
	for li := srcLen; li < len(c.Lines()); li++ {
		markSpans(c, li, "NEWJOB", code.Love)
	}
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
