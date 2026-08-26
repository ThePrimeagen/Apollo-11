// landing: the portable landing scene from scenes/landing. A huge moon
// horizon as a colored floor and the north-facing lander coming down
// onto it — booster full, then ¾, ½, ¼, then off on the pad. Pad dust
// starts at DustStart and blows for DustRun. Play rebuilds from the
// current knobs; j/k select a knob, h/l walk it 50ms. q quits.
//
//	p / enter / space   play from the top
//	j / k               select knob
//	h / l               −50ms / +50ms
//	s                   save knobs to scenes/landing/config.json
//	q                   quit
//
//	go run ./cmd/landing
//	go run ./cmd/landing -seconds 15
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW   = 72
	defaultH   = 28
	minW       = 10
	minH       = 4
	frameMs    = 1000.0 / 30
	statusRows = 4
)

// applySky loads a tuned sky config and makes it the active sky. A
// missing file quietly keeps the stock sky; a broken file is an error
// worth stopping for.
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

// applyPuff loads a tuned dust config and makes it the active kick.
// A missing file quietly keeps the stock puff; a broken file is an
// error worth stopping for.
func applyPuff(path string) error {
	c, err := dust.LoadPuff(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return dust.UsePuff(c)
}

func forcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}

type model struct {
	w, h    int
	show    *landing.Show
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	cursor  landing.Knob
	seconds float64
	elapsed float64
	path    string
	note    string
}

func newModel(seconds float64) model {
	show := landing.New(nil)
	play := screenplay.New(screenplay.Entry{Name: "landing", Scene: show})
	play.Start()
	return model{
		w:       defaultW,
		h:       defaultH,
		show:    show,
		play:    play,
		screen:  screenplay.NewScreen(defaultW, defaultH-statusRows),
		seconds: seconds,
		path:    landing.DefaultConfigPath,
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
	return m
}

func (m model) move(delta int) model {
	n := int(landing.KnobCount)
	m.cursor = landing.Knob((int(m.cursor) + delta%n + n) % n)
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
	if err := landing.Use(m.show.Cfg); err != nil {
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
	help := dim + pad("landing   p play  j/k select  h/l ±50ms  s save  q quit", w) + reset
	if m.note != "" {
		help = dim + pad(m.note, w) + reset
	}
	rows := []string{help}
	for i, label := range []string{"land", "dust start", "dust run"} {
		sec := m.show.Cfg.LandSeconds
		switch landing.Knob(i) {
		case landing.KnobDustStart:
			sec = m.show.Cfg.DustStart
		case landing.KnobDustRun:
			sec = m.show.Cfg.DustRun
		}
		marker, color := "  ", dim
		if landing.Knob(i) == m.cursor {
			marker, color = "> ", hot
		}
		rows = append(rows, color+pad(fmt.Sprintf("%s%-11s %6.3fs", marker, label, sec), w)+reset)
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
	skyPath := flag.String("stars", "components/stars/config.json",
		"sky config JSON (adjuststars); a missing file keeps the stock sky")
	puffPath := flag.String("dust", "components/dust/config.json",
		"dust puff JSON (adjustdust); a missing file keeps the stock kick")
	cfgPath := flag.String("config", landing.DefaultConfigPath,
		"landing timing JSON; a missing file keeps the stock knobs")
	flag.Parse()
	skyFile := menu.Resolve(*skyPath)
	puffFile := menu.Resolve(*puffPath)
	cfgFile := menu.Resolve(*cfgPath)
	if err := applySky(skyFile); err != nil {
		fmt.Fprintln(os.Stderr, "landing:", err)
		os.Exit(1)
	}
	if err := applyPuff(puffFile); err != nil {
		fmt.Fprintln(os.Stderr, "landing:", err)
		os.Exit(1)
	}
	c, err := landing.LoadOrDefault(cfgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "landing:", err)
		os.Exit(1)
	}
	if err := landing.Use(c); err != nil {
		fmt.Fprintln(os.Stderr, "landing:", err)
		os.Exit(1)
	}
	m := newModel(*seconds)
	m.path = cfgFile
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "landing:", err)
		os.Exit(1)
	}
}
