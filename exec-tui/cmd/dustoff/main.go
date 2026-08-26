// dustoff: the landing kick-up demo. Something drops, and dust blows
// out of the floor to both sides — two mirrored swirl engines climbing
// 15° above horizontal, leftward and rightward, with a still 8-column
// gap between the nozzles. Half the specks curve out, half sweep one
// full cartoon-wind loop. Heavy dust wears gray shade blocks; the thin
// fringe wears braille with computed dots. The kick reads the same
// JSON the dust-off editor saves.
//
//	q         quit
//
//	go run ./cmd/dustoff
//	go run ./cmd/dustoff -seconds 20
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
		Name: "dust off",
		Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					dust.NewCloud(11),
				}
			},
		},
	})
}

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
	stage := strings.Split(m.screen.Render(), "\n")
	for len(stage) < h {
		stage = append(stage, "")
	}
	status := " dust off   q quit"
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
	puffPath := flag.String("config", dustDefaultConfig,
		"dust puff JSON (adjustdust); a missing file keeps the stock kick")
	flag.Parse()
	if err := applyPuff(*puffPath); err != nil {
		fmt.Fprintln(os.Stderr, "dustoff:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(newModel(*seconds), opts...); err != nil {
		fmt.Fprintln(os.Stderr, "dustoff:", err)
		os.Exit(1)
	}
}

const dustDefaultConfig = "components/dust/config.json"
