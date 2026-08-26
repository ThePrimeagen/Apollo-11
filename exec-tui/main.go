// exec-tui: the front door to every Apollo-11 lab. Running it opens a
// scrollable launcher menu (j/k move, enter runs, q quits) listing the
// screenplay premiere, the flame and stars configurators, the legacy
// Executive sim, and the rest of the labs. The sim itself lives behind
// the LEGACY EXEC TUI entry. See ROADMAP.md.
package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustflame"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/editor"
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
	if e.Pkg != "" {
		return runLocal(e)
	}
	switch e.ID {
	case "legacy":
		_, err := tea.NewProgram(ui.NewModel(sim.New()), colorOpts()...).Run()
		return err
	case "flame":
		return runFlame()
	case "editor":
		return runEditor()
	}
	return fmt.Errorf("unknown in-process program %q", e.ID)
}

// runFlame opens the heat-threshold configurator on the fire
// component's own config — the same JSON the standalone
// cmd/adjustflame runner edits.
func runFlame() error {
	root, err := ownModuleRoot()
	if err != nil {
		return err
	}
	m, err := adjustflame.Open(filepath.Join(root, filepath.FromSlash(adjustflame.DefaultConfigPath)))
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m, colorOpts()...).Run()
	return err
}

// runEditor opens the LM sprite editor on the shipped size-4 atlas.
// In-process so a bad file returns the load error on the menu instead
// of a bare `go run` exit status.
func runEditor() error {
	root, err := ownModuleRoot()
	if err != nil {
		return err
	}
	path := filepath.Join(root, filepath.FromSlash(editor.DefaultAtlasPath))
	if _, err := os.Stat(path); err != nil {
		if cand := filepath.Join(editor.FindAssetsDir(), "lm-4.json"); cand != path {
			if _, err := os.Stat(cand); err == nil {
				path = cand
			}
		}
	}
	m, err := editor.Open(path)
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(m, colorOpts()...).Run()
	return err
}

// ownModuleRoot finds this module's root, so in-module programs and
// their relative config paths work no matter where the launcher runs.
func ownModuleRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return menu.ModuleRoot(cwd)
}

// runLocal hands the terminal to one of our own cmd/ programs via
// `go run` from the module root, so every default config path — all
// relative to the root — keeps working.
func runLocal(e menu.Entry) error {
	root, err := ownModuleRoot()
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "run", e.Pkg)
	cmd.Dir = root
	var captured bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &captured)
	err = cmd.Run()
	if err == nil {
		return nil
	}
	if detail := lastNonEmptyLines(captured.String(), 6); detail != "" {
		return fmt.Errorf("%w: %s", err, detail)
	}
	return err
}

func lastNonEmptyLines(s string, n int) string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, " · ")
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
