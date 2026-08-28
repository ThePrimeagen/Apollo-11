// coreset2: the Core Sets Two scene from scenes/coreset2, standalone.
// The house opens on scene one's held frame — the PRIORITY word over
// its fifteen bits — then the roster lands (six jobs, six different
// priorities), the stage sweeps clean, the EJSCAN loop reveals as
// code, and the scan plays twice: five core sets with the full word
// math beside each (priority + VAC address, 000 for NOVAC) walked
// comparison by comparison until the third box down is SELECTED, then
// the redo with three SERVICER copies at PRIO 20 climbing the real
// VAC addresses — the newest copy always wins, the old ones starve as
// stubs. The scene plays itself and holds on the lesson.
//
//	space / p / enter   replay from the top
//	q / ctrl+c          quit
//
//	go run ./cmd/coreset2
//	go run ./cmd/coreset2 -seconds 45
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset2"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW   = 100
	defaultH   = 30
	minW       = 10
	minH       = 4
	frameMs    = 1000.0 / 30
	statusRows = 1
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
	show    *coreset2.Show
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	seconds float64
	elapsed float64
}

func newModel(seconds float64) model {
	show := coreset2.New()
	play := screenplay.New(screenplay.Entry{Name: "Core Sets Two", Scene: show})
	play.Start()
	return model{
		w:       defaultW,
		h:       defaultH,
		show:    show,
		play:    play,
		screen:  screenplay.NewScreen(defaultW, defaultH-statusRows),
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

// replay is Stop then Start: a fresh director, back to the pickup.
func (m model) replay() model {
	m.show.Stop()
	m.show.Start()
	return m
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.screen.Resize(max(m.w, minW), max(m.h-statusRows, minH))
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
	body := strings.Join(stage, "\n") + "\n" + m.status(w)
	lines := strings.Split(body, "\n")
	winH := m.h
	if winH < 1 {
		winH = defaultH
	}
	for len(lines) < winH {
		lines = append(lines, "")
	}
	if len(lines) > winH {
		lines = lines[:winH]
	}
	v := tea.NewView(strings.Join(lines, "\n"))
	v.AltScreen = true
	return v
}

func (m model) status(w int) string {
	dim := "\x1b[38;5;240m"
	reset := "\x1b[0m"
	return dim + pad(" Core Sets Two   space replay   q quit", w) + reset
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
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "coreset2:", err)
		os.Exit(1)
	}
}
