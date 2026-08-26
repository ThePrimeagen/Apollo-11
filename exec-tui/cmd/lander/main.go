// lander: an ASCII lunar-lander demo of the Apollo 11 powered descent.
// The LM stays upright and lowers continuously as mission time runs; ALT and
// VEL interpolate between the real flight events, and the five program
// alarms appear at their true moments. [ and ] change the playback rate,
// '.' pauses, q quits.
package main

import (
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander/descent"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

type scriptEvent struct {
	timeSec float64
	altFt   float64
	velFps  float64
	phase   string
	alarm   string // "" when the event is not an executive overflow
	caption string
}

// script is the flight, PDI-relative (times/altitudes per Cherry's event log
// and Eyles; velocities approximate).
var script = []scriptEvent{
	{0, 49971, 5560, "P63 BRAKING", "", "PDI — DPS ignition at 10%"},
	{26, 48000, 5460, "P63 BRAKING", "", "throttle to full — guidance enabled"},
	{232, 42426, 3366, "P63 BRAKING", "", "yaw maneuver — windows up"},
	{274, 39000, 2745, "P63 BRAKING", "", "landing radar: data good"},
	{304, 35706, 2521, "P63 BRAKING", "", "Buzz keys V16N68 — DELTAH monitor"},
	{316, 33500, 2280, "P63 BRAKING", "1202", "PROG ALARM 1202 — no core sets"},
	{358, 30900, 1826, "P63 BRAKING", "1202", "1202 again — monitor re-keyed"},
	{384, 23393, 1481, "P63 BRAKING", "", "throttle down — right on time"},
	{506, 7400, 506, "P64 APPROACH", "", "high gate — LPD active"},
	{552, 3000, 130, "P64 APPROACH", "1201", "PROG ALARM 1201 — no VAC areas"},
	{578, 2000, 90, "P64 APPROACH", "1202", "PROG ALARM 1202"},
	{594, 770, 60, "P64 APPROACH", "1202", "PROG ALARM 1202 — 770 ft"},
	{603, 650, 50, "P64 APPROACH", "", "Armstrong: AUTO → ATT HOLD"},
	{615, 430, 30, "P66 LANDING", "", "P66 — rate of descent mode"},
	{757, 0, 0, "P66 LANDED", "", "CONTACT LIGHT — the Eagle has landed"},
}

const defaultScale = 20 // mission seconds per wall second

type demoModel struct {
	t      float64 // mission time, seconds
	scale  float64 // mission seconds per wall second
	paused bool
	tick   int // animation frame counter
}

func newDemoModel() demoModel { return demoModel{scale: defaultScale} }

// advance moves mission time by wall milliseconds at the playback rate.
func (m *demoModel) advance(wallMs float64) {
	if m.paused {
		return
	}
	m.t += wallMs / 1000 * m.scale
	if end := script[len(script)-1].timeSec; m.t > end {
		m.t = end
	}
}

// lerp interpolates a value between the neighboring script events.
func lerp(t, t0, t1, v0, v1 float64) float64 {
	if t1 <= t0 {
		return v1
	}
	f := (t - t0) / (t1 - t0)
	return v0 + f*(v1-v0)
}

// state assembles the lander view for the current mission time: meters
// interpolated between events, alarms accumulated, the craft always upright
// until it is down.
func (m demoModel) state() descent.State {
	last := script[0]
	next := script[len(script)-1]
	for i := range script {
		if script[i].timeSec <= m.t {
			last = script[i]
			if i+1 < len(script) {
				next = script[i+1]
			} else {
				next = script[i]
			}
		}
	}
	end := script[len(script)-1].timeSec
	st := descent.State{
		AltFt:     lerp(m.t, last.timeSec, next.timeSec, last.altFt, next.altFt),
		VelFps:    lerp(m.t, last.timeSec, next.timeSec, last.velFps, next.velFps),
		TimeSec:   m.t,
		LandInSec: end - m.t,
		Tick:      m.tick,
		Phase:     last.phase,
		Event:     last.caption,
	}
	st.Attitude = descent.Vertical
	if st.AltFt <= 0 {
		st.AltFt = 0
		st.Attitude = descent.Landed
	}
	for _, ev := range script {
		if ev.alarm != "" && ev.timeSec <= m.t {
			st.Alarms = append(st.Alarms, descent.Alarm{Code: ev.alarm, AltFt: ev.altFt})
		}
	}
	return st
}

type frameMsg struct{}

const frameMs = 33.34

func tick() tea.Cmd {
	return tea.Tick(time.Duration(frameMs*1e6)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func (m demoModel) Init() tea.Cmd { return tick() }

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case frameMsg:
		m.tick++
		m.advance(frameMs)
		return m, tick()
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if r, ok := keyRune(msg); ok {
			switch r {
			case 'q':
				return m, tea.Quit
			case '[':
				if m.scale > 1.25 {
					m.scale /= 2
				}
			case ']':
				if m.scale < 320 {
					m.scale *= 2
				}
			case 'r':
				if m.scale == 1 {
					m.scale = defaultScale
				} else {
					m.scale = 1
				}
			case '.':
				m.paused = !m.paused
			}
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

func (m demoModel) View() tea.View {
	dim := "\x1b[38;5;240m"
	reset := "\x1b[0m"
	status := fmt.Sprintf("%.0f× time · [ ] speed · [r] realtime · [.] pause · [q] quit", m.scale)
	if m.paused {
		status = "PAUSED · " + status
	}
	v := tea.NewView(descent.Render(m.state()) + "\n" + dim + status + reset + "\n")
	v.AltScreen = true
	return v
}

func main() {
	if _, err := termreset.Run(newDemoModel()); err != nil {
		fmt.Fprintln(os.Stderr, "lander:", err)
		os.Exit(1)
	}
}
