package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// Rose Pine
var (
	rpBase    = lipgloss.Color("#191724")
	rpSurface = lipgloss.Color("#1f1d2e")
	rpOverlay = lipgloss.Color("#26233a")
	rpMuted   = lipgloss.Color("#6e6a86")
	rpSubtle  = lipgloss.Color("#908caa")
	rpText    = lipgloss.Color("#e0def4")
	rpLove    = lipgloss.Color("#eb6f92")
	rpGold    = lipgloss.Color("#f6c177")
	rpRose    = lipgloss.Color("#ebbcba")
	rpPine    = lipgloss.Color("#31748f")
	rpFoam    = lipgloss.Color("#9ccfd8")
	rpIris    = lipgloss.Color("#c4a7e7")
)

const (
	nCores = 8
	nVACs  = 5

	periodS          = 2.00
	servicerNeedS    = 1.80
	fadeDuration     = 500 * time.Millisecond
	tickInterval     = 16 * time.Millisecond
	playRealDuration = 8 * time.Second
)

type mode int

const (
	modeStep mode = iota
	modePlay
)

type tickMsg time.Time

type board struct {
	cores [nCores]int // owner id, or -1
	vacs  [nVACs]int
	next  int
	alarm int
}

func newBoard() board {
	b := board{next: 0, alarm: 0}
	for i := range b.cores {
		b.cores[i] = -1
	}
	for i := range b.vacs {
		b.vacs[i] = -1
	}
	return b
}

func (b *board) claimPair() (ok bool, alarm int) {
	vac := -1
	for i := range b.vacs {
		if b.vacs[i] < 0 {
			vac = i
			break
		}
	}
	if vac < 0 {
		b.alarm = 1201
		return false, 1201
	}
	core := -1
	for i := range b.cores {
		if b.cores[i] < 0 {
			core = i
			break
		}
	}
	if core < 0 {
		b.alarm = 1202
		return false, 1202
	}
	id := b.next
	b.next++
	b.vacs[vac] = id
	b.cores[core] = id
	return true, 0
}

func (b *board) clear() {
	*b = newBoard()
}

// Event = one MEMORY_LEAK / timeline beat.
type event struct {
	marker   string
	title    string
	blurb    string
	code     string
	apply    func(*board) // mutates board when stepped onto
	playable bool         // show cycle bars emphasis
}

