// gunfire: the one-shot Doom muzzle-flame demo — the red flame that
// comes out when the shotgun goes off. An empty stage and the trigger
// on the space bar: one squeeze and the flame leaps from the muzzle
// at low center screen, a white-hot heart wrapped in tongues that
// cool bright yellow through orange and red to a maroon ember, with
// Doom's dimmer second flash frame pulsing a beat later — then the
// stage goes dark again, because a gunshot is a trigger, not a clock.
// The aim goes any which direction: left/h and right/l swing the
// muzzle 15° a press around the full circle — the status line reads
// the aim out — and the flame's lift stays world-up, so a sideways or
// downward shot still curls its tongues skyward. The demo auto-fires
// once shortly after boot so tapes show the flame, and it reads the
// same JSON the gunfire tuner saves.
//
//	space, f      fire
//	left/h        swing the aim counterclockwise
//	right/l       swing it back
//	q             quit
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

	// aimStepDeg is one press of an aim key: 24 presses is a full turn.
	aimStepDeg = 15
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
		case "left", "h":
			swingAim(aimStepDeg)
			return m, nil
		case "right", "l":
			swingAim(-aimStepDeg)
			return m, nil
		}
	}
	return m, nil
}

// swingAim turns the active blast's muzzle by delta degrees, wrapping
// around the circle, and puts it in effect — the very next volley
// flies the new way.
func swingAim(delta float64) {
	c := gunfire.ActiveBlast()
	c.AngleDeg = wrapDeg(c.AngleDeg + delta)
	_ = gunfire.UseBlast(c)
}

// wrapDeg folds an angle into the circle's signed half turns.
func wrapDeg(a float64) float64 {
	for a > 180 {
		a -= 360
	}
	for a < -180 {
		a += 360
	}
	return a
}

func (m model) View() tea.View {
	m.play.Render(m.screen)
	w, h := m.screen.Size()
	stage := strings.Split(m.screen.Render(), "\n")
	for len(stage) < h {
		stage = append(stage, "")
	}
	status := fmt.Sprintf(" gunfire   space fire   ←/→ aim %g°   q quit", gunfire.ActiveBlast().AngleDeg)
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
