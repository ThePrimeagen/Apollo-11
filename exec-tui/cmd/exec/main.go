package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/sim"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
	"github.com/theprimeagen/apollo-11/exec-tui/ui"
)

func main() {
	var opts []tea.ProgramOption
	if p, ok := ui.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(ui.NewModel(sim.New()), opts...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
