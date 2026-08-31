// lunarcloseup: 02. Walkthrough — the composable five-scene bill from
// shows/lunarcloseup. The house opens on "pause": the still sky
// alone, for as long as the audience likes. Space cuts to "Lunar
// Lander Close-Up": the zoomed-in Apollo craft slides in from the
// right at once, hull only, cold engine, the sky surging from rest
// to a 1.25 peak then settling to cruise. Space cuts to "fire": the
// parked craft lights the booster, eases downward, and the stars
// slow by 60% over five seconds. Space cuts to "fall": the north-facing lander, fire
// down, drops from the top of the stage to the bottom. Space cuts to
// "landing": a huge moon horizon as a colored floor (five rows high
// in the middle, one row at the edges) and the lander coming down
// onto it — booster full, then ¾, ½, ¼, then off on the pad. The
// moment the booster starts slowing the craft, a mirrored dust cloud
// blows out of the pad on both sides of the bell and keeps running
// through booster-off. Space on the last scene ends the show. One
// stars.Continuity seeds every scene's sky, so no cut ever jumps a
// star.
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

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
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
	puffPath := flag.String("dust", "components/dust/config.json",
		"dust puff JSON (adjustdust); a missing file keeps the stock kick")
	cfgPath := flag.String("config", landing.DefaultConfigPath,
		"landing timing JSON; a missing file keeps the stock knobs")
	flag.Parse()
	if err := applySky(menu.Resolve(*skyPath)); err != nil {
		fmt.Fprintln(os.Stderr, "lunarcloseup:", err)
		os.Exit(1)
	}
	if err := applyPuff(menu.Resolve(*puffPath)); err != nil {
		fmt.Fprintln(os.Stderr, "lunarcloseup:", err)
		os.Exit(1)
	}
	c, err := landing.LoadOrDefault(menu.Resolve(*cfgPath))
	if err != nil {
		fmt.Fprintln(os.Stderr, "lunarcloseup:", err)
		os.Exit(1)
	}
	if err := landing.Use(c); err != nil {
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
