// Package ui renders the AGC Executive simulation as an interactive TUI:
// free compute up top, the long task timelines on the left, the 8 core set
// and 5 VAC area boxes on the right, and the instruction dashes below.
package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/theprimeagen/apollo-11/button-lab/button"
	"github.com/theprimeagen/apollo-11/dsky-lab/dsky"
	"github.com/theprimeagen/apollo-11/exec-tui/sim"
)

// dskyState maps the engine onto the DSKY panel: verb/noun as keyed, PROG
// from the phase, registers from the flight values, PROG/RESTART lamps from
// the alarms, ALT/VEL from the landing-radar lock.
func (m Model) dskyState() dsky.State {
	e := m.eng
	d := e.DSKY()
	st := dsky.State{Verb: d.Verb, Noun: d.Noun, CompActy: e.CompActy()}
	ph := e.Phase()
	switch ph {
	case sim.P63:
		st.Prog = "63"
	case sim.P64:
		st.Prog = "64"
	case sim.P66:
		st.Prog = "66"
	}
	if ph != sim.P00 {
		st.R1, st.R2, st.R3 = d.R1, d.R2, d.R3
	}
	st.Lights = dsky.Lights{
		Prog:    e.ProgLamp(),
		Restart: e.RestartRecently(1500),
		Alt:     ph != sim.P00 && !e.LandingRadarAcquired(),
		Vel:     ph != sim.P00 && !e.LandingRadarAcquired(),
	}
	// Right after a restart the panel shows the failure the way the crew
	// read it: V05 N09 with the FAILREG codes, unsigned, in the registers.
	if fr := e.FailReg(); len(fr) > 0 && e.RestartRecently(2500) {
		st.Verb, st.Noun = "05", "09"
		regs := []*string{&st.R1, &st.R2, &st.R3}
		for i := range regs {
			*regs[i] = ""
			if i < len(fr) {
				*regs[i] = " 0" + fr[i]
			}
		}
	}
	return st
}

// ForceColorIfRequested forces a 256-color profile when CLICOLOR_FORCE is
// set — profile detection fails in detached ptys (tmux capture, CI), which
// would otherwise strip every color from recordings.
func ForceColorIfRequested() {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		lipgloss.SetColorProfile(termenv.ANSI256)
	}
}

// FrameMsg advances the simulation by one wall-clock frame (~33ms).
type FrameMsg struct{}

const frameWallMs = 33.34

// fake typing cadence in AGC milliseconds: a human types ~230-330ms apart in
// REAL time (which is AGC time), so wall spacing scales with playback speed.
var neilCadenceAGC = []float64{270, 230, 300, 270, 330, 230, 300}

type pendingKey struct {
	key    byte
	dueAGC float64 // absolute AGC time at which the key lands
}

// Model is the bubbletea model.
type Model struct {
	eng     *sim.Engine
	w, h    int
	paused  bool
	typing  bool
	pending []pendingKey
	sel     int // selected switch: 0 DESCENT, 1 DELTAH, 2 RR STEAL
	zoom    int // timeline zoom level index (see zoomBPC)
}

// NewModel wraps an engine.
func NewModel(e *sim.Engine) Model {
	return Model{eng: e, w: 120, h: 40}
}

// Init starts the frame ticker.
func (m Model) Init() tea.Cmd { return frameTick() }

func frameTick() tea.Cmd {
	return tea.Tick(time.Duration(frameWallMs*1e6)*time.Nanosecond, func(time.Time) tea.Msg {
		return FrameMsg{}
	})
}

// Paused reports pause state.
func (m Model) Paused() bool { return m.paused }

// TypingMode reports whether user keys are DSKY keys.
func (m Model) TypingMode() bool { return m.typing }

// PendingKeys is how many fake keystrokes are still queued.
func (m Model) PendingKeys() int { return len(m.pending) }

// Selected is the index of the selected toggle card.
func (m Model) Selected() int { return m.sel }

// queueKeys enqueues DSKY keystrokes at a human cadence in AGC time.
func (m *Model) queueKeys(keys string) {
	due := m.eng.AGCTimeMs()
	if len(m.pending) > 0 {
		due = m.pending[len(m.pending)-1].dueAGC // queue behind earlier typing
	}
	for i, k := range []byte(keys) {
		due += neilCadenceAGC[i%len(neilCadenceAGC)]
		m.pending = append(m.pending, pendingKey{k, due})
	}
}

