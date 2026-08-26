package lander

// The Apollo lander owns its art: the hand-drawn default atlas and the
// editable JSON both live here, so the sprite editor and every scene
// read the same home. Tests written before the move.

import (
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

func TestDefaultAtlas(t *testing.T) {
	t.Run("happy: every size and heading is drawn", func(t *testing.T) {
		a := DefaultAtlas()
		if a == nil {
			t.Fatal("DefaultAtlas() returned nil")
		}
		for _, sz := range sprite.Sizes {
			for _, h := range HeadingsFor(sz) {
				sp, ok := a.Frame(sz, h)
				if !ok {
					t.Fatalf("size %d heading %s missing from the default atlas", sz, h)
				}
				if err := sp.Validate(); err != nil {
					t.Fatalf("size %d heading %s: %v", sz, h, err)
				}
				w, hh := sz.Dim()
				if sp.Width != w || sp.Height != hh {
					t.Fatalf("size %d heading %s is %dx%d, want %dx%d",
						sz, h, sp.Width, sp.Height, w, hh)
				}
			}
		}
	})
	t.Run("unhappy: callers get their own copy — edits never leak back", func(t *testing.T) {
		a := DefaultAtlas()
		sp := a.MustFrame(sprite.Size1, sprite.N)
		orig := sp.At(2, 6).Ch
		sp.Set(2, 6, sprite.Cell{Ch: 'X', FG: 15, BG: -1})
		if got := DefaultAtlas().MustFrame(sprite.Size1, sprite.N).At(2, 6).Ch; got != orig {
			t.Fatalf("a caller's edit leaked into the shared art: %q, want %q", got, orig)
		}
	})
}
