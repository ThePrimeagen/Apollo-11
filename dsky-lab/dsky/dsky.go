// Package dsky renders a compact, vertical terminal DSKY — the Apollo
// Display and Keyboard unit — built for embedding into larger TUIs. A gray
// plastic bezel (Game Boy / DS style) frames the whole unit. Top: the four
// story-relevant lights (PROG, RESTART, and the LM's ALT and VEL
// landing-radar lights). Middle: the electroluminescent display — COMP ACTY,
// PROG, VERB, NOUN in seven-segment digits, and three signed five-digit
// registers. Bottom: the 16-key keypad (VERB/NOUN/CLR/ENTR and the
// digit / sign keys). Digits being keyed render dull orange; a depressed
// key lights the same orange so a keystroke is obvious.
//
// Output is raw ANSI 256-color (no terminal profile detection, so captures
// always keep color) and the footprint is a constant Width×Height grid in
// every state. Render is pure.
package dsky

import "strings"

// Fixed footprint: 1-cell gray bezel around a 23-wide face, display on
// top, keypad under the registers.
const (
	Width  = 25
	Height = 30
	innerW = Width - 2
)

// Palette (xterm-256).
const (
	colSeg     = 48  // electroluminescent green segments
	colTyping  = 172 // dull orange: digits being keyed / key held down
	colLabel   = 245 // panel labels
	colRule    = 29  // dim green separators
	colLitBG   = 220 // amber caution light background
	colLitFG   = 16  // caution light text
	colDarkBox = 234 // unlit light-box background
	colDarkFG  = 240 // unlit light-box label
	colActyBG  = 40  // COMP ACTY green
	colFrame   = 244 // gray plastic outline
	colBody    = 236 // gray plastic fill
	colWell    = 232 // recessed display well
	colKeyBG   = 241 // unpressed key — a shade above the plastic
	colKeyFG   = 252 // unpressed key legend
)

// Lights is the compact caution panel: the alarm lamps and the LM's
// landing-radar lights — the ones that tell the descent story.
type Lights struct {
	Prog    bool
	Restart bool
	Alt     bool
	Vel     bool
}

// Key is a DSKY keypad key. Digit keys are '0'..'9'; the rest match the
// engine's PressKey bytes: V, N, E, C, +, -. Zero means none.
type Key byte

const (
	KeyNone  Key = 0
	KeyVerb  Key = 'V'
	KeyNoun  Key = 'N'
	KeyEntr  Key = 'E'
	KeyClr   Key = 'C'
	KeyPlus  Key = '+'
	KeyMinus Key = '-'
)

// Typing marks which display fields are currently being keyed. Those
// digits render dull orange instead of electroluminescent green.
type Typing struct {
	Verb, Noun, Prog bool
}

