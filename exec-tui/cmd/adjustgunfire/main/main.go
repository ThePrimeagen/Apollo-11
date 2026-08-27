// adjustgunfire reads the gunfire blast-config JSON, tunes the
// one-shot Doom-shotgun blast live — every heading firing at once as
// a compass rose beside a paged panel of every knob, the way the
// flame config plays all eight courses — and writes the file back on
// save. tab flips pages (aim, core, then each heading), j/k select,
// h/l change, [/] take bigger steps, f fires now, s saves and quits,
// q quits.
//
//	go run ./cmd/adjustgunfire/main
//	go run ./cmd/adjustgunfire/main -seconds 15
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustgunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

func main() {
	path := flag.String("config", adjustgunfire.DefaultConfigPath, "gunfire blast JSON")
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := adjustgunfire.Open(*path, *seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "adjustgunfire:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := adjustgunfire.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "adjustgunfire:", err)
		os.Exit(1)
	}
}
