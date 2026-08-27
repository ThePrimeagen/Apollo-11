// america: the portable America scene from scenes/america. The house
// opens on pure black; the full-screened American flag fades in
// slowly, and once it is fully in, the very large bald eagle crosses
// the stage right to left with the flag flying beneath it. After the
// flyover the flag flies alone. p (or space, or enter) replays from
// the top — back to black. q quits.
//
//	p / enter / space   play from the top
//	q                   quit
//
//	go run ./cmd/america
//	go run ./cmd/america -seconds 25
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/america"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW = 72
	defaultH = 28
	minW     = 10
	minH     = 4
	frameMs  = 1000.0 / 30
)

// forcedColorProfile mirrors the other runners: CLICOLOR_FORCE keeps
// the colors alive in detached ptys (tmux capture, CI).
func forcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}

type model struct {
	w, h    int
	show    *america.Show
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	seconds float64
	elapsed float64
}

func newModel(seconds float64) model {
	show := america.New()
	play := screenplay.New(screenplay.Entry{Name: "America", Scene: show})
	play.Start()
	return model{
		w:       defaultW,
		h:       defaultH,
		show:    show,
		play:    play,
		screen:  screenplay.NewScreen(defaultW, defaultH-1),
		seconds: seconds,
	}
}

type frameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

func (m model) Init() tea.Cmd { return tick() }

// replay is Stop then Start: a fresh cast, back to black.
func (m model) replay() model {
	m.show.Stop()
	m.show.Start()
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		// The screen tracks the stage: everything above the status line.
		m.screen.Resize(max(m.w, minW), max(m.h, minH)-1)
		return m, nil
	case frameMsg:
		dt := frameMs / 1000
		m.elapsed += dt
		m.play.Update(dt)
		if m.seconds > 0 && m.elapsed >= m.seconds {
			m.play.Stop()
			return m, tea.Quit
		}
		return m, tick()
	case tea.KeyPressMsg:
		switch strings.ToLower(msg.String()) {
		case "ctrl+c", "q":
			m.play.Stop()
			return m, tea.Quit
		case " ", "space", "enter", "p":
			return m.replay(), nil
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	m.play.Render(m.screen)
	w, h := m.screen.Size()
	stage := strings.Split(m.screen.Render(), "\n")
	for len(stage) < h {
		stage = append(stage, "")
	}
	status := fmt.Sprintf(" %s — flag fades in, eagle crosses   p replay · q quit",
		m.play.CurrentName())
	dim := "\x1b[38;5;240m"
	reset := "\x1b[0m"
	body := strings.Join(stage, "\n") + "\n" + dim + pad(status, w) + reset
	v := tea.NewView(body)
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
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(newModel(*seconds), opts...); err != nil {
		fmt.Fprintln(os.Stderr, "america:", err)
		os.Exit(1)
	}
}
