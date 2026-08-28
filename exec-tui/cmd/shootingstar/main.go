// shootingstar: the shooting-star tuner from scenes/shootingstar — a
// larger star with a persist-particle trail falling right-to-left
// (high right to low left). Path can still walk a circle or a square
// so the tail is readable. Live knobs: path, size, random
// size, speed, count, period, min/max life, nozzle, peak, taper. The scene itself
// (the viewer, the bill) always flies that right-to-left fall.
// Play rebuilds from the current knobs. q quits.
//
//	p / enter / space   play from the top
//	j / k               select knob
//	h / l               tune it down / up (live)
//	s                   save knobs to scenes/shootingstar/config.json
//	q                   quit
//
//	go run ./cmd/shootingstar
//	go run ./cmd/shootingstar -seconds 15
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
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/shootingstar"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW   = 72
	defaultH   = 28
	minW       = 10
	minH       = 4
	frameMs    = 1000.0 / 30
	statusRows = 1 + int(shootingstar.KnobCount)
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
	show    *shootingstar.Show
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	cursor  shootingstar.Knob
	seconds float64
	elapsed float64
	path    string
	note    string
}

func newModel(seconds float64, random bool) model {
	var show *shootingstar.Show
	if random {
		show = shootingstar.New(nil)
	} else {
		show = shootingstar.NewPreview(nil)
	}
	play := screenplay.New(screenplay.Entry{Name: "shootingstar", Scene: show})
	play.Start()
	return model{
		w:       defaultW,
		h:       defaultH,
		show:    show,
		play:    play,
		screen:  screenplay.NewScreen(defaultW, defaultH-statusRows),
		seconds: seconds,
		path:    shootingstar.DefaultConfigPath,
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
	n := int(shootingstar.KnobCount)
	m.cursor = shootingstar.Knob((int(m.cursor) + delta%n + n) % n)
	return m
}

func (m model) nudge(dir int) model {
	m.show.Cfg.Nudge(m.cursor, dir)
	_ = shootingstar.Use(m.show.Cfg)
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
	if err := shootingstar.Use(m.show.Cfg); err != nil {
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
			return m.nudge(1), nil
		case "h", "left":
			return m.nudge(-1), nil
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

func (m model) knobValue(i shootingstar.Knob) string {
	c := m.show.Cfg
	switch i {
	case shootingstar.KnobPath:
		return fmt.Sprintf("%8s", c.Path)
	case shootingstar.KnobSize:
		return fmt.Sprintf("%8d", c.Size)
	case shootingstar.KnobRandomSize:
		if c.RandomSize {
			return "      on"
		}
		return "     off"
	case shootingstar.KnobSpeed:
		return fmt.Sprintf("%7.1f", c.Speed)
	case shootingstar.KnobSpawn:
		return fmt.Sprintf("%8d", c.Count)
	case shootingstar.KnobPeriod:
		return fmt.Sprintf("%7.3fs", c.Period)
	case shootingstar.KnobMinLife, shootingstar.KnobMaxLife:
		return fmt.Sprintf("%7.2fs", c.Value(i))
	case shootingstar.KnobNozzle:
		return fmt.Sprintf("%8.1f", c.Nozzle)
	case shootingstar.KnobPeak:
		return fmt.Sprintf("%8.1f", c.Peak)
	case shootingstar.KnobTaper:
		return fmt.Sprintf("%8.2f", c.Taper)
	default:
		return ""
	}
}

func (m model) status(w int) []string {
	dim := "\x1b[38;5;240m"
	hot := "\x1b[38;5;214m"
	reset := "\x1b[0m"
	help := dim + pad("shooting star   p play  j/k select  h/l tune  s save  q quit", w) + reset
	if m.note != "" {
		help = dim + pad(m.note, w) + reset
	}
	rows := []string{help}
	for i := shootingstar.Knob(0); i < shootingstar.KnobCount; i++ {
		marker, color := "  ", dim
		if i == m.cursor {
			marker, color = "> ", hot
		}
		rows = append(rows, color+pad(fmt.Sprintf("%s%-12s %s", marker, shootingstar.KnobLabel(i), m.knobValue(i)), w)+reset)
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
	random := flag.Bool("random", false, "fly the scene fall (right-to-left) even if the knobs say circle or square")
	skyPath := flag.String("stars", "components/stars/config.json",
		"sky config JSON (adjuststars); a missing file keeps the stock sky")
	cfgPath := flag.String("config", shootingstar.DefaultConfigPath,
		"shooting-star knobs JSON; a missing file keeps the stock knobs")
	flag.Parse()
	skyFile := menu.Resolve(*skyPath)
	cfgFile := menu.Resolve(*cfgPath)
	if err := applySky(skyFile); err != nil {
		fmt.Fprintln(os.Stderr, "shootingstar:", err)
		os.Exit(1)
	}
	c, err := shootingstar.LoadOrDefault(cfgFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "shootingstar:", err)
		os.Exit(1)
	}
	if err := shootingstar.Use(c); err != nil {
		fmt.Fprintln(os.Stderr, "shootingstar:", err)
		os.Exit(1)
	}
	m := newModel(*seconds, *random)
	m.path = cfgFile
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "shootingstar:", err)
		os.Exit(1)
	}
}
