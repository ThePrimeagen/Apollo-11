// Package code is a component that just displays code. You hand it
// lines and the language they are written in and it paints them as a
// still Rose Pine card — the syntax coloring is private to the
// component and keyed by the language, so a language it does not
// know stays plain text instead of guessing weirdly. Tabs expand on
// the 8-column stops of the AGC listings. An optional gutter numbers
// the non-empty lines in five-digit octal. Marks are the
// highlighting: the caller says which span of which line to
// highlight and in what color, and the mark wins over the syntax
// ink. Dim is the vignette ramp a scroller leans on: level 0 is the
// bright palette, three deepening rungs sink every hue toward the
// base, and past level 3 nothing paints. The card never moves —
// moving it is somebody else's job.
package code

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/theprimeagen/apollo-11/exec-tui/components/danzig"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// Lang names the language a card is written in. The coloring behind
// each name is private; an unknown Lang paints plain text.
type Lang string

const (
	// LangAGC is the AGC assembly / interpretive listing: labels in
	// column 0, the opcode field at column 16, operands and the
	// pair's far ops from column 24, # comments.
	LangAGC Lang = "agc"
	// LangPseudo is the danzig card's pseudocode dialect.
	LangPseudo Lang = "pseudo"
)

// The Rose Pine palette, as the xterm-256 indexes the danzig card
// already paints. These are the inks marks may ask for.
const (
	Base  = danzig.Base256
	Text  = danzig.Text256
	Muted = danzig.Muted256
	Gold  = danzig.Gold256
	Foam  = danzig.Foam256
	Iris  = danzig.Iris256
	Rose  = danzig.Rose256
	Love  = 168
)

// ramps is the vignette: index 0 the bright ink, then three
// deepening rungs. Every hue keeps its family for two rungs and
// sinks into the same barely-there gray on the third.
var ramps = map[int][4]int{
	Text:  {Text, 103, 60, 237},
	Muted: {Muted, 60, 59, 237},
	Gold:  {Gold, 137, 95, 237},
	Foam:  {Foam, 66, 23, 237},
	Iris:  {Iris, 97, 60, 237},
	Rose:  {Rose, 138, 95, 237},
	Love:  {Love, 132, 96, 237},
}

// Dim is ink fg at a vignette level: 0 (and below) is the bright ink
// itself, levels 1-3 sink it, and past 3 nothing paints. No-color
// stays no-color; an ink off the palette follows the text ramp.
func Dim(fg, level int) int {
	if fg < 0 {
		return -1
	}
	if level <= 0 {
		return fg
	}
	if level > 3 {
		return -1
	}
	ramp, ok := ramps[fg]
	if !ok {
		ramp = ramps[Text]
	}
	return ramp[level]
}

// tabStop is the AGC listing's grid.
const tabStop = 8

// gutterW is the octal address plus the two spaces after it.
const gutterW = 7

// mark is one caller-chosen highlight span.
type mark struct {
	line, start, end, ink int
}

// Code is one still card of code.
type Code struct {
	lang   Lang
	lines  []string
	marks  []mark
	base   int
	w, h   int
	staged bool
}

// New is a card of lines written in lang, tabs expanded.
func New(lang Lang, lines []string) *Code {
	c := &Code{lang: lang, base: -1}
	for _, line := range lines {
		c.lines = append(c.lines, expandTabs(line))
	}
	return c
}

