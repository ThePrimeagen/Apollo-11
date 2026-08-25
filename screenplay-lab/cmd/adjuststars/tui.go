package adjuststars

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

const (
	defaultW = 72
	defaultH = 28
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
	return Model{}
}

// FrameMsg advances the sky one frame.
type FrameMsg struct{}

func tick() tea.Cmd {
	return nil
}

// Init schedules the first frame.
func (m Model) Init() tea.Cmd { return nil }

// Update runs the clock and the knobs.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View is the rendered screen, exactly window height.
func (m Model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true
	return v
}

// ForcedColorProfile mirrors exec-tui: profile detection fails in
// detached ptys (tmux capture, CI), which would strip every color from
// a recording. When CLICOLOR_FORCE is set the program runs with an
// ANSI256 profile; otherwise detection is left alone.
func ForcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return 0, false
	}
	return 0, false
}
