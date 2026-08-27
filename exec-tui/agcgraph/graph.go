// Package agcgraph is the graphs screen: three CPU lanes over a 180-column
// window covering 2.000 s of machine time — no header chrome, a 20-column
// label gutter, light-gray vertical gridlines marking time (every 100 ms,
// brighter on the seconds), and the same three switches on the bottom row.
//
// Each lane is three rows tall; every column is a small vertical bar — the
// fraction of that ~11 ms slice the CPU spent in the lane's class:
//
//	VAC JOBS          jobs holding a core set AND a VAC area
//	CORESET JOBS      jobs holding a core set only
//	NO-PRIORITY OPS   tasks & interrupts: cpu only, no memory
//
// The screen opens FROZEN on one complete prerun 2 s cycle, so the anatomy
// of the guidance cycle sits still and readable. space runs and freezes;
// d / 1 / r flick the switches (and unfreeze); q quits.
package agcgraph

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	msim "github.com/theprimeagen/apollo-11/msim"
)

// frameMS is wall (and sim) milliseconds per frame: 20 fps, 1:1 time.
const frameMS = 50

// windowMS is the plotted span: 2.000 s across the plot columns.
const windowMS = 2000

// gutter is the label column budget.
const gutter = 20

// maxPlot is the plot column budget: 180 columns for the 2 s window.
const maxPlot = 180

type frameMsg struct{}

func frameTick() tea.Cmd {
	return tea.Tick(frameMS*time.Millisecond, func(time.Time) tea.Msg { return frameMsg{} })
}

// Model is the graphs screen over one live machine.
type Model struct {
	live   *msim.Live
	w, h   int
	frozen bool
}

// New pre-runs exactly one 2 s cycle and opens frozen on it.
func New(l *msim.Live) Model {
	l.StepMS(windowMS)
	return Model{live: l, w: 200, h: 40, frozen: true}
}

func (m Model) Init() tea.Cmd { return frameTick() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case frameMsg:
		if !m.frozen {
			m.live.StepMS(frameMS)
		}
		return m, frameTick()
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch msg.Code {
		case 'q':
			return m, tea.Quit
		case tea.KeySpace:
			m.frozen = !m.frozen
		case 'd':
			m.live.SetDescent(!m.live.DescentOn())
			m.frozen = false
		case '1':
			m.live.SetMonitor(!m.live.MonitorOn())
			m.frozen = false
		case 'r':
			m.live.SetRadar(!m.live.RadarOn())
			m.frozen = false
		}
	}
	return m, nil
}

var (
	sLabel  = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Bold(true)
	sVac    = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	sCore   = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	sOps    = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	sGrid   = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	sGridS  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sOn     = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	sOff    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	sSwitch = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
)

var blockRunes = []rune("▁▂▃▄▅▆▇█")

type lane struct {
	label string
	style lipgloss.Style
	class func(msim.Sample) msim.Nanos
}

var lanes = []lane{
	{"VAC JOBS", sVac, func(s msim.Sample) msim.Nanos { return s.VacNs }},
	{"CORESET JOBS", sCore, func(s msim.Sample) msim.Nanos { return s.CoreNs }},
	{"NO-PRIORITY OPS", sOps, func(s msim.Sample) msim.Nanos { return s.OpsNs }},
}

// column is one plotted slice: the bar level (0..24 eighths over three
// rows) and its grid marking.
type column struct {
	level int
	grid  int // 0 none, 1 light (100 ms), 2 strong (1 s)
}

// columns buckets the last windowMS of samples into `plot` columns for one
// lane class.
func (m Model) columns(plot int, class func(msim.Sample) msim.Nanos) []column {
	e := m.live.Engine()
	samples := e.Samples()
	endMs := int(e.Now() / msim.Millisecond)
	startMs := endMs - windowMS
	out := make([]column, plot)
	for i := 0; i < plot; i++ {
		loMs := startMs + i*windowMS/plot
		hiMs := startMs + (i+1)*windowMS/plot
		if hiMs == loMs {
			hiMs = loMs + 1
		}
		// grid: does a 100 ms boundary land in [lo, hi)?
		if lo100, hi100 := ceilDiv(loMs, 100), ceilDiv(hiMs-1, 100); lo100 <= (hiMs-1)/100 && hi100 >= 0 {
			if b := lo100 * 100; b >= loMs && b < hiMs {
				out[i].grid = 1
				if b%1000 == 0 {
					out[i].grid = 2
				}
			}
		}
		var busy msim.Nanos
		for ms := loMs; ms < hiMs; ms++ {
			if ms < 0 || ms >= len(samples) {
				continue
			}
			busy += class(samples[ms])
		}
		span := msim.Nanos(hiMs-loMs) * msim.Millisecond
		lvl := int((busy*24 + span/2) / span)
		if busy > 0 && lvl == 0 {
			lvl = 1 // sub-slice work must stay visible
		}
		if lvl > 24 {
			lvl = 24
		}
		out[i].level = lvl
	}
	return out
}

func ceilDiv(a, b int) int {
	if a >= 0 {
		return (a + b - 1) / b
	}
	return a / b
}

// laneRows renders one lane's three rows for the given columns.
func laneRows(l lane, cols []column) [3]string {
	var rows [3]string
	for t := 0; t < 3; t++ {
		var b strings.Builder
		for _, c := range cols {
			cell := c.level - (2-t)*8
			switch {
			case cell >= 8:
				b.WriteString(l.style.Render("█"))
			case cell > 0:
				b.WriteString(l.style.Render(string(blockRunes[cell-1])))
			case c.grid == 2:
				b.WriteString(sGridS.Render("│"))
			case c.grid == 1:
				b.WriteString(sGrid.Render("│"))
			default:
				b.WriteString(" ")
			}
		}
		rows[t] = b.String()
	}
	return rows
}

func onOff(on bool) string {
	if on {
		return sOn.Render("ON ")
	}
	return sOff.Render("OFF")
}

func (m Model) View() tea.View {
	g := gutter
	if m.w < g+10 {
		g = 0
	}
	plot := m.w - g
	if plot > maxPlot {
		plot = maxPlot
	}
	if plot < 1 {
		plot = 1
	}

	pad := strings.Repeat(" ", g)
	var lines []string
	for _, l := range lanes {
		rows := laneRows(l, m.columns(plot, l.class))
		for t := 0; t < 3; t++ {
			left := pad
			if t == 1 && g > 0 {
				left = sLabel.Render(fmt.Sprintf("%-*s", g, l.label))
			}
			lines = append(lines, left+rows[t])
		}
		lines = append(lines, "")
	}

	state := "space run"
	if !m.frozen {
		state = "space freeze"
	}
	lines = append(lines, " "+
		sSwitch.Render("[d] DESCENT ")+onOff(m.live.DescentOn())+"    "+
		sSwitch.Render("[1] 1668 ")+onOff(m.live.MonitorOn())+"    "+
		sSwitch.Render("[r] RADAR STEAL ")+onOff(m.live.RadarOn())+"      "+
		sDim.Render(state+" · q quit"))

	if len(lines) > m.h && m.h > 0 {
		lines = lines[:m.h]
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}