// State is everything the DSKY shows. Empty strings blank their fields.
// Registers are "+DDDDD"/"-DDDDD"; malformed values render blank.
type State struct {
	Prog, Verb, Noun string
	R1, R2, R3       string
	CompActy         bool
	Flash            bool // VERB/NOUN flash (hidden on the off blink phase)
	Lights           Lights
	// Pressed is the keypad key currently held down. It lights dull
	// orange so a keystroke is visible. Zero means none.
	Pressed Key
	Typing  Typing
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

// validReg reports whether a register string is "+DDDDD", "-DDDDD", or
// " DDDDD" (blank sign — the real DSKY showed alarm codes unsigned).
func validReg(s string) bool {
	if len(s) != 6 || (s[0] != '+' && s[0] != '-' && s[0] != ' ') {
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
		switch v[0] {
		case '+':
			sign = " + "
		case '-':
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

// lightBox renders one 11-wide caution light (fits the 23-wide well).
func lightBox(l *line, label string, lit bool) {
	const boxW = 11
	pad := boxW - len(label)
	if pad < 0 {
		pad = 0
		label = label[:boxW]
	}
	text := strings.Repeat(" ", pad/2) + label + strings.Repeat(" ", pad-pad/2)
	if label == "" {
		l.add(text, -1, colWell)
		return
	}
	if lit {
		l.add(text, colLitFG, colLitBG)
	} else {
		l.add(text, colDarkFG, colDarkBox)
	}
}

func (l *line) frameStart() { l.add("│", colFrame, colBody) }

func (l *line) finish(fillBG int) {
	for l.width < Width-1 {
		l.add(" ", -1, fillBG)
	}
	l.add("│", colFrame, colBody)
}

func borderRow(left, right rune) *line {
	l := &line{}
	l.add(string(left), colFrame, colBody)
	l.add(strings.Repeat("─", innerW), colFrame, colBody)
	l.add(string(right), colFrame, colBody)
	return l
}

type keyCap struct {
	key   Key
	label string // exactly 5 cells
}

var keypad = [][]keyCap{
	{{KeyVerb, "VERB "}, {KeyNoun, "NOUN "}, {KeyClr, " CLR "}, {KeyEntr, "ENTR "}},
	{{KeyPlus, "  +  "}, {'7', "  7  "}, {'8', "  8  "}, {'9', "  9  "}},
	{{KeyMinus, "  -  "}, {'4', "  4  "}, {'5', "  5  "}, {'6', "  6  "}},
	{{'0', "  0  "}, {'1', "  1  "}, {'2', "  2  "}, {'3', "  3  "}},
}

func paintKeypad(l *line, row []keyCap, pressed Key) {
	l.frameStart()
	for i, k := range row {
		if i > 0 {
			l.add(" ", -1, colBody)
		}
		fg, bg := colKeyFG, colKeyBG
		if pressed != KeyNone && pressed == k.key {
			fg, bg = colLitFG, colTyping
		}
		l.add(k.label, fg, bg)
	}
	l.finish(colBody)
}

func fieldColor(typing bool) int {
	if typing {
		return colTyping
	}
	return colSeg
}

func wellPad(l *line, to int) {
	for l.width < to {
		l.add(" ", -1, colWell)
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

	rows = append(rows, borderRow('┌', '┐'))

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
		l.frameStart()
		lightBox(l, lr.left, lr.litL)
		l.add(" ", -1, colWell)
		lightBox(l, lr.right, lr.litR)
		l.finish(colWell)
	}

	rule := func() {
		l := newLine()
		l.frameStart()
		l.add(strings.Repeat("─", innerW), colRule, colWell)
		l.finish(colWell)
	}
	rule()

	// --- COMP ACTY + PROG --------------------------------------------------
	verb, noun := s.Verb, s.Noun
	if s.Flash && !blinkOn {
		verb, noun = "", ""
	}
	compFG, compBG := colDarkFG, colWell
	if s.CompActy {
		compFG, compBG = colLitFG, colActyBG
	}
	progCol := fieldColor(s.Typing.Prog)
	verbCol := fieldColor(s.Typing.Verb)
	nounCol := fieldColor(s.Typing.Noun)
	{
		l := newLine()
		l.frameStart()
		l.add(" COMP ", compFG, compBG)
		wellPad(l, Width-1-4)
		l.add("PROG", colLabel, colWell)
		l.finish(colWell)
	}
	for row := 0; row < 3; row++ {
		l := newLine()
		l.frameStart()
		if row == 0 {
			l.add(" ACTY ", compFG, compBG)
		}
		wellPad(l, Width-1-7)
		l.add(twoDigits(s.Prog, row), progCol, colWell)
		l.finish(colWell)
	}

	// --- VERB + NOUN --------------------------------------------------------
	{
		l := newLine()
		l.frameStart()
		l.add("VERB", colLabel, colWell)
		wellPad(l, Width-1-4)
		l.add("NOUN", colLabel, colWell)
		l.finish(colWell)
	}
	for row := 0; row < 3; row++ {
		l := newLine()
		l.frameStart()
		l.add(twoDigits(verb, row), verbCol, colWell)
		wellPad(l, Width-1-7)
		l.add(twoDigits(noun, row), nounCol, colWell)
		l.finish(colWell)
	}

	// --- registers ----------------------------------------------------------
	for _, reg := range []string{s.R1, s.R2, s.R3} {
		rule()
		for row := 0; row < 3; row++ {
			l := newLine()
			l.frameStart()
			l.add(regRow(reg, row), colSeg, colWell)
			l.finish(colWell)
		}
	}

	// --- bezel between the screen well and the keypad ---------------------
	{
		l := newLine()
		l.frameStart()
		l.finish(colBody)
	}
	for _, row := range keypad {
		paintKeypad(newLine(), row, s.Pressed)
	}

	rows = append(rows, borderRow('└', '┘'))

	outs := make([]string, len(rows))
	for i, l := range rows {
		l.pad(Width)
		outs[i] = l.String()
	}
	return strings.Join(outs, "\n")
}
