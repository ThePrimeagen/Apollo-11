// agctop: the AGC Executive command screen — every process on the
// millisecond Executive, live, grouped by what it holds (VAC jobs,
// coreset jobs, no-priority operations), with the three cockpit
// switches on the bottom row:
//
//	d   DESCENT      the whole P63 job chain
//	1   1668         Buzz's V16N68 DELTAH monitor
//	r   RADAR STEAL  the RR CDU counter theft
//	q   quit
//
//	go run ./cmd/agctop
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/agctop"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
	"github.com/theprimeagen/apollo-11/exec-tui/ui"
	msim "github.com/theprimeagen/apollo-11/msim"
)

func main() {
	var opts []tea.ProgramOption
	if p, ok := ui.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(agctop.New(msim.NewLive()), opts...); err != nil {
		fmt.Fprintln(os.Stderr, "agctop:", err)
		os.Exit(1)
	}
}
