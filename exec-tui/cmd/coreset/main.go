// coreset: the Core Set scene from scenes/coreset, standalone — and
// the scene's tuner. The memory unit — the core set panel beside the
// VAC panel — drains away to one survivor, the unnumbered CORE SET
// box glides to the top center, lands, settles, and only then do its
// twelve words build as a bar (MPAC…MPAC+6, MODE, LOC, BANKSET,
// PUSHLOC, PRIORITY); the PRIORITY word then breaks open into its
// fifteen bits: six of priority over nine of VAC address — SERVICER
// at PRIO 20 in VAC1 at 400, OCT 20400. The scene plays itself and
// holds on the bits.
//
// Nine live knobs retune it, the same way the America runner tunes:
// unit hold, fade beat, dissolve, move, settle, word beat, word
// hold, zoom fade and zoom glide — every one in seconds, stepping
// 50ms. Play rebuilds from the current knobs; s saves them to
// scenes/coreset/config.json and installs them as the Active timing.
//
//	space / p / enter   replay from the top
//	j / k               select knob
//	h / l               nudge the knob down / up one step
//	s                   save knobs to scenes/coreset/config.json
//	q / ctrl+c          quit
//
//	go run ./cmd/coreset
//	go run ./cmd/coreset -seconds 30
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
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW   = 100
	defaultH   = 30
	minW       = 10
	minH       = 4
	frameMs    = 1000.0 / 30
	statusRows = 1 + int(coreset.KnobCount)
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
	show    *coreset.Show
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	cursor  coreset.Knob
	seconds float64
	elapsed float64
	path    string
	note    string
}

func newModel(seconds float64) model {
	show := coreset.New()
	play := screenplay.New(screenplay.Entry{Name: "Core Set", Scene: show})
	play.Start()
	return model{
		w:       defaultW,
		h:       defaultH,
		show:    show,
		play:    play,
		screen:  screenplay.NewScreen(defaultW, defaultH-statusRows),
		seconds: seconds,
		path:    coreset.DefaultConfigPath,
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

// replay is Stop then Start: a fresh director from the current knobs,
// back to the memory unit.
func (m model) replay() model {
	m.show.Stop()
	m.show.Start()
	return m
}

// move walks the knob cursor with wrap.
func (m model) move(delta int) model {
	n := int(coreset.KnobCount)
	m.cursor = coreset.Knob((int(m.cursor) + delta%n + n) % n)
	return m
}

// save writes the knobs to the config path and makes them the Active
// timing, so the launcher's next Core Set plays them too.
func (m model) save() model {
	if m.path == "" {
		m.note = "no config path"
		return m
	}
	if err := m.show.Cfg.Save(m.path); err != nil {
		m.note = "save failed: " + err.Error()
		return m
	}
	if err := coreset.Use(m.show.Cfg); err != nil {
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
	help := dim + pad(" Core Set   p replay  j/k select  h/l adjust  s save  q quit", w) + reset
	if m.note != "" {
		help = dim + pad(" "+m.note, w) + reset
	}
	rows := []string{help}
	for i := coreset.Knob(0); i < coreset.KnobCount; i++ {
		marker, color := "  ", dim
		if i == m.cursor {
			marker, color = "> ", hot
		}
		rows = append(rows, color+pad(fmt.Sprintf("%s%-11s %s", marker, coreset.KnobLabel(i), m.show.Cfg.Display(i)), w)+reset)
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
	cfgPath := flag.String("config", coreset.DefaultConfigPath,
		"Core Set timing JSON; a missing file keeps the stock knobs")
	flag.Parse()
	cfgFile := menu.Resolve(*cfgPath)
	c, err := coreset.LoadOrDefault(cfgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "coreset:", err)
		os.Exit(1)
	}
	if err := coreset.Use(c); err != nil {
		fmt.Fprintln(os.Stderr, "coreset:", err)
		os.Exit(1)
	}
	m := newModel(*seconds)
	m.path = cfgFile
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "coreset:", err)
		os.Exit(1)
	}
}
