package adjuststars

import (
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

const (
	defaultW = 72
	defaultH = 28
	minW     = 10
	minH     = 4
	frameMs  = 1000.0 / 30
)

// Model is the bubbletea shell around the tuner scene: it owns the
// screen, forwards keys to the knobs, and puts the rendered grid on
// the terminal.
type Model struct {
	w, h    int
	play    *screenplay.Screenplay
	tuner   *Tuner
	screen  *screenplay.Screen
	seconds float64
	elapsed float64
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
		case "h", "left":
			m.tuner.Nudge(-1)
		}
	}
	return m, nil
}

// View is the rendered screen, exactly window height.
func (m Model) View() tea.View {
	m.play.Render(m.screen)
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
