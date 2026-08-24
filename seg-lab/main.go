// seg-lab: a standalone terminal viewer for the font package.
//
//	font.Render(s, 3)  // height units 1–5
//
//	tab        cycle height 1→2→3→4→5→1
//	1–5        set the height unit
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

	"github.com/theprimeagen/apollo-11/seg-lab/font"
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
	text   string
	height int
}

func newDemo() demoModel {
	return demoModel{text: "HELLO WORLD", height: 3}
}

func (m demoModel) Init() tea.Cmd { return nil }

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab:
			m.height++
			if m.height > 5 {
				m.height = 1
			}
		case tea.KeyEsc:
			m.text = ""
		case tea.KeyBackspace:
			if m.text != "" {
				rs := []rune(m.text)
				m.text = string(rs[:len(rs)-1])
			}
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				if r >= '1' && r <= '5' {
					m.height = int(r - '0')
					continue
				}
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
	b.WriteString(fmt.Sprintf("%s  ·  height %d%s\n", fg(colDim), m.height, reset))
	b.WriteString(fg(colDim) + fmt.Sprintf("font.Render(s, %d)", m.height) + reset + "\n\n")

	if m.text == "" {
		b.WriteString(fg(colNote) + "alphabet catalog" + reset + "\n\n")
		b.WriteString(fg(colSeg) + catalog(m.height) + reset + "\n")
	} else {
		b.WriteString(fg(colSeg) + font.Render(m.text, m.height) + reset + "\n")
	}

	b.WriteString("\n" + fg(colDim) + "tab / 1-5 height · type to edit · backspace · esc clear · ctrl-c quit" + reset + "\n")
	return b.String()
}

func catalog(height int) string {
	if height >= 3 {
		return font.Render("ABCDEFGHIJKLM", height) + "\n\n" +
			font.Render("NOPQRSTUVWXYZ", height)
	}
	return font.Render("ABCDEFGHIJKLMNOPQRSTUVWXYZ", height) + "\n\n" +
		font.Render("0123456789", height)
}

func main() {
	if _, err := tea.NewProgram(newDemo(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "seg-lab:", err)
		os.Exit(1)
	}
}
