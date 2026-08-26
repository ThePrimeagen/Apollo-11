// gunfire: the one-shot shotgun demo, tuned to read like the Doom
// shotgun. A bare stage, a stub of a barrel at the muzzle, and the
// trigger on the space bar: one squeeze blooms the white-hot flash,
// throws the classic seven pellets in a tight fan, sprays sparks that
// cool yellow → orange → ember, and a beat later curls gray gunsmoke
// up out of the barrel — then the stage goes quiet again, because a
// gunshot is a trigger, not a clock. The demo auto-fires once shortly
// after boot so tapes show the shot, and it reads the same JSON the
// gunfire tuner saves.
//
//	space, f  fire
//	q         quit
//
//	go run ./cmd/gunfire
//	go run ./cmd/gunfire -seconds 6
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/components/particle"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

const (
	defaultW = 80
	defaultH = 24
	minW     = 10
	minH     = 4
	frameMs  = 1000.0 / 30

	// autoFireAt is the one free squeeze after boot, so tapes and the
	// impatient both see the shot without touching a key.
	autoFireAt = 0.4
)

func applyBlast(path string) error {
	c, err := gunfire.LoadBlast(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return gunfire.UseBlast(c)
}

func forcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}

type model struct {
	w, h      int
	play      *screenplay.Screenplay
	screen    *screenplay.Screen
	blast     *gunfire.Blast
	seconds   float64
	elapsed   float64
	autoFired bool
}

func newModel(seconds float64) model {
	blast := gunfire.NewBlast(11)
	play := screenplay.New(screenplay.Entry{
		Name: "gunfire",
		Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{blast}
			},
		},
	})
	play.Start()
	return model{
		w:       defaultW,
		h:       defaultH,
		play:    play,
		screen:  screenplay.NewScreen(defaultW, defaultH-1),
		blast:   blast,
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
		if !m.autoFired && m.elapsed >= autoFireAt && m.blast.Fire() {
			m.autoFired = true
		}
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
		case "space", " ", "f":
			m.blast.Fire()
			return m, nil
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	m.play.Render(m.screen)
	m.barrel()
	w, h := m.screen.Size()
	stage := strings.Split(m.screen.Render(), "\n")
	for len(stage) < h {
		stage = append(stage, "")
	}
	status := " gunfire   space fire   q quit"
	dim := "\x1b[38;5;240m"
	reset := "\x1b[0m"
	body := strings.Join(stage, "\n") + "\n" + dim + pad(status, w) + reset
	v := tea.NewView(body)
	v.AltScreen = true
	return v
}

// barrel draws the two-cell gun stub whose mouth is the blast's
// muzzle, so the shot visibly leaves something.
func (m model) barrel() {
	c := gunfire.ActiveBlast()
	w, h := m.screen.Size()
	uw := float64(w)*particle.CellWidthUnits - 0.01
	uh := float64(h)*particle.CellHeightUnits - 0.01
	cell := particle.CellOf(c.MuzzleX*uw, c.MuzzleY*uh)
	m.screen.PutCell(cell.Col-2, cell.Row, '━', 245, -1)
	m.screen.PutCell(cell.Col-1, cell.Row, '━', 250, -1)
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
	blastPath := flag.String("config", gunfireDefaultConfig,
		"gunfire blast JSON (adjustgunfire); a missing file keeps the stock shotgun")
	flag.Parse()
	if err := applyBlast(*blastPath); err != nil {
		fmt.Fprintln(os.Stderr, "gunfire:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(newModel(*seconds), opts...); err != nil {
		fmt.Fprintln(os.Stderr, "gunfire:", err)
		os.Exit(1)
	}
}

const gunfireDefaultConfig = "components/gunfire/config.json"
