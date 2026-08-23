package button

// Tests written FIRST. The component contract: fixed-size cell grids, raw
// ANSI 256-color output (no profile detection — captures always keep color),
// and the 1960s tactile states the lab exists to explore. Every test has a
// happy and an unhappy path.

import (
	"regexp"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string { return ansiRE.ReplaceAllString(s, "") }

func gridOf(t *testing.T, b Button) []string {
	t.Helper()
	return strings.Split(b.Render(), "\n")
}

func fgCode(n int) string { return "38;5;" + itoa(n) }
func bgCode(n int) string { return "48;5;" + itoa(n) }

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
// geometry: fixed grids, no layout shift between states
// ---------------------------------------------------------------------------

func TestGeometry(t *testing.T) {
	t.Run("happy: every style renders exactly its declared size", func(t *testing.T) {
		for _, st := range []Style{Panel, HalfCell, Protrude, Switch} {
			b := New("X", st)
			w, h := b.Size()
			lines := gridOf(t, b)
			if len(lines) != h {
				t.Fatalf("style %v: %d lines, want %d", st, len(lines), h)
			}
			for i, l := range lines {
				if got := len([]rune(plain(l))); got != w {
					t.Fatalf("style %v line %d: width %d, want %d (%q)", st, i, got, w, plain(l))
				}
			}
		}
	})
	t.Run("unhappy: pressing or focusing must never change the footprint", func(t *testing.T) {
		for _, st := range []Style{Panel, HalfCell, Protrude, Switch} {
			b := New("X", st)
			base := plainDims(b)
			b.Focused = true
			if plainDims(b) != base {
				t.Fatalf("style %v: focus changed the footprint", st)
			}
			b.On = true
			if plainDims(b) != base {
				t.Fatalf("style %v: pressing changed the footprint", st)
			}
		}
	})
}

func plainDims(b Button) [2]int {
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
// PANEL: 6x3 face, half-cursor bezel, half-fill off, 3/4-fill lit ring
// ---------------------------------------------------------------------------

func TestPanelStates(t *testing.T) {
	t.Run("happy: bezel uses half-height blocks top and bottom, shade sides", func(t *testing.T) {
		b := New("X", Panel)
		lines := gridOf(t, b)
		if !strings.Contains(plain(lines[0]), "▄") {
			t.Fatal("panel top bezel must be lower-half blocks (half-cursor sliver)")
		}
		if !strings.Contains(plain(lines[len(lines)-1]), "▀") {
			t.Fatal("panel bottom bezel must be upper-half blocks")
		}
		if !strings.Contains(plain(lines[1]), "░") {
			t.Fatal("panel side bezel must be the barely-filled ░ shade")
		}
	})
	t.Run("happy: off face is half-filled dark red", func(t *testing.T) {
		b := New("X", Panel)
		out := b.Render()
		if !strings.Contains(plain(out), "▒") {
			t.Fatal("off face must use the half-fill ▒ shade")
		}
		if !strings.Contains(out, fgCode(ColorOffFace)) {
			t.Fatal("off face must be dark red")
		}
		if strings.Contains(out, fgCode(ColorOn)) {
			t.Fatal("off face must not contain the lit orange")
		}
	})
	t.Run("happy: lit face has a 3/4-fill orange ring and a hot center", func(t *testing.T) {
		b := New("X", Panel)
		b.On = true
		out := b.Render()
		if !strings.Contains(plain(out), "▓") {
			t.Fatal("lit ring must use the 3/4-fill ▓ shade")
		}
		if !strings.Contains(out, fgCode(ColorOn)) || !strings.Contains(out, fgCode(ColorOnHot)) {
			t.Fatal("lit face needs both the orange ring and the orangey-white center")
		}
		mid := gridOf(t, b)[2]
		if !strings.Contains(mid, fgCode(ColorOnHot)) || !strings.Contains(plain(mid), "█") {
			t.Fatal("the hot center must sit in the middle row as full blocks")
		}
	})
	t.Run("unhappy: focus lifts only the bezel, never the face", func(t *testing.T) {
		b := New("X", Panel)
		dim := b.Render()
		b.Focused = true
		lit := b.Render()
		if !strings.Contains(lit, fgCode(ColorBezelFocus)) {
			t.Fatal("focused bezel must brighten")
		}
		if strings.Contains(dim, fgCode(ColorBezelFocus)) {
			t.Fatal("unfocused bezel must stay dim")
		}
		if !strings.Contains(lit, fgCode(ColorOffFace)) {
			t.Fatal("focus must not change the face color")
		}
	})
}

// ---------------------------------------------------------------------------
// HALFCELL: one cell, two colors — ▄ with face foreground on bezel background
// ---------------------------------------------------------------------------

func TestHalfCellTwoColorTrick(t *testing.T) {
	t.Run("happy: the top row carries fg AND bg color in the same cell", func(t *testing.T) {
		b := New("X", HalfCell)
		top := gridOf(t, b)[0]
		if !strings.Contains(plain(top), "▄") {
			t.Fatal("halfcell top row must be lower-half blocks")
		}
		if !strings.Contains(top, fgCode(ColorOffFace)) || !strings.Contains(top, bgCode(ColorBezel)) {
			t.Fatal("the ▄ cell must combine face foreground with bezel background — the two-color trick")
		}
	})
	t.Run("happy: lighting it recolors both halves of the trick row", func(t *testing.T) {
		b := New("X", HalfCell)
		b.On = true
		top := gridOf(t, b)[0]
		if !strings.Contains(top, fgCode(ColorOn)) {
			t.Fatal("lit halfcell must show orange in the trick row")
		}
	})
	t.Run("unhappy: the body row is a single color, no background bleed", func(t *testing.T) {
		b := New("X", HalfCell)
		body := gridOf(t, b)[1]
		if strings.Contains(body, bgCode(ColorBezel)) {
			t.Fatal("the full-block body row must not carry the bezel background")
		}
	})
}

// ---------------------------------------------------------------------------
// PROTRUDE: sticks out half a cell; pressing sinks it and lights it
// ---------------------------------------------------------------------------

func TestProtrudePressIn(t *testing.T) {
	t.Run("happy: off, the red cap sticks up and hides the bezel bar", func(t *testing.T) {
		b := New("X", Protrude)
		top := gridOf(t, b)[0]
		if !strings.Contains(plain(top), "▄") {
			t.Fatal("the protruding cap must be a half-height block")
		}
		if !strings.Contains(top, fgCode(ColorOffFace)) {
			t.Fatal("the raised cap must be dark red")
		}
		if strings.Contains(strings.TrimPrefix(top, sideBezelPrefix(b)), fgCode(ColorBezel)+"m") {
			t.Fatal("while raised, the top bezel bar must be hidden behind the cap")
		}
	})
	t.Run("happy: pressed, the cap drops — the bezel appears and the face lights", func(t *testing.T) {
		b := New("X", Protrude)
		b.On = true
		lines := gridOf(t, b)
		if !strings.Contains(lines[0], fgCode(ColorBezel)) {
			t.Fatal("pressed: the gray bezel bar must appear where the cap was")
		}
		if strings.Contains(lines[0], fgCode(ColorOffFace)) || strings.Contains(lines[0], fgCode(ColorOn)) {
			t.Fatal("pressed: no button color may remain in the top half-row")
		}
		if !strings.Contains(lines[1], fgCode(ColorOn)) {
			t.Fatal("pressed: the face must light orange")
		}
		if !strings.Contains(lines[2], fgCode(ColorOnShade)) {
			t.Fatal("pressed: the lower face row shades darker for depth")
		}
	})
	t.Run("unhappy: off, the face rows are red, not orange", func(t *testing.T) {
		b := New("X", Protrude)
		out := b.Render()
		if strings.Contains(out, fgCode(ColorOn)) || strings.Contains(out, fgCode(ColorOnShade)) {
			t.Fatal("an unpressed protrude button must show no orange at all")
		}
	})
}

func sideBezelPrefix(b Button) string { return "" }

// ---------------------------------------------------------------------------
// SWITCH: cockpit toggle — lever position and two-tone glow
// ---------------------------------------------------------------------------

func TestSwitchLever(t *testing.T) {
	t.Run("happy: off, the lever sits at the bottom in the ~50%-darker red", func(t *testing.T) {
		b := New("X", Switch)
		lines := gridOf(t, b)
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
		b := New("X", Switch)
		b.On = true
		lines := gridOf(t, b)
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
			b := New("X", Switch)
			b.On = on
			b.Focused = true
			if !strings.Contains(b.Render(), fgCode(ColorBezelFocus)) {
				t.Fatalf("focused switch frame must brighten (on=%v)", on)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// behavior
// ---------------------------------------------------------------------------

func TestToggle(t *testing.T) {
	t.Run("happy: Toggle flips the state both ways", func(t *testing.T) {
		b := New("X", Panel)
		b.Toggle()
		if !b.On {
			t.Fatal("first toggle must switch on")
		}
		b.Toggle()
		if b.On {
			t.Fatal("second toggle must switch off")
		}
	})
	t.Run("unhappy: Render is pure — it never mutates the button", func(t *testing.T) {
		b := New("X", Switch)
		before := b
		_ = b.Render()
		if b != before {
			t.Fatal("Render must not change any state")
		}
	})
}
