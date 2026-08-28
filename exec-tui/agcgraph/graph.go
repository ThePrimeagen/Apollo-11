// Package agcgraph is the graphs screen: a STILL — 2.5 seconds of "here is
// what the CPU operates with" under the current switch states, never
// animated. No header chrome: one row per process that consumed CPU,
// grouped under VAC JOBS / CORESET JOBS / NO-PRIORITY OPS headers, names
// inside a 20-column gutter, light-gray vertical gridlines every 100 ms
// (brighter on the seconds) and a HARD WHITE line on the 2.00 s guidance
// boundary, then a plain-text legend describing every process that ran —
//
//	DOWNRUPT: 25.0ms total :: wakes up every 20ms and runs for 0.2ms
//
// — and four switches on the bottom row. The SERVICER is entered exactly
// ONCE per portrait — the only process that does not repeat — so its row
// is the single ~1.36 s pass stretching toward the white line as load is
// switched on: descent alone fits, the radar steal is the knife edge, and
// the 1668 monitor or the P64 approach guidance push the pass PAST the
// boundary. 1668 and P64 cannot share the DSKY: keying either drops the
// other. Every toggle re-simulates a fresh 2.5 s snapshot; with everything
// off only the hardware cadences remain. The opening portrait is the
// healthy CPU: descent on, monitor off, radar steal off, approach off.
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

// boundaryMS is the guidance boundary the hard white line marks: the
// instant the next READACCS arrives and a finished SERVICER would have
// already reached ENDOFJOB.
const boundaryMS = 2000

// gutter is the label column budget.
const gutter = 20

// maxPlot is the plot column budget.
const maxPlot = 180

// Model is the graphs screen: one frozen snapshot per switch configuration.
type Model struct {
	live     *msim.Live
	w, h     int
	descent  bool
	monitor  bool
	radar    bool
	approach bool
}

// New opens on the healthy portrait: descent on, monitor off, steal off,
// approach off.
func New() Model {
	m := Model{w: 200, h: 45, descent: true}
	m.rebuild()
	return m
}

// rebuild re-simulates a fresh 2.5 s snapshot under the current switches.
// The portrait's fixed rules: the SERVICER is entered once (everything
// else keeps its timer), and the theft sweep rides its worst-case crest —
// the RESEARCH.md "worst 2 s window" — instead of the flight window's
// floor dwell.
func (m *Model) rebuild() {
	l := msim.NewLive()
	l.SetRadar(m.radar)
	l.SetDescent(m.descent)
	l.SetServicerOneShot(true)
	l.SetApproach(m.approach)
	l.Engine().SetTheftPhaseMS(msim.TheftPeakPhaseMS)
	if m.monitor {
		// the monitor is already up as the portrait opens, on the flight's
		// ENTR phase: each 1 Hz refresh lands .985 into its second, the
		// second one straddling the white line
		msim.StartMonitor(l.Engine(), -15*msim.Millisecond)
	}
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
			// the DELTAH monitor owns the DSKY — the approach's landing
			// display cannot be up at the same time
			m.monitor = !m.monitor
			if m.monitor {
				m.approach = false
			}
			m.rebuild()
		case 'r':
			m.radar = !m.radar
			m.rebuild()
		case 'p':
			// P64's flashing V06N64 owns the DSKY — keying the approach
			// drops the monitor
			m.approach = !m.approach
			if m.approach {
				m.monitor = false
			}
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
	sBound  = lipgloss.NewStyle().Foreground(lipgloss.Color("231")).Bold(true)
	sDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sName   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sOn     = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	sOff    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	sSwitch = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
)

var blockRunes = []rune("▁▂▃▄▅▆▇█")

// process groups: who holds what while consuming the CPU.
const (
	groupVac = iota
	groupCore
	groupOps
)

var groupLabels = [...]string{"VAC JOBS", "CORESET JOBS", "NO-PRIORITY OPS"}
var groupStyles = [...]lipgloss.Style{sVac, sCore, sOps}

// proc is one describable process: its lane group under the current
// switches, how it is activated, and how often.
type proc struct {
	name   string
	period string
	count  func(*msim.Engine) int
	group  func(Model) int
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

func fixed(g int) func(Model) int { return func(Model) int { return g } }

// procs is the catalog, in display order. A row (and its legend line)
// appears only when the process consumed CPU in the window. MAKEPLAY moves
// with the display form: the P63 static V06N63 is NOVAC, the approach's
// flashing V06N64 holds a VAC while it sleeps awaiting PRO.
var procs = []proc{
	{"SERVICER", "2s", spawns("SERVICER"), fixed(groupVac)},
	{"MAKEPLAY", "2s", spawns("MAKEPLAY"), func(m Model) int {
		if m.approach {
			return groupVac
		}
		return groupCore
	}},
	{"HIGATJOB", "high gate", spawns("HIGATJOB"), fixed(groupVac)},
	{"LRHJOB", "2s", spawns("LRHJOB"), fixed(groupCore)},
	{"LRVJOB", "2s", spawns("LRVJOB"), fixed(groupCore)},
	{"MONDO", "1s", spawns("MONDO"), fixed(groupCore)},
	{"CHARIN", "keystroke", spawns("CHARIN"), fixed(groupCore)},
	{"1/GYRO", "2s", spawns("1/GYRO"), fixed(groupCore)},
	{"READACCS", "2s", tasks("READACCS"), fixed(groupOps)},
	{"R10,R11", "250ms", tasks("R10,R11"), fixed(groupOps)},
	{"LRHTASK", "2s", tasks("LRHTASK"), fixed(groupOps)},
	{"LRVTASK", "2s", tasks("LRVTASK"), fixed(groupOps)},
	{"HIGATASK", "high gate", tasks("HIGATASK"), fixed(groupOps)},
	{"MONREQ", "1s", tasks("MONREQ"), fixed(groupOps)},
	{"DAP", "100ms", rupts("DAP"), fixed(groupOps)},
	{"T4RUPT", "120ms", rupts("T4RUPT"), fixed(groupOps)},
	{"DOWNRUPT", "20ms", rupts("DOWNRUPT"), fixed(groupOps)},
}

// column is one plotted slice: the bar level (0..8 eighths of one row) and
// its grid marking.
type column struct {
	level int
	grid  int // 0 none, 1 light (100 ms), 2 strong (1 s)
}

// columns buckets the snapshot's window into `plot` columns for one
// per-sample series.
func (m Model) columns(plot int, series func(msim.Sample) msim.Nanos) []column {
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
			busy += series(samples[ms])
		}
		span := msim.Nanos(hiMs-loMs) * msim.Millisecond
		lvl := int((busy*8 + span/2) / span)
		if busy > 0 && lvl == 0 {
			lvl = 1 // sub-slice work must stay visible
		}
		if lvl > 8 {
			lvl = 8
		}
		out[i].level = lvl
	}
	return out
}

