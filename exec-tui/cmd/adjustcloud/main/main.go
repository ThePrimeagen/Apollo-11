package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustcloud"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

func main() {
	path := flag.String("config", adjustcloud.DefaultConfigPath, "cloud JSON")
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	flag.Parse()
	m, err := adjustcloud.Open(*path, *seconds)
	if err != nil {
		fmt.Fprintln(os.Stderr, "adjustcloud:", err)
		os.Exit(1)
	}
	var opts []tea.ProgramOption
	if p, ok := adjustcloud.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fmt.Fprintln(os.Stderr, "adjustcloud:", err)
		os.Exit(1)
	}
}
