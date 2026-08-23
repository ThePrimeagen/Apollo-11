// button-lab: a playground for the cockpit toggle switch used by exec-tui.
// Three switches; h/l to move, space or enter to flick, q to quit.
// See button/button.go for the reusable component.
package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/theprimeagen/apollo-11/button-lab/button"
)

type labModel struct {
	switches []button.Switch
	col      int
}

func newLab() labModel {
	m := labModel{switches: []button.Switch{
		button.NewSwitch("MSTR"),
		button.NewSwitch("PGNS"),
		button.NewSwitch("AGS"),
	}}
	m.syncFocus()
	return m
}

func (m *labModel) syncFocus() {
	for i := range m.switches {
		m.switches[i].Focused = i == m.col
	}
}

func (m labModel) Init() tea.Cmd { return nil }

func (m labModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter, tea.KeySpace:
			m.switches[m.col].Toggle()
			return m, nil
		case tea.KeyLeft:
			m.col = (m.col + 2) % 3
		case tea.KeyRight:
			m.col = (m.col + 1) % 3
		}
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'q':
				return m, tea.Quit
			case 'h':
				m.col = (m.col + 2) % 3
			case 'l':
				m.col = (m.col + 1) % 3
			}
		}
		m.syncFocus()
	}
	return m, nil
}

var (
	dim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bright = lipgloss.NewStyle().Foreground(lipgloss.Color("223")).Bold(true)
	title  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
)

func (m labModel) View() string {
	var b strings.Builder
	b.WriteString(title.Render("SWITCH LAB · cockpit toggles") +
		dim.Render("   h/l move · space/enter flick · q quit"))
	b.WriteString("\n\n")
	var cols []string
	for _, sw := range m.switches {
		w, _ := sw.Size()
		label := dim.Render(sw.Label)
		if sw.Focused {
			label = bright.Render(sw.Label)
		}
		cols = append(cols, lipgloss.JoinVertical(lipgloss.Center,
			sw.Render(), lipgloss.PlaceHorizontal(w, lipgloss.Center, label)))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cols[0], "   ", cols[1], "   ", cols[2]))
	b.WriteString("\n")
	return b.String()
}

func main() {
	if _, err := tea.NewProgram(newLab(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "button-lab:", err)
		os.Exit(1)
	}
}