func events() []event {
	return []event{
		{
			marker: "MAP",
			title:  "The pool that ran dry",
			blurb:  "8 core sets + 5 VAC areas. Radar steals TIME only — zero words.",
			code: `Vac   vac_1 .. vac_5;     // 5 workspaces
Core  core_1 .. core_8;   // 8 job slots

// radar touches CDUT/CDUS — time only, no alloc
`,
			apply: func(b *board) { b.clear() },
		},
		{
			marker: "BASELINE",
			title:  "Healthy cycle — S0 owns one pair",
			blurb:  "One SERVICER. Plenty of spare slots. Would finish before +2s if alone.",
			code: `servicer = start_job(prio=20);  // claims one vac + one core

// CORE  S0 · · · · · · ·
// VAC   S0 · · · ·
`,
			apply: func(b *board) {
				b.clear()
				b.claimPair()
			},
		},
		{
			marker: "MEMORY_LEAK1",
			title:  "GOREADAX — re-arm for exactly 2.00s",
			blurb:  "No check that the old SERVICER finished. Memory unchanged.",
			code: `void goreadax(void) {
    schedule(READACCS, after=2.00s);
    // never asks: is old SERVICER done?
}
`,
			apply: nil,
		},
		{
			marker: "MEMORY_LEAK2",
			title:  "T3RUPT — timer fires on time",
			blurb:  "Radar stole CPU at CDUT/CDUS; TIME3 still punctual. Demand fixed.",
			code: `void t3rupt(void) {          // hardware clock
    run(READACCS);           // always on time
    // radar slowed jobs — not this timer
}
`,
			apply:    nil,
			playable: true,
		},
		{
			marker: "MEMORY_LEAK3",
			title:  "READACCS — start another cycle",
			blurb:  "Short task: read PIPAs. Does not look for older SERVICER copies.",
			code: `void readaccs(void) {
    read_pipas();
    start_job(SERVICER, prio=20);  // always a NEW copy
}
`,
			apply: nil,
		},
		{
			marker: "MEMORY_LEAK4",
			title:  "FINDVAC — brand-new SERVICER",
			blurb:  "Never resumes S0. Asks for a fresh VAC + core set.",
			code: `void start_job(SERVICER) {
    vac  = find_vac();    // MEMORY_LEAK5
    core = find_core();   // MEMORY_LEAK6
    // does not reuse any unfinished SERVICER
}
`,
			apply: nil,
		},
		{
			marker: "MEMORY_LEAK5",
			title:  "find_vac — claim or 1201",
			blurb:  "Scan five VAC flags. First free wins. None free → alarm.",
			code: `int find_vac(void) {
    if (!vac_1.in_use) { vac_1.in_use = true; return 1; }
    if (!vac_2.in_use) { vac_2.in_use = true; return 2; }
    if (!vac_3.in_use) { vac_3.in_use = true; return 3; }
    if (!vac_4.in_use) { vac_4.in_use = true; return 4; }
    if (!vac_5.in_use) { vac_5.in_use = true; return 5; }
    return 1201;   // no VAC areas
}
`,
			apply: func(b *board) {
				if b.next < 1 {
					b.clear()
					b.claimPair()
				}
				if b.next == 1 {
					b.claimPair() // S1 — shows second VAC claimed
				}
			},
		},
		{
			marker: "MEMORY_LEAK6",
			title:  "find_core — claim or 1202",
			blurb:  "Scan eight core sets. First free wins. None free → alarm.",
			code: `int find_core(void) {
    if (!core_1.in_use) { core_1.in_use = true; return 1; }
    if (!core_2.in_use) { core_2.in_use = true; return 2; }
    if (!core_3.in_use) { core_3.in_use = true; return 3; }
    if (!core_4.in_use) { core_4.in_use = true; return 4; }
    if (!core_5.in_use) { core_5.in_use = true; return 5; }
    if (!core_6.in_use) { core_6.in_use = true; return 6; }
    if (!core_7.in_use) { core_7.in_use = true; return 7; }
    if (!core_8.in_use) { core_8.in_use = true; return 8; }
    return 1202;   // no core sets
}
`,
			apply: nil, // board already has S0+S1 from LEAK5
		},
		{
			marker: "MEMORY_LEAK7",
			title:  "All jobs share the 2.00s — SERVICER last",
			blurb:  "LR/MONDO/gyro/… preempt prio 20. Press p to watch every lane.",
			code: `// who runs in one 2.00s period (prio high → low)
RR_ECDU   // HW steal ~0.30s
LRHJOB 32
LRVJOB 32
CHARIN 30 / MONDO 30
1/GYRO 21
MAKEPLAY 20
SERVICER 20   // leftovers ~1.20s, needs 1.80s

do_nav(); do_guidance(); do_throttle();
`,
			apply:    nil,
			playable: true,
		},
		{
			marker: "MEMORY_LEAK8",
			title:  "SERVEXIT — finish line missed",
			blurb:  "Still above ENDOFJOB when next cycle arms. Stub keeps 55 words.",
			code: `void servicer(void) {
    ...
    // next READACCS already started another copy
    // we never reach:
    end_of_job();
}
`,
			apply: nil,
		},
		{
			marker: "MEMORY_LEAK9",
			title:  "end_of_job — the only release",
			blurb:  "Frees the pair. Overload stubs never get here in time.",
			code: `void end_of_job(void) {
    core.in_use = false;
    vac.in_use  = false;
}
`,
			apply: nil,
		},
		{
			marker: "COMPOUND",
			title:  "Compounding — one pair per 2s",
			blurb:  "Each overloaded cycle claims another pair. Step to fill the pool.",
			code: `every 2.00s:
    start_job(SERVICER);   // old copies still in_use
    // S0, S1, S2, … keep their vac+core
`,
			apply: func(b *board) {
				for b.next < 3 {
					if ok, _ := b.claimPair(); !ok {
						return
					}
				}
			},
		},
		{
			marker: "FILL→1201",
			title:  "Pool full — ALARM 1201",
			blurb:  "All five VAC areas busy. find_vac falls through.",
			code: `int find_vac(void) {
    if (!vac_1.in_use) { ...; return 1; }
    if (!vac_2.in_use) { ...; return 2; }
    if (!vac_3.in_use) { ...; return 3; }
    if (!vac_4.in_use) { ...; return 4; }
    if (!vac_5.in_use) { ...; return 5; }
    return 1201;   // ← we hit this
}
`,
			apply: func(b *board) {
				b.clear()
				for {
					ok, alarm := b.claimPair()
					if !ok {
						b.alarm = alarm
						return
					}
					if b.next >= 5 {
						ok, alarm = b.claimPair()
						if !ok {
							b.alarm = alarm
						}
						return
					}
				}
			},
		},
		{
			marker: "RESTART",
			title:  "BAILOUT — clear the desk",
			blurb:  "Software restart frees cores/VACs. Engine & PIPAs kept running.",
			code: `void bailout(int alarm) {
    light_PROG();
    clear_all_jobs();     // stubs gone
    restart(SERVICER);    // one fresh copy
}
`,
			apply: func(b *board) {
				b.clear()
				b.claimPair()
			},
		},
	}
}

