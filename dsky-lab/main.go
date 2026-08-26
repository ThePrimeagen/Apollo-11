// dsky-lab: a standalone terminal DSKY, replaying the Apollo 11 descent
// displays so you can watch the monitor work:
//
//	d — the crew keys V37E 63E: P63 engages, V06 N63 shows velocity,
//	    altitude-rate and altitude ticking once per second; ALT + VEL burn
//	    until the landing radar locks. Each keystroke lights the keypad.
//	n — Buzz keys V16N68E: R1 slant range, R2 time-to-go, R3 DELTAH
//	    refreshing at 1Hz with COMP ACTY pulsing on each computation
//	a — executive overflow: PROG lights, RESTART flashes, the unprotected
//	    monitor is dropped and the display snaps back to V06 N63
//	r — RSET clears the caution lamps · q — quit
//	0–9 V N E C — press the matching DSKY key, the way typing mode does
//	    in the Executive sim. Digits being keyed turn dull orange.
package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

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

	entering   byte // 'V', 'N', or 0 — same as the engine's PressKey
	progSel    bool
	pressed    dsky.Key
	pressUntil time.Duration
}

func newDemo() *demo { return &demo{} }

func (d *demo) at(deltaMs float64, fn func(*demo)) {
	d.queue = append(d.queue, event{d.t + time.Duration(deltaMs)*time.Millisecond, fn})
}

const keyHold = 180 * time.Millisecond

func (d *demo) queueKeys(seq string) {
	delay := 50.0
	for _, r := range seq {
		k := dsky.Key(r)
		d.at(delay, func(d *demo) { d.key(k) })
		delay += 150
	}
}

// key is one DSKY keystroke — the same contract as the engine's PressKey.
// The matching keypad button stays depressed for keyHold so the press is
// visible; digits of the field being entered render dull orange.
func (d *demo) key(k dsky.Key) {
	d.pressed = k
	d.pressUntil = d.t + keyHold
	switch k {
	case dsky.KeyVerb:
		d.entering = 'V'
		d.verb = ""
		d.progSel = false
	case dsky.KeyNoun:
		d.entering = 'N'
		d.noun = ""
		d.progSel = false
	case dsky.KeyClr:
		d.entering = 0
		d.progSel = false
	case dsky.KeyEntr:
		d.entering = 0
		if d.progSel {
			d.progSel = false
			if d.noun == "63" && !d.started {
				d.started = true
				d.startAt = d.t
				d.verb, d.noun = "06", "63"
			}
			break
		}
		if d.verb == "37" {
			d.progSel = true
			d.entering = 'N'
			d.noun = ""
			break
		}
		if d.verb == "16" && d.noun == "68" && d.started && !d.monitor {
			d.monitor = true
			d.monStart = d.t
		}
	default:
		if k >= '0' && k <= '9' {
			switch d.entering {
			case 'V':
				if len(d.verb) < 2 {
					d.verb += string(k)
				}
			case 'N':
				if len(d.noun) < 2 {
					d.noun += string(k)
				}
			}
		}
	}
}

// press handles one lab key. Digits, V/N/E/C, and +/− go to the keypad
// the way typing mode does in the Executive sim. d/n/a/r are the
// scripted story beats — and those scripts now type on the pad too.
func (d *demo) press(k byte) {
	switch k {
	case 'd':
		if d.started || len(d.queue) > 0 {
			return
		}
		d.queueKeys("V37E63E")
	case 'n', 'N':
		if d.entering != 0 {
			d.key(dsky.KeyNoun)
			return
		}
		if !d.started || d.monitor || len(d.queue) > 0 {
			return
		}
		d.queueKeys("V16N68E")
	case 'a':
		if !d.started {
			return
		}
		d.progLamp = true
		d.restartUntil = d.t + 1200*time.Millisecond
		d.monitor = false
		d.verb, d.noun = "06", "63" // the restart reverts the display
		d.entering = 0
		d.progSel = false
	case 'r':
		d.progLamp = false
	default:
		switch {
		case k >= '0' && k <= '9', k == '+' || k == '-':
			d.key(dsky.Key(k))
		case k == 'v' || k == 'V':
			d.key(dsky.KeyVerb)
		case k == 'e' || k == 'E':
			d.key(dsky.KeyEntr)
		case k == 'c' || k == 'C':
			d.key(dsky.KeyClr)
		}
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
	if d.t < d.pressUntil {
		s.Pressed = d.pressed
	}
	s.Typing.Verb = d.entering == 'V'
	s.Typing.Noun = d.entering == 'N'
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
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if r, ok := keyRune(msg); ok {
			if r == 'q' {
				return m, tea.Quit
			}
			m.d.press(byte(r))
		}
		if msg.Code == tea.KeyEnter {
			m.d.press('E')
		}
	}
	return m, nil
}

// keyRune returns the single printable rune of a key press, if any.
func keyRune(msg tea.KeyPressMsg) (rune, bool) {
	rs := []rune(msg.Text)
	if len(rs) != 1 {
		return 0, false
	}
	return rs[0], true
}

func (m model) View() tea.View {
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
	b.WriteString(pad + dim + "[d] descent  [n] V16N68  [a] alarm  [r] rset  0-9/V/N/E keypad  [q] quit" + reset + "\n")
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func main() {
	if _, err := tea.NewProgram(model{d: newDemo()}).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "dsky-lab:", err)
		os.Exit(1)
	}
}
