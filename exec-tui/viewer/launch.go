package viewer

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustarmed"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustcloud"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustdust"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustflame"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustgunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustparticle"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustsky"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/editor"
	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

func colorOpts() []tea.ProgramOption {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return []tea.ProgramOption{tea.WithColorProfile(colorprofile.ANSI256)}
	}
	return nil
}

// Launch opens the editor e names and blocks until it quits. Components
// land in the ASCII editor, particles in their tuner, scenes in their
// runner. The caller resumes the viewer afterwards.
func Launch(ed Edit) error {
	path := menu.Resolve(ed.Path)
	opts := colorOpts()
	switch ed.Kind {
	case KindComponent:
		if ed.Program != "" {
			return launchParticle(ed.Program, path, opts)
		}
		m, err := editor.Open(path)
		if err != nil {
			return err
		}
		_, err = termreset.Run(m, opts...)
		return err
	case KindParticle:
		return launchParticle(ed.Program, path, opts)
	case KindScene:
		return runLocal(ed.Program)
	default:
		return fmt.Errorf("viewer: unknown edit kind %q", ed.Kind)
	}
}

func launchParticle(program, path string, opts []tea.ProgramOption) error {
	switch program {
	case "./cmd/adjustgunfire/main":
		m, err := adjustgunfire.Open(path, 0)
		if err != nil {
			return err
		}
		_, err = termreset.Run(m, opts...)
		return err
	case "./cmd/adjustflame/main":
		m, err := adjustflame.Open(path)
		if err != nil {
			return err
		}
		_, err = termreset.Run(m, opts...)
		return err
	case "./cmd/adjustdust/main":
		m, err := adjustdust.Open(path, 0)
		if err != nil {
			return err
		}
		_, err = termreset.Run(m, opts...)
		return err
	case "./cmd/adjustparticle/main":
		m, err := adjustparticle.Open(path, 0)
		if err != nil {
			return err
		}
		_, err = termreset.Run(m, opts...)
		return err
	case "./cmd/adjustsky/main":
		m, err := adjustsky.Open(path, 0)
		if err != nil {
			return err
		}
		_, err = termreset.Run(m, opts...)
		return err
	case "./cmd/adjustcloud/main":
		m, err := adjustcloud.Open(path, 0)
		if err != nil {
			return err
		}
		_, err = termreset.Run(m, opts...)
		return err
	case "./cmd/adjustarmed/main":
		m, err := adjustarmed.Open(path, 0)
		if err != nil {
			return err
		}
		_, err = termreset.Run(m, opts...)
		return err
	default:
		return runLocal(program)
	}
}

func runLocal(pkg string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := menu.ModuleRoot(cwd)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "run", pkg)
	cmd.Dir = root
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err = cmd.Run()
	termreset.Restore()
	return err
}
