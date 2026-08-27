// Package agcgraph is the graphs screen: a STILL — 2.5 seconds of "here is
// what the CPU operates with" under the current switch states, never
// animated. No header chrome: three lanes (each three rows tall) over a
// 180-column window, labels inside a 20-column gutter, light-gray vertical
// gridlines every 100 ms (brighter on the seconds), then a plain-text
// legend describing every job that ran in the interval —
//
//	DOWNRUPT: 25.0ms total :: wakes up every 20ms and runs for 0.2ms
//
// — and the same three switches on the bottom row. Every toggle
// re-simulates a fresh 2.5 s snapshot under the new configuration; with
// everything off only the hardware cadences remain. The opening portrait
// is the healthy CPU: descent on, monitor off, radar steal off.
package agcgraph

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	msim "github.com/theprimeagen/apollo-11/msim"
)

// windowMS is the portrait span: 2.5 s across the plot columns.
const windowMS = 2500

// gutter is the label column budget.
const gutter = 20

// maxPlot is the plot column budget.
const maxPlot = 180

// Model is the graphs screen: one frozen snapshot per switch configuration.
type Model struct {
	live    *msim.Live
	w, h    int
	descent bool
	monitor bool
	radar   bool
}

// New opens on the healthy portrait: descent on, monitor off, steal off.
func New() Model {
	m := Model{w: 200, h: 45, descent: true, monitor: false, radar: false}
	m.rebuild()
	return m
}

// rebuild re-simulates a fresh 2.5 s snapshot under the current switches.
func (m *Model) rebuild() {
	l := msim.NewLive()
	l.SetRadar(m.radar)
	l.SetDescent(m.descent)
	l.SetMonitor(m.monitor)
	l.StepMS(windowMS)
	m.live = l
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
			m.descent = !m.descent
			m.rebuild()
		case '1':
			m.monitor = !m.monitor
			m.rebuild()
		case 'r':
			m.radar = !m.radar
			m.rebuild()
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
	sName   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
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

// columns buckets the snapshot's window into `plot` columns for one class.
func (m Model) columns(plot int, class func(msim.Sample) msim.Nanos) []column {
	samples := m.live.Engine().Samples()
	out := make([]column, plot)
	for i := 0; i < plot; i++ {
		loMs := i * windowMS / plot
		hiMs := (i + 1) * windowMS / plot
		if hiMs == loMs {
			hiMs = loMs + 1
		}
		if b := (loMs + 99) / 100 * 100; b >= loMs && b < hiMs {
			out[i].grid = 1
			if b%1000 == 0 {
				out[i].grid = 2
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

// legendRow is one describable process: how it is activated and how often.
type legendRow struct {
	name   string
	period string
	count  func(*msim.Engine) int
}

func spawns(name string) func(*msim.Engine) int {
	return func(e *msim.Engine) int { return e.SpawnCount(name) }
}

func tasks(name string) func(*msim.Engine) int {
	return func(e *msim.Engine) int { return e.TaskFires(name) }
}

func rupts(name string) func(*msim.Engine) int {
	return func(e *msim.Engine) int { return e.InterruptFires(name) }
}

var legendRows = []legendRow{
	{"SERVICER", "2s", spawns("SERVICER")},
	{"MAKEPLAY", "2s", spawns("MAKEPLAY")},
	{"MONDO", "1s", spawns("MONDO")},
	{"LRHJOB", "2s", spawns("LRHJOB")},
	{"LRVJOB", "2s", spawns("LRVJOB")},
	{"1/GYRO", "2s", spawns("1/GYRO")},
	{"CHARIN", "keystroke", spawns("CHARIN")},
	{"READACCS", "2s", tasks("READACCS")},
	{"R10,R11", "250ms", tasks("R10,R11")},
	{"LRHTASK", "2s", tasks("LRHTASK")},
	{"LRVTASK", "2s", tasks("LRVTASK")},
	{"MONREQ", "1s", tasks("MONREQ")},
	{"DAP", "100ms", rupts("DAP")},
	{"T4RUPT", "120ms", rupts("T4RUPT")},
	{"DOWNRUPT", "20ms", rupts("DOWNRUPT")},
}

func ms1(n msim.Nanos) string {
	return fmt.Sprintf("%.1fms", float64(n)/1e6)
}

// legend lists every process that ran in the window, with its totals.
func (m Model) legend() []string {
	e := m.live.Engine()
	var out []string
	for _, r := range legendRows {
		busy := e.BusyNs(r.name)
		if busy <= 0 {
			continue
		}
		n := r.count(e)
		avg := busy
		if n > 0 {
			avg = busy / msim.Nanos(n)
		}
		out = append(out,
			sName.Render(fmt.Sprintf(" %s:", r.name))+
				sDim.Render(fmt.Sprintf(" %s total :: wakes up every %s and runs for %s",
					ms1(busy), r.period, ms1(avg))))
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

	lines = append(lines, m.legend()...)
	lines = append(lines, "")
	lines = append(lines, " "+
		sSwitch.Render("[d] DESCENT ")+onOff(m.descent)+"    "+
		sSwitch.Render("[1] 1668 ")+onOff(m.monitor)+"    "+
		sSwitch.Render("[r] RADAR STEAL ")+onOff(m.radar)+"      "+
		sDim.Render("q quit"))

	if len(lines) > m.h && m.h > 0 {
		lines = lines[:m.h]
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}
