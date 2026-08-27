// Package agctop is the command screen: every process on the millisecond
// Executive (msim), live, in the three groups that matter —
//
//	VAC JOBS                 jobs holding a core set AND a VAC area
//	CORESET JOBS             jobs holding a core set only
//	NO-PRIORITY OPERATIONS   tasks & interrupts: cpu only, no memory
//
// — with three switches on the bottom: [d] DESCENT (the whole P63 job
// chain), [1] 1668 (Buzz's V16N68 DELTAH monitor), [r] RADAR STEAL (the
// RR CDU counter theft). The machine runs in real time, one wall
// millisecond per simulated millisecond.
package agctop

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	msim "github.com/theprimeagen/apollo-11/msim"
)

// frameMS is the wall (and sim) milliseconds per frame: 20 fps, 1:1 time.
const frameMS = 50

type frameMsg struct{}

func frameTick() tea.Cmd {
	return tea.Tick(frameMS*time.Millisecond, func(time.Time) tea.Msg { return frameMsg{} })
}

// Model is the command screen over one live machine.
type Model struct {
	live *msim.Live
	w, h int
}

// New builds the screen over the given live machine.
func New(l *msim.Live) Model { return Model{live: l, w: 100, h: 40} }

func (m Model) Init() tea.Cmd { return frameTick() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case frameMsg:
		m.live.StepMS(frameMS)
		return m, frameTick()
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch msg.Code {
		case 'q':
			return m, tea.Quit
		case 'd':
			m.live.SetDescent(!m.live.DescentOn())
		case '1':
			m.live.SetMonitor(!m.live.MonitorOn())
		case 'r':
			m.live.SetRadar(!m.live.RadarOn())
		}
	}
	return m, nil
}

var (
	sTitle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	sHead   = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true)
	sDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	sName   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	sRun    = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	sPark   = lipgloss.NewStyle().Foreground(lipgloss.Color("178"))
	sSleep  = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
	sAlarm  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	sOn     = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	sOff    = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	sSwitch = lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Bold(true)
)

// t0GETSeconds is GET 102:37:55 — the window's origin (PDI+290 s).
const t0GETSeconds = 102*3600 + 37*60 + 55

func getStamp(ns msim.Nanos) string {
	s := t0GETSeconds + int(ns/msim.Second)
	return fmt.Sprintf("%d:%02d:%02d", s/3600, (s%3600)/60, s%60)
}

func bar(n, max int) string {
	if n < 0 {
		n = 0
	}
	if n > max {
		n = max
	}
	return strings.Repeat("#", n) + strings.Repeat(".", max-n)
}

// coreCatalog is the coreset-only (NOVAC) job set of the locked P63
// configuration, with the reason each one holds memory.
var coreCatalog = []struct{ name, note string }{
	{"MONDO", "the V16N68 refresh, 1 Hz while keyed"},
	{"MAKEPLAY", "the display job; blocked sleepers hold their core"},
	{"LRHJOB", "radar altitude gate — sleeps ~95 ms across the boundary"},
	{"LRVJOB", "radar velocity samples — sleeps ~500 ms"},
	{"1/GYRO", "gyro compensation, ~1/s"},
	{"CHARIN", "one job per DSKY keystroke"},
}

// opsCatalog is the no-priority layer: cadences and waitlist tasks.
var opsCatalog = []struct {
	name   string
	period string
	task   bool // waitlist task (TaskFires) vs hardware cadence (InterruptFires)
}{
	{"DAP", "100 ms", false},
	{"T4RUPT", "120 ms", false},
	{"DOWNRUPT", "20 ms", false},
	{"READACCS", "2 s", true},
	{"R10,R11", "250 ms", true},
	{"LRHTASK", "per cycle", true},
	{"LRVTASK", "per cycle", true},
	{"MONREQ", "1 s", true},
}

func onOff(on bool) string {
	if on {
		return sOn.Render("ON ")
	}
	return sOff.Render("OFF")
}

