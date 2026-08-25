// screenplay-lab: a two-scene premiere for the screenplay component.
//
// Scene 1, "arrival": the Apollo craft slides in from the right wing
// over a drifting starfield, parks at center stage, and bobbles on a
// slow one-cell sine. Scene 2, "the end": the height-5 banner card,
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

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/stars-lab/stars"

	"github.com/theprimeagen/apollo-11/screenplay-lab/cast"
	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

const (
	defaultW = 72
	defaultH = 28
	frameMs  = 1000.0 / 30
)

// premiere is the bill: arrival, then the end card.
func premiere() *screenplay.Screenplay {
	return screenplay.New(
		&screenplay.Scene{
			Name: "arrival",
			Cast: []screenplay.Actor{
				cast.NewStarfield(stars.Drift),
				cast.NewShip(11),
			},
		},
		&screenplay.Scene{
			Name: "the end",
			Cast: []screenplay.Actor{
				cast.NewStarfield(stars.Drift),
				mustTitle("THE END", 5),
			},
		},
	)
}

// mustTitle fails at boot, not on stage: the bill is static, so a bad
// card is a programming error.
func mustTitle(text string, height int) *cast.Title {
	t, err := cast.NewTitle(text, height)
	if err != nil {
		panic(err)
	}
	return t
}

type model struct {
	w, h    int
	play    *screenplay.Screenplay
	seconds float64
	elapsed float64
}

func newModel(seconds float64) model {
	return model{
		w:       defaultW,
		h:       defaultH,
		play:    premiere(),
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
		return m, nil
	case frameMsg:
		dt := frameMs / 1000
		m.elapsed += dt
		m.play.Advance(dt)
		if m.seconds > 0 && m.elapsed >= m.seconds {
			return m, tea.Quit
		}
		return m, tick()
	case tea.KeyMsg:
		switch {
		case msg.Type == tea.KeyCtrlC:
			return m, tea.Quit
		case msg.Type == tea.KeySpace:
			m.play.Next()
			return m, nil
		case msg.Type == tea.KeyRunes && len(msg.Runes) == 1:
			switch msg.Runes[0] {
			case 'q':
				return m, tea.Quit
			case ' ':
				m.play.Next()
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	w, h := m.w, m.h
	if w < 10 {
		w = 10
	}
	if h < 4 {
		h = 4
	}
	st := screenplay.NewStage(w, h-1)
	sc := m.play.Current()
	sc.Paint(st)
	name := ""
	if sc != nil {
		name = sc.Name
	}
	status := fmt.Sprintf(" scene %d/%d — %s   space next scene · q quit",
		m.play.SceneIndex()+1, m.play.Len(), name)
	dim := "\x1b[38;5;240m"
	reset := "\x1b[0m"
	return st.Render() + "\n" + dim + pad(status, w) + reset
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
	flag.Parse()
	m := newModel(*seconds)
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "screenplay-lab:", err)
		os.Exit(1)
	}
}