// engage flips the selected switch the way the crew would have.
func (m *Model) engage() {
	switch m.sel {
	case 0: // DESCENT — select P63 on the DSKY (V37E 63E)
		if m.eng.Phase() == sim.P00 && len(m.pending) == 0 {
			m.queueKeys("V37E63E")
		}
	case 1: // DELTAH — Buzz's V16 N68 monitor; V34 flicks it back off
		if len(m.pending) != 0 {
			return
		}
		if !m.eng.MonitorActive() {
			m.queueKeys("V16N68E")
		} else {
			m.queueKeys("V34E")
		}
	case 2: // RR STEAL — the mode switch, instant
		m.eng.SetRadarBug(!m.eng.RadarBug())
	}
}

// TimeScale is AGC ms per wall ms.
func (m Model) TimeScale() float64 { return m.eng.WallToAGC() }

// Update handles input and frames.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case FrameMsg:
		if !m.paused {
			m.eng.AdvanceWall(frameWallMs)
			for len(m.pending) > 0 && m.eng.AGCTimeMs() >= m.pending[0].dueAGC {
				m.eng.PressKey(m.pending[0].key)
				m.pending = m.pending[1:]
			}
		}
		return m, frameTick()
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.typing {
		switch msg.Type {
		case tea.KeyEsc:
			m.typing = false
			return m, nil
		case tea.KeyEnter:
			m.eng.PressKey('E')
			return m, nil
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
		if len(msg.Runes) == 1 {
			r := msg.Runes[0]
			switch {
			case r >= '0' && r <= '9':
				m.eng.PressKey(byte(r))
			case r == 'v' || r == 'V':
				m.eng.PressKey('V')
			case r == 'n' || r == 'N':
				m.eng.PressKey('N')
			case r == 'e' || r == 'E':
				m.eng.PressKey('E')
			case r == 'c' || r == 'C':
				m.eng.PressKey('C')
			}
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeySpace, tea.KeyEnter:
		m.engage()
		return m, nil
	case tea.KeyLeft:
		m.sel = (m.sel + 2) % 3
		return m, nil
	case tea.KeyRight:
		m.sel = (m.sel + 1) % 3
		return m, nil
	}
	if len(msg.Runes) != 1 {
		return m, nil
	}
	switch msg.Runes[0] {
	case 'q':
		return m, tea.Quit
	case '.':
		m.paused = !m.paused
	case ' ':
		m.engage()
	case 'h':
		m.sel = (m.sel + 2) % 3
	case 'l':
		m.sel = (m.sel + 1) % 3
	case 'z':
		m.zoom = (m.zoom + 1) % len(zoomBPC)
	case 'd':
		m.eng.StartDescent()
	case 'n':
		m.sel = 1
		m.engage()
	case 't':
		m.typing = true
	case 'r':
		m.eng.SetRadarBug(!m.eng.RadarBug())
	case 'p':
		m.eng.PingRadar()
	case '6':
		m.eng.EnterP64()
	case 'a':
		m.eng.AttHold()
	case '[', '-':
		m.eng.SetWallToAGC(maxf(0.0125, m.eng.WallToAGC()/2))
	case ']', '+', '=':
		m.eng.SetWallToAGC(minf(2.0, m.eng.WallToAGC()*2))
	case 'x':
		m.eng.Reset()
		m.pending = nil
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

var (
	cAmber  = lipgloss.Color("214")
	cGreen  = lipgloss.Color("83")
	cDimGrn = lipgloss.Color("29")
	cRed    = lipgloss.Color("196")
	cYellow = lipgloss.Color("220")
	cCyan   = lipgloss.Color("87")
	cBlue   = lipgloss.Color("75")
	cMag    = lipgloss.Color("213")
	cOrange = lipgloss.Color("208")
	cGray   = lipgloss.Color("245")
	cDim    = lipgloss.Color("240")
	cWhite  = lipgloss.Color("255")

	sTitle = lipgloss.NewStyle().Foreground(cAmber).Bold(true)
	sDim   = lipgloss.NewStyle().Foreground(cDim)
	sAlarm = lipgloss.NewStyle().Foreground(cWhite).Background(cRed).Bold(true)
	sLamp  = lipgloss.NewStyle().Foreground(lipgloss.Color("16")).Background(cYellow).Bold(true)
)

type rowSpec struct {
	label string
	c     sim.Consumer
	color lipgloss.Color
}

var rows = []rowSpec{
	{"SERVICER", sim.CServicer, cGreen},
	{"MONITOR", sim.CMonitor, cYellow},
	{"CHARIN", sim.CCharin, cMag},
	{"RR READ", sim.CRRRead, cCyan},
	{"LR READ", sim.CLRRead, cBlue},
	{"GYRO", sim.CGyro, cWhite},
	{"DAP", sim.CDAP, cOrange},
	{"T4RUPT", sim.CT4Rupt, cGray},
	{"DOWNLINK", sim.CDownrupt, cGray},
	{"RR STEAL", sim.CSteal, cRed},
	{"PIPA CTR", sim.CPipa, cGray},
	{"RESTART", sim.CRestart, cRed},
	{"IDLE", sim.CIdle, cDimGrn},
}

func shortOwner(name string) string {
	switch name {
	case "SERVICER":
		return "SERV"
	case "CHARIN":
		return "CHAR"
	case "MONITOR":
		return "MON"
	case "RR READ":
		return "RR"
	case "LR READ":
		return "LR"
	case "LRHJOB":
		return "LRH"
	case "LRVJOB":
		return "LRV"
	case "HIGATJOB":
		return "HGAT"
	case "GYRO COMP":
		return "GYRO"
	default:
		if len(name) > 4 {
			return name[:4]
		}
		return name
	}
}

// View renders the whole screen.
func (m Model) View() string {
	if m.w < 20 || m.h < 5 {
		return "AGC EXECUTIVE — terminal too small"
	}
	var b strings.Builder
	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	left := lipgloss.JoinVertical(lipgloss.Left, m.viewLeft(), "", m.viewPools())
	panel := dsky.Render(m.dskyState(), true)
	switches := m.viewSwitches()
	rightW := lipgloss.Width(switches)
	if dsky.Width > rightW {
		rightW = dsky.Width
	}
	right := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.PlaceHorizontal(rightW, lipgloss.Right, panel),
		"",
		lipgloss.PlaceHorizontal(rightW, lipgloss.Right, switches),
	)
	gap := m.w - lipgloss.Width(left) - rightW
	if gap < 1 {
		gap = 1
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", gap), right)
	b.WriteString(body)
	return b.String()
}

// viewSwitches renders the three switches as a tight bank that fits
// completely under the 25-cell DSKY. State shows in the label color: light
// gray when off, amber when on; focus shows on the switch frame itself.
// h/l selects, space (or enter) flicks.
func (m Model) viewSwitches() string {
	e := m.eng
	specs := []struct {
		label string
		on    bool
	}{
		{"DESCENT", e.Phase() != sim.P00},
		{"DELTAH", e.MonitorActive()},
		{"RR STEAL", e.RadarBug()},
	}
	sGrayLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	sOnLabel := lipgloss.NewStyle().Foreground(cAmber).Bold(true)
	cols := make([]string, 3)
	for i, sp := range specs {
		sw := button.NewSwitch(sp.label)
		sw.On = sp.on
		sw.Focused = i == m.sel
		style := sGrayLabel
		if sp.on {
			style = sOnLabel
		}
		colW := len(sp.label)
		center := func(s string) string { return lipgloss.PlaceHorizontal(colW, lipgloss.Center, s) }
		cols[i] = lipgloss.JoinVertical(lipgloss.Left,
			center(sw.Render()),
			center(style.Render(sp.label)),
		)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cols[0], " ", cols[1], " ", cols[2])
}

// viewHeader is ONE line: the effective free compute — idle minus deficit —
// which goes NEGATIVE under overload. Breakage shows on the DSKY (PROG lamp
// + V05 N09 alarm codes), not as header text.
func (m Model) viewHeader() string {
	a := m.eng.Accounting()
	free := a.IdlePct - a.DeficitPct

	freeColor := cGreen
	switch {
	case free < 0:
		freeColor = cRed
	case free < 15:
		freeColor = cYellow
	}
	freeStyle := lipgloss.NewStyle().Foreground(freeColor).Bold(true)

	barW := 30
	fill := int(free / 100 * float64(barW))
	if fill > barW {
		fill = barW
	}
	if fill < 0 {
		fill = 0
	}
	bar := freeStyle.Render(strings.Repeat("█", fill)) + sDim.Render(strings.Repeat("░", barW-fill))
	line := freeStyle.Render(fmt.Sprintf("FREE COMPUTE %+6.1f%%", free)) + " " + bar
	if m.typing {
		line += " " + sAlarm.Render(" TYPING ")
	}
	if m.paused {
		line += " " + sAlarm.Render(" PAUSED ")
	}
	return line
}

// zoomBPC lists the buckets-per-cell zoom levels the z key cycles through:
// 50ms bars (default), 80ms bars, 40ms bars.
var zoomBPC = []int{5, 8, 4}

// bpc is the current buckets-per-cell.
func (m Model) bpc() int { return zoomBPC[m.zoom%len(zoomBPC)] }

// cellMs is the AGC time one bar covers at the current zoom.
func (m Model) cellMs() float64 { return float64(m.bpc()) * sim.BucketMs }

// cellsFor holds the visible window constant (the old half-width 40ms track)
// while each bar covers more time: fewer, denser bars.
func cellsFor(w, bpc int) int {
	base := clampi(w/2-9, 20, 160) // cells at the reference 40ms zoom
	return clampi(base*4/bpc, 20, 160)
}

// gridBGColor is the ruler's background tint (xterm-256 index).
const gridBGColor = "240"

func (m Model) viewLeft() string {
	bucketsPerCell := m.bpc()
	trackW := cellsFor(m.w, bucketsPerCell)
	buckets := m.eng.History(trackW*bucketsPerCell + bucketsPerCell)
	// Anchor cells to ABSOLUTE groups and render only complete groups: a
	// cell, once drawn, must never change content — it may only scroll.
	// Grouping "the most recent N" re-shuffled every close and blinked.
	if first := m.eng.BucketsClosed() - len(buckets); first%bucketsPerCell != 0 {
		buckets = buckets[bucketsPerCell-first%bucketsPerCell:]
	}
	if rem := len(buckets) % bucketsPerCell; rem != 0 {
		buckets = buckets[:len(buckets)-rem]
	}

	// absolute cell index of the first rendered group (for 2s gridlines)
	absCell0 := (m.eng.BucketsClosed() - len(buckets)) / bucketsPerCell

	// the 2s ruler: a lighter BACKGROUND behind the bars, so full blocks
	// cover it, shades let it glow through, and blanks show it plainly
	gridEvery := int(sim.CyclePeriodMs / (float64(bucketsPerCell) * sim.BucketMs))

	e := m.eng
	var b strings.Builder
	for _, r := range rows {
		style := lipgloss.NewStyle().Foreground(r.color)
		gridStyle := style.Background(lipgloss.Color(gridBGColor))
		used := e.UsedMs(r.c)
		if used > 9999 {
			used = 9999
		}
		b.WriteString(style.Render(fmt.Sprintf("%-9s%4.0fms ", r.label, used)))
		type cell struct {
			ch   rune
			grid bool
		}
		cells := make([]cell, 0, trackW)
		// left-pad when history is younger than the track
		missing := trackW - len(buckets)/bucketsPerCell
		for i := 0; i < missing; i++ {
			cells = append(cells, cell{' ', false})
		}
		for i, ci := 0, 0; i+bucketsPerCell <= len(buckets); i, ci = i+bucketsPerCell, ci+1 {
			mask := uint32(0)
			dominant := false
			for j := i; j < i+bucketsPerCell; j++ {
				mask |= buckets[j].Mask
				dominant = dominant || buckets[j].Dominant == r.c
			}
			onGrid := (absCell0+ci)%gridEvery == 0
			switch {
			case dominant:
				cells = append(cells, cell{'█', onGrid}) // block covers the ruler
			case mask&(1<<uint(r.c)) != 0:
				cells = append(cells, cell{'░', onGrid}) // ruler glows through the shade
			default:
				cells = append(cells, cell{' ', onGrid}) // bare ruler
			}
		}
		if len(cells) > trackW {
			cells = cells[len(cells)-trackW:]
		}
		// emit runs so ruler cells carry the background tint
		run, runGrid := []rune{}, false
		flush := func() {
			if len(run) == 0 {
				return
			}
			if runGrid {
				b.WriteString(gridStyle.Render(string(run)))
			} else {
				b.WriteString(style.Render(string(run)))
			}
			run = run[:0]
		}
		for _, c := range cells {
			if c.grid != runGrid {
				flush()
				runGrid = c.grid
			}
			run = append(run, c.ch)
		}
		flush()
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func (m Model) boxFor(label string, s sim.SlotState, w int) string {
	border := lipgloss.RoundedBorder()
	if !s.Busy {
		content := sDim.Render(fmt.Sprintf("%-4s", label)) + sDim.Render("free")
		return lipgloss.NewStyle().Border(border).BorderForeground(cDim).Width(w).Render(content)
	}
	if s.Stub {
		// Abandoned copy: superseded, starving, never reaching ENDOFJOB —
		// this slot is leaked memory.
		content := lipgloss.NewStyle().Foreground(cWhite).Bold(true).Render(fmt.Sprintf("%-4s", label)) +
			lipgloss.NewStyle().Foreground(cRed).Bold(true).Render(fmt.Sprintf("STUB·%d", s.Prio))
		return lipgloss.NewStyle().Border(border).BorderForeground(cRed).Width(w).Render(content)
	}
	color := cGreen
	switch s.Owner {
	case "CHARIN":
		color = cMag
	case "MONITOR":
		color = cYellow
	case "RR READ":
		color = cCyan
	case "LR READ", "LRHJOB", "LRVJOB", "HIGATJOB":
		color = cBlue
	case "GYRO COMP":
		color = cWhite
	}
	content := lipgloss.NewStyle().Foreground(cWhite).Bold(true).Render(fmt.Sprintf("%-4s", label)) +
		lipgloss.NewStyle().Foreground(color).Bold(true).Render(fmt.Sprintf("%s·%d", shortOwner(s.Owner), s.Prio))
	return lipgloss.NewStyle().Border(border).BorderForeground(color).Width(w).Render(content)
}

// viewPools renders the Executive's memory where the log used to live: the
// eight core sets as two stacks of four (CS1–CS4 beside CS5–CS8), and the
// five VACs as one stack alongside.
func (m Model) viewPools() string {
	e := m.eng
	cores := e.CoreSets()
	vacs := e.VACs()
	boxW := 12

	busyC, busyV := 0, 0
	var coreL, coreR, vacCol []string
	for i, s := range cores {
		if s.Busy {
			busyC++
		}
		box := m.boxFor(fmt.Sprintf("CS%d", i+1), s, boxW)
		if i < 4 {
			coreL = append(coreL, box)
		} else {
			coreR = append(coreR, box)
		}
	}
	for i, s := range vacs {
		if s.Busy {
			busyV++
		}
		vacCol = append(vacCol, m.boxFor(fmt.Sprintf("VC%d", i+1), s, boxW))
	}

	poolStyle := sDim
	if busyC >= 7 || busyV >= 4 {
		poolStyle = lipgloss.NewStyle().Foreground(cRed).Bold(true)
	}
	coreTitle := sTitle.Render("CORE SETS") + " " + poolStyle.Render(fmt.Sprintf("%d/8", busyC))
	if busyC == 8 {
		coreTitle += " " + sAlarm.Render("→ 1202")
	}
	vacTitle := sTitle.Render("VAC AREAS") + " " + poolStyle.Render(fmt.Sprintf("%d/5", busyV))
	if busyV == 5 {
		vacTitle += " " + sAlarm.Render("→ 1201")
	}

	coreGrid := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left, coreL...),
		lipgloss.JoinVertical(lipgloss.Left, coreR...),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left, coreTitle, coreGrid),
		"  ",
		lipgloss.JoinVertical(lipgloss.Left, vacTitle, lipgloss.JoinVertical(lipgloss.Left, vacCol...)),
	)
}

func clampi(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