func expandTabs(line string) string {
	var b strings.Builder
	col := 0
	for _, r := range line {
		if r == '\t' {
			n := tabStop - col%tabStop
			b.WriteString(strings.Repeat(" ", n))
			col += n
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

// Gutter numbers the card's non-empty lines in five-digit octal,
// counting up from base.
func (c *Code) Gutter(base int) *Code {
	if c == nil {
		return c
	}
	c.base = base
	return c
}

// Mark highlights [start, end) of line in ink, over the syntax
// coloring. Spans off the card clamp quietly.
func (c *Code) Mark(line, start, end, ink int) *Code {
	if c == nil {
		return c
	}
	c.marks = append(c.marks, mark{line, start, end, ink})
	return c
}

// Lines is the expanded content, without the gutter — what marks and
// callers measure against.
func (c *Code) Lines() []string {
	if c == nil {
		return nil
	}
	return c.lines
}

// Size is the card's own art size: the widest expanded line (plus
// the gutter when on) by the line count.
func (c *Code) Size() (w, h int) {
	if c == nil || len(c.lines) == 0 {
		return 0, 0
	}
	for _, line := range c.lines {
		if n := len([]rune(line)); n > w {
			w = n
		}
	}
	if c.base >= 0 {
		w += gutterW
	}
	return w, len(c.lines)
}

// Art paints the card at its own size, bright.
func (c *Code) Art() sprite.Sprite {
	w, h := c.Size()
	if w == 0 || h == 0 {
		return sprite.Sprite{}
	}
	art := sprite.New(w, h)
	for r := 0; r < h; r++ {
		for col := 0; col < w; col++ {
			art.Set(r, col, sprite.Cell{Ch: ' ', FG: -1, BG: Base})
		}
	}
	addr := c.base
	for r, line := range c.lines {
		left := 0
		if c.base >= 0 {
			if strings.TrimSpace(line) != "" {
				for i, ch := range fmt.Sprintf("%05o  ", addr) {
					art.Set(r, i, sprite.Cell{Ch: ch, FG: Muted, BG: Base})
				}
				addr++
			}
			left = gutterW
		}
		runes := []rune(line)
		inks := c.paint(line)
		for i, ch := range runes {
			art.Set(r, left+i, sprite.Cell{Ch: ch, FG: inks[i], BG: Base})
		}
		for _, m := range c.marks {
			if m.line != r {
				continue
			}
			lo, hi := m.start, m.end
			if lo < 0 {
				lo = 0
			}
			if hi > len(runes) {
				hi = len(runes)
			}
			for i := lo; i < hi; i++ {
				cell := art.At(r, left+i)
				cell.FG = m.ink
				art.Set(r, left+i, cell)
			}
		}
	}
	return art
}

// paint is the private coloring: one ink per rune of the expanded
// line, keyed by the card's language.
func (c *Code) paint(line string) []int {
	switch c.lang {
	case LangAGC:
		return paintAGC(line)
	case LangPseudo:
		return paintPseudo(line)
	default:
		inks := make([]int, len([]rune(line)))
		for i := range inks {
			inks[i] = Text
		}
		return inks
	}
}

// numberTok is a numeric AGC operand: digits, an optional D scale
// suffix, trailing punctuation tolerated.
var numberTok = regexp.MustCompile(`^[0-9]+D?[:,]*$`)

// paintAGC colors one listing line: comments muted from the # on,
// column-0 labels foam, the first code token iris when it sits in
// the opcode field (before column 24), numbers gold, symbols foam,
// and bare punctuation rose.
func paintAGC(line string) []int {
	runes := []rune(line)
	inks := make([]int, len(runes))
	for i := range inks {
		inks[i] = Text
	}
	end := len(runes)
	for i, r := range runes {
		if r == '#' {
			end = i
			for j := i; j < len(runes); j++ {
				inks[j] = Muted
			}
			break
		}
	}
	first := true
	i := 0
	for i < end {
		if runes[i] == ' ' {
			i++
			continue
		}
		j := i
		for j < end && runes[j] != ' ' {
			j++
		}
		tok := string(runes[i:j])
		ink := Rose
		hasLetter := strings.IndexFunc(tok, unicode.IsLetter) >= 0
		switch {
		case i == 0:
			// a label leads the line; the opcode still follows
			ink = Foam
		case numberTok.MatchString(tok):
			ink = Gold
			first = false
		case first && hasLetter && i < 24:
			ink = Iris
			first = false
		case hasLetter:
			ink = Foam
			first = false
		}
		for k := i; k < j; k++ {
			inks[k] = ink
		}
		i = j
	}
	return inks
}

// keywords is the pseudocode dialect's iris set — the danzig card's,
// plus the C-style words the allocation walks speak. "new" is NOT a
// keyword: it is the walked function's own variable.
var keywords = map[string]bool{
	"if":       true,
	"for":      true,
	"continue": true,
	"pick":     true,
	"run":      true,
	"swap":     true,
	"first":    true,
	"free":     true,
	"return":   true,
	"throw":    true,
}

func isIdentStart(r rune) bool { return unicode.IsLetter(r) || r == '_' }
func isIdent(r rune) bool      { return isIdentStart(r) || unicode.IsDigit(r) }

// isLabel is an all-caps symbol of at least two runes.
func isLabel(s string) bool {
	n := 0
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			n++
		case r >= '0' && r <= '9':
		default:
			return false
		}
	}
	return n >= 2 || (n == 1 && len(s) >= 2)
}

// paintPseudo colors the danzig dialect: comments muted, keywords
// iris, labels foam, numbers gold, operators rose, idents text.
func paintPseudo(line string) []int {
	runes := []rune(line)
	inks := make([]int, len(runes))
	i := 0
	for i < len(runes) {
		r := runes[i]
		switch {
		case r == ' ' || r == '\t':
			inks[i] = Text
			i++
		case r == '#':
			for j := i; j < len(runes); j++ {
				inks[j] = Muted
			}
			i = len(runes)
		case r >= '0' && r <= '9':
			j := i
			for j < len(runes) && runes[j] >= '0' && runes[j] <= '9' {
				j++
			}
			for k := i; k < j; k++ {
				inks[k] = Gold
			}
			i = j
		case isIdentStart(r):
			j := i
			for j < len(runes) && isIdent(runes[j]) {
				j++
			}
			word := string(runes[i:j])
			ink := Text
			switch {
			case keywords[word]:
				ink = Iris
			case isLabel(word):
				ink = Foam
			}
			for k := i; k < j; k++ {
				inks[k] = ink
			}
			i = j
		default:
			inks[i] = Rose
			i++
		}
	}
	return inks
}

// Start pins the stage the still card centers on.
func (c *Code) Start(w, h int) {
	if c == nil {
		return
	}
	c.w, c.h = w, h
	c.staged = true
}

// Update is a no-op: the card never moves itself.
func (c *Code) Update(dt float64) {}

// Render paints the card centered on a stage-sized sprite. Before
// Start and after Stop the stage is empty.
func (c *Code) Render() sprite.Sprite {
	if c == nil || !c.staged || c.w < 1 || c.h < 1 {
		return sprite.Sprite{}
	}
	stage := sprite.New(c.w, c.h)
	art := c.Art()
	sprite.Blit(stage, (c.w-art.Width)/2, (c.h-art.Height)/2, art)
	return stage
}

// Stop clears the staging.
func (c *Code) Stop() {
	if c == nil {
		return
	}
	c.staged = false
}
