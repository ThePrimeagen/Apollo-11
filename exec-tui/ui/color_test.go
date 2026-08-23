package ui

// t44 — screen recordings run in detached ptys where profile detection
// fails; CLICOLOR_FORCE must force a color profile so captures keep color.

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestForceColorIfRequested(t *testing.T) {
	t.Run("happy: CLICOLOR_FORCE=1 forces the ANSI256 profile", func(t *testing.T) {
		prev := lipgloss.ColorProfile()
		defer lipgloss.SetColorProfile(prev)
		t.Setenv("CLICOLOR_FORCE", "1")
		lipgloss.SetColorProfile(termenv.Ascii)
		ForceColorIfRequested()
		if got := lipgloss.ColorProfile(); got != termenv.ANSI256 {
			t.Fatalf("CLICOLOR_FORCE must force ANSI256, got %v", got)
		}
	})
	t.Run("unhappy: unset leaves the detected profile alone", func(t *testing.T) {
		prev := lipgloss.ColorProfile()
		defer lipgloss.SetColorProfile(prev)
		t.Setenv("CLICOLOR_FORCE", "")
		lipgloss.SetColorProfile(termenv.Ascii)
		ForceColorIfRequested()
		if got := lipgloss.ColorProfile(); got != termenv.Ascii {
			t.Fatalf("without CLICOLOR_FORCE the profile must not change, got %v", got)
		}
	})
}
