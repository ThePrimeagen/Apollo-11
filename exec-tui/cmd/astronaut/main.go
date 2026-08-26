// astronaut: the moonwalk loop. An original NES-styled astronaut —
// big helmet, gold visor, life-support pack — runs in from the left
// on the classic three-frame stride, hops a joy jump, leaps onto the
// flagpole, slides down it on two alternating grips, and stands at
// the base before the loop restarts.
//
//	space / enter / p   replay from the top
//	q / ctrl+c          quit
//
//	go run ./cmd/astronaut
//	go run ./cmd/astronaut -seconds 12
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW   = 72
	defaultH   = 26
	frameMs    = 1000.0 / 30
	statusRows = 1
)

// groundedY is the sprite top row that parks the boots on the floor.
func groundedY(stageH int) int { return 0 }

// poleCol is the flagpole's stage column.
func poleCol(stageW int) int { return 0 }

// timelineAt is the whole choreography as a pure function of time:
// which pose plays and where its sprite's top-left sits on the stage.
func timelineAt(stageW, stageH int, t float64) (pose sprite.Heading, x, y int) {
	return "", 0, 0
}

// cycleSeconds is how long one full loop of the route takes.
func cycleSeconds(stageW, stageH int) float64 { return 1 }

type model struct {
	w, h    int
	clock   float64
	seconds float64
	atlas   *sprite.Atlas
}

func newModel(seconds float64) (model, error) {
	return model{}, fmt.Errorf("astronaut: not implemented")
}

type frameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func (m model) Init() tea.Cmd { return tick() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	return v
}

func forcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}

func main() {
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := newModel(*seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "astronaut:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "astronaut:", err)
		os.Exit(1)
	}
}
