// premiere: the four-scene showcase for the screenplay component.
//
// Scene 1, "arrival": three seconds of drifting sky, then the Apollo
// craft slides in from the right wing over a starfield that translates
// with it — every star speeds up on the same ease-out cubic the hull
// flies, so the whole scene rushes left as the ship comes in, then the
// sky settles back into its own drift once the craft parks. Hull only,
// cold engine. Scene 2, "dsky": the
// parked craft stays, the right third of the sky wipes away one column
// at a time (~500ms), and the DSKY docks in that space. Scene 3,
// "descent orbit": the pixelated moon under a wide dotted ring — the
// descent path — and a lone gold marker riding the ring eastward over
// the top of a perfectly still sky: where the craft was, and why it
// flies sideways. Scene 4, "the end": the height-5 banner card,
// centered under the same sky.
//
//	space     cut to the next scene
//	q         quit
//
//	go run .
//	go run . -seconds 30
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dsky"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/components/title"

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

// premiere is the bill: arrival, then the DSKY dock, then the descent
// orbit around the moon, then the end card. Each scene's cast of
// components is assembled when its curtain rises, not before.
func premiere() *screenplay.Screenplay {
	return screenplay.New(
		screenplay.Entry{Name: "arrival", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().SlideIn(lander.FlyInSeconds, lander.BodyCols).Hold(lander.FlyInHoldSeconds),
					lander.NewShip(11).Dark().Hold(lander.FlyInHoldSeconds),
				}
			},
		}},
		screenplay.Entry{Name: "dsky", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Dock(dsky.Width, dsky.WipeSeconds),
					lander.NewShip(11).Dark().Parked(),
					dsky.NewPanel(dsky.MonitorState()),
				}
			},
		}},
		screenplay.Entry{Name: "descent orbit", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield().Still(),
					moon.New(),
				}
			},
		}},
		screenplay.Entry{Name: "the end", Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield(),
					mustTitle("THE END", 5),
				}
			},
		}},
	)
}

// mustTitle fails at boot, not on stage: the bill is static, so a bad
// card is a programming error.
func mustTitle(text string, height int) *title.Title {
	t, err := title.New(text, height)
	if err != nil {
		panic(err)
	}
	return t
}

// applySky loads a tuned sky config (the file adjuststars saves) and
// makes it the active sky. A missing file quietly keeps the stock sky
// — the premiere just works with or without a tuning session behind
// it. A broken file is an error worth stopping for.
func applySky(path string) (bool, error) {
	c, err := stars.LoadSky(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := stars.UseSky(c); err != nil {
		return false, err
	}
	return true, nil
}

// forcedColorProfile mirrors exec-tui: profile detection fails in
// detached ptys (tmux capture, CI), which would strip every color from
// a recording. When CLICOLOR_FORCE is set the program runs with an
// ANSI256 profile; otherwise detection is left alone.
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
	play := premiere()
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
			m.play.Next()
			return m, nil
		default:
			if rs := []rune(msg.Text); len(rs) == 1 {
				switch rs[0] {
				case 'q':
					m.play.Stop()
					return m, tea.Quit
				case ' ':
					m.play.Next()
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
	if _, err := applySky(*skyPath); err != nil {
		fmt.Fprintln(os.Stderr, "premiere:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	m := newModel(*seconds)
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "premiere:", err)
		os.Exit(1)
	}
}
