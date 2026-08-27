// agcgraph: the graphs screen — a still of 2.5 seconds of machine time
// under the current switches: three CPU lanes (VAC jobs, coreset jobs,
// no-priority operations) and a text legend for every job that ran.
// Never animated; every toggle re-simulates the portrait.
//
//	d   DESCENT — the whole P63 job chain
//	1   1668 — Buzz's V16N68 DELTAH monitor
//	r   RADAR STEAL — the RR CDU counter theft
//	q   quit
//
//	go run ./cmd/agcgraph
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/agcgraph"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
	"github.com/theprimeagen/apollo-11/exec-tui/ui"
)

func main() {
	var opts []tea.ProgramOption
	if p, ok := ui.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(agcgraph.New(), opts...); err != nil {
		fmt.Fprintln(os.Stderr, "agcgraph:", err)
		os.Exit(1)
	}
}
