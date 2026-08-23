package button

// Tests written FIRST. The component contract: a fixed-size cockpit toggle
// switch rendered as raw ANSI 256-color (no profile detection — captures
// always keep color). Lever down = ~50%-darker red; flicked up = lit in two
// orange tones with a dim glow left in the slot. Every test has a happy and
// an unhappy path.

import (
	"regexp"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string { return ansiRE.ReplaceAllString(s, "") }

func fgCode(n int) string { return "38;5;" + itoa(n) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

// ---------------------------------------------------------------------------
// geometry: a fixed grid, no layout shift between states
// ---------------------------------------------------------------------------

func TestGeometry(t *testing.T) {
	t.Run("happy: the switch renders exactly its declared size", func(t *testing.T) {
		b := NewSwitch("X")
		w, h := b.Size()
		lines := strings.Split(b.Render(), "\n")
		if len(lines) != h {
			t.Fatalf("%d lines, want %d", len(lines), h)
		}
		for i, l := range lines {
			if got := len([]rune(plain(l))); got != w {
				t.Fatalf("line %d: width %d, want %d (%q)", i, got, w, plain(l))
			}
		}
	})
	t.Run("unhappy: flicking or focusing must never change the footprint", func(t *testing.T) {
		b := NewSwitch("X")
		base := plainDims(b)
		b.Focused = true
		if plainDims(b) != base {
			t.Fatal("focus changed the footprint")
		}
		b.On = true
		if plainDims(b) != base {
			t.Fatal("flicking changed the footprint")
		}
	})
}

func plainDims(b Switch) [2]int {
	lines := strings.Split(b.Render(), "\n")
	w := 0
	for _, l := range lines {
		if n := len([]rune(plain(l))); n > w {
			w = n
		}
	}
	return [2]int{w, len(lines)}
}

// ---------------------------------------------------------------------------
// lever states
// ---------------------------------------------------------------------------

func TestSwitchLever(t *testing.T) {
	t.Run("happy: off, the lever sits at the bottom in the ~50%-darker red", func(t *testing.T) {
		b := NewSwitch("X")
		lines := strings.Split(b.Render(), "\n")
		bottom := lines[len(lines)-2] + lines[len(lines)-1]
		if !strings.Contains(bottom, fgCode(ColorLeverOff)) {
			t.Fatal("off lever must occupy the lower rows")
		}
		// The held-down lever is a proper dark red (xterm 88 ≈ half the
		// brightness of the original dull pink 131), per design review.
		if !strings.Contains(bottom, fgCode(88)) {
			t.Fatal("off lever must be the darkened red (xterm 88)")
		}
		if strings.Contains(b.Render(), fgCode(131)) {
			t.Fatal("the old brighter dull-pink lever (xterm 131) must be gone")
		}
		if strings.Contains(b.Render(), fgCode(ColorOn)) {
			t.Fatal("an off switch must show no lit orange")
		}
	})
	t.Run("happy: on, the lever flips up and lights in two orange tones", func(t *testing.T) {
		b := NewSwitch("X")
		b.On = true
		lines := strings.Split(b.Render(), "\n")
		top := lines[0] + lines[1]
		if !strings.Contains(top, fgCode(ColorOnTip)) || !strings.Contains(top, fgCode(ColorOn)) {
			t.Fatal("lit lever must carry both orange tones (tip + body) up top")
		}
		bottom := lines[len(lines)-2] + lines[len(lines)-1]
		if !strings.Contains(bottom, fgCode(ColorOnGlow)) {
			t.Fatal("the vacated slot must keep a dim orange glow")
		}
	})
	t.Run("unhappy: focus brightens the frame on and off alike", func(t *testing.T) {
		for _, on := range []bool{false, true} {
			b := NewSwitch("X")
			b.On = on
			b.Focused = true
			if !strings.Contains(b.Render(), fgCode(ColorBezelFocus)) {
				t.Fatalf("focused switch frame must brighten (on=%v)", on)
			}
			b.Focused = false
			if strings.Contains(b.Render(), fgCode(ColorBezelFocus)) {
				t.Fatalf("unfocused switch frame must stay dim (on=%v)", on)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// behavior
// ---------------------------------------------------------------------------

func TestToggle(t *testing.T) {
	t.Run("happy: Toggle flips the state both ways", func(t *testing.T) {
		b := NewSwitch("X")
		b.Toggle()
		if !b.On {
			t.Fatal("first toggle must switch on")
		}
		b.Toggle()
		if b.On {
			t.Fatal("second toggle must switch off")
		}
	})
	t.Run("unhappy: Render is pure — it never mutates the switch", func(t *testing.T) {
		b := NewSwitch("X")
		before := b
		_ = b.Render()
		if b != before {
			t.Fatal("Render must not change any state")
		}
	})
}
