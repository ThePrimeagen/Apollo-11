// seg-lab: a standalone terminal segmented-letter viewer.
//
// Unicode only encodes segmented digits (U+1FBF0–U+1FBF9). Letters have no
// codepoints, so 7-seg and 14-seg compose them from box-drawing strokes.
//
//	tab        cycle unicode / 7-seg / 14-seg
//	type       edit the message (q is a letter)
//	backspace  delete
//	esc        clear — shows the A–Z catalog
//	ctrl-c     quit
package main

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/seg-lab/seg"
)

const (
	colSeg   = 48
	colDim   = 240
	colTitle = 214
	colNote  = 245
)

func fg(n int) string { return fmt.Sprintf("\x1b[38;5;%dm", n) }

const reset = "\x1b[0m"

type demoModel struct {
	text  string
	style seg.Style
}

func newDemo() demoModel {
	return demoModel{text: "APOLLO 11", style: seg.StyleFourteen}
}

func (m demoModel) Init() tea.Cmd { return nil }

func (m demoModel) nextStyle() seg.Style {
	switch m.style {
	case seg.StyleFourteen:
		return seg.StyleUnicode
	case seg.StyleUnicode:
		return seg.StyleSeven
	default:
		return seg.StyleFourteen
	}
}

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab:
			m.style = m.nextStyle()
		case tea.KeyEsc:
			m.text = ""
		case tea.KeyBackspace:
			if m.text != "" {
				rs := []rune(m.text)
				m.text = string(rs[:len(rs)-1])
			}
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				if unicode.IsPrint(r) && r != '\t' {
					m.text += string(unicode.ToUpper(r))
				}
			}
		}
	}
	return m, nil
}

func (m demoModel) View() string {
	var b strings.Builder
	b.WriteString(fg(colTitle) + "SEGMENTED LETTER VIEWER" + reset)
	b.WriteString(fg(colDim) + "  ·  " + m.style.String() + reset + "\n\n")

	b.WriteString(fg(colNote) + "Unicode digits U+1FBF0–U+1FBF9  " + reset)
	b.WriteString(fg(colSeg) + seg.Render("0123456789", seg.StyleUnicode) + reset + "\n")
	b.WriteString(fg(colDim) + "No segmented letter codepoints exist. 7-seg / 14-seg are composed." + reset + "\n\n")

	if m.text == "" {
		b.WriteString(fg(colNote) + "alphabet catalog" + reset + "\n\n")
		b.WriteString(fg(colSeg) + catalog(m.style) + reset + "\n")
	} else {
		b.WriteString(fg(colSeg) + seg.Render(m.text, m.style) + reset + "\n")
		if m.style == seg.StyleUnicode {
			b.WriteString("\n" + fg(colDim) + "letters stay blank in unicode — tab to 7-seg or 14-seg" + reset + "\n")
		} else if m.style == seg.StyleSeven {
			b.WriteString("\n" + fg(colDim) + "7-seg cannot draw K M V W X — tab to 14-seg for those" + reset + "\n")
		}
	}

	b.WriteString("\n" + fg(colDim) + "tab style · type to edit · backspace · esc clear · ctrl-c quit" + reset + "\n")
	return b.String()
}

func catalog(style seg.Style) string {
	switch style {
	case seg.StyleUnicode:
		return seg.Render("0123456789", style)
	case seg.StyleSeven:
		return seg.Render("ABCDEFGHIJ", style) + "\n\n" +
			seg.Render("LNOPQRSTUY", style) + "\n\n" +
			seg.Render("Z0123456789", style)
	default:
		return seg.Render("ABCDEFGHIJKLM", style) + "\n\n" +
			seg.Render("NOPQRSTUVWXYZ", style) + "\n\n" +
			seg.Render("0123456789", style)
	}
}

func main() {
	if _, err := tea.NewProgram(newDemo(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "seg-lab:", err)
		os.Exit(1)
	}
}
