// astronaut: the moonwalk tuner. Plays the scenes/moonwalk show — the
// crate climb, the pole-top landing, the American flag rising as he
// slides, and the closing camera pan to the lunar rover — with every
// scene knob adjustable live.
//
//	j / k               select a knob
//	h / l               nudge it down / up
//	s                   save knobs to scenes/moonwalk/config.json
//	space / enter / p   replay from the top
//	q / ctrl+c          quit
//
//	go run ./cmd/astronaut
//	go run ./cmd/astronaut -seconds 20
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW = 84
	defaultH = 30
	frameMs  = 1000.0 / 30
)

type model struct {
	w, h    int
	clock   float64
	seconds float64
	atlas   *sprite.Atlas
	cfg     moonwalk.Config
	knob    moonwalk.Knob
	path    string
	note    string
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
