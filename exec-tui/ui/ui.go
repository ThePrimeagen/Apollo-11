// Package ui renders the AGC Executive simulation as an interactive TUI:
// free compute up top, the long task timelines on the left, the 8 core set
// and 5 VAC area boxes on the right, and the instruction dashes below.
package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
)

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

	seenAlarms int
	flashLeft  int
	lastAlarm  sim.Alarm
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
		if n := len(m.eng.Alarms()); n > m.seenAlarms {
			m.seenAlarms = n
			m.lastAlarm = m.eng.Alarms()[n-1]
			m.flashLeft = 90
		} else if m.flashLeft > 0 {
			m.flashLeft--
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
	case tea.KeySpace:
		m.paused = !m.paused
		return m, nil
	}
	if len(msg.Runes) != 1 {
		return m, nil
	}
	switch msg.Runes[0] {
	case 'q':
		return m, tea.Quit
	case ' ':
		m.paused = !m.paused
	case 'd':
		m.eng.StartDescent()
	case 'l':
		m.eng.AcquireLandingRadar()
	case 'n':
		due := m.eng.AGCTimeMs()
		if len(m.pending) > 0 {
			due = m.pending[len(m.pending)-1].dueAGC // queue behind earlier typing
		}
		for i, k := range []byte("V16N68E") {
			due += neilCadenceAGC[i%len(neilCadenceAGC)]
			m.pending = append(m.pending, pendingKey{k, due})
		}
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
		m.seenAlarms = 0
		m.flashLeft = 0
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
	b.WriteString(m.viewCycleBar())
	b.WriteString("\n")

	left := m.viewLeft()
	right := m.viewBoxes()
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
	b.WriteString(body)
	b.WriteString("\n")
	b.WriteString(m.viewKeyBar())
	return b.String()
}

func fmtAGC(ms float64) string {
	s := ms / 1000.0
	return fmt.Sprintf("T+%06.2fs", s)
}

func (m Model) viewHeader() string {
	e := m.eng
	a := e.Accounting()
	free := a.IdlePct

	freeColor := cGreen
	switch {
	case free < 5:
		freeColor = cRed
	case free < 15:
		freeColor = cYellow
	}
	freeStyle := lipgloss.NewStyle().Foreground(freeColor).Bold(true)

	// free compute mini-bar
	barW := 20
	fill := int(free/100*float64(barW) + 0.5)
	if fill > barW {
		fill = barW
	}
	bar := freeStyle.Render(strings.Repeat("█", fill)) + sDim.Render(strings.Repeat("░", barW-fill))

	speed := fmt.Sprintf("1s wall = %.0fms AGC", e.WallToAGC()*1000)
	title := sTitle.Render("AGC EXECUTIVE · LUMINARY 099")
	clock := sDim.Render(fmtAGC(e.AGCTimeMs())) + "  " + sTitle.Render(e.Phase().String())
	// Armed-load badges: at a glance, which of the three overload
	// ingredients are on (the V16N68 monitor shows as MON 1Hz by the DSKY).
	if e.LandingRadarAcquired() {
		clock += "  " + lipgloss.NewStyle().Foreground(cBlue).Bold(true).Render("LR LOCK")
	}
	if e.RadarBug() {
		clock += "  " + lipgloss.NewStyle().Foreground(cRed).Bold(true).Render("RR BUG")
	}
	clock += "  " + sDim.Render(speed)
	if m.paused {
		clock += "  " + sAlarm.Render(" PAUSED ")
	}

	line1 := title + "   " + clock
	deficitStyle := sDim
	if a.DeficitPct > 0 {
		deficitStyle = lipgloss.NewStyle().Foreground(cRed).Bold(true)
	}
	line2 := freeStyle.Render(fmt.Sprintf("FREE COMPUTE %5.1f%%", free)) + " " + bar +
		sDim.Render(fmt.Sprintf("  duty %4.1f%%  steal %4.1f%%  ",
			a.JobsPct+a.InterruptsPct+a.RestartPct, a.StealPct)) +
		deficitStyle.Render(fmt.Sprintf("deficit %4.1f%%", a.DeficitPct))

	if e.ProgLamp() {
		line2 += "  " + sLamp.Render(" PROG ")
	}
	if fr := e.FailReg(); len(fr) > 0 {
		line2 += " " + lipgloss.NewStyle().Foreground(cRed).Bold(true).Render("FAILREG "+strings.Join(fr, " "))
	}

	if m.flashLeft > 0 && (m.flashLeft/8)%2 == 0 {
		what := "NO CORE SETS AVAILABLE"
		if m.lastAlarm.Code == "1201" {
			what = "NO VAC AREAS AVAILABLE"
		}
		line1 = sAlarm.Render(fmt.Sprintf(" ⚠ PROG ALARM %s — EXECUTIVE OVERFLOW: %s — BAILOUT → RESTART ", m.lastAlarm.Code, what))
	}
	return line1 + "\n" + line2
}