// laneRow renders one process's row: bars over gridlines, with the hard
// white boundary line cutting through everything — bars included.
func laneRow(st lipgloss.Style, cols []column, bcol int) string {
	var b strings.Builder
	for i, c := range cols {
		switch {
		case i == bcol:
			b.WriteString(sBound.Render("│"))
		case c.level > 0:
			b.WriteString(st.Render(string(blockRunes[c.level-1])))
		case c.grid == 2:
			b.WriteString(sGridS.Render("│"))
		case c.grid == 1:
			b.WriteString(sGrid.Render("│"))
		default:
			b.WriteString(" ")
		}
	}
	return b.String()
}

// running reports whether the process consumed any CPU in the window.
func running(e *msim.Engine, name string) bool { return e.BusyNs(name) > 0 }

func ms1(n msim.Nanos) string {
	return fmt.Sprintf("%.1fms", float64(n)/1e6)
}

// legend lists every process that ran in the window, with its totals, in
// the same order as the lanes.
func (m Model) legend() []string {
	e := m.live.Engine()
	var out []string
	for g := range groupLabels {
		for _, p := range procs {
			if p.group(m) != g || !running(e, p.name) {
				continue
			}
			busy := e.BusyNs(p.name)
			n := p.count(e)
			avg := busy
			if n > 0 {
				avg = busy / msim.Nanos(n)
			}
			out = append(out,
				sName.Render(fmt.Sprintf(" %s:", p.name))+
					sDim.Render(fmt.Sprintf(" %s total :: wakes up every %s and runs for %s",
						ms1(busy), p.period, ms1(avg))))
		}
	}
	return out
}

func onOff(on bool) string {
	if on {
		return sOn.Render("ON ")
	}
	return sOff.Render("OFF")
}

// axisRow anchors an "Nms" label at every other gridline column (every
// 200 ms), skipping any label that would not fit or would collide.
func axisRow(plot int) string {
	cells := make([]rune, plot)
	for i := range cells {
		cells[i] = ' '
	}
	for t := 0; t < windowMS; t += 200 {
		col := t * plot / windowMS
		label := []rune(fmt.Sprintf("%dms", t))
		if col+len(label) > plot {
			continue
		}
		free := true
		lo := col - 1
		if lo < 0 {
			lo = 0
		}
		for _, r := range cells[lo : col+len(label)] {
			if r != ' ' {
				free = false
				break
			}
		}
		if !free {
			continue
		}
		copy(cells[col:], label)
	}
	return sGridS.Render(string(cells))
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
	bcol := boundaryMS * plot / windowMS

	e := m.live.Engine()
	pad := strings.Repeat(" ", g)
	gutterCell := func(st lipgloss.Style, text string) string {
		if g == 0 {
			return ""
		}
		if len(text) > g {
			text = text[:g]
		}
		return st.Render(fmt.Sprintf("%-*s", g, text))
	}

	none := func(msim.Sample) msim.Nanos { return 0 }
	byName := func(name string) func(msim.Sample) msim.Nanos {
		return func(s msim.Sample) msim.Nanos { return s.ByName[name] }
	}

	var lines []string
	for gi, label := range groupLabels {
		lines = append(lines, gutterCell(sLabel, label)+laneRow(sGrid, m.columns(plot, none), bcol))
		for _, p := range procs {
			if p.group(m) != gi || !running(e, p.name) {
				continue
			}
			lines = append(lines,
				gutterCell(groupStyles[gi], " "+p.name)+
					laneRow(groupStyles[gi], m.columns(plot, byName(p.name)), bcol))
		}
	}
	lines = append(lines, pad+axisRow(plot))

	lines = append(lines, "")
	lines = append(lines, m.legend()...)
	lines = append(lines, "")
	lines = append(lines, " "+
		sSwitch.Render("[d] DESCENT ")+onOff(m.descent)+"    "+
		sSwitch.Render("[1] 1668 ")+onOff(m.monitor)+"    "+
		sSwitch.Render("[r] RADAR STEAL ")+onOff(m.radar)+"    "+
		sSwitch.Render("[p] P64 ")+onOff(m.approach)+"      "+
		sDim.Render("q quit"))

	if len(lines) > m.h && m.h > 0 {
		lines = lines[:m.h]
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}
