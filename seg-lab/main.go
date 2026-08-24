// seg-lab: a standalone terminal segmented-letter viewer.
//
// Unicode only encodes segmented digits (U+1FBF0–U+1FBF9). Letters have no
// codepoints, so 7-seg and 14-seg compose them from box-drawing strokes.
//
//	tab        cycle alpha / unicode / 7-seg / 14-seg
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

	tea "charm.land/bubbletea/v2"

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
	return demoModel{text: "HELLO WORLD", style: seg.StyleAlpha}
}

func (m demoModel) Init() tea.Cmd { return nil }

func (m demoModel) nextStyle() seg.Style {
	switch m.style {
	case seg.StyleAlpha:
		return seg.StyleUnicode
	case seg.StyleUnicode:
		return seg.StyleSeven
	case seg.StyleSeven:
		return seg.StyleFourteen
	default:
		return seg.StyleAlpha
	}
}

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch msg.Code {
		case tea.KeyTab:
			m.style = m.nextStyle()
		case tea.KeyEsc:
			m.text = ""
		case tea.KeyBackspace:
			if m.text != "" {
				rs := []rune(m.text)
				m.text = string(rs[:len(rs)-1])
			}
		case tea.KeySpace:
			// v1 delivered space as a non-rune key, so it never typed;
			// the port keeps that behavior.
		default:
			for _, r := range msg.Text {
				if unicode.IsPrint(r) && r != '\t' {
					m.text += string(unicode.ToUpper(r))
				}
			}
		}
	}
	return m, nil
}

func (m demoModel) View() tea.View {
	var b strings.Builder
	b.WriteString(fg(colTitle) + "SEGMENTED LETTER VIEWER" + reset)
	b.WriteString(fg(colDim) + "  ·  " + m.style.String() + reset + "\n\n")

	b.WriteString(fg(colNote) + "Unicode digits U+1FBF0–U+1FBF9  " + reset)
	b.WriteString(fg(colSeg) + seg.Render("0123456789", seg.StyleUnicode) + reset + "\n")
	b.WriteString(fg(colDim) + "No official letter codepoints. alpha = 14-seg font (U+E000–U+E019)." + reset + "\n\n")

	if m.text == "" {
		b.WriteString(fg(colNote) + "alphabet catalog" + reset + "\n\n")
		b.WriteString(fg(colSeg) + catalog(m.style) + reset + "\n")
	} else {
		b.WriteString(fg(colSeg) + seg.Render(m.text, m.style) + reset + "\n")
		if m.style == seg.StyleUnicode {
			b.WriteString("\n" + fg(colDim) + "letters stay blank in unicode — tab to alpha" + reset + "\n")
		} else if m.style == seg.StyleSeven {
			b.WriteString("\n" + fg(colDim) + "7-seg cannot draw K M V W X — tab to 14-seg or alpha" + reset + "\n")
		} else if m.style == seg.StyleAlpha {
			b.WriteString("\n" + fg(colDim) + "one cell per letter (needs Segmented Alpha font)" + reset + "\n")
		}
	}

	b.WriteString("\n" + fg(colDim) + "tab style · type to edit · backspace · esc clear · ctrl-c quit" + reset + "\n")
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func catalog(style seg.Style) string {
	switch style {
	case seg.StyleUnicode:
		return seg.Render("0123456789", style)
	case seg.StyleAlpha:
		return seg.Render("ABCDEFGHIJKLMNOPQRSTUVWXYZ", style) + "\n" +
			seg.Render("0123456789", style)
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
	if _, err := tea.NewProgram(newDemo()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "seg-lab:", err)
		os.Exit(1)
	}
}