func (m Model) viewCycleBar() string {
	e := m.eng
	elapsed := e.CycleElapsedMs()
	if elapsed > sim.CyclePeriodMs {
		elapsed = sim.CyclePeriodMs
	}
	barW := clampi(m.w-60, 10, 50)
	fill := int(elapsed / sim.CyclePeriodMs * float64(barW))
	if fill > barW {
		fill = barW
	}
	var bar string
	if e.Phase() == sim.P00 {
		bar = sDim.Render(strings.Repeat("·", barW)) + sDim.Render("  (no guidance cycle — idle)")
	} else {
		bar = lipgloss.NewStyle().Foreground(cGreen).Render(strings.Repeat("█", fill)) +
			sDim.Render(strings.Repeat("─", barW-fill)) +
			sDim.Render(fmt.Sprintf(" %4.2fs/2.00s", elapsed/1000))
	}

	d := e.DSKY()
	dsky := sTitle.Render("DSKY") + " " +
		lipgloss.NewStyle().Foreground(cGreen).Bold(true).Render(fmt.Sprintf("V%-2s N%-2s", pad2(d.Verb), pad2(d.Noun)))
	if d.R3 != "" {
		dsky += lipgloss.NewStyle().Foreground(cGreen).Render("  R3 " + d.R3)
	}
	if e.MonitorActive() {
		dsky += " " + lipgloss.NewStyle().Foreground(cYellow).Render("MON 1Hz")
	}
	if m.typing {
		dsky += " " + sAlarm.Render(" TYPING ")
	}
	return sDim.Render("2s CYCLE ") + bar + "   " + dsky
}

func pad2(s string) string {
	for len(s) < 2 {
		s += "–"
	}
	return s
}

