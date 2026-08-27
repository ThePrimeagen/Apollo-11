// america: the portable America scene from scenes/america. The house
// opens on pure black; the full-screened American flag fades in
// fast, and once it is fully in, the very large bald eagle crosses
// the stage right to left with the flag flying beneath it and this
// scene's own armed composite — stock one shotgun on the leading
// talon. After the flyover the flag flies alone.
//
// Eleven live knobs retune the scene, the same way the landing runner
// tunes: flag fade, eagle delay, eagle cross (the eagle's speed),
// eagle start / eagle end (where the flight begins and ends, as
// fractions of the full off-right-to-off-left span), left/right on
// (whether each talon carries a gun), left/right shots (how many
// shells that gun fires across one crossing) and left/right aim
// (which of the eight compass points the barrel faces). The time
// knobs step 50ms, the path knobs 0.05 of the span, the on knobs
// flip, the shots one shell, the aims one compass point with wrap.
// Play rebuilds from the current knobs; s saves them to
// scenes/america/config.json.
//
//	p / enter / space   play from the top
//	j / k               select knob
//	h / l               nudge the knob down / up one step
//	s                   save knobs to scenes/america/config.json
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

	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/america"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW   = 72
	defaultH   = 28
	minW       = 10
	minH       = 4
	frameMs    = 1000.0 / 30
	statusRows = 1 + int(america.KnobCount)
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
	cursor  america.Knob
	seconds float64
	elapsed float64
	path    string
	note    string
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
		screen:  screenplay.NewScreen(defaultW, defaultH-statusRows),
		seconds: seconds,
		path:    america.DefaultConfigPath,
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

// replay is Stop then Start: a fresh cast from the current knobs,
// back to black.
func (m model) replay() model {
	m.show.Stop()
	m.show.Start()
	return m
}

// move walks the knob cursor with wrap.
func (m model) move(delta int) model {
	n := int(america.KnobCount)
	m.cursor = america.Knob((int(m.cursor) + delta%n + n) % n)
	return m
}

// save writes the knobs to the config path and makes them the Active
// timing, so the launcher's next America plays them too.
func (m model) save() model {
	if m.path == "" {
		m.note = "no config path"
		return m
	}
	if err := m.show.Cfg.Save(m.path); err != nil {
		m.note = "save failed: " + err.Error()
		return m
	}
	if err := america.Use(m.show.Cfg); err != nil {
		m.note = "save failed: " + err.Error()
		return m
	}
	m.note = "saved"
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
		case "j", "down":
			return m.move(1), nil
		case "k", "up":
			return m.move(-1), nil
		case "l", "right":
			m.show.Cfg.Nudge(m.cursor, 1)
			return m, nil
		case "h", "left":
			m.show.Cfg.Nudge(m.cursor, -1)
			return m, nil
		case "s":
			return m.save(), nil
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
	body := strings.Join(stage, "\n") + "\n" + strings.Join(m.status(w), "\n")
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

func (m model) status(w int) []string {
	dim := "\x1b[38;5;240m"
	hot := "\x1b[38;5;214m"
	reset := "\x1b[0m"
	help := dim + pad(" America   p replay  j/k select  h/l adjust  s save  q quit", w) + reset
	if m.note != "" {
		help = dim + pad(" "+m.note, w) + reset
	}
	rows := []string{help}
	for i := america.Knob(0); i < america.KnobCount; i++ {
		marker, color := "  ", dim
		if i == m.cursor {
			marker, color = "> ", hot
		}
		rows = append(rows, color+pad(fmt.Sprintf("%s%-11s %s", marker, america.KnobLabel(i), m.show.Cfg.Display(i)), w)+reset)
	}
	return rows
}

func pad(s string, w int) string {
	// strip the width of ANSI so the pad matches the cell count
	plain := s
	for _, seq := range []string{"\x1b[38;5;240m", "\x1b[38;5;214m", "\x1b[0m"} {
		plain = strings.ReplaceAll(plain, seq, "")
	}
	r := []rune(plain)
	if len(r) >= w {
		return string(r[:w])
	}
	return s + strings.Repeat(" ", w-len(r))
}

func main() {
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	cfgPath := flag.String("config", america.DefaultConfigPath,
		"America timing JSON; a missing file keeps the stock knobs")
	flag.Parse()
	cfgFile := menu.Resolve(*cfgPath)
	c, err := america.LoadOrDefault(cfgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "america:", err)
		os.Exit(1)
	}
	if err := america.Use(c); err != nil {
		fmt.Fprintln(os.Stderr, "america:", err)
		os.Exit(1)
	}
	m := newModel(*seconds)
	m.path = cfgFile
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "america:", err)
		os.Exit(1)
	}
}
