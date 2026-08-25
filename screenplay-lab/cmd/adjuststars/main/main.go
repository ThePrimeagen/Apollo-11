// adjuststars tunes the starfield live: the whole sky plays behind a
// panel of eight numbers — a fly delay and a density for each of the
// four star layers. j/k select, h/l change, q quits.
//
//	go run ./cmd/adjuststars/main
//	go run ./cmd/adjuststars/main -seconds 15
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/screenplay-lab/cmd/adjuststars"
)

func main() {
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	var opts []tea.ProgramOption
	if p, ok := adjuststars.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	m := adjuststars.NewModel(*seconds)
	if _, err := tea.NewProgram(m, opts...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "adjuststars:", err)
		os.Exit(1)
	}
}
