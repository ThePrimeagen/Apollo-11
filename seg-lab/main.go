// seg-lab: a standalone terminal viewer for the font package.
//
//	font.Render(s, font.Small)  // regular writing
//	font.Render(s, font.Large)  // large writing
//
//	tab        small ↔ large
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
	text string
	size font.Size
}

func newDemo() demoModel {
	return demoModel{text: "HELLO WORLD", size: font.Large}
}

func (m demoModel) Init() tea.Cmd { return nil }

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyTab:
			if m.size == font.Large {
				m.size = font.Small
			} else {
				m.size = font.Large
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
	b.WriteString(fg(colDim) + "  ·  " + m.size.String() + reset + "\n")
	sizeName := "Small"
	if m.size == font.Large {
		sizeName = "Large"
	}
	b.WriteString(fg(colDim) + "font.Render(s, font." + sizeName + ")" + reset + "\n\n")

	if m.text == "" {
		b.WriteString(fg(colNote) + "alphabet catalog" + reset + "\n\n")
		b.WriteString(fg(colSeg) + catalog(m.size) + reset + "\n")
	} else {
		b.WriteString(fg(colSeg) + font.Render(m.text, m.size) + reset + "\n")
	}

	b.WriteString("\n" + fg(colDim) + "tab size · type to edit · backspace · esc clear · ctrl-c quit" + reset + "\n")
	return b.String()
}

func catalog(size font.Size) string {
	if size == font.Large {
		return font.Render("ABCDEFGHIJKLM", size) + "\n\n" +
			font.Render("NOPQRSTUVWXYZ", size)
	}
	return font.Render("ABCDEFGHIJKLMNOPQRSTUVWXYZ", size) + "\n\n" +
		font.Render("0123456789", size)
}

func main() {
	if _, err := tea.NewProgram(newDemo(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "seg-lab:", err)
		os.Exit(1)
	}
}
