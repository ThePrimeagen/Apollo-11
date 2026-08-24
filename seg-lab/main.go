// seg-lab: a standalone terminal segmented-letter viewer.
//
// Unicode only has segmented digits (U+1FBF0–U+1FBF9). Letters are composed.
package main

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/seg-lab/seg"
)

type demoModel struct {
	text  string
	style seg.Style
}

func newDemo() demoModel { return demoModel{} }

func (m demoModel) Init() tea.Cmd { return nil }

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }

func (m demoModel) View() string { return "" }
