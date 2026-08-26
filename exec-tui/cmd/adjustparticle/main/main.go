// adjustparticle reads the nyan trail-config JSON, tunes the rainbow
// particle plume live — a parked nyan cat over the sky and a panel of
// engine knobs — and writes the file back on save. j/k select, h/l
// change, [/] take bigger steps, s saves and quits, q quits.
//
//	go run ./cmd/adjustparticle/main
//	go run ./cmd/adjustparticle/main -seconds 15
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustparticle"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

func main() {
	path := flag.String("config", adjustparticle.DefaultConfigPath, "particle trail JSON")
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := adjustparticle.Open(*path, *seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "adjustparticle:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := adjustparticle.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "adjustparticle:", err)
		os.Exit(1)
	}
}