func healthyEvents() []event {
	return []event{
		{
			marker: "HEALTHY",
			title:  "Nominal descent — margin intact",
			blurb:  "No RR theft, no V16 N68. SERVICER finishes every 2.00s and frees memory.",
			code: `// checklist: RR in LGC (or zeroed)
// no monitor verb keyed up
// duty cycle < 100% of the 2.00s period
`,
			apply: func(b *board) { b.clear() },
		},
		{
			marker: "START",
			title:  "One SERVICER — one vac + one core",
			blurb:  "Same allocator as overload. Difference is: this copy will finish.",
			code: `servicer = start_job(prio=20);

// CORE  S0 · · · · · · ·
// VAC   S0 · · · ·
`,
			apply: func(b *board) {
				b.clear()
				b.claimPair()
			},
		},
		{
			marker: "RUN",
			title:  "2.00s cycle — SERVICER gets 1.80s",
			blurb:  "Higher jobs still run, but leftovers cover the full need. Press p.",
			code: `// CPU this period (healthy)
RR_ECDU    0.00s
LR / gyro  small
SERVICER   1.80s   // == need

do_nav(); do_guidance(); do_throttle();
`,
			apply:    nil,
			playable: true,
		},
		{
			marker: "DONE",
			title:  "end_of_job — success",
			blurb:  "Reaches the finish line before the next READACCS. Pool stays stable.",
			code: `void servicer(void) {
    ...
    end_of_job();   // reached in time
}

void end_of_job(void) {
    core.in_use = false;
    vac.in_use  = false;
}
`,
			apply: func(b *board) {
				b.clear() // freed
			},
		},
		{
			marker: "NEXT",
			title:  "Next cycle reuses the same slots",
			blurb:  "No stubs. find_vac / find_core succeed on the same pool. No alarm.",
			code: `every 2.00s:
    start_job(SERVICER);  // claims free vac+core
    ...
    end_of_job();         // returns them
// steady state — no leak
`,
			apply: func(b *board) {
				b.clear()
				b.claimPair()
			},
		},
	}
}

func (m *model) loadScenario(sc scenario) {
	m.scenario = sc
	m.jobs = jobsFor(sc)
	m.bursts = buildBursts(sc)
	m.jobFocus = -1
	m.playing = false
	m.simT = 0
	m.idx = 0
	if sc == scenarioHealthy {
		m.events = healthyEvents()
	} else {
		m.events = events()
	}
	m.board = newBoard()
	if m.events[0].apply != nil {
		m.events[0].apply(&m.board)
	}
	m.status = fmt.Sprintf("%s — n step · p play · j/k jobs · h toggle · q quit",
		scenarioName(sc))
}

type model struct {
	width, height int
	fadeAt        time.Time
	fadeDone      bool

	scenario scenario
	mode     mode
	events   []event
	idx      int
	board    board
	jobs     []landingJob
	bursts   []burst
	jobFocus int // -1 = show step code; else index into jobs

	playStart time.Time
	simT      float64
	playing   bool

	status string
}

