// exec-tui: an interactive simulation of the Apollo 11 LM guidance
// computer's Executive during the powered descent. See ROADMAP.md.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
	"github.com/theprimeagen/apollo-11/exec-tui/ui"
)

func main() {
	engine := sim.New()
	p := tea.NewProgram(ui.NewModel(engine), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "exec-tui:", err)
		os.Exit(1)
	}
}
