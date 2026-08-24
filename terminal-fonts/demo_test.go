package main

// Demo harness tests, written first. The demo prints the A-Z catalog at
// every supported height. Heights 1-3 fit one line; heights 4 and 5 split
// the alphabet across two blocks (A-M, N-Z) separated by a blank line.

import (
	"errors"
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
)

func TestRenderABC(t *testing.T) {
	t.Run("happy: height 1 is the plain alphabet on a single line", func(t *testing.T) {
		out, err := renderABC(1)
		if err != nil {
			t.Fatalf("height 1: %v", err)
		}
		if out != "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
			t.Fatalf("height 1 must be the alphabet itself, got %q", out)
		}
	})
	t.Run("happy: heights 2 and 3 are single blocks of uniform rows", func(t *testing.T) {
		for _, h := range []int{2, 3} {
			out, err := renderABC(h)
			if err != nil {
				t.Fatalf("height %d: %v", h, err)
			}
			lines := strings.Split(out, "\n")
			if len(lines) != h {
				t.Fatalf("height %d: %d lines, want %d", h, len(lines), h)
			}
			for i, l := range lines {
				if len(l) != len(lines[0]) {
					t.Fatalf("height %d line %d: ragged width %d vs %d", h, i, len(l), len(lines[0]))
				}
			}
		}
	})
	t.Run("happy: heights 4 and 5 split A-M and N-Z across two blocks", func(t *testing.T) {
		for _, h := range []int{4, 5} {
			out, err := renderABC(h)
			if err != nil {
				t.Fatalf("height %d: %v", h, err)
			}
			lines := strings.Split(out, "\n")
			if len(lines) != 2*h+1 {
				t.Fatalf("height %d: %d lines, want %d (two blocks and a separator)", h, len(lines), 2*h+1)
			}
			if lines[h] != "" {
				t.Fatalf("height %d: line %d must be the blank separator, got %q", h, h, lines[h])
			}
			_, wantTop, err := termfont.Render(h, "ABCDEFGHIJKLM")
			if err != nil {
				t.Fatalf("height %d top block: %v", h, err)
			}
			_, wantBottom, err := termfont.Render(h, "NOPQRSTUVWXYZ")
			if err != nil {
				t.Fatalf("height %d bottom block: %v", h, err)
			}
			for i := 0; i < h; i++ {
				if len(lines[i]) != wantTop {
					t.Fatalf("height %d top row %d: width %d, want %d", h, i, len(lines[i]), wantTop)
				}
				if len(lines[h+1+i]) != wantBottom {
					t.Fatalf("height %d bottom row %d: width %d, want %d", h, i, len(lines[h+1+i]), wantBottom)
				}
			}
		}
	})
	t.Run("unhappy: heights outside 1..5 refuse to render the catalog", func(t *testing.T) {
		for _, h := range []int{0, 6} {
			out, err := renderABC(h)
			if err == nil {
				t.Fatalf("height %d must error, got output %q", h, out)
			}
			if !errors.Is(err, termfont.ErrInvalidHeight) {
				t.Fatalf("height %d: error must wrap termfont.ErrInvalidHeight, got %v", h, err)
			}
			if out != "" {
				t.Fatalf("height %d: failed catalog must be empty, got %q", h, out)
			}
		}
	})
}