func initialModel() model {
	m := model{
		fadeAt:   time.Now(),
		jobFocus: -1,
		mode:     modeStep,
	}
	m.loadScenario(scenarioOverload)
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), tea.EnterAltScreen)
}

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		if !m.fadeDone {
			if time.Since(m.fadeAt) >= fadeDuration {
				m.fadeDone = true
			}
		}
		if m.playing {
			elapsed := time.Since(m.playStart)
			prog := float64(elapsed) / float64(playRealDuration)
			if prog >= 1 {
				m.simT = periodS
				m.playing = false
				if m.scenario == scenarioHealthy {
					m.board.clear()
					m.status = "HEALTHY cycle OK — SERVICER finished, pool freed · p replay · h overload"
				} else {
					m.status = "OVERLOAD cycle — SERVICER late · n step · p replay · h healthy"
				}
			} else {
				m.simT = prog * periodS
				// Healthy: show S0 held during the run
				if m.scenario == scenarioHealthy && m.board.next == 0 {
					m.board.claimPair()
				}
			}
		}
		return m, tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "h":
			if m.scenario == scenarioHealthy {
				m.loadScenario(scenarioOverload)
			} else {
				m.loadScenario(scenarioHealthy)
			}
			return m, nil
		case "j":
			if m.jobFocus < 0 {
				m.jobFocus = 0
			} else {
				m.jobFocus = (m.jobFocus + 1) % len(m.jobs)
			}
			m.status = fmt.Sprintf("job %d/%d  %s  · j/k browse · esc back to steps",
				m.jobFocus+1, len(m.jobs), m.jobs[m.jobFocus].Name)
			return m, nil
		case "k":
			if m.jobFocus < 0 {
				m.jobFocus = len(m.jobs) - 1
			} else {
				m.jobFocus--
				if m.jobFocus < 0 {
					m.jobFocus = len(m.jobs) - 1
				}
			}
			m.status = fmt.Sprintf("job %d/%d  %s  · j/k browse · esc back to steps",
				m.jobFocus+1, len(m.jobs), m.jobs[m.jobFocus].Name)
			return m, nil
		case "esc":
			m.jobFocus = -1
			m.status = fmt.Sprintf("%s — n step · p play · j/k jobs · h toggle",
				scenarioName(m.scenario))
			return m, nil
		case "n", "right", " ", "enter":
			if m.playing {
				return m, nil
			}
			m.jobFocus = -1
			m.mode = modeStep
			if m.idx < len(m.events)-1 {
				m.idx++
				if m.events[m.idx].apply != nil {
					m.events[m.idx].apply(&m.board)
				}
			}
			m.status = fmt.Sprintf("%s step %d/%d — n next · p play · h toggle · j/k jobs",
				scenarioName(m.scenario), m.idx+1, len(m.events))
			return m, nil
		case "left", "b":
			if m.playing {
				return m, nil
			}
			m.jobFocus = -1
			if m.idx > 0 {
				m.idx--
				m.board = newBoard()
				for i := 0; i <= m.idx; i++ {
					if m.events[i].apply != nil {
						m.events[i].apply(&m.board)
					}
				}
			}
			m.status = fmt.Sprintf("%s step %d/%d", scenarioName(m.scenario), m.idx+1, len(m.events))
			return m, nil
		case "p":
			m.mode = modePlay
			m.playing = true
			m.playStart = time.Now()
			m.simT = 0
			m.jobFocus = -1
			m.board = newBoard()
			m.board.claimPair() // S0 for the cycle
			m.status = fmt.Sprintf("playing %s 2.00s cycle…", scenarioName(m.scenario))
			return m, nil
		case "r":
			m.loadScenario(m.scenario)
			m.status = "reset · " + scenarioName(m.scenario)
			return m, nil
		}
	}
	return m, nil
}

func fadeT(m model) float64 {
	if m.fadeDone {
		return 1
	}
	t := float64(time.Since(m.fadeAt)) / float64(fadeDuration)
	if t > 1 {
		return 1
	}
	if t < 0 {
		return 0
	}
	return t
}

