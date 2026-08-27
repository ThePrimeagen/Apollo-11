package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustarmed"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

func main() {
	path := flag.String("config", adjustarmed.DefaultConfigPath, "armed-eagle JSON")
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := adjustarmed.Open(*path, *seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "adjustarmed:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := adjustarmed.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "adjustarmed:", err)
		os.Exit(1)
	}
}
