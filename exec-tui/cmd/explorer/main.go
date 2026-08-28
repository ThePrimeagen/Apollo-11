// explorer: the Big E scene from scenes/explorer — the moon-sized IE
// logo under blinking stars, plus one shooting star — and the editable
// screen for its knobs. j/k select a knob, h/l tune it LIVE (the sky
// and the flying meteor read the knobs on the next frame): min/max
// cycle move 250ms at a time, min/max fade 50ms, every twinkle knob
// railed and no pair crossing, and the ten star knobs walk the
// shooting-star tuner's own steps. The scene opens on the shooting-
// star scene's saved knobs when its own config carries no star
// section, and s pins the scene's copy. q quits.
//
//	p / enter / space   play from the top
//	j / k               select knob
//	h / l               tune it down / up (live)
//	s                   save knobs to scenes/explorer/config.json
//	q                   quit
//
//	go run ./cmd/explorer
//	go run ./cmd/explorer -seconds 15
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
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/explorer"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/shootingstar"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW   = 72
	defaultH   = 38
	minW       = 10
	minH       = 4
	frameMs    = 1000.0 / 30
	statusRows = 1 + int(explorer.KnobCount)
)

func forcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}

type model struct {
	w, h    int
	show    *explorer.Show
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	cursor  explorer.Knob
	seconds float64
	elapsed float64
	path    string
	note    string
}

func newModel(seconds float64) model {
	show := explorer.New(nil)
	play := screenplay.New(screenplay.Entry{Name: "explorer", Scene: show})
	play.Start()
	return model{
		w:       defaultW,
		h:       defaultH,
		show:    show,
		play:    play,
		screen:  screenplay.NewScreen(defaultW, defaultH-statusRows),
		seconds: seconds,
		path:    explorer.DefaultConfigPath,
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
	n := int(explorer.KnobCount)
	m.cursor = explorer.Knob((int(m.cursor) + delta%n + n) % n)
	return m
}

// nudge walks the selected knob and pushes the result onto the stage
// at once — the breathing and the flying meteor retune live, no
// replay needed. A nudged knob is valid by construction, so the push
// cannot be refused.
func (m model) nudge(dir int) model {
	m.show.Cfg.Nudge(m.cursor, dir)
	m.show.Retune()
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
	if err := explorer.Use(m.show.Cfg); err != nil {
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

// knobValue paints one knob's reading: seconds for the twinkle
// ranges, the shooting-star tuner's own units for the star knobs.
func (m model) knobValue(i explorer.Knob) string {
	c := m.show.Cfg
	switch i {
	case explorer.KnobStarSize:
		return fmt.Sprintf("%7d", c.Star.Size)
	case explorer.KnobStarRandomSize:
		if c.Star.RandomSize {
			return "     on"
		}
		return "    off"
	case explorer.KnobStarSpeed:
		return fmt.Sprintf("%7.1f", c.Star.Speed)
	case explorer.KnobStarCount:
		return fmt.Sprintf("%7d", c.Star.Count)
	case explorer.KnobStarPeriod:
		return fmt.Sprintf("%6.3fs", c.Star.Period)
	case explorer.KnobStarMinLife, explorer.KnobStarMaxLife:
		return fmt.Sprintf("%6.2fs", c.Value(i))
	case explorer.KnobStarNozzle:
		return fmt.Sprintf("%7.1f", c.Star.Nozzle)
	case explorer.KnobStarPeak:
		return fmt.Sprintf("%7.1f", c.Star.Peak)
	case explorer.KnobStarTaper:
		return fmt.Sprintf("%7.2f", c.Star.Taper)
	default:
		return fmt.Sprintf("%6.2fs", c.Value(i))
	}
}

func (m model) status(w int) []string {
	dim := "\x1b[38;5;240m"
	hot := "\x1b[38;5;214m"
	reset := "\x1b[0m"
	help := dim + pad("big e   p play  j/k select  h/l tune (live)  s save  q quit", w) + reset
	if m.note != "" {
		help = dim + pad(m.note, w) + reset
	}
	rows := []string{help}
	for i := explorer.Knob(0); i < explorer.KnobCount; i++ {
		marker, color := "  ", dim
		if i == m.cursor {
			marker, color = "> ", hot
		}
		rows = append(rows, color+pad(fmt.Sprintf("%s%-16s %s", marker, explorer.KnobLabel(i), m.knobValue(i)), w)+reset)
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
	cfgPath := flag.String("config", explorer.DefaultConfigPath,
		"explorer knobs JSON; a missing file keeps the stock knobs")
	starPath := flag.String("star", shootingstar.DefaultConfigPath,
		"shooting-star knobs JSON the Big E inherits while its own config carries no star section")
	flag.Parse()
	cfgFile := menu.Resolve(*cfgPath)
	star, err := shootingstar.LoadOrDefault(menu.Resolve(*starPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, "explorer:", err)
		os.Exit(1)
	}
	c, err := explorer.LoadOrInherit(cfgFile, star)
	if err != nil {
		fmt.Fprintln(os.Stderr, "explorer:", err)
		os.Exit(1)
	}
	if err := explorer.Use(c); err != nil {
		fmt.Fprintln(os.Stderr, "explorer:", err)
		os.Exit(1)
	}
	m := newModel(*seconds)
	m.path = cfgFile
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "explorer:", err)
		os.Exit(1)
	}
}