func (m model) View() string {
	ft := fadeT(m)
	if m.width == 0 {
		return ""
	}

	// Fade: background moves #000000 → rose-pine base over 500ms
	bg := blendHex("#000000", "#191724", ft)
	fg := blendHex("#000000", "#e0def4", ft)

	leftW := m.width * 52 / 100
	if leftW < 40 {
		leftW = m.width / 2
	}
	rightW := m.width - leftW - 3
	if rightW < 30 {
		rightW = 30
		leftW = m.width - rightW - 3
	}

	left := m.viewLeft(leftW, ft)
	right := m.viewRight(rightW, ft)

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftW).Render(left),
		lipgloss.NewStyle().Foreground(blendHex("#000000", "#26233a", ft)).Render(" │ "),
		lipgloss.NewStyle().Width(rightW).Render(right),
	)

	help := lipgloss.NewStyle().Foreground(blendHex("#000000", "#6e6a86", ft)).Render(m.status)
	body := lipgloss.JoinVertical(lipgloss.Left, cols, "", help)

	return lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Width(m.width).
		Height(m.height).
		Render(body)
}

func blendHex(from, to string, t float64) lipgloss.Color {
	if t <= 0 {
		return lipgloss.Color(from)
	}
	if t >= 1 {
		return lipgloss.Color(to)
	}
	cf, _ := colorful.Hex(from)
	ct, _ := colorful.Hex(to)
	return lipgloss.Color(cf.BlendRgb(ct, t).Hex())
}

func (m model) viewLeft(width int, ft float64) string {
	_ = ft
	ev := m.events[m.idx]
	badgeColor := rpLove
	if m.scenario == scenarioHealthy {
		badgeColor = rpFoam
	}
	badge := lipgloss.NewStyle().Foreground(badgeColor).Bold(true).Render(
		"[" + scenarioName(m.scenario) + "]",
	)
	header := lipgloss.NewStyle().Foreground(rpIris).Render("APOLLO 11 · LGC  ") + badge

	title := lipgloss.NewStyle().Foreground(rpGold).Bold(true).Render(
		fmt.Sprintf("%s  %s", ev.marker, ev.title),
	)
	blurb := lipgloss.NewStyle().Foreground(rpSubtle).Width(width - 2).Render(ev.blurb)

	board := m.renderBoard(width)
	bars := m.renderBars(width)

	parts := []string{header, "", title, blurb, "", board, "", bars}
	return lipgloss.NewStyle().Width(width).Render(strings.Join(parts, "\n"))
}

func (m model) renderBoard(width int) string {
	var coreParts, vacParts []string
	cell := func(owner int, hot bool) string {
		sty := lipgloss.NewStyle().Padding(0, 1)
		if owner < 0 {
			return sty.Foreground(rpMuted).Render(" · ")
		}
		label := fmt.Sprintf("S%d", owner)
		c := sty.Foreground(rpFoam).Bold(true)
		if hot {
			c = sty.Foreground(rpGold).Bold(true).Background(rpOverlay)
		}
		if m.board.alarm != 0 {
			c = sty.Foreground(rpLove).Bold(true)
		}
		return c.Render(label)
	}

	// highlight newest owner
	newest := m.board.next - 1
	for i, o := range m.board.cores {
		_ = i
		coreParts = append(coreParts, cell(o, o == newest && o >= 0))
	}
	for _, o := range m.board.vacs {
		vacParts = append(vacParts, cell(o, o == newest && o >= 0))
	}

	label := lipgloss.NewStyle().Foreground(rpPine).Bold(true)
	alarm := ""
	if m.board.alarm != 0 {
		alarm = "\n" + lipgloss.NewStyle().Foreground(rpLove).Bold(true).
			Render(fmt.Sprintf("PROG · ALARM %d", m.board.alarm))
	}

	s := fmt.Sprintf("%s\nCORE %s\n%s\nVAC  %s%s",
		label.Render("ERASABLE BOARD"),
		strings.Join(coreParts, ""),
		label.Render(""),
		strings.Join(vacParts, ""),
		alarm,
	)
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rpOverlay).
		Padding(0, 1).
		Width(width - 2)
	return box.Render(s)
}

// barCells converts a fill fraction into whole '█' cells plus one shade rune
// ('░' '▒' '▓') for the sub-cell remainder, so any nonzero amount is visible
// instead of truncating to nothing. Returns partial == 0 when there is no
// fractional cell. full + one partial cell never exceeds cells.
func barCells(frac float64, cells int) (full int, partial rune) {
	if cells <= 0 || frac <= 0 {
		return 0, 0
	}
	if frac >= 1 {
		return cells, 0
	}
	exact := frac * float64(cells)
	full = int(exact)
	rem := exact - float64(full)
	switch {
	case rem <= 0:
		return full, 0
	case rem < 1.0/3:
		return full, '░'
	case rem < 2.0/3:
		return full, '▒'
	default:
		return full, '▓'
	}
}

