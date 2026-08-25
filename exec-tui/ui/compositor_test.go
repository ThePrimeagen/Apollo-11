package ui

// The point of the v2 migration: lipgloss now ships a cell-based
// compositor, so scenes compose as z-ordered layers instead of string
// arithmetic. These tests pin the behavior the exec-tui scene work will
// build on: transparency-free blitting by z, and safe degenerate cases.

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestCompositorZOrder(t *testing.T) {
	t.Run("happy: the higher z layer wins where layers overlap", func(t *testing.T) {
		base := lipgloss.NewLayer("AAA\nAAA\nAAA")
		top := lipgloss.NewLayer("B").X(1).Y(1).Z(1)
		out := lipgloss.NewCompositor(base, top).Render()
		lines := strings.Split(out, "\n")
		if len(lines) != 3 {
			t.Fatalf("composite must keep the base 3 rows, got %d:\n%s", len(lines), out)
		}
		if got := []rune(lines[1])[1]; got != 'B' {
			t.Fatalf("cell (1,1) must show the z=1 layer, got %q", string(got))
		}
		if got := []rune(lines[0])[0]; got != 'A' {
			t.Fatalf("uncovered cells must keep the base layer, got %q", string(got))
		}
	})
	t.Run("happy: offsets place a sprite inside the scene", func(t *testing.T) {
		scene := lipgloss.NewCompositor(
			lipgloss.NewLayer("....\n....\n...."),
			lipgloss.NewLayer("LM").X(1).Y(2).Z(2),
		).Render()
		lines := strings.Split(scene, "\n")
		if !strings.HasPrefix(lines[2], ".LM") {
			t.Fatalf("sprite must land at x=1 y=2, got %q", lines[2])
		}
	})
	t.Run("unhappy: an empty compositor renders without panicking", func(t *testing.T) {
		out := lipgloss.NewCompositor().Render()
		if strings.TrimSpace(out) != "" {
			t.Fatalf("no layers must render nothing, got %q", out)
		}
	})
	t.Run("unhappy: an empty layer neither panics nor grows the scene", func(t *testing.T) {
		out := lipgloss.NewCompositor(
			lipgloss.NewLayer("XX"),
			lipgloss.NewLayer("").X(1).Y(0).Z(5),
		).Render()
		if !strings.Contains(out, "XX") {
			t.Fatalf("the base layer must survive an empty overlay, got %q", out)
		}
	})
}
