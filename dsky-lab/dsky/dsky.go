// Package dsky renders a compact, vertical terminal DSKY — the Apollo
// Display and Keyboard unit — built for embedding into larger TUIs. Top: the
// four story-relevant lights only (PROG, RESTART, and the LM's ALT and VEL
// landing-radar lights). Below: the electroluminescent display — COMP ACTY,
// PROG, VERB, NOUN in seven-segment digits, and three signed five-digit
// registers separated by thin lines.
//
// Output is raw ANSI 256-color (no terminal profile detection, so captures
// always keep color) and the footprint is a constant Width×Height grid in
// every state. Render is pure.
package dsky

import "strings"

// Fixed footprint.
const (
	Width  = 25
	Height = 23
)

// Palette (xterm-256).
const (
	colSeg     = 48  // electroluminescent green segments
	colLabel   = 245 // panel labels
	colRule    = 29  // dim green separators
	colLitBG   = 220 // amber caution light background
	colLitFG   = 16  // caution light text
	colDarkBox = 234 // unlit light-box background
	colDarkFG  = 240 // unlit light-box label
	colActyBG  = 40  // COMP ACTY green
)

// Lights is the compact caution panel: the alarm lamps and the LM's
// landing-radar lights — the ones that tell the descent story.
type Lights struct {
	Prog    bool
	Restart bool
	Alt     bool
	Vel     bool
}

// State is everything the DSKY shows. Empty strings blank their fields.
// Registers are "+DDDDD"/"-DDDDD"; malformed values render blank.
type State struct {
	Prog, Verb, Noun string
	R1, R2, R3       string
	CompActy         bool
	Flash            bool // VERB/NOUN flash (hidden on the off blink phase)
	Lights           Lights
}

// segFont is the classic thin-stroke seven-segment shape per digit.
var segFont = map[rune][3]string{
	'0': {" _ ", "| |", "|_|"},
	'1': {"   ", "  |", "  |"},
	'2': {" _ ", " _|", "|_ "},
	'3': {" _ ", " _|", " _|"},
	'4': {"   ", "|_|", "  |"},
	'5': {" _ ", "|_ ", " _|"},
	'6': {" _ ", "|_ ", "|_|"},
	'7': {" _ ", "  |", "  |"},
	'8': {" _ ", "|_|", "|_|"},
	'9': {" _ ", "|_|", " _|"},
}

var segBlank = [3]string{"   ", "   ", "   "}

// SegRows returns the three seven-segment rows for one digit. Unknown
// runes (including spaces) return blanks.
func SegRows(r rune) [3]string {
	if rows, ok := segFont[r]; ok {
		return rows
	}
	return segBlank
}

func fg(n int) string { return "\x1b[38;5;" + itoa(n) + "m" }
func bg(n int) string { return "\x1b[48;5;" + itoa(n) + "m" }

const reset = "\x1b[0m"

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

// line accumulates styled text while tracking the visible width.
type line struct {
	out   strings.Builder
	width int
}

func (l *line) add(text string, fgc, bgc int) {
	if text == "" {
		return
	}
	if bgc >= 0 {
		l.out.WriteString(bg(bgc))
	}
	if fgc >= 0 {
		l.out.WriteString(fg(fgc))
	}
	l.out.WriteString(text)
	l.out.WriteString(reset)
	l.width += len([]rune(text))
}

func (l *line) pad(to int) {
	for l.width < to {
		l.out.WriteString(" ")
		l.width++
	}
}

func (l *line) String() string { return l.out.String() }

// twoDigits renders a 2-character field as seven-segment rows (7 wide).
func twoDigits(s string, row int) string {
	for len(s) < 2 {
		s = " " + s
	}
	rs := []rune(s)
	return SegRows(rs[0])[row] + " " + SegRows(rs[1])[row]
}

// validReg reports whether a register string is "+DDDDD" or "-DDDDD".
func validReg(s string) bool {
	if len(s) != 6 || (s[0] != '+' && s[0] != '-') {
		return false
	}
	for _, c := range s[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// regRow renders one row of a signed register (23 wide).
func regRow(v string, row int) string {
	if !validReg(v) {
		return strings.Repeat(" ", 23)
	}
	sign := "   "
	if row == 1 {
		if v[0] == '+' {
			sign = " + "
		} else {
			sign = " − "
		}
	}
	out := sign + " "
	for i, r := range v[1:] {
		if i > 0 {
			out += " "
		}
		out += SegRows(r)[row]
	}
	return out
}

// lightBox renders one 12-wide caution light.
func lightBox(l *line, label string, lit bool) {
	pad := 12 - len(label)
	text := strings.Repeat(" ", pad/2) + label + strings.Repeat(" ", pad-pad/2)
	if label == "" {
		l.add(text, -1, -1)
		return
	}
	if lit {
		l.add(text, colLitFG, colLitBG)
	} else {
		l.add(text, colDarkFG, colDarkBox)
	}
}

// Render draws the DSKY. blinkOn is the flash phase: when State.Flash is
// set, VERB and NOUN digits are visible only while blinkOn is true.
func Render(s State, blinkOn bool) string {
	rows := make([]*line, 0, Height)
	newLine := func() *line {
		l := &line{}
		rows = append(rows, l)
		return l
	}

	// --- caution lights: the four that matter, two columns ----------------
	lightRows := []struct {
		left, right string
		litL, litR  bool
	}{
		{"PROG", "RESTART", s.Lights.Prog, s.Lights.Restart},
		{"ALT", "VEL", s.Lights.Alt, s.Lights.Vel},
	}
	for _, lr := range lightRows {
		l := newLine()
		lightBox(l, lr.left, lr.litL)
		l.add(" ", -1, -1)
		lightBox(l, lr.right, lr.litR)
		l.pad(Width)
	}

	rule := func() {
		l := newLine()
		l.add(strings.Repeat("─", Width), colRule, -1)
	}
	rule()

	// --- COMP ACTY + PROG --------------------------------------------------
	verb, noun := s.Verb, s.Noun
	if s.Flash && !blinkOn {
		verb, noun = "", ""
	}
	compFG, compBG := colDarkFG, -1
	if s.CompActy {
		compFG, compBG = colLitFG, colActyBG
	}
	{
		l := newLine()
		l.add(" COMP ", compFG, compBG)
		l.pad(Width - 4)
		l.add("PROG", colLabel, -1)
	}
	for row := 0; row < 3; row++ {
		l := newLine()
		if row == 0 {
			l.add(" ACTY ", compFG, compBG)
		}
		l.pad(Width - 7)
		l.add(twoDigits(s.Prog, row), colSeg, -1)
	}

	// --- VERB + NOUN --------------------------------------------------------
	{
		l := newLine()
		l.add("VERB", colLabel, -1)
		l.pad(Width - 4)
		l.add("NOUN", colLabel, -1)
	}
	for row := 0; row < 3; row++ {
		l := newLine()
		l.add(twoDigits(verb, row), colSeg, -1)
		l.pad(Width - 7)
		l.add(twoDigits(noun, row), colSeg, -1)
	}

	// --- registers ----------------------------------------------------------
	for _, reg := range []string{s.R1, s.R2, s.R3} {
		rule()
		for row := 0; row < 3; row++ {
			l := newLine()
			l.add(regRow(reg, row), colSeg, -1)
			l.pad(Width)
		}
	}

	outs := make([]string, len(rows))
	for i, l := range rows {
		l.pad(Width)
		outs[i] = l.String()
	}
	return strings.Join(outs, "\n")
}