func barLine(label string, filled, total float64, width int, fill lipgloss.Color, ghost float64) string {
	inner := width - 28
	if inner < 8 {
		inner = 8
	}
	frac := 0.0
	if total > 0 {
		frac = filled / total
	}
	full, part := barCells(frac, inner)
	partial := ""
	used := full
	if part != 0 {
		partial = string(part)
		used++
	}
	ghostN := 0
	if total > 0 && ghost > filled {
		ghostN = int((ghost/total)*float64(inner)) - used
		if ghostN < 0 {
			ghostN = 0
		}
		if used+ghostN > inner {
			ghostN = inner - used
		}
	}
	rest := inner - used - ghostN
	if rest < 0 {
		rest = 0
	}

	fillSt := lipgloss.NewStyle().Foreground(fill)
	ghostSt := lipgloss.NewStyle().Foreground(rpMuted)
	emptySt := lipgloss.NewStyle().Foreground(rpOverlay)

	bar := fillSt.Render(strings.Repeat("█", full)+partial) +
		ghostSt.Render(strings.Repeat("░", ghostN)) +
		emptySt.Render(strings.Repeat("·", rest))

	lab := lipgloss.NewStyle().Foreground(rpSubtle).Width(9).Render(label)
	nums := lipgloss.NewStyle().Foreground(rpText).Render(
		fmt.Sprintf(" %4.2fs", filled),
	)
	return lab + " " + bar + nums
}

func (m model) renderBars(width int) string {
	t := m.simT
	if m.mode == modeStep && !m.playing {
		ev := m.events[m.idx]
		if ev.playable || ev.marker == "MEMORY_LEAK7" || ev.marker == "MEMORY_LEAK8" || ev.marker == "RUN" || ev.marker == "DONE" || ev.marker == "NEXT" {
			t = periodS
		} else {
			t = 0
		}
	}

	p := t / periodS
	if p > 1 {
		p = 1
	}

	spent := cpuSpentBy(m.jobs, t)
	active := activeJobsAt(t, m.bursts)

	title := lipgloss.NewStyle().Foreground(rpRose).Bold(true).Render(
		fmt.Sprintf("ONE 2.00s CYCLE — %s", scenarioName(m.scenario)),
	)
	clock := lipgloss.NewStyle().Foreground(rpGold).Render(
		fmt.Sprintf("playhead  t = %.2fs / %.2fs", t, periodS),
	)

	inner := width - 8
	if inner < 20 {
		inner = 20
	}
	pos := int(p * float64(inner))
	if pos > inner {
		pos = inner
	}
	scrub := lipgloss.NewStyle().Foreground(rpFoam).Render(strings.Repeat("━", pos)) +
		lipgloss.NewStyle().Foreground(rpLove).Render("█") +
		lipgloss.NewStyle().Foreground(rpOverlay).Render(strings.Repeat("─", max(0, inner-pos)))

	act := lipgloss.NewStyle().Foreground(rpMuted).Render("running now: ")
	if len(active) == 0 {
		act += lipgloss.NewStyle().Foreground(rpSubtle).Render("(gap)")
	} else {
		var parts []string
		for _, i := range active {
			j := m.jobs[i]
			parts = append(parts,
				lipgloss.NewStyle().Foreground(lipgloss.Color(j.ColorHex)).Bold(true).Render(j.Name),
			)
		}
		act += strings.Join(parts, "  ")
	}

	var bars []string
	for i, j := range m.jobs {
		ghost := 0.0
		label := j.Name
		if j.Name == "SERVICER" && m.scenario == scenarioOverload {
			ghost = servicerNeedS
		}
		if m.jobFocus == i {
			label = "▸" + j.Name
		}
		line := barLine(label, spent[i], periodS, width, lipgloss.Color(j.ColorHex), ghost)
		if m.jobFocus >= 0 && m.jobFocus != i {
			line = lipgloss.NewStyle().Faint(true).Render(line)
		}
		bars = append(bars, line)
	}

	gantt := m.renderGantt(width, t)

	svcGot := m.jobs[len(m.jobs)-1].CPUSec
	outcome := ""
	if p >= 1 || (!m.playing && t >= periodS) {
		if m.scenario == scenarioHealthy {
			outcome = "\n" + lipgloss.NewStyle().Foreground(rpFoam).Bold(true).Render(
				fmt.Sprintf("SUCCESS  SERVICER got %.2fs  need %.2fs  → end_of_job()",
					svcGot, servicerNeedS),
			)
		} else {
			outcome = "\n" + lipgloss.NewStyle().Foreground(rpLove).Bold(true).Render(
				fmt.Sprintf("SHORTFALL  SERVICER got %.2fs  need %.2fs  → misses ENDOFJOB",
					svcGot, servicerNeedS),
			)
		}
	}

	hint := lipgloss.NewStyle().Foreground(rpMuted).Render(
		"p play · j/k highlight job · h healthy/overload",
	)

	ghostNote := "SERVICER ░ = still short of 1.80s need"
	if m.scenario == scenarioHealthy {
		ghostNote = "SERVICER bar fills the full 1.80s need — finishes clean"
	}

	body := strings.Join([]string{
		title,
		clock,
		scrub,
		act,
		"",
		gantt,
		"",
		lipgloss.NewStyle().Foreground(rpPine).Bold(true).Render("CPU consumed in period"),
		strings.Join(bars, "\n"),
		lipgloss.NewStyle().Foreground(rpMuted).Render(ghostNote),
		outcome,
		"",
		hint,
	}, "\n")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rpPine).
		Padding(0, 1).
		Width(width - 2)
	return box.Render(body)
}

