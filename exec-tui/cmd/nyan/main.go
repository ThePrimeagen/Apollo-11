// nyan: the pop-tart cat demo. A drifting starfield, a nyan cat
// flying left-to-right with a live rainbow particle trail. The trail
// reads the same JSON the particle editor saves.
//
//	q         quit
//
//	go run ./cmd/nyan
//	go run ./cmd/nyan -seconds 20
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/nyan"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW = 80
	defaultH = 24
	minW     = 10
	minH     = 4
	frameMs  = 1000.0 / 30
)

func bill() *screenplay.Screenplay {
	return screenplay.New(screenplay.Entry{
		Name: "nyan",
		Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield(),
					nyan.NewCat(11),
				}
			},
		},
	})
}

func applyTrail(path string) error {
	c, err := nyan.LoadTrail(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nyan.UseTrail(c)
}

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
	play    *screenplay.Screenplay
	screen  *screenplay.Screen
	seconds float64
	elapsed float64
}

func newModel(seconds float64) model {
	play := bill()
	play.Start()
	return model{
		w:       defaultW,
		h:       defaultH,
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

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
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
		if msg.String() == "ctrl+c" {
			m.play.Stop()
			return m, tea.Quit
		}
		if rs := []rune(msg.Text); len(rs) == 1 && rs[0] == 'q' {
			m.play.Stop()
			return m, tea.Quit
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
	status := " nyan   q quit"
	dim := "\x1b[38;5;240m"
	reset := "\x1b[0m"
	body := strings.Join(sky, "\n") + "\n" + dim + pad(status, w) + reset
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
	trailPath := flag.String("config", nyanDefaultConfig, "particle trail JSON (adjustparticle)")
	skyPath := flag.String("stars", "components/stars/config.json",
		"sky config JSON (adjuststars); a missing file keeps the stock sky")
	flag.Parse()
	if err := applyTrail(*trailPath); err != nil {
		fmt.Fprintln(os.Stderr, "nyan:", err)
		os.Exit(1)
	}
	if err := applySky(*skyPath); err != nil {
		fmt.Fprintln(os.Stderr, "nyan:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(newModel(*seconds), opts...); err != nil {
		fmt.Fprintln(os.Stderr, "nyan:", err)
		os.Exit(1)
	}
}

const nyanDefaultConfig = "components/nyan/config.json"
