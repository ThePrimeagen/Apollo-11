// lander-lab: an exploratory ASCII lunar-lander view of the Apollo 11
// powered descent. Step through the flight with h/l — every throttle event,
// the yaw, radar lock, Buzz's monitor, all five program alarms at their real
// times and altitudes, ATT HOLD, P66, touchdown. q quits.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/lander-lab/lander"
)

type scriptEvent struct {
	timeSec  float64
	altFt    float64
	velFps   float64
	phase    string
	attitude lander.Attitude
	alarm    string // "" when the event is not an executive overflow
	caption  string
}

// script is the flight, PDI-relative (times/altitudes per Cherry's event log
// and Eyles; velocities approximate).
var script = []scriptEvent{
	{0, 49971, 5560, "P63 BRAKING", lander.Horizontal, "", "PDI — DPS ignition at 10%"},
	{26, 48000, 5460, "P63 BRAKING", lander.Horizontal, "", "throttle to full — guidance enabled"},
	{232, 42426, 3366, "P63 BRAKING", lander.Horizontal, "", "yaw maneuver — windows up"},
	{274, 39000, 2745, "P63 BRAKING", lander.Horizontal, "", "landing radar: data good"},
	{304, 35706, 2521, "P63 BRAKING", lander.Horizontal, "", "Buzz keys V16N68 — DELTAH monitor"},
	{316, 33500, 2280, "P63 BRAKING", lander.Horizontal, "1202", "PROG ALARM 1202 — no core sets"},
	{358, 30900, 1826, "P63 BRAKING", lander.Horizontal, "1202", "1202 again — monitor re-keyed"},
	{384, 23393, 1481, "P63 BRAKING", lander.Horizontal, "", "throttle down — right on time"},
	{506, 7400, 506, "P64 APPROACH", lander.Tilted, "", "high gate — pitch over, LPD active"},
	{552, 3000, 130, "P64 APPROACH", lander.Tilted, "1201", "PROG ALARM 1201 — no VAC areas"},
	{578, 2000, 90, "P64 APPROACH", lander.Tilted, "1202", "PROG ALARM 1202"},
	{594, 770, 60, "P64 APPROACH", lander.Tilted, "1202", "PROG ALARM 1202 — 770 ft"},
	{603, 650, 50, "P64 APPROACH", lander.Tilted, "", "Armstrong: AUTO → ATT HOLD"},
	{615, 430, 30, "P66 LANDING", lander.Vertical, "", "P66 — rate of descent mode"},
	{757, 0, 0, "P66 LANDED", lander.Landed, "", "CONTACT LIGHT — the Eagle has landed"},
}

type demoModel struct {
	idx int
}

func newDemoModel() demoModel { return demoModel{} }

// state assembles the lander view for the current step, with every alarm
// reached so far as a persistent marker.
func (m demoModel) state() lander.State {
	ev := script[m.idx]
	st := lander.State{
		AltFt: ev.altFt, VelFps: ev.velFps, TimeSec: ev.timeSec,
		Phase: ev.phase, Attitude: ev.attitude, Event: ev.caption,
	}
	for i := 0; i <= m.idx; i++ {
		if script[i].alarm != "" {
			st.Alarms = append(st.Alarms, lander.Alarm{Code: script[i].alarm, AltFt: script[i].altFt})
		}
	}
	return st
}

func (m demoModel) Init() tea.Cmd { return nil }

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyLeft:
			if m.idx > 0 {
				m.idx--
			}
		case tea.KeyRight:
			if m.idx < len(script)-1 {
				m.idx++
			}
		}
		if len(msg.Runes) == 1 {
			switch msg.Runes[0] {
			case 'q':
				return m, tea.Quit
			case 'h':
				if m.idx > 0 {
					m.idx--
				}
			case 'l':
				if m.idx < len(script)-1 {
					m.idx++
				}
			}
		}
	}
	return m, nil
}

func (m demoModel) View() string {
	dim := "\x1b[38;5;240m"
	reset := "\x1b[0m"
	return lander.Render(m.state()) + "\n" +
		dim + fmt.Sprintf("step %d/%d · [h/l] step · [q] quit", m.idx+1, len(script)) + reset + "\n"
}

func main() {
	if _, err := tea.NewProgram(newDemoModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "lander-lab:", err)
		os.Exit(1)
	}
}
