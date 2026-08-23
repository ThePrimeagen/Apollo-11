// button-lab: a playground for cockpit-style terminal buttons. Four styles,
// three buttons each; h/l/j/k to move, enter or space to press, q to quit.
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

type labRow struct {
	name    string
	blurb   string
	buttons []button.Button
}

type labModel struct {
	rows []labRow
	row  int
	col  int
}

func newLab() labModel {
	mk := func(st button.Style, labels ...string) []button.Button {
		out := make([]button.Button, len(labels))
		for i, l := range labels {
			out[i] = button.New(l, st)
		}
		return out
	}
	m := labModel{rows: []labRow{
		{"PANEL", "6×3 face · half-cursor bezel · ▒ off, ▓ ring + hot center on", mk(button.Panel, "ARM", "RADAR", "N68")},
		{"HALF-CELL", "two colors in one cell: ▄ face-on-bezel", mk(button.HalfCell, "SCE", "AUX", "TLM")},
		{"PROTRUDE", "sticks up half a cell — pressing sinks it in and lights it", mk(button.Protrude, "STAGE", "ABORT", "TEST")},
		{"SWITCH", "cockpit toggle: lever down dull red, flick up lit two-tone", mk(button.Switch, "MSTR", "PGNS", "AGS")},
	}}
	m.syncFocus()
	return m
}

func (m *labModel) syncFocus() {
	for ri := range m.rows {
		for ci := range m.rows[ri].buttons {
			m.rows[ri].buttons[ci].Focused = ri == m.row && ci == m.col
		}
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
			m.rows[m.row].buttons[m.col].Toggle()
			return m, nil
		case tea.KeyLeft:
			m.col = (m.col + 2) % 3
		case tea.KeyRight:
			m.col = (m.col + 1) % 3
		case tea.KeyUp:
			m.row = (m.row + len(m.rows) - 1) % len(m.rows)
		case tea.KeyDown:
			m.row = (m.row + 1) % len(m.rows)
		}
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'q':
				return m, tea.Quit
			case 'h':
				m.col = (m.col + 2) % 3
			case 'l':
				m.col = (m.col + 1) % 3
			case 'k':
				m.row = (m.row + len(m.rows) - 1) % len(m.rows)
			case 'j':
				m.row = (m.row + 1) % len(m.rows)
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

func centered(s string, w int) string {
	if len(s) > w {
		s = s[:w]
	}
	pad := w - len(s)
	return strings.Repeat(" ", pad/2) + s + strings.Repeat(" ", pad-pad/2)
}

func (m labModel) View() string {
	var b strings.Builder
	b.WriteString(title.Render("BUTTON LAB · 1960s cockpit controls") +
		dim.Render("   h/l j/k move · enter/space press · q quit"))
	b.WriteString("\n\n")
	for ri, row := range m.rows {
		name := dim.Render(fmt.Sprintf("%-9s", row.name))
		if ri == m.row {
			name = title.Render(fmt.Sprintf("%-9s", row.name))
		}
		b.WriteString(name + dim.Render(" "+row.blurb) + "\n")
		var cols []string
		for _, bt := range row.buttons {
			w, _ := bt.Size()
			label := dim.Render(centered(bt.Label, w))
			if bt.Focused {
				label = bright.Render(centered(bt.Label, w))
			}
			cols = append(cols, bt.Render()+"\n"+label)
		}
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cols[0], "   ", cols[1], "   ", cols[2]))
		b.WriteString("\n\n")
	}
	return b.String()
}

func main() {
	if _, err := tea.NewProgram(newLab(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "button-lab:", err)
		os.Exit(1)
	}
}
