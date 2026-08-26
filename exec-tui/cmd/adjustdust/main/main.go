// adjustdust reads the dust-off puff-config JSON, tunes the landing
// kick live — two mirrored swirl engines behind a panel of knobs —
// and writes the file back on save. j/k select, h/l change, [/] take
// bigger steps, s saves and quits, q quits.
//
//	go run ./cmd/adjustdust/main
//	go run ./cmd/adjustdust/main -seconds 15
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustdust"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

func main() {
	path := flag.String("config", adjustdust.DefaultConfigPath, "dust puff JSON")
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := adjustdust.Open(*path, *seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "adjustdust:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := adjustdust.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "adjustdust:", err)
		os.Exit(1)
	}
}
