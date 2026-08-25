// screenplay-lab: a two-scene premiere for the screenplay component.
//
// Scene 1, "arrival": the Apollo craft slides in from the right wing over
// a drifting starfield, parks at center stage, and bobs on a slow sine.
// Scene 2, "the end": the height-5 banner card, centered.
//
//	space     cut to the next scene
//	q         quit
//
//	go run . -seconds 20
package main

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/screenplay-lab/screenplay"
)

type model struct {
	w, h    int
	play    *screenplay.Screenplay
	seconds float64
	elapsed float64
}

// premiere is the two-scene bill: arrival, then the end card.
func premiere() *screenplay.Screenplay {
	return screenplay.New()
}

func newModel(seconds float64) model {
	return model{}
}

type frameMsg struct{}

const frameMs = 1000.0 / 30

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m model) View() string {
	return ""
}

func main() {
}
