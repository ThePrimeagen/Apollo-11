// exec-tui: an interactive simulation of the Apollo 11 LM guidance
// computer's Executive during the powered descent. See ROADMAP.md.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
	"github.com/theprimeagen/apollo-11/exec-tui/ui"
)

func main() {
	var opts []tea.ProgramOption
	if p, ok := ui.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	engine := sim.New()
	p := tea.NewProgram(ui.NewModel(engine), opts...)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "exec-tui:", err)
		os.Exit(1)
	}
}
