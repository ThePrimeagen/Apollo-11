package sprite

// Tests written FIRST. An Animation is the simplest thing that can
// play: an ordered list of sprites and a frame rate. Frame(i) walks
// the list in order and wraps; At(t) turns seconds into a frame index
// via FPS. AnimationFrom builds one from named atlas frames, in the
// order the caller asks for, and refuses to invent missing frames.

import (
	"strings"
	"testing"
)

// glyphAt is the whole first row of a sprite as a plain string — a
// cheap fingerprint for "which frame is this".
func glyphAt(sp Sprite) string {
	if sp.Height == 0 {
		return ""
	}
	return sp.GlyphRows()[0]
}

// stamp makes a 3x1 sprite whose first row spells the tag.
func stamp(tag string) Sprite {
	sp := New(3, 1)
	for i, r := range []rune(tag) {
		sp.Set(0, i, Cell{Ch: r, FG: 255, BG: -1})
	}
	return sp
}

func TestAnimationOrder(t *testing.T) {
	t.Run("happy: Frame walks the list in order and wraps", func(t *testing.T) {
		a := Animation{Frames: []Sprite{stamp("one"), stamp("two"), stamp("tre")}}
		want := []string{"one", "two", "tre", "one", "two", "tre"}
		for i, w := range want {
			if got := glyphAt(a.Frame(i)); got != w {
				t.Fatalf("Frame(%d) = %q, want %q — the list must play in order and wrap", i, got, w)
			}
		}
	})
	t.Run("happy: a negative index still lands on a real frame", func(t *testing.T) {
		a := Animation{Frames: []Sprite{stamp("one"), stamp("two")}}
		got := glyphAt(a.Frame(-1))
		if got != "one" && got != "two" {
			t.Fatalf("Frame(-1) = %q, must wrap onto a real frame, never crash or blank", got)
		}
	})
	t.Run("unhappy: an empty animation renders an empty sprite, never panics", func(t *testing.T) {
		var a Animation
		sp := a.Frame(3)
		if sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("empty animation returned %dx%d, want empty", sp.Width, sp.Height)
		}
		if a.Len() != 0 {
			t.Fatalf("empty animation Len() = %d, want 0", a.Len())
		}
	})
}

func TestAnimationClock(t *testing.T) {
	t.Run("happy: At(t) advances by FPS", func(t *testing.T) {
		a := Animation{Frames: []Sprite{stamp("one"), stamp("two"), stamp("tre")}, FPS: 10}
		if got := glyphAt(a.At(0)); got != "one" {
			t.Fatalf("At(0) = %q, want the first frame", got)
		}
		if got := glyphAt(a.At(0.10)); got != "two" {
			t.Fatalf("At(0.10) at 10fps = %q, want the second frame", got)
		}
		if got := glyphAt(a.At(0.35)); got != "one" {
			t.Fatalf("At(0.35) at 10fps = %q, want the wrap back to the first frame", got)
		}
	})
	t.Run("happy: FPS <= 0 falls back to the default rate", func(t *testing.T) {
		a := Animation{Frames: []Sprite{stamp("one"), stamp("two")}}
		step := 1.0/DefaultAnimationFPS + 1e-9
		if got := glyphAt(a.At(step)); got != "two" {
			t.Fatalf("At(one default step) = %q, want the second frame", got)
		}
	})
	t.Run("unhappy: time before the curtain clamps to the first frame", func(t *testing.T) {
		a := Animation{Frames: []Sprite{stamp("one"), stamp("two")}, FPS: 10}
		if got := glyphAt(a.At(-3)); got != "one" {
			t.Fatalf("At(-3) = %q, want the first frame", got)
		}
	})
}

func TestAnimationFrom(t *testing.T) {
	atlas := func() *Atlas {
		a := &Atlas{Palette: append([]PaletteEntry(nil), DefaultPalette...)}
		a.SetFrame(Size1, Heading("run1"), stamp("on1"))
		a.SetFrame(Size1, Heading("run2"), stamp("on2"))
		a.SetFrame(Size1, Heading("run3"), stamp("on3"))
		return a
	}
	t.Run("happy: frames come out in the exact order asked for", func(t *testing.T) {
		names := []Heading{"run3", "run1", "run2"}
		anim, err := AnimationFrom(atlas(), Size1, names, 12)
		if err != nil {
			t.Fatalf("AnimationFrom: %v", err)
		}
		if anim.FPS != 12 {
			t.Fatalf("FPS = %v, want 12", anim.FPS)
		}
		want := []string{"on3", "on1", "on2"}
		for i, w := range want {
			if got := glyphAt(anim.Frame(i)); got != w {
				t.Fatalf("frame %d = %q, want %q — order is the caller's, not the atlas's", i, got, w)
			}
		}
	})
	t.Run("unhappy: a missing frame is an error naming the frame, never invented", func(t *testing.T) {
		_, err := AnimationFrom(atlas(), Size1, []Heading{"run1", "ghost"}, 12)
		if err == nil {
			t.Fatal("a missing frame must be an error")
		}
		if !strings.Contains(err.Error(), "ghost") {
			t.Fatalf("the error must name the missing frame, got %v", err)
		}
	})
	t.Run("unhappy: a nil atlas is an error, not a panic", func(t *testing.T) {
		if _, err := AnimationFrom(nil, Size1, []Heading{"run1"}, 12); err == nil {
			t.Fatal("a nil atlas must be an error")
		}
	})
}
