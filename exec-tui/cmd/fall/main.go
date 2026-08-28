// fall: the portable spacelander fall from scenes/fall. The north-
// facing lander dropping top to bottom under twinkling stars. One
// live knob: drop duration (±50ms). Play rebuilds from the current
// knobs; j/k select, h/l walk it. q quits.
//
//	p / enter / space   play from the top
//	j / k               select knob
//	h / l               −50ms / +50ms
//	s                   save knobs to scenes/fall/config.json
//	q                   quit
//
//	go run ./cmd/fall
//	go run ./cmd/fall -seconds 15
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/fall"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW   = 72
	defaultH   = 28
	minW       = 10
	minH       = 4
	frameMs    = 1000.0 / 30
	statusRows = 1 + int(fall.KnobCount)
)

func applySky(path string) error {
	c, err := stars.LoadSky(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return stars.UseSky(c)
}

func forcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}

type model struct {
	w, h    int
	show    *fall.Show
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	cursor  fall.Knob
	seconds float64
	elapsed float64
	path    string
	note    string
}

func newModel(seconds float64) model {
	show := fall.New(nil)
	play := screenplay.New(screenplay.Entry{Name: "fall", Scene: show})
	play.Start()
	return model{
		w:       defaultW,
		h:       defaultH,
		show:    show,
		play:    play,
		screen:  screenplay.NewScreen(defaultW, defaultH-statusRows),
		seconds: seconds,
		path:    fall.DefaultConfigPath,
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

func (m model) replay() model {
	m.show.Stop()
	m.show.Start()
	if m.screen != nil {
		m.play.Render(m.screen)
	}
	return m
}

func (m model) move(delta int) model {
	n := int(fall.KnobCount)
	m.cursor = fall.Knob((int(m.cursor) + delta%n + n) % n)
	return m
}

func (m model) save() model {
	if m.path == "" {
		m.note = "no config path"
		return m
	}
	if err := m.show.Cfg.Save(m.path); err != nil {
		m.note = "save failed: " + err.Error()
		return m
	}
	if err := fall.Use(m.show.Cfg); err != nil {
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
	sky := strings.Split(m.screen.Render(), "\n")
	for len(sky) < h {
		sky = append(sky, "")
	}
	body := strings.Join(sky, "\n") + "\n" + strings.Join(m.status(w), "\n")
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
	help := dim + pad("fall   p play  j/k select  h/l ±50ms  s save  q quit", w) + reset
	if m.note != "" {
		help = dim + pad(m.note, w) + reset
	}
	rows := []string{help}
	for i := fall.Knob(0); i < fall.KnobCount; i++ {
		marker, color := "  ", dim
		if i == m.cursor {
			marker, color = "> ", hot
		}
		rows = append(rows, color+pad(fmt.Sprintf("%s%-11s %6.3fs", marker, fall.KnobLabel(i), m.show.Cfg.Value(i)), w)+reset)
	}
	return rows
}

func pad(s string, w int) string {
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
	skyPath := flag.String("stars", "components/stars/config.json",
		"sky config JSON (adjuststars); a missing file keeps the stock sky")
	cfgPath := flag.String("config", fall.DefaultConfigPath,
		"fall timing JSON; a missing file keeps the stock knobs")
	flag.Parse()
	skyFile := menu.Resolve(*skyPath)
	cfgFile := menu.Resolve(*cfgPath)
	if err := applySky(skyFile); err != nil {
		fmt.Fprintln(os.Stderr, "fall:", err)
		os.Exit(1)
	}
	c, err := fall.LoadOrDefault(cfgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fall:", err)
		os.Exit(1)
	}
	if err := fall.Use(c); err != nil {
		fmt.Fprintln(os.Stderr, "fall:", err)
		os.Exit(1)
	}
	m := newModel(*seconds)
	m.path = cfgFile
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "fall:", err)
		os.Exit(1)
	}
}
