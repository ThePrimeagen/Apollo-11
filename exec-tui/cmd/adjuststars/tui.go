package adjuststars

import (
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	defaultW = 72
	defaultH = 28
	minW     = 10
	minH     = 4
	frameMs  = 1000.0 / 30
)

// Model is the bubbletea shell around the tuner scene: it owns the
// screen, forwards keys to the knobs, applies every change as the
// active sky, and puts the rendered grid on the terminal.
type Model struct {
	Path    string
	Saved   bool
	w, h    int
	play    *screenplay.Screenplay
	tuner   *Tuner
	screen  *screenplay.Screen
	seconds float64
	elapsed float64
	note    string
}

// Open reads the sky-config JSON at path, makes it the active sky, and
// seeds the tuner from it. A missing or invalid file is an error.
func Open(path string, seconds float64) (Model, error) {
	c, err := stars.LoadSky(path)
	if err != nil {
		return Model{}, err
	}
	if err := stars.UseSky(c); err != nil {
		return Model{}, err
	}
	m := NewModel(seconds)
	m.Path = path
	return m, nil
}

// sky is the config the knobs currently describe.
func (m Model) sky() stars.SkyConfig {
	return stars.SkyConfig{
		Delay:   m.tuner.Delays[:],
		Density: m.tuner.Densities[:],
	}
}

// NewModel opens the tuner scene, optionally auto-quitting after
// seconds (for tapes).
func NewModel(seconds float64) Model {
	tuner := NewTuner()
	play := screenplay.New(screenplay.Entry{Name: "adjust stars", Scene: tuner})
	play.Start()
	return Model{
		w:       defaultW,
		h:       defaultH,
		play:    play,
		tuner:   tuner,
		screen:  screenplay.NewScreen(defaultW, defaultH),
		seconds: seconds,
	}
}

// FrameMsg advances the sky one frame.
type FrameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return FrameMsg{}
	})
}

// Init schedules the first frame.
func (m Model) Init() tea.Cmd { return tick() }

// Update runs the clock and the knobs.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.screen.Resize(max(m.w, minW), max(m.h, minH))
		return m, nil
	case FrameMsg:
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
		case "j", "down":
			m.tuner.Move(1)
		case "k", "up":
			m.tuner.Move(-1)
		case "l", "right":
			m.tuner.Nudge(1)
			m.apply()
		case "h", "left":
			m.tuner.Nudge(-1)
			m.apply()
		case "s":
			if m.Path == "" {
				m.note = "no config path — open the tool with -config"
				return m, nil
			}
			if err := m.sky().Save(m.Path); err != nil {
				m.note = "save failed: " + err.Error()
				return m, nil
			}
			m.Saved = true
			m.play.Stop()
			return m, tea.Quit
		}
	}
	return m, nil
}

// apply puts the knobs in effect as the active sky, live, so any
// tuned scene in the process follows along. The rails sit inside the
// config bounds, so this cannot fail.
func (m Model) apply() {
	_ = stars.UseSky(m.sky())
}

// View is the rendered screen, exactly window height, with the save
// note (if any) pinned under the panel.
func (m Model) View() tea.View {
	m.play.Render(m.screen)
	if m.note != "" {
		putText(m.screen, 0, 1+Rows, m.note+" ", 203)
	}
	_, h := m.screen.Size()
	rows := strings.Split(m.screen.Render(), "\n")
	for len(rows) < h {
		rows = append(rows, "")
	}
	v := tea.NewView(strings.Join(rows, "\n"))
	v.AltScreen = true
	return v
}

// ForcedColorProfile mirrors exec-tui: profile detection fails in
// detached ptys (tmux capture, CI), which would strip every color from
// a recording. When CLICOLOR_FORCE is set the program runs with an
// ANSI256 profile; otherwise detection is left alone.
func ForcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}
