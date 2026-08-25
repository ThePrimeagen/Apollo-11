// screenplay-lab: a two-scene premiere for the screenplay component.
//
// Scene 1, "arrival": the Apollo craft slides in from the right wing
// over a drifting starfield, parks at center stage, and bobbles on a
// slow one-cell sine. Scene 2, "the end": the height-5 banner card,
// centered under the same sky.
//
//	space     cut to the next scene
//	q         quit
//
//	go run .
//	go run . -seconds 30
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/stars-lab/stars"

	"github.com/theprimeagen/apollo-11/screenplay-lab/cast"
	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

const (
	defaultW = 72
	defaultH = 28
	frameMs  = 1000.0 / 30
)

// premiere is the bill: arrival, then the end card. Each scene's cast
// is assembled when its curtain rises, not before.
func premiere() *screenplay.Screenplay {
	_ = stars.Drift
	_ = cast.NewShip
	return screenplay.New()
}

type model struct {
	w, h    int
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	seconds float64
	elapsed float64
}

func newModel(seconds float64) model {
	return model{}
}

type frameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	return v
}

func pad(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}

func main() {
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m := newModel(*seconds)
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "screenplay-lab:", err)
		os.Exit(1)
	}
}
