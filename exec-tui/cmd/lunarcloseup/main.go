// lunarcloseup: the lunar lander close-up screenplay — the composable
// one-scene bill from shows/lunarcloseup, a copy of the premiere's
// arrival. The house opens on "Lunar Lander Close-Up": three seconds
// of drifting sky, then the zoomed-in Apollo craft slides in from the
// right wing over a starfield that translates with it — hull only,
// cold engine — parks and bobbles at center stage. Space on the last
// scene ends the show — nothing left.
//
//	space     next scene (past the last one, the show ends)
//	q         quit
//
//	go run ./cmd/lunarcloseup
//	go run ./cmd/lunarcloseup -seconds 15
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
	"github.com/theprimeagen/apollo-11/exec-tui/shows/lunarcloseup"

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

// forcedColorProfile mirrors the premiere: CLICOLOR_FORCE keeps the
// colors alive in detached ptys (tmux capture, CI).
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
	play := screenplay.Compose(lunarcloseup.Bill())
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

// cut advances the bill; past the last scene there is nothing left,
// so the show ends.
func (m model) cut() (tea.Model, tea.Cmd) {
	if m.play.Next() {
		return m, nil
	}
	m.play.Stop()
	return m, tea.Quit
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		// The screen tracks the sky: everything above the status line.
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
		switch {
		case msg.String() == "ctrl+c":
			m.play.Stop()
			return m, tea.Quit
		case msg.Code == tea.KeySpace:
			return m.cut()
		default:
			if rs := []rune(msg.Text); len(rs) == 1 {
				switch rs[0] {
				case 'q':
					m.play.Stop()
					return m, tea.Quit
				case ' ':
					return m.cut()
				}
			}
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
	status := fmt.Sprintf(" scene %d/%d — %s   space next scene · q quit",
		m.play.SceneIndex()+1, m.play.Len(), m.play.CurrentName())
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
	skyPath := flag.String("stars", "components/stars/config.json",
		"sky config JSON (adjuststars); a missing file keeps the stock sky")
	flag.Parse()
	if err := applySky(*skyPath); err != nil {
		fmt.Fprintln(os.Stderr, "lunarcloseup:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(newModel(*seconds), opts...); err != nil {
		fmt.Fprintln(os.Stderr, "lunarcloseup:", err)
		os.Exit(1)
	}
}
