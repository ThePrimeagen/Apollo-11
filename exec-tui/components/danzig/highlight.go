package danzig

import (
	"fmt"
	"strings"
	"unicode"
)

type rgb struct{ r, g, b int }

var (
	rpBase    = rgb{25, 23, 36}    // #191724
	rpMuted   = rgb{110, 106, 134} // #6e6a86
	rpText    = rgb{224, 222, 244} // #e0def4
	rpGold    = rgb{246, 193, 119} // #f6c177
	rpFoam    = rgb{156, 207, 216} // #9ccfd8
	rpIris    = rgb{196, 167, 231} // #c4a7e7
	rpRose    = rgb{235, 188, 186} // #ebbcba
	rpOverlay = rgb{38, 35, 58}    // #26233a
)

var keywords = map[string]bool{
	"if":       true,
	"for":      true,
	"continue": true,
	"pick":     true,
	"run":      true,
	"swap":     true,
	"first":    true,
	"free":     true,
}

func (c rgb) fg() string { return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.r, c.g, c.b) }
func (c rgb) bg() string { return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", c.r, c.g, c.b) }

func (c rgb) xterm() int {
	switch c {
	case rpGold:
		return Gold256
	case rpFoam:
		return Foam256
	case rpIris:
		return Iris256
	case rpMuted:
		return Muted256
	case rpRose:
		return Rose256
	case rpOverlay:
		return Overlay256
	case rpBase:
		return Base256
	default:
		return Text256
	}
}

func fgOf(k Kind) rgb {
	switch k {
	case KindComment:
		return rpMuted
	case KindKeyword:
		return rpIris
	case KindLabel:
		return rpFoam
	case KindNumber:
		return rpGold
	case KindOp:
		return rpRose
	case KindSpace:
		return rpText
	default:
		return rpText
	}
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdent(r rune) bool {
	return isIdentStart(r) || unicode.IsDigit(r)
}

func isLabel(s string) bool {
	n := 0
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			n++
		case r >= '0' && r <= '9':
			// VAC1, S0
		default:
			return false
		}
	}
	return n >= 2 || (n == 1 && len(s) >= 2)
}

// TokenizeLine splits one source line into lossless tokens.
func TokenizeLine(line string) []Token {
	runes := []rune(line)
	var toks []Token
	i := 0
	for i < len(runes) {
		r := runes[i]
		if r == ' ' || r == '\t' {
			j := i + 1
			for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t') {
				j++
			}
			toks = append(toks, Token{KindSpace, string(runes[i:j])})
			i = j
			continue
		}
		if r == '#' {
			toks = append(toks, Token{KindComment, string(runes[i:])})
			break
		}
		if r >= '0' && r <= '9' {
			j := i + 1
			for j < len(runes) && runes[j] >= '0' && runes[j] <= '9' {
				j++
			}
			toks = append(toks, Token{KindNumber, string(runes[i:j])})
			i = j
			continue
		}
		if isIdentStart(r) {
			j := i + 1
			for j < len(runes) && isIdent(runes[j]) {
				j++
			}
			word := string(runes[i:j])
			kind := KindIdent
			switch {
			case keywords[word]:
				kind = KindKeyword
			case isLabel(word):
				kind = KindLabel
			}
			toks = append(toks, Token{kind, word})
			i = j
			continue
		}
		toks = append(toks, Token{KindOp, string(r)})
		i++
	}
	return toks
}

type cell struct {
	ch rune
	fg rgb
}

func innerWidth(src string) int {
	w := len([]rune(Title))
	for _, line := range strings.Split(src, "\n") {
		if n := len([]rune(line)); n > w {
			w = n
		}
	}
	return w
}

func sourceLines(src string) []string {
	if src == "" {
		return nil
	}
	return strings.Split(src, "\n")
}

func paintLine(line string, width int) []cell {
	out := make([]cell, 0, width)
	for _, tok := range TokenizeLine(line) {
		fg := fgOf(tok.Kind)
		for _, r := range tok.Text {
			out = append(out, cell{r, fg})
		}
	}
	for len(out) < width {
		out = append(out, cell{' ', rpText})
	}
	if len(out) > width {
		out = out[:width]
	}
	return out
}

func frame(src string) [][]cell {
	inner := innerWidth(src)
	// 1-cell pad each side
	rowW := inner + 2
	title := paintLine(Title, inner)
	for i := range title {
		title[i].fg = rpGold
	}

	var rows [][]cell
	rows = append(rows, borderRow('╭', '─', '╮', rowW))
	rows = append(rows, contentRow(title, inner))
	rows = append(rows, contentRow(paintLine("", inner), inner))
	for _, line := range sourceLines(src) {
		rows = append(rows, contentRow(paintLine(line, inner), inner))
	}
	rows = append(rows, borderRow('╰', '─', '╯', rowW))
	return rows
}

func borderRow(left, mid, right rune, rowW int) []cell {
	out := make([]cell, 0, rowW+2)
	out = append(out, cell{left, rpOverlay})
	for i := 0; i < rowW; i++ {
		out = append(out, cell{mid, rpOverlay})
	}
	out = append(out, cell{right, rpOverlay})
	return out
}

func contentRow(inner []cell, innerW int) []cell {
	out := make([]cell, 0, innerW+4)
	out = append(out, cell{'│', rpOverlay})
	out = append(out, cell{' ', rpText})
	out = append(out, inner...)
	out = append(out, cell{' ', rpText})
	out = append(out, cell{'│', rpOverlay})
	return out
}

// Highlight paints src as a Rose Pine truecolor card. Empty src still
// renders the title chrome.
func Highlight(src string) string {
	rows := frame(src)
	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		for _, c := range row {
			b.WriteString(rpBase.bg())
			b.WriteString(c.fg.fg())
			b.WriteRune(c.ch)
		}
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// CardWidth is the rendered column count of the default Source card.
func CardWidth() int { return innerWidth(Source) + 4 }

// CardHeight is the rendered row count of the default Source card.
func CardHeight() int { return len(sourceLines(Source)) + 4 }
