// dsky-lab: a standalone terminal DSKY, replaying the Apollo 11 descent
// displays so you can watch the monitor work:
//
//	d — the crew keys V37E 63E: P63 engages, V06 N63 shows velocity,
//	    altitude-rate and altitude ticking once per second; ALT + VEL burn
//	    until the landing radar locks
//	n — Buzz keys V16N68E: R1 slant range, R2 time-to-go, R3 DELTAH
//	    refreshing at 1Hz with COMP ACTY pulsing on each computation
//	a — executive overflow: PROG lights, RESTART flashes, the unprotected
//	    monitor is dropped and the display snaps back to V06 N63
//	r — RSET clears the caution lamps · q — quit
package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/dsky-lab/dsky"
)

// ---------------------------------------------------------------------------
// The scripted flight (deterministic, time-driven — display values only)
// ---------------------------------------------------------------------------

type event struct {
	at time.Duration
	fn func(*demo)
}

type demo struct {
	t       time.Duration // mission clock
	queue   []event       // pending keystroke script
	started bool
	startAt time.Duration
	verb    string
	noun    string

	monitor  bool
	monStart time.Duration

	progLamp     bool
	restartUntil time.Duration
}

func newDemo() *demo { return &demo{} }

func (d *demo) at(deltaMs float64, fn func(*demo)) {
	d.queue = append(d.queue, event{d.t + time.Duration(deltaMs)*time.Millisecond, fn})
}

// press handles one lab key.
func (d *demo) press(k byte) {
	switch k {
	case 'd':
		if d.started || len(d.queue) > 0 {
			return
		}
		d.at(200, func(d *demo) { d.verb = "37" })
		d.at(900, func(d *demo) { d.noun = "63" })
		d.at(1400, func(d *demo) { // ENTR
			d.started = true
			d.startAt = d.t
			d.verb, d.noun = "06", "63"
		})
	case 'n':
		if !d.started || d.monitor || len(d.queue) > 0 {
			return
		}
		d.at(200, func(d *demo) { d.verb = "16" })
		d.at(900, func(d *demo) { d.noun = "68" })
		d.at(1400, func(d *demo) { // ENTR
			d.monitor = true
			d.monStart = d.t
		})
	case 'a':
		if !d.started {
			return
		}
		d.progLamp = true
		d.restartUntil = d.t + 1200*time.Millisecond
		d.monitor = false
		d.verb, d.noun = "06", "63" // the restart reverts the display
	case 'r':
		d.progLamp = false
	}
}

// advance moves the mission clock, firing due keystrokes in order.
func (d *demo) advance(ms float64) {
	target := d.t + time.Duration(ms)*time.Millisecond
	for len(d.queue) > 0 && d.queue[0].at <= target {
		ev := d.queue[0]
		d.queue = d.queue[1:]
		d.t = ev.at
		ev.fn(d)
	}
	d.t = target
}

const lrLockAfter = 8 * time.Second

// state derives the DSKY face from the mission clock. Displays update on
// whole seconds — the real cadence of the display routines.
func (d *demo) state() dsky.State {
	s := dsky.State{Verb: d.verb, Noun: d.noun}
	s.Lights.Prog = d.progLamp
	s.Lights.Restart = d.t < d.restartUntil
	if !d.started {
		return s
	}
	s.Prog = "63"
	elapsed := (d.t - d.startAt).Seconds()
	es := math.Floor(elapsed)

	radarLocked := d.t >= d.startAt+lrLockAfter
	s.Lights.Alt = !radarLocked
	s.Lights.Vel = !radarLocked

	// COMP ACTY pulses at the top of each second while computing.
	s.CompActy = math.Mod(elapsed, 1.0) < 0.28

	if d.monitor {
		// V16 N68: R1 slant range (.1 nmi), R2 time-to-go (min/sec),
		// R3 DELTAH (ft) converging as radar data folds into the state.
		em := math.Floor((d.t - d.monStart).Seconds())
		s.R1 = fmt.Sprintf("+%05.0f", math.Max(0, 1405-3*em))
		ttg := int(math.Max(0, 215-em))
		s.R2 = fmt.Sprintf("+%03d%02d", ttg/60, ttg%60)
		s.R3 = fmt.Sprintf("-%05.0f", math.Max(900, 2900-55*em))
		return s
	}

	// V06 N63: velocity, altitude-rate, altitude.
	s.R1 = fmt.Sprintf("+%05.0f", math.Max(0, 5559-6.2*es))
	s.R2 = fmt.Sprintf("-%05.0f", math.Min(99999, 84+0.4*es))
	s.R3 = fmt.Sprintf("+%05.0f", math.Max(0, 49971-84*es))
	return s
}

// caption narrates the current phase for the audience.
func (d *demo) caption() string {
	switch {
	case !d.started && len(d.queue) > 0:
		return "keying V37E 63E — selecting the descent program"
	case !d.started:
		return "idle — press d to key the descent"
	case d.progLamp:
		return "EXECUTIVE OVERFLOW — restart dropped the monitor (r to RSET)"
	case d.monitor:
		return "V16 N68: R1 range ·.1nmi  R2 time-to-go m:ss  R3 DELTAH ft — 1Hz"
	case len(d.queue) > 0:
		return "keying V16N68E — Buzz wants DELTAH"
	case d.t < d.startAt+lrLockAfter:
		return "P63 running — ALT/VEL lit until the landing radar locks"
	default:
		return "P63 running, radar locked — press n for Buzz's monitor"
	}
}

// ---------------------------------------------------------------------------
// bubbletea shell
// ---------------------------------------------------------------------------

type frameMsg struct{}

const frameMs = 33.34

type model struct {
	d     *demo
	frame int
}

func (m model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(time.Duration(frameMs*1e6)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case frameMsg:
		m.frame++
		m.d.advance(frameMs)
		return m, tick()
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if len(msg.Runes) == 1 {
			if msg.Runes[0] == 'q' {
				return m, tea.Quit
			}
			m.d.press(byte(msg.Runes[0]))
		}
	}
	return m, nil
}

func (m model) View() string {
	dim := "\x1b[38;5;240m"
	reset := "\x1b[0m"
	blinkOn := (m.frame/12)%2 == 0

	panel := dsky.Render(m.d.state(), blinkOn)
	pad := "   "
	var b strings.Builder
	b.WriteString("\n")
	for _, l := range strings.Split(panel, "\n") {
		b.WriteString(pad + l + "\n")
	}
	b.WriteString("\n" + pad + dim + m.d.caption() + reset + "\n")
	b.WriteString(pad + dim + "[d] descent  [n] V16N68  [a] alarm  [r] rset  [q] quit" + reset + "\n")
	return b.String()
}

func main() {
	if _, err := tea.NewProgram(model{d: newDemo()}, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "dsky-lab:", err)
		os.Exit(1)
	}
}
