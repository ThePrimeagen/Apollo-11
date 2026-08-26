package sprite

// Tests written FIRST. Atlas frames have always been keyed by compass
// heading, but a heading is just a name — an animation atlas keys its
// frames by pose ("run1", "jump", "pole2"). Unmarshal already accepts
// any name; these tests make Marshal and FrameNames honor them too,
// so the editor can open, edit, and save a pose atlas without silently
// dropping frames.

import (
	"strings"
	"testing"
)

// poseAtlas is a small atlas keyed by pose names, plus one compass
// frame so ordering between the two families is observable.
func poseAtlas() *Atlas {
	a := &Atlas{Palette: append([]PaletteEntry(nil), DefaultPalette...)}
	a.SetFrame(Size1, Heading("run2"), stamp("rn2"))
	a.SetFrame(Size1, Heading("run1"), stamp("rn1"))
	a.SetFrame(Size1, Heading("jump"), stamp("jmp"))
	a.SetFrame(Size1, N, stamp("nnn"))
	return a
}

func TestNamedFrameRoundTrip(t *testing.T) {
	t.Run("happy: pose-named frames survive Marshal and Unmarshal", func(t *testing.T) {
		raw, err := poseAtlas().Marshal()
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		back, err := Unmarshal(raw)
		if err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		for _, name := range []Heading{"run1", "run2", "jump", N} {
			sp, ok := back.Frame(Size1, name)
			if !ok {
				t.Fatalf("frame %q was dropped on the round trip", name)
			}
			want := map[Heading]string{"run1": "rn1", "run2": "rn2", "jump": "jmp", N: "nnn"}[name]
			if got := glyphAt(sp); got != want {
				t.Fatalf("frame %q came back as %q, want %q", name, got, want)
			}
		}
	})
	t.Run("happy: compass-only atlases keep their exact file shape", func(t *testing.T) {
		a := &Atlas{Palette: append([]PaletteEntry(nil), DefaultPalette...)}
		a.SetFrame(Size1, N, stamp("nnn"))
		a.SetFrame(Size1, E, stamp("eee"))
		raw, err := a.Marshal()
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		doc := string(raw)
		if !strings.Contains(doc, `"N"`) || !strings.Contains(doc, `"E"`) {
			t.Fatalf("compass frames must still be written: %s", doc)
		}
		if strings.Contains(doc, `"run1"`) {
			t.Fatal("no invented frames on a compass-only atlas")
		}
	})
	t.Run("unhappy: a bad pose frame still errors with its name", func(t *testing.T) {
		bad := []byte(`{"palette":[],"frames":{"1":{"run1":{"glyphs":["ab"],"fg":["S"],"bg":[".."]}}}}`)
		_, err := Unmarshal(bad)
		if err == nil {
			t.Fatal("ragged masks must be an error")
		}
		if !strings.Contains(err.Error(), "run1") {
			t.Fatalf("the error must name the bad frame, got %v", err)
		}
	})
}

func TestFrameNames(t *testing.T) {
	t.Run("happy: compass canonical order first, then extra names sorted", func(t *testing.T) {
		a := poseAtlas()
		a.SetFrame(Size1, E, stamp("eee"))
		got := a.FrameNames(Size1)
		want := []Heading{N, E, "jump", "run1", "run2"}
		if len(got) != len(want) {
			t.Fatalf("FrameNames = %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("FrameNames[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
			}
		}
	})
	t.Run("unhappy: a nil atlas or empty size lists nothing, never panics", func(t *testing.T) {
		var a *Atlas
		if got := a.FrameNames(Size1); len(got) != 0 {
			t.Fatalf("nil atlas FrameNames = %v, want empty", got)
		}
		b := &Atlas{}
		if got := b.FrameNames(Size3); len(got) != 0 {
			t.Fatalf("empty atlas FrameNames = %v, want empty", got)
		}
	})
}
