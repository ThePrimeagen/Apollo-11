package adjustgunfire

import (
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/gunfire"

	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	defaultW = 80
	defaultH = 28
	minW     = 10
	minH     = 4
	frameMs  = 1000.0 / 30

	// The tool keeps squeezing the trigger so the blast is always in
	// the air while you turn knobs: one quick shot after boot, then
	// one about every Doom shotgun cadence and a half.
	firstFireAt = 0.25
	refireEvery = 1.6
)

// Model is the bubbletea shell around the tuner: the live blast plus
// the paged knob panel. Every change is applied as the active blast.
type Model struct {
	Path     string
	Saved    bool
	w, h     int
	play     *screenplay.Screenplay
	tuner    *Tuner
	blast    *gunfire.Blast
	screen   *screenplay.Screen
	seconds  float64
	elapsed  float64
	nextFire float64
	note     string
}

// Open reads the blast-config JSON at path, makes it the active
// blast, and seeds the tuner from it. A missing or invalid file is
// an error.
func Open(path string, seconds float64) (Model, error) {
	c, err := gunfire.LoadBlast(path)
	if err != nil {
		return Model{}, err
	}
	if err := gunfire.UseBlast(c); err != nil {
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
	blast := gunfire.NewBlast(7)
	play := screenplay.New(screenplay.Entry{
		Name: "adjust gunfire",
		Scene: &screenplay.Ensemble{
			Assemble: func() []screenplay.Component {
				return []screenplay.Component{blast}
			},
		},
	})
	play.Start()
	return Model{
		w:        defaultW,
		h:        defaultH,
		play:     play,
		tuner:    tuner,
		blast:    blast,
		screen:   screenplay.NewScreen(defaultW, defaultH),
		seconds:  seconds,
		nextFire: firstFireAt,
	}
}

// FrameMsg advances the blast one frame.
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
		if m.elapsed >= m.nextFire && m.blast.Fire() {
			m.nextFire = m.elapsed + refireEvery
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
		case "tab":
			m.tuner.Flip(1)
		case "shift+tab":
			m.tuner.Flip(-1)
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
		case "f", "space", " ":
			m.blast.Fire()
		case "s":
			if m.Path == "" {
				m.note = "no config path — open the tool with -config"
				return m, nil
			}
			if err := m.tuner.Blast.Save(m.Path); err != nil {
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
	if err := gunfire.UseBlast(m.tuner.Blast); err != nil {
		m.note = err.Error()
		return
	}
	m.note = ""
}

func (m Model) View() tea.View {
	m.play.Render(m.screen)
	m.panel()
	if m.note != "" {
		putText(m.screen, 0, 3+len(pageMeta(m.tuner.Page)), m.note+" ", 203)
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
	putText(m.screen, 0, 0, "adjust gunfire  tab page  j/k pick  h/l turn  [/] ×10  f fire  s save  q quit ", 245)
	x := 0
	for i, name := range pageNames {
		label, fg := "  "+name+"  ", 245
		if i == m.tuner.Page {
			label, fg = " ["+name+"] ", 214
		}
		putText(m.screen, x, 1, label, fg)
		x += len([]rune(label))
	}
	knobs := pageMeta(m.tuner.Page)
	for i := range knobs {
		marker, fg := "  ", 250
		if i == m.tuner.Cursor {
			marker, fg = "> ", 214
		}
		row := marker + fmtPad(knobs[i].label, 12) + " " + formatKnob(m.tuner.Page, i, m.tuner.get(i)) + " "
		putText(m.screen, 0, 2+i, row, fg)
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