func (m Model) viewLeft() string {
	labelW := 9
	rightW := 29
	trackW := clampi(m.w-labelW-rightW-3, 20, 160)
	buckets := m.eng.History(trackW*2 + 2)
	// Anchor bucket pairs to ABSOLUTE parity and render only complete pairs:
	// a cell, once drawn, must never change content — it may only scroll.
	// Pairing "the most recent N" re-shuffled every close and blinked.
	if first := m.eng.BucketsClosed() - len(buckets); first%2 != 0 {
		buckets = buckets[1:]
	}
	if len(buckets)%2 != 0 {
		buckets = buckets[:len(buckets)-1]
	}

	var b strings.Builder
	b.WriteString(sDim.Render(fmt.Sprintf("%-*s", labelW, "")) +
		sDim.Render(fmt.Sprintf("◀ %0.1fs of AGC time, 20ms/cell", float64(trackW)*2*sim.BucketMs/1000)))
	b.WriteString("\n")
	for _, r := range rows {
		style := lipgloss.NewStyle().Foreground(r.color)
		b.WriteString(lipgloss.NewStyle().Foreground(r.color).Render(fmt.Sprintf("%-*s", labelW, r.label)))
		cells := make([]rune, 0, trackW)
		// left-pad when history is younger than the track
		missing := trackW - len(buckets)/2
		for i := 0; i < missing; i++ {
			cells = append(cells, ' ')
		}
		for i := 0; i < len(buckets); i += 2 {
			mask := buckets[i].Mask
			dominant := buckets[i].Dominant == r.c
			if i+1 < len(buckets) {
				mask |= buckets[i+1].Mask
				dominant = dominant || buckets[i+1].Dominant == r.c
			}
			switch {
			case dominant:
				cells = append(cells, '█') // owned most of this slice
			case mask&(1<<uint(r.c)) != 0:
				// Full-cell shade, not a vertically-cut block: '█▂' pairs
				// composited into boot-shaped artifacts on screen.
				cells = append(cells, '░') // ran, but only briefly
			default:
				cells = append(cells, ' ')
			}
		}
		if len(cells) > trackW {
			cells = cells[len(cells)-trackW:]
		}
		b.WriteString(style.Render(string(cells)))
		b.WriteString("\n")
	}

	// stats + event log fill the space under the timelines
	e := m.eng
	b.WriteString("\n")
	stats := sDim.Render(fmt.Sprintf("cycles %d   servicer copies %d   restarts %d   alarms %d",
		e.CycleCount(), e.ServicerCopies(), e.RestartCount(), len(e.Alarms())))
	if stubs := e.StubCount(); stubs > 0 {
		stats += lipgloss.NewStyle().Foreground(cRed).Bold(true).
			Render(fmt.Sprintf("   ⚠ %d stubs leaked (%d words never freed)", stubs, stubs*55))
	} else if e.KnifeEdge() {
		// Flight truth: with the theft active but no monitor, Eagle flew a
		// quiet knife edge for ~5 minutes — margin gone, nothing overrun.
		stats += lipgloss.NewStyle().Foreground(cYellow).
			Render("   ⚠ knife edge: margin ≈ 0, nothing overrun yet — one straw breaks it: [n] monitor · [6] P64")
	} else if e.RecoveredRecently(5000) {
		stats += lipgloss.NewStyle().Foreground(cYellow).
			Render("   ⟳ overrun recovering — a superseded SERVICER finished late and freed its pair")
	}
	b.WriteString(stats)
	b.WriteString("\n\n")
	evs := e.Events()
	n := clampi(m.h-22, 3, 20)
	if len(evs) < n {
		n = len(evs)
	}
	for _, ev := range evs[len(evs)-n:] {
		style := sDim
		switch ev.Kind {
		case sim.EvAlarm, sim.EvRestart:
			style = lipgloss.NewStyle().Foreground(cRed)
		case sim.EvLeak:
			style = lipgloss.NewStyle().Foreground(cRed).Bold(true)
		case sim.EvRecover:
			style = lipgloss.NewStyle().Foreground(cGreen)
		case sim.EvHint:
			style = lipgloss.NewStyle().Foreground(cCyan)
		case sim.EvBug:
			style = lipgloss.NewStyle().Foreground(cOrange)
		case sim.EvMonitorOn, sim.EvMonitorOff:
			style = lipgloss.NewStyle().Foreground(cYellow)
		}
		b.WriteString(style.Render(fmt.Sprintf("%s  %s", fmtAGC(ev.AGCTimeMs), ev.Text)))
		b.WriteString("\n")
	}
	return lipgloss.NewStyle().Width(labelW + trackW).Render(b.String())
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

func (m Model) viewBoxes() string {
	e := m.eng
	cores := e.CoreSets()
	vacs := e.VACs()

	compact := m.h < 31
	boxW := 12

	var coreCol, vacCol []string
	coreCol = append(coreCol, sTitle.Render("CORE SETS"))
	vacCol = append(vacCol, sTitle.Render("VAC AREAS"))

	busyC, busyV := 0, 0
	for i, s := range cores {
		label := fmt.Sprintf("CS%d", i+1)
		if s.Busy {
			busyC++
		}
		if compact {
			coreCol = append(coreCol, m.slotLine(label, s))
		} else {
			coreCol = append(coreCol, m.boxFor(label, s, boxW))
		}
	}
	for i, s := range vacs {
		label := fmt.Sprintf("VC%d", i+1)
		if s.Busy {
			busyV++
		}
		if compact {
			vacCol = append(vacCol, m.slotLine(label, s))
		} else {
			vacCol = append(vacCol, m.boxFor(label, s, boxW))
		}
	}
	poolStyle := sDim
	if busyC >= 7 || busyV >= 4 {
		poolStyle = lipgloss.NewStyle().Foreground(cRed).Bold(true)
	}
	vacCol = append(vacCol, "")
	vacCol = append(vacCol, poolStyle.Render(fmt.Sprintf("CORE %d/8", busyC)))
	vacCol = append(vacCol, poolStyle.Render(fmt.Sprintf("VAC  %d/5", busyV)))
	if busyC == 8 {
		vacCol = append(vacCol, sAlarm.Render("→ 1202"))
	} else if busyV == 5 {
		vacCol = append(vacCol, sAlarm.Render("→ 1201"))
	}

	left := lipgloss.JoinVertical(lipgloss.Left, coreCol...)
	right := lipgloss.JoinVertical(lipgloss.Left, vacCol...)
	return lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
}

func (m Model) slotLine(label string, s sim.SlotState) string {
	if !s.Busy {
		return sDim.Render(fmt.Sprintf("%-4s ·free", label))
	}
	if s.Stub {
		return lipgloss.NewStyle().Foreground(cRed).Bold(true).Render(fmt.Sprintf("%-4s █STUB·%d", label, s.Prio))
	}
	return lipgloss.NewStyle().Foreground(cGreen).Render(fmt.Sprintf("%-4s █%s·%d", label, shortOwner(s.Owner), s.Prio))
}

func (m Model) viewKeyBar() string {
	if m.typing {
		return sAlarm.Render(" DSKY TYPING ") +
			sDim.Render(" your keys cost real compute ─ 0-9 v n e(ENTR) c(CLR) ─ try v16n68e ─ ") +
			sTitle.Render("esc") + sDim.Render(" to leave")
	}
	e := m.eng
	hint := func(k, what string) string {
		return sDim.Render("─ ") + sTitle.Render("["+k+"]") + " " + sDim.Render(what) + " "
	}
	// A latched control renders bright with a ✓ while its state is on, so
	// the bar itself answers "what is running right now?".
	on := func(k, what string) string {
		s := lipgloss.NewStyle().Foreground(cGreen).Bold(true)
		return sDim.Render("─ ") + s.Render("["+k+"] "+what+" ✓") + " "
	}
	latched := func(active bool, k, what string) string {
		if active {
			return on(k, what)
		}
		return hint(k, what)
	}
	phase := e.Phase()
	line1 := latched(phase != sim.P00, "d", "descent") +
		latched(e.LandingRadarAcquired(), "l", "radar lock") +
		latched(e.MonitorActive(), "n", "neil types") +
		hint("t", "you type") +
		latched(e.RadarBug(), "r", "RR bug") +
		hint("p", "ping radar") +
		latched(phase == sim.P64, "6", "P64") +
		latched(phase == sim.P66, "a", "att-hold")
	line2 := hint("space", "pause") + hint("-", "slow") + hint("+", "fast") +
		hint("x", "reset") + hint("q", "quit")
	if m.w >= 175 {
		return line1 + line2
	}
	return line1 + "\n" + line2
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
