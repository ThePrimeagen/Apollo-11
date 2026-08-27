package editor

// Tests written FIRST. An animation atlas keys its frames by pose name
// ("run1", "jump", "pole2") instead of compass heading. The editor
// must treat those like any other frame: open onto one, cycle through
// them with [ and ], list them in the frames popup, and — above all —
// never drop them on save.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// poseTestAtlas is a small pose-keyed atlas: three named frames at
// size 1, no compass frames at all.
func poseTestAtlas() *sprite.Atlas {
	a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
	for _, name := range []sprite.Heading{"run1", "run2", "jump"} {
		sp := sprite.New(16, 8)
		sp.Set(0, 0, sprite.Cell{Ch: []rune(name)[0], FG: 255, BG: -1})
		a.SetFrame(sprite.Size1, name, sp)
	}
	return a
}

func TestPoseFrames(t *testing.T) {
	t.Run("happy: opening a pose atlas snaps onto a pose frame", func(t *testing.T) {
		m := New(poseTestAtlas(), "")
		m.snapToExistingFrame()
		if m.Size != sprite.Size1 {
			t.Fatalf("snapped to size %d, want 1", m.Size)
		}
		if _, ok := m.Atlas.Frame(m.Size, m.Heading); !ok {
			t.Fatalf("snapped to heading %q which has no frame", m.Heading)
		}
	})
	t.Run("happy: ] and [ cycle through the pose frames", func(t *testing.T) {
		m := New(poseTestAtlas(), "")
		m.Size = sprite.Size1
		m.Heading = "jump"
		seen := map[sprite.Heading]bool{m.Heading: true}
		for i := 0; i < 2; i++ {
			m = send(m, key(']'))
			if _, ok := m.Atlas.Frame(m.Size, m.Heading); !ok {
				t.Fatalf("] landed on %q which has no frame", m.Heading)
			}
			seen[m.Heading] = true
		}
		if len(seen) != 3 {
			t.Fatalf("cycling visited %d pose frames, want all 3 (%v)", len(seen), seen)
		}
		wrapped := m.Heading
		m = send(m, key(']'))
		if m.Heading == wrapped {
			t.Fatal("] on the last pose must wrap, not stall")
		}
		m = send(m, key('['))
		if m.Heading != wrapped {
			t.Fatalf("[ must step back to %q, got %q", wrapped, m.Heading)
		}
	})
	t.Run("happy: the frames popup lists pose names", func(t *testing.T) {
		m := New(poseTestAtlas(), "")
		m.snapToExistingFrame()
		m.Win = WinFrames
		m.TermW, m.TermH = 100, 40
		v := m.View().Content
		for _, want := range []string{"run1", "run2", "jump"} {
			if !strings.Contains(v, want) {
				t.Fatalf("frames popup is missing pose %q", want)
			}
		}
	})
	t.Run("happy: saving a pose atlas keeps every pose on disk", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "poses.json")
		m := New(poseTestAtlas(), path)
		m.snapToExistingFrame()
		if err := m.Save(); err != nil {
			t.Fatalf("Save: %v", err)
		}
		back, err := sprite.LoadFile(path)
		if err != nil {
			t.Fatalf("reload: %v", err)
		}
		for _, name := range []sprite.Heading{"run1", "run2", "jump"} {
			sp, ok := back.Frame(sprite.Size1, name)
			if !ok {
				t.Fatalf("save dropped pose %q — the editor destroyed the animation", name)
			}
			if sp.At(0, 0).Ch != []rune(name)[0] {
				t.Fatalf("pose %q lost its cells on the round trip", name)
			}
		}
	})
	t.Run("unhappy: a compass-only atlas cycles exactly as before", func(t *testing.T) {
		m := newEd(t)
		m.Size = sprite.Size1
		m.Heading = sprite.N
		m = send(m, key(']'))
		if m.Heading != sprite.NE {
			t.Fatalf("] from N = %q, want NE — compass order must not change", m.Heading)
		}
		m = send(m, key('['))
		if m.Heading != sprite.N {
			t.Fatalf("[ back = %q, want N", m.Heading)
		}
	})
	t.Run("unhappy: cycling with no frames at all never panics", func(t *testing.T) {
		a := &sprite.Atlas{Palette: append([]sprite.PaletteEntry(nil), sprite.DefaultPalette...)}
		m := New(a, "")
		m = send(m, key(']'))
		m = send(m, key('['))
		_ = m.View()
	})
}
