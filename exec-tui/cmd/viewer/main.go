// viewer: cycle every component, particle effect, and scene. e opens
// the matching editor — ASCII for components, the particle tuner for
// a particle effect, the scene tuner for a scene — and closing that
// sub-edit resumes on the same item.
//
//	n / j / right   next
//	p / k / left    previous
//	e               edit
//	f               fullscreen — just the item, no chrome
//	q               quit
//
//	go run ./cmd/viewer
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
	"github.com/theprimeagen/apollo-11/exec-tui/viewer"
)

func colorOpts() []tea.ProgramOption {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return []tea.ProgramOption{tea.WithColorProfile(colorprofile.ANSI256)}
	}
	return nil
}

func main() {
	idx := 0
	for {
		got, err := termreset.Run(viewer.New(idx), colorOpts()...)
		if err != nil {
			fmt.Fprintln(os.Stderr, "viewer:", err)
			os.Exit(1)
		}
		m, ok := got.(viewer.Model)
		if !ok {
			return
		}
		ed, want := m.ChosenEdit()
		if !want {
			return
		}
		if err := viewer.Launch(ed); err != nil {
			fmt.Fprintln(os.Stderr, "viewer:", err)
		}
		idx = m.Index()
	}
}
