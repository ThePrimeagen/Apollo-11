// agcgraph: the graphs screen — a still of 2.5 seconds of machine time
// under the current switches: one CPU lane per process, grouped as VAC
// jobs, coreset jobs and no-priority operations, a hard white line on the
// 2.00 s guidance boundary, and a text legend for every process that ran.
// The SERVICER is entered exactly once per portrait — everything else
// keeps its timer — so its single ~1.36 s pass visibly stretches past the
// white line as load is switched on. Never animated; every toggle
// re-simulates the portrait.
//
//	d   DESCENT — the whole P63 job chain
//	1   1668 — Buzz's V16N68 DELTAH monitor (drops P64)
//	r   RADAR STEAL — the RR CDU counter theft
//	p   P64 — the approach: REDESIG guidance, HIGATJOB, the flashing
//	    V06N64 (drops 1668)
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
