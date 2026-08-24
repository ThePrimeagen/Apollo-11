// Package adjustflame is the TUI that edits fire heat thresholds.
// j/k walk the rungs, h/l change the selected amount (0..500), s
// writes the JSON config and quits. The program starts by reading
// that JSON file.
package adjustflame

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/lander-lab/components/fire"
)

// Model is the adjusting-flame item: one cursor, eight thresholds.
type Model struct {
	Path       string
	Thresholds []int
	Cursor     int
	Err        string
	Saved      bool
}

// Open reads the JSON config at path. A missing or invalid file is an error.
func Open(path string) (Model, error) {
	c, err := fire.LoadHeat(path)
	if err != nil {
		return Model{}, err
	}
	return Model{
		Path:       path,
		Thresholds: append([]int(nil), c.Thresholds...),
	}, nil
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch strings.ToLower(key.String()) {
	case "j", "down":
		if m.Cursor < len(m.Thresholds)-1 {
			m.Cursor++
		}
	case "k", "up":
		if m.Cursor > 0 {
			m.Cursor--
		}
	case "l", "right":
		m = m.nudge(1)
	case "h", "left":
		m = m.nudge(-1)
	case "s":
		if err := m.save(); err != nil {
			m.Err = err.Error()
			m.Saved = false
			return m, nil
		}
		m.Err = ""
		m.Saved = true
		return m, tea.Quit
	}
	return m, nil
}

func (m Model) nudge(delta int) Model {
	if m.Cursor < 0 || m.Cursor >= len(m.Thresholds) {
		return m
	}
	n := m.Thresholds[m.Cursor] + delta
	if n < fire.MinThreshold {
		n = fire.MinThreshold
	}
	if n > fire.MaxThreshold {
		n = fire.MaxThreshold
	}
	m.Thresholds[m.Cursor] = n
	return m
}

func (m Model) save() error {
	return fire.HeatConfig{Thresholds: append([]int(nil), m.Thresholds...)}.Save(m.Path)
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString("adjust flame\n")
	b.WriteString("j/k select   h/l change (0..500)   s save+quit\n\n")
	if len(m.Thresholds) == 0 {
		b.WriteString("(no thresholds)\n")
		return b.String()
	}
	rungs := fire.Bands()
	for i, n := range m.Thresholds {
		mark := " "
		if i == m.Cursor {
			mark = ">"
		}
		name, glyph := "?", ' '
		if i < len(rungs) {
			name = rungs[i].Name
			glyph = rungs[i].Glyph
		}
		fmt.Fprintf(&b, "%s %c  %-22s %3d\n", mark, glyph, name, n)
	}
	if m.Err != "" {
		fmt.Fprintf(&b, "\nerror: %s\n", m.Err)
	}
	return b.String()
}
