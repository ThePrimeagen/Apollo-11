// Package adjustarmed is the armed-eagle tuner: the live composite
// (bird, talon shotguns, gunfire) behind a panel of the flight and
// gun knobs. j/k pick a knob, h/l change it, p replays, s saves the
// component's config and quits.
package adjustarmed

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/armed"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const DefaultConfigPath = "components/armed/config.json"

const (
	defaultW = 80
	defaultH = 28
	minW     = 10
	minH     = 4
	frameMs  = 1000.0 / 30
	nKnobs   = int(armed.KnobCount)
)

type Tuner struct {
	Cfg    armed.Config
	Cursor int
}

func NewTuner() *Tuner {
	return &Tuner{Cfg: armed.Active()}
}

func (t *Tuner) Move(delta int) {
	if t == nil {
		return
	}
	n := nKnobs
	t.Cursor = ((t.Cursor+delta)%n + n) % n
}

func (t *Tuner) Nudge(steps int) {
	if t == nil {
		return
	}
	t.Cfg.Nudge(armed.Knob(t.Cursor), steps)
}

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

func Open(path string, seconds float64) (Model, error) {
	c, err := armed.Load(path)
	if err != nil {
		return Model{}, err
	}
	if err := armed.Use(c); err != nil {
		return Model{}, err
	}
	m := NewModel(seconds)
	m.Path = path
	return m, nil
}

func NewModel(seconds float64) Model {
	tuner := NewTuner()
	play := screenplay.New(screenplay.Entry{
		Name: "adjust armed",
		Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{armed.New()}
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

type FrameMsg struct{}

func tick() tea.Cmd {
	ns := float64(frameMs) * 1e6
	return tea.Tick(time.Duration(ns)*time.Nanosecond, func(time.Time) tea.Msg {
		return FrameMsg{}
	})
}

func (m Model) Init() tea.Cmd { return tick() }

func (m *Model) apply() {
	if err := armed.Use(m.tuner.Cfg); err != nil {
		m.note = err.Error()
		return
	}
	m.note = ""
	m.play.Stop()
	m.play.Start()
}

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
		case " ", "space", "enter", "p":
			m.apply()
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
			if err := m.tuner.Cfg.Save(m.Path); err != nil {
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

func (m Model) View() tea.View {
	m.play.Render(m.screen)
	putText(m.screen, 0, 0, "adjust armed   p replay  j/k select  h/l change  s save+quit  q quit ", 245)
	for i := 0; i < nKnobs; i++ {
		marker, fg := "  ", 250
		if i == m.tuner.Cursor {
			marker, fg = "> ", 214
		}
		row := marker + fmt.Sprintf("%-11s %s ", armed.KnobLabel(armed.Knob(i)), m.tuner.Cfg.Display(armed.Knob(i)))
		putText(m.screen, 0, 1+i, row, fg)
	}
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

func putText(scr *screenplay.Screen, x, y int, text string, fg int) {
	for i, r := range []rune(text) {
		scr.PutCell(x+i, y, r, fg, -1)
	}
}

func ForcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}
