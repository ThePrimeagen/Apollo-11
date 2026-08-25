// exec-tui: the front door to every Apollo-11 lab. Running it opens a
// scrollable launcher menu (j/k move, enter runs, q quits) listing the
// screenplay premiere, the flame and stars configurators, the legacy
// Executive sim, and the rest of the labs. The sim itself lives behind
// the LEGACY EXEC TUI entry. See ROADMAP.md.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustflame"
	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/sim"
	"github.com/theprimeagen/apollo-11/exec-tui/ui"
)

func main() {
	status := ""
	for {
		e, ok := pick(status)
		if !ok {
			return
		}
		status = ""
		if err := launch(e); err != nil {
			status = e.ID + ": " + err.Error()
		}
	}
}

// pick runs the menu program and returns the chosen entry, if any.
func pick(status string) (menu.Entry, bool) {
	p := tea.NewProgram(menu.New(menu.Catalog(), status), colorOpts()...)
	got, err := p.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "exec-tui:", err)
		os.Exit(1)
	}
	return got.(menu.Model).Chosen()
}

// colorOpts carries the CLICOLOR_FORCE escape hatch into every program
// the launcher starts in-process.
func colorOpts() []tea.ProgramOption {
	var opts []tea.ProgramOption
	if p, ok := ui.ForcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	return opts
}

// launch blocks until the chosen program exits; its error lands on the
// menu's status line instead of killing the launcher.
func launch(e menu.Entry) error {
	if e.Module != "" {
		return runExternal(e)
	}
	switch e.ID {
	case "legacy":
		_, err := tea.NewProgram(ui.NewModel(sim.New()), colorOpts()...).Run()
		return err
	case "flame":
		return runFlame()
	}
	return fmt.Errorf("unknown in-process program %q", e.ID)
}

// runFlame opens the heat-threshold configurator on the same JSON the
// standalone cmd/adjustflame runner edits.
func runFlame() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir, err := menu.LocateModule(cwd, "exec-tui")
	if err != nil {
		return err
	}
	m, err := adjustflame.Open(filepath.Join(dir, "cmd", "adjustflame", "flame.json"))
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m, colorOpts()...).Run()
	return err
}

// runExternal hands the terminal to a sibling lab via `go run`, from the
// lab's own module directory so its relative config paths keep working.
func runExternal(e menu.Entry) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir, err := menu.LocateModule(cwd, e.Module)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "run", e.Pkg)
	cmd.Dir = dir
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
