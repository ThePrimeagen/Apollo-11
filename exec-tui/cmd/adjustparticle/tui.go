package adjustparticle

import (
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/nyan"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	defaultW = 80
	defaultH = 28
	minW     = 10
	minH     = 4
	frameMs  = 1000.0 / 30
)

// Model is the bubbletea shell around the tuner: sky + parked nyan
// plus the knob panel. Every change is applied as the active trail.
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

// Open reads the trail-config JSON at path, makes it the active
// trail, and seeds the tuner from it. A missing or invalid file is
// an error.
func Open(path string, seconds float64) (Model, error) {
	c, err := nyan.LoadTrail(path)
	if err != nil {
		return Model{}, err
	}
	if err := nyan.UseTrail(c); err != nil {
		return Model{}, err
	}
	m := NewModel(seconds)
	m.Path = path
	return m, nil
}

// NewModel opens the tuner scene, optionally auto-quitting after
// seconds (for tapes).
func NewModel(seconds float64) Model {
	tuner := NewTuner()
	play := screenplay.New(screenplay.Entry{
		Name: "adjust particle",
		Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{
					stars.NewTunedStarfield(),
					nyan.NewParked(7),
				}
			},
		},
	})
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

// FrameMsg advances the plume one frame.
type FrameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return FrameMsg{}
	})
}

func (m Model) Init() tea.Cmd { return tick() }

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
		case "]":
			m.tuner.Nudge(10)
			m.apply()
		case "[":
			m.tuner.Nudge(-10)
			m.apply()
		case "s":
			if m.Path == "" {
				m.note = "no config path — open the tool with -config"
				return m, nil
			}
			if err := m.tuner.Trail.Save(m.Path); err != nil {
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

func (m Model) apply() {
	if err := nyan.UseTrail(m.tuner.Trail); err != nil {
		m.note = err.Error()
		return
	}
	m.note = ""
}

func (m Model) View() tea.View {
	m.play.Render(m.screen)
	m.panel()
	if m.note != "" {
		putText(m.screen, 0, 2+nKnobs, m.note+" ", 203)
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

func (m Model) panel() {
	putText(m.screen, 0, 0, "adjust particle   j/k select  h/l change  [/] ×10  s save+quit  q quit ", 245)
	for i := 0; i < nKnobs; i++ {
		marker, fg := "  ", 250
		if i == m.tuner.Cursor {
			marker, fg = "> ", 214
		}
		row := marker + fmtPad(knobMeta[i].label, 10) + " " + formatKnob(knob(i), m.tuner.get(knob(i))) + " "
		putText(m.screen, 0, 1+i, row, fg)
	}
}

func fmtPad(s string, n int) string {
	r := []rune(s)
	if len(r) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(r))
}

func putText(scr *screenplay.Screen, x, y int, text string, fg int) {
	for i, r := range []rune(text) {
		scr.PutCell(x+i, y, r, fg, -1)
	}
}

// ForcedColorProfile mirrors exec-tui: detached ptys strip color
// unless CLICOLOR_FORCE is set.
func ForcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}
