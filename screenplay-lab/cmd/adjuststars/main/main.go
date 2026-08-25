// adjuststars reads the sky-config JSON, tunes the starfield live —
// the whole sky plays behind a panel of eight numbers, a fly delay and
// a density for each of the four star layers — and writes the file
// back on save. j/k select, h/l change, s saves and quits, q quits.
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
	path := flag.String("config", "cmd/adjuststars/stars.json", "sky config JSON")
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := adjuststars.Open(*path, *seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "adjuststars:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := adjuststars.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := tea.NewProgram(m, opts...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "adjuststars:", err)
		os.Exit(1)
	}
}
