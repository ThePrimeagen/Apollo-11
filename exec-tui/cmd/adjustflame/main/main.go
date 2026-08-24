// adjustflame reads heat-threshold JSON, lets you edit the rungs in a
// TUI (hjkl + s), and writes the file back on save.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustflame"
)

func main() {
	path := flag.String("config", "cmd/adjustflame/flame.json", "heat threshold JSON")
	flag.Parse()
	m, err := adjustflame.Open(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