// laneCoverage returns, for each of laneW cells across one 2.00s period, the
// fraction [0,1] of that cell's time covered by the job's bursts. Bursts are
// clamped to the period; overlaps saturate at 1.
func laneCoverage(bursts []burst, jobIndex, laneW int) []float64 {
	cover := make([]float64, max(laneW, 0))
	if laneW <= 0 {
		return cover
	}
	cellDur := periodS / float64(laneW)
	for _, b := range bursts {
		if b.JobIndex != jobIndex {
			continue
		}
		start := math.Max(b.Start, 0)
		end := math.Min(b.End, periodS)
		if end <= start {
			continue
		}
		first := int(start / cellDur)
		if first >= laneW {
			continue
		}
		last := int(math.Ceil(end/cellDur)) - 1
		if last >= laneW {
			last = laneW - 1
		}
		for x := first; x <= last; x++ {
			cellStart := float64(x) * cellDur
			lo := math.Max(cellStart, start)
			hi := math.Min(cellStart+cellDur, end)
			if hi > lo {
				cover[x] += (hi - lo) / cellDur
			}
		}
	}
	// Snap float noise so a boundary-grazing burst doesn't light a stray cell
	// and a fully covered cell reads exactly 1.
	for i := range cover {
		switch {
		case cover[i] < 1e-9:
			cover[i] = 0
		case cover[i] > 1-1e-9:
			cover[i] = 1
		}
	}
	return cover
}

// coverageRune maps cell coverage to a shade so partial compute in one column
// stays visible without pretending to be a full block.
func coverageRune(c float64) rune {
	switch {
	case c <= 0:
		return '·'
	case c < 0.25:
		return '░'
	case c < 0.5:
		return '▒'
	case c < 0.9:
		return '▓'
	default:
		return '█'
	}
}

