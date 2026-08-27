package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustsky"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

func main() {
	path := flag.String("config", adjustsky.DefaultConfigPath, "sky JSON")
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := adjustsky.Open(*path, *seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "adjustsky:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := adjustsky.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "adjustsky:", err)
		os.Exit(1)
	}
}