func (m Model) View() tea.View {
	e := m.live.Engine()
	now := e.Now()
	var b strings.Builder

	// --- header
	b.WriteString(sTitle.Render("AGC EXECUTIVE — COMMAND SCREEN"))
	b.WriteString(sDim.Render(fmt.Sprintf("   GET %s   t=%.1fs", getStamp(now), float64(now)/1e9)))
	b.WriteString("\n")
	theft := "off"
	if m.live.RadarOn() {
		theft = fmt.Sprintf("%.1f%%", 100*float64(e.TheftNs())/float64(now+1))
	}
	b.WriteString(fmt.Sprintf(" cores |%s| %d/8   vacs |%s| %d/5   theft %s   restarts %d\n",
		bar(e.CoresHeld(), 8), e.CoresHeld(), bar(e.VACsHeld(), 5), e.VACsHeld(),
		theft, e.RestartCount()))
	if alarms := e.Alarms(); len(alarms) > 0 {
		a := alarms[len(alarms)-1]
		code := "1202 NO CORE SETS"
		if a.Code == 1201 {
			code = "1201 NO VAC AREAS"
		}
		line := fmt.Sprintf(" ALARM %s — %q denied at t=%.1fs (cores %d/8, vacs %d/5)",
			code, a.Requester, float64(a.At)/1e9, a.CoresHeld, a.VACsHeld)
		if now-a.At < 4*msim.Second {
			b.WriteString(sAlarm.Render(line) + "\n")
		} else {
			b.WriteString(sDim.Render(line) + "\n")
		}
	} else {
		b.WriteString(sDim.Render(" no alarms") + "\n")
	}
	b.WriteString("\n")

	views := e.SlotViews()

	// --- VAC JOBS: live slots holding a VAC area
	b.WriteString(sHead.Render(" VAC JOBS") + sDim.Render(" — core set + VAC area") + "\n")
	vacRows := 0
	for _, v := range views {
		if v.Name == "" || v.VAC < 0 {
			continue
		}
		vacRows++
		b.WriteString(fmt.Sprintf("  slot%d  %s  prio %d  %s  %s %d/%d  vac %d\n",
			v.Slot, sName.Render(fmt.Sprintf("%-9s", v.Name)), v.Prio,
			stateStr(v.State), sDim.Render("ip"), v.IP, v.Len, v.VAC))
	}
	if vacRows == 0 {
		b.WriteString(sDim.Render("  —") + "\n")
	}
	b.WriteString("\n")

	// --- CORESET JOBS: the NOVAC catalog with live state
	b.WriteString(sHead.Render(" CORESET JOBS") + sDim.Render(" — core set only") + "\n")
	for _, c := range coreCatalog {
		state := m.novacState(views, c.name, now)
		b.WriteString(fmt.Sprintf("  %s %s   %s\n",
			sName.Render(fmt.Sprintf("%-9s", c.name)), state,
			sDim.Render(c.note)))
	}
	b.WriteString("\n")

	// --- NO-PRIORITY OPERATIONS
	b.WriteString(sHead.Render(" NO-PRIORITY OPERATIONS") +
		sDim.Render(" — tasks & interrupts, cpu only, no memory") + "\n")
	for _, o := range opsCatalog {
		fires := e.TaskFires(o.name)
		if !o.task {
			fires = e.InterruptFires(o.name)
		}
		last := "—"
		if at := e.LastFired(o.name); at >= 0 {
			last = fmt.Sprintf("%.1fs ago", float64(now-at)/1e9)
		}
		b.WriteString(fmt.Sprintf("  %s every %-9s fired %6dx   %s\n",
			sName.Render(fmt.Sprintf("%-9s", o.name)), o.period, fires, sDim.Render(last)))
	}
	b.WriteString("\n")

	// --- the switches
	b.WriteString(" " +
		sSwitch.Render("[d] DESCENT ")+onOff(m.live.DescentOn())+"    "+
		sSwitch.Render("[1] 1668 ")+onOff(m.live.MonitorOn())+"    "+
		sSwitch.Render("[r] RADAR STEAL ")+onOff(m.live.RadarOn())+"      "+
		sDim.Render("q quit"))

	v := tea.NewView(clamp(b.String(), m.w, m.h))
	v.AltScreen = true
	return v
}

// novacState renders a coreset job's live state: its slot state when it
// holds a core set, a fading "ran ...s ago" when recently active, or a dash.
func (m Model) novacState(views []msim.SlotView, name string, now msim.Nanos) string {
	for _, v := range views {
		if v.Name != name || v.VAC >= 0 {
			continue
		}
		return fmt.Sprintf("prio %2d  %s", v.Prio, stateStr(v.State))
	}
	if at := m.live.Engine().LastRan(name); at >= 0 && now-at < 5*msim.Second {
		return sDim.Render(fmt.Sprintf("ran %.1fs ago    ", float64(now-at)/1e9))
	}
	return sDim.Render("—               ")
}

func stateStr(s msim.JobState) string {
	switch s {
	case msim.JobRunning:
		return sRun.Render("RUNNING ")
	case msim.JobParked:
		return sPark.Render("parked  ")
	case msim.JobSleeping:
		return sSleep.Render("sleeping")
	case msim.JobWaiting:
		return sDim.Render("waiting ")
	}
	return sDim.Render("—       ")
}

// clamp cuts the frame to the window so tiny terminals stay safe.
func clamp(s string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i, l := range lines {
		if lipgloss.Width(l) > w {
			lines[i] = lipgloss.NewStyle().MaxWidth(w).Render(l)
		}
	}
	return strings.Join(lines, "\n")
}