func (m model) renderGantt(width int, t float64) string {
	// 16 cols of prefix+name+prio, plus 4 the surrounding box consumes
	// (width inset + padding) — wider lanes word-wrap off their labels.
	laneW := width - 20
	if laneW < 24 {
		laneW = 24
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Foreground(rpPine).Bold(true).Render("SCHEDULE (0 … 2.00s)"))

	ph := -1
	if p := int(t / periodS * float64(laneW)); p >= 0 && p < laneW {
		ph = p
	}

	for ji, j := range m.jobs {
		lane := make([]rune, laneW)
		for x, c := range laneCoverage(m.bursts, ji, laneW) {
			lane[x] = coverageRune(c)
		}

		focused := m.jobFocus == ji
		dimOthers := m.jobFocus >= 0 && !focused

		nameSt := lipgloss.NewStyle().Foreground(lipgloss.Color(j.ColorHex)).Width(9)
		laneSt := lipgloss.NewStyle().Foreground(lipgloss.Color(j.ColorHex))
		playSt := lipgloss.NewStyle().Foreground(rpLove).Bold(true)
		if focused {
			nameSt = nameSt.Bold(true).Background(rpOverlay)
			laneSt = laneSt.Bold(true)
		}
		if dimOthers {
			nameSt = nameSt.Faint(true)
			laneSt = laneSt.Faint(true)
			playSt = playSt.Faint(true)
		}

		// Playhead is a solid full-height block — a '◆' diamond next to '█'
		// bursts composited into boot-shaped artifacts.
		laneStr := laneSt.Render(string(lane))
		if ph >= 0 {
			laneStr = laneSt.Render(string(lane[:ph])) +
				playSt.Render("█") +
				laneSt.Render(string(lane[ph+1:]))
		}

		prio := "HW"
		if j.Prio > 0 {
			prio = fmt.Sprintf("%2d", j.Prio)
		}
		prioS := lipgloss.NewStyle().Foreground(rpMuted).Width(3).Render(prio)
		prefix := "  "
		if focused {
			prefix = "▸ "
		}
		lines = append(lines, prefix+nameSt.Render(j.Name)+" "+prioS+" "+laneStr)
	}
	return strings.Join(lines, "\n")
}

func (m model) viewRight(width int, ft float64) string {
	_ = ft
	if m.jobFocus >= 0 && m.jobFocus < len(m.jobs) {
		return m.viewJobExplain(width, m.jobs[m.jobFocus])
	}
	return m.viewStepCode(width)
}

func (m model) viewJobExplain(width int, j landingJob) string {
	head := lipgloss.NewStyle().Foreground(lipgloss.Color(j.ColorHex)).Bold(true).
		Render("JOB  ·  " + j.Name)
	sub := lipgloss.NewStyle().Foreground(rpMuted).Render("j/k next job · esc back to step code")

	prio := "hardware (not an Executive job)"
	if j.Prio > 0 {
		prio = fmt.Sprintf("priority %d · %s", j.Prio, j.Kind)
	}
	meta := lipgloss.NewStyle().Foreground(rpGold).Render(prio)

	cadence := lipgloss.NewStyle().Foreground(rpSubtle).Render("cadence: " + j.Cadence)
	cpu := lipgloss.NewStyle().Foreground(rpText).Render(
		fmt.Sprintf("CPU this period (%s): %.2fs", scenarioName(m.scenario), j.CPUSec),
	)

	body := lipgloss.NewStyle().
		Foreground(rpText).
		Background(rpSurface).
		Padding(1, 2).
		Width(width - 2).
		Render(j.Explain)

	return lipgloss.JoinVertical(lipgloss.Left,
		head, sub, "", meta, cadence, cpu, "", body,
	)
}

func (m model) viewStepCode(width int) string {
	ev := m.events[m.idx]
	head := lipgloss.NewStyle().Foreground(rpGold).Bold(true).Render("C  ·  " + ev.marker)
	sub := lipgloss.NewStyle().Foreground(rpMuted).Render("abstract C · j/k for job explain")

	codeSt := lipgloss.NewStyle().
		Foreground(rpText).
		Background(rpSurface).
		Padding(1, 2).
		Width(width - 2)

	lines := strings.Split(strings.TrimRight(ev.code, "\n"), "\n")
	var rendered []string
	for _, ln := range lines {
		trim := strings.TrimSpace(ln)
		if strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "/*") {
			rendered = append(rendered, lipgloss.NewStyle().Foreground(rpMuted).Render(ln))
		} else if strings.Contains(ln, "1201") || strings.Contains(ln, "1202") || strings.Contains(ln, "ALARM") {
			rendered = append(rendered, lipgloss.NewStyle().Foreground(rpLove).Bold(true).Render(ln))
		} else if strings.Contains(ln, "end_of_job") || strings.Contains(ln, "SUCCESS") {
			rendered = append(rendered, lipgloss.NewStyle().Foreground(rpFoam).Render(ln))
		} else {
			rendered = append(rendered, lipgloss.NewStyle().Foreground(rpText).Render(ln))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, head, sub, "", codeSt.Render(strings.Join(rendered, "\n")))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
