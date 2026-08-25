package ui

// t44 — screen recordings run in detached ptys where profile detection
// fails; CLICOLOR_FORCE must force a color profile so captures keep color.
// In bubbletea v2 the profile is a program option instead of a lipgloss
// global, so the contract is a (profile, force?) pair fed to NewProgram.

import (
	"testing"

	"github.com/charmbracelet/colorprofile"
)

func TestForcedColorProfile(t *testing.T) {
	t.Run("happy: CLICOLOR_FORCE=1 forces the ANSI256 profile", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "1")
		p, ok := ForcedColorProfile()
		if !ok {
			t.Fatal("CLICOLOR_FORCE must force a profile")
		}
		if p != colorprofile.ANSI256 {
			t.Fatalf("CLICOLOR_FORCE must force ANSI256, got %v", p)
		}
	})
	t.Run("unhappy: unset leaves the detected profile alone", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "")
		if _, ok := ForcedColorProfile(); ok {
			t.Fatal("without CLICOLOR_FORCE no profile must be forced")
		}
	})
}
