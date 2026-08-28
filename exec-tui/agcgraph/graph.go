// Package agcgraph is the graphs screen: a STILL — 2.5 seconds of "here is
// what the CPU operates with" under the current switch states, never
// animated. The screen is a COMPOSITE built on the standalone cpugraph
// component: the component owns the portrait — one row per process that
// consumed CPU, grouped under VAC JOBS / CORESET JOBS / NO-PRIORITY OPS /
// COUNTER THEFT headers, names inside a 20-column gutter, light-gray
// gridlines, a HARD WHITE line on the 2.00 s guidance boundary, the
// millisecond axis — and the whole switch API. This screen adds the
// surrounding information: a plain-text legend describing every process
// that ran —
//
//	DOWNRUPT: 25.0ms total :: wakes up every 20ms and runs for 0.2ms
//
// — and four switches on the bottom row. The SERVICER is entered exactly
// ONCE per portrait — the only process that does not repeat — so its row
// is the single ~1.36 s pass stretching toward the white line as load is
// switched on: descent alone fits, the radar steal is the knife edge, and
// the 1668 monitor or the P64 approach guidance push the pass PAST the
// boundary — where its bar turns RED, the overflow to come made visible.
// 1668 and P64 cannot share the DSKY: keying either drops the other.
// Every toggle re-simulates a fresh 2.5 s snapshot; with everything off
// only the hardware cadences remain. The opening portrait is the healthy
// CPU: descent on, monitor off, radar steal off, approach off.
package agcgraph

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/cpugraph"
	msim "github.com/theprimeagen/apollo-11/msim"
)

// Model is the graphs screen: the graph component plus the legend and
// the switch row — one frozen snapshot per switch configuration.
type Model struct {
	graph *cpugraph.Graph
	w, h  int
}

// New opens on the healthy portrait: descent on, monitor off, steal off,
// approach off.
func New() Model {
	return Model{w: 200, h: 45, graph: cpugraph.New()}
}

// Init schedules nothing: the screen never animates.
func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch msg.Code {
		case 'q':
			return m, tea.Quit
		case 'd':
			m.graph.SetDescent(!m.graph.Descent())
		case '1':
			// the DELTAH monitor owns the DSKY — the component's API
			// drops the approach display when the monitor is keyed
			m.graph.SetMonitor(!m.graph.Monitor())
		case 'r':
			m.graph.SetRadar(!m.graph.Radar())
		case 'p':
			// P64's flashing V06N64 owns the DSKY — the component's API
			// drops the monitor when the approach is keyed
			m.graph.SetApproach(!m.graph.Approach())
		}
	}
	return m, nil
}

var (
	sDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sName   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sOn     = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	sOff    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	sSwitch = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
)

func ms1(n msim.Nanos) string {
	return fmt.Sprintf("%.1fms", float64(n)/1e6)
}

// legend lists every process that ran in the window, with its totals, in
// the same order as the lanes — and the theft's stolen total last. The
// component hands over the data; the composite writes the prose.
func (m Model) legend() []string {
	var out []string
	for _, p := range m.graph.Running() {
		avg := p.Busy
		if p.Fires > 0 {
			avg = p.Busy / msim.Nanos(p.Fires)
		}
		out = append(out,
			sName.Render(fmt.Sprintf(" %s:", p.Name))+
				sDim.Render(fmt.Sprintf(" %s total :: wakes up every %s and runs for %s",
					ms1(p.Busy), p.Period, ms1(avg))))
	}
	if stolen := m.graph.Stolen(); stolen > 0 {
		out = append(out,
			sName.Render(" RR CDU:")+
				sDim.Render(fmt.Sprintf(" %s total :: hardware counter steal — 12,800 pulses/s, time only, zero memory",
					ms1(stolen))))
	}
	return out
}

func onOff(on bool) string {
	if on {
		return sOn.Render("ON ")
	}
	return sOff.Render("OFF")
}

func (m Model) View() tea.View {
	lines := m.graph.Rows(m.w)

	lines = append(lines, "")
	lines = append(lines, m.legend()...)
	lines = append(lines, "")
	lines = append(lines, " "+
		sSwitch.Render("[d] DESCENT ")+onOff(m.graph.Descent())+"    "+
		sSwitch.Render("[1] 1668 ")+onOff(m.graph.Monitor())+"    "+
		sSwitch.Render("[r] RADAR STEAL ")+onOff(m.graph.Radar())+"    "+
		sSwitch.Render("[p] P64 ")+onOff(m.graph.Approach())+"      "+
		sDim.Render("q quit"))

	if len(lines) > m.h && m.h > 0 {
		lines = lines[:m.h]
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}
