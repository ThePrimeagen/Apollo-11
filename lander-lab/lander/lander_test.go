package lander

// Tests written FIRST. The component contract: a fixed 40×30 cell descent
// view — phase + time up top, an altitude-scaled fall (square-root scale so
// the final thousand feet stay readable), the LM sprite rotating with the
// phase (horizontal in P63, pitched over in P64, vertical in P66, flame off
// once landed), a four-glyph starfield that scrolls right-to-left while
// flying and freezes on the surface, persistent alarm markers at their
// altitudes, the lunar surface across the bottom, an event caption
// underneath. Raw ANSI 256-color, pure Render. Happy + unhappy throughout.

import (
	"regexp"
	"sort"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string { return ansiRE.ReplaceAllString(s, "") }

func render(s State) []string {
	return strings.Split(Render(s), "\n")
}

func base() State {
	return State{
		AltFt: 49971, VelFps: 5560, TimeSec: 0,
		Phase: "P63 BRAKING", Attitude: Horizontal,
		Event: "PDI — ignition",
	}
}

// ---------------------------------------------------------------------------
// geometry & purity
// ---------------------------------------------------------------------------

func TestGeometry(t *testing.T) {
	t.Run("happy: every state renders exactly Width x Height", func(t *testing.T) {
		states := []State{
			base(),
			{AltFt: 770, VelFps: 55, TimeSec: 594, Phase: "P64 APPROACH", Attitude: Tilted,
				Alarms: []Alarm{{"1202", 33500}, {"1201", 3000}, {"1202", 770}}},
			{AltFt: 0, Phase: "P66 LANDED", Attitude: Landed, Event: "CONTACT LIGHT"},
		}
		for i, s := range states {
			ls := render(s)
			if len(ls) != Height {
				t.Fatalf("state %d: %d lines, want %d", i, len(ls), Height)
			}
			for j, l := range ls {
				if got := len([]rune(plain(l))); got != Width {
					t.Fatalf("state %d line %d: width %d, want %d (%q)", i, j, got, Width, plain(l))
				}
			}
		}
	})
	t.Run("unhappy: absurd altitudes clamp instead of breaking the grid", func(t *testing.T) {
		for _, alt := range []float64{-500, 999999} {
			s := base()
			s.AltFt = alt
			if got := len(render(s)); got != Height {
				t.Fatalf("alt %v broke the grid: %d lines", alt, got)
			}
		}
	})
	t.Run("unhappy: Render never mutates its input", func(t *testing.T) {
		s := base()
		s.Alarms = []Alarm{{"1202", 33500}}
		before := s.Alarms[0]
		_ = Render(s)
		if s.Alarms[0] != before {
			t.Fatal("Render must be pure")
		}
	})
}

// ---------------------------------------------------------------------------
// the fall: altitude maps to rows, sqrt-scaled, moon at the bottom
// ---------------------------------------------------------------------------

// spriteRow finds the first row containing the LM body glyph.
func spriteRow(t *testing.T, s State) int {
	t.Helper()
	for i, l := range render(s) {
		if strings.Contains(plain(l), "██") {
			return i
		}
	}
	t.Fatal("lander sprite not found")
	return -1
}

func TestAltitudeScale(t *testing.T) {
	t.Run("happy: lower altitude renders lower on screen", func(t *testing.T) {
		high, mid, low := base(), base(), base()
		mid.AltFt = 7400
		low.AltFt = 770
		rh, rm, rl := spriteRow(t, high), spriteRow(t, mid), spriteRow(t, low)
		if !(rh < rm && rm < rl) {
			t.Fatalf("rows must descend with altitude: 49971→%d, 7400→%d, 770→%d", rh, rm, rl)
		}
	})
	t.Run("happy: the sqrt scale keeps the last thousand feet apart", func(t *testing.T) {
		a, b := base(), base()
		a.AltFt = 3000
		b.AltFt = 770
		if spriteRow(t, a) == spriteRow(t, b) {
			t.Fatal("3000ft and 770ft must land on different rows")
		}
	})
	t.Run("happy: the moon surface spans the bottom", func(t *testing.T) {
		ls := render(base())
		surface := plain(ls[Height-2])
		if !strings.ContainsAny(surface, "▁▂▃▄") {
			t.Fatalf("the surface row must be lunar terrain, got %q", surface)
		}
	})
	t.Run("happy: touchdown parks the lander on the surface", func(t *testing.T) {
		s := base()
		s.AltFt = 0
		s.Attitude = Landed
		if got := spriteRow(t, s); got < Height-6 {
			t.Fatalf("a landed LM must sit at the surface, got row %d", got)
		}
	})
}

// ---------------------------------------------------------------------------
// rotation: the sprite tracks the phase attitude
// ---------------------------------------------------------------------------

func TestAttitudes(t *testing.T) {
	t.Run("happy: each attitude has its own two-stage silhouette", func(t *testing.T) {
		s := base()
		s.Attitude = Horizontal
		v := plain(Render(s))
		if !strings.Contains(v, "◢▟▓▓▙") {
			t.Fatal("horizontal braking must show the rigidly rotated descent stage")
		}
		if !strings.Contains(v, "≈") {
			t.Fatal("the P63 burn needs its plume")
		}
		s.Attitude = Tilted
		v = plain(Render(s))
		if !strings.Contains(v, "◢▟▓██▓▙") {
			t.Fatal("the pitched-over stack must render rigidly rotated in P64")
		}
		s.Attitude = Vertical
		v = plain(Render(s))
		if !strings.Contains(v, "▟▄▙") {
			t.Fatal("the vertical silhouette must show the descent engine bell")
		}
		if strings.Count(v, "≈")+strings.Count(v, "~") < 3 {
			t.Fatal("the vertical descent needs a straight-down plume under the bell")
		}
		if !strings.Contains(v, "▟▓████▓▙") {
			t.Fatal("the foil descent stage must be wider than the cabin")
		}
	})
	t.Run("happy: sprites and color masks agree — rectangular, plume-aligned", func(t *testing.T) {
		for att, sp := range sprites {
			mask, ok := colorMasks[att]
			if !ok {
				t.Fatalf("attitude %v missing a color mask", att)
			}
			for i := range sp {
				sr, mr := []rune(sp[i]), []rune(mask[i])
				if len(sr) != 13 || len(mr) != 13 {
					t.Fatalf("attitude %v row %d: sprite %d / mask %d runes, want 13", att, i, len(sr), len(mr))
				}
				for j := range sr {
					if (sr[j] == '~') != (mr[j] == 'P') {
						t.Fatalf("attitude %v row %d col %d: plume cells must map to P", att, i, j)
					}
				}
			}
		}
	})
	t.Run("happy: the craft renders in materials — gold foil, windows, steel", func(t *testing.T) {
		s := base()
		s.Attitude = Vertical
		out := Render(s)
		for _, code := range []string{"38;5;178", "38;5;24", "38;5;245", "38;5;208"} {
			if !strings.Contains(out, code) {
				t.Fatalf("vertical LM must carry material color %s", code)
			}
		}
	})
	t.Run("happy: legs and footpads render in every attitude (above the surface)", func(t *testing.T) {
		for _, a := range []Attitude{Horizontal, Tilted, Vertical, Landed} {
			s := base()
			s.Attitude = a
			sky := strings.Join(render(s)[:Height-2], "\n")
			if !strings.Contains(plain(sky), "▁") {
				t.Fatalf("attitude %v must show footpads", a)
			}
		}
	})
	t.Run("unhappy: a landed LM shows no flame", func(t *testing.T) {
		s := base()
		s.AltFt = 0
		s.Attitude = Landed
		if strings.Contains(plain(Render(s)), "≈") {
			t.Fatal("the engine is off after touchdown — no flame anywhere")
		}
	})
}

// ---------------------------------------------------------------------------
// alarm markers: persistent, at their own altitudes
// ---------------------------------------------------------------------------

func TestAlarmMarkers(t *testing.T) {
	t.Run("happy: markers render their codes at distinct altitudes", func(t *testing.T) {
		s := base()
		s.AltFt = 770
		s.Alarms = []Alarm{{"1202", 33500}, {"1201", 3000}}
		v := render(s)
		row1202, row1201 := -1, -1
		for i, l := range v {
			p := plain(l)
			if strings.Contains(p, "1202") {
				row1202 = i
			}
			if strings.Contains(p, "1201") {
				row1201 = i
			}
		}
		if row1202 < 0 || row1201 < 0 {
			t.Fatal("both alarm markers must render")
		}
		if row1202 >= row1201 {
			t.Fatalf("the 33,500ft 1202 must sit above the 3,000ft 1201 (%d vs %d)", row1202, row1201)
		}
	})
	t.Run("happy: markers at colliding altitudes nudge apart — all five stay visible", func(t *testing.T) {
		s := base()
		s.Alarms = []Alarm{
			{"1202", 33500}, {"1202", 30900}, // these round to the same row
			{"1201", 3000}, {"1202", 2000}, {"1202", 770},
		}
		rows := 0
		for _, l := range render(s) {
			p := plain(l)
			if strings.Contains(p, "1202") || strings.Contains(p, "1201") {
				rows++
			}
		}
		if rows != 5 {
			t.Fatalf("all five alarm markers must render on their own rows, got %d", rows)
		}
	})
	t.Run("unhappy: no alarms, no markers", func(t *testing.T) {
		if v := plain(Render(base())); strings.Contains(v, "1202") || strings.Contains(v, "1201") {
			t.Fatal("no markers may render before any alarm")
		}
	})
}

// ---------------------------------------------------------------------------
// captions
// ---------------------------------------------------------------------------

func TestCountdown(t *testing.T) {
	t.Run("happy: the touchdown countdown renders and ticks with the state", func(t *testing.T) {
		s := base()
		s.LandInSec = 128
		if !strings.Contains(plain(Render(s)), "▼ 128s") {
			t.Fatal("the countdown must render as ▼ NNNs")
		}
		s.LandInSec = 127
		if !strings.Contains(plain(Render(s)), "▼ 127s") {
			t.Fatal("the countdown must follow the state")
		}
	})
	t.Run("unhappy: a landed craft shows no countdown", func(t *testing.T) {
		s := base()
		s.AltFt = 0
		s.Attitude = Landed
		s.LandInSec = 0
		if strings.Contains(plain(Render(s)), "▼") {
			t.Fatal("no countdown after touchdown")
		}
	})
}

func TestPlumeFlicker(t *testing.T) {
	t.Run("happy: the plume animates across ticks while the body holds still", func(t *testing.T) {
		s := base()
		s.Attitude = Vertical
		s.Tick = 0
		a := plain(Render(s))
		s.Tick = 1
		b := plain(Render(s))
		if a == b {
			t.Fatal("the plume must flicker frame to frame")
		}
		if stripAnim(a) != stripAnim(b) {
			t.Fatal("only the plume and the starfield may animate — the body must hold perfectly still")
		}
	})
	t.Run("unhappy: a landed craft does not flicker", func(t *testing.T) {
		s := base()
		s.AltFt = 0
		s.Attitude = Landed
		s.Tick = 0
		a := Render(s)
		s.Tick = 1
		if a != Render(s) {
			t.Fatal("nothing may animate after engine cutoff")
		}
	})
}

// Four one-cell background stars, far/dim → near/bright. They are glyphs
// only — no per-star twinkle. Motion comes from the field flying past.
var skyStars = []rune{'·', '˚', '*', '✦'}

func stripAnim(v string) string {
	v = strings.ReplaceAll(v, "≈", " ")
	v = strings.ReplaceAll(v, "~", " ")
	for _, g := range skyStars {
		v = strings.ReplaceAll(v, string(g), " ")
	}
	return v
}

func glyphCols(line string, g rune) []int {
	var cols []int
	for i, r := range []rune(plain(line)) {
		if r == g {
			cols = append(cols, i)
		}
	}
	return cols
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	as, bs := append([]int(nil), a...), append([]int(nil), b...)
	sort.Ints(as)
	sort.Ints(bs)
	for i := range as {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

// shiftLeft wraps a sky column one cell left. Column 0 is the altitude
// axis, so the starfield lives in [1, Width) and wraps 1 ← Width-1.
func shiftLeft(col int) int {
	col--
	if col < 1 {
		return Width - 1
	}
	return col
}

func shiftCols(cols []int, steps int) []int {
	out := make([]int, len(cols))
	copy(out, cols)
	for n := 0; n < steps; n++ {
		for i := range out {
			out[i] = shiftLeft(out[i])
		}
	}
	return out
}

func TestStarGlyphs(t *testing.T) {
	t.Run("happy: four distinct one-cell star glyphs", func(t *testing.T) {
		if len(skyStars) != 4 {
			t.Fatalf("want 4 background stars, got %d", len(skyStars))
		}
		seen := map[rune]bool{}
		for _, g := range skyStars {
			if seen[g] {
				t.Fatalf("duplicate star glyph %q", string(g))
			}
			seen[g] = true
			if n := len([]rune(string(g))); n != 1 {
				t.Fatalf("star %q must be one cell, got %d runes", string(g), n)
			}
		}
	})
	t.Run("happy: every glyph appears in the sky at PDI", func(t *testing.T) {
		v := plain(Render(base()))
		for _, g := range skyStars {
			if !strings.ContainsRune(v, g) {
				t.Fatalf("background star %q missing from the sky", string(g))
			}
		}
	})
	t.Run("unhappy: stars never sit in the telemetry header", func(t *testing.T) {
		ls := render(base())
		for _, row := range []int{0, 1} {
			line := plain(ls[row])
			for _, g := range skyStars {
				if strings.ContainsRune(line, g) {
					t.Fatalf("star %q sat in header row %d (%q)", string(g), row, line)
				}
			}
		}
	})
}

func TestStarfieldSky(t *testing.T) {
	t.Run("happy: stars live in the sky, never on the moon or the caption", func(t *testing.T) {
		ls := render(base())
		surface, caption := plain(ls[Height-2]), plain(ls[Height-1])
		for _, g := range skyStars {
			if strings.ContainsRune(surface, g) {
				t.Fatalf("star %q sat on the lunar surface", string(g))
			}
			if strings.ContainsRune(caption, g) {
				t.Fatalf("star %q sat on the event caption", string(g))
			}
		}
	})
	t.Run("happy: stars never overwrite the lander hull", func(t *testing.T) {
		s := base()
		s.Attitude = Vertical
		v := plain(Render(s))
		if !strings.Contains(v, "██") {
			t.Fatal("the lander hull must still render in front of the stars")
		}
		if !strings.Contains(v, "▟▓████▓▙") {
			t.Fatal("the foil descent stage must sit in front of the starfield")
		}
	})
	t.Run("unhappy: stars never overwrite an alarm marker", func(t *testing.T) {
		s := base()
		s.Alarms = []Alarm{{"1202", 33500}}
		if !strings.Contains(plain(Render(s)), "◄ 1202") {
			t.Fatal("the alarm marker must stay intact in front of the starfield")
		}
	})
	t.Run("unhappy: the altitude axis is not a star column", func(t *testing.T) {
		for i, l := range render(base()) {
			if i < 2 || i >= Height-2 {
				continue
			}
			runes := []rune(plain(l))
			if len(runes) == 0 {
				continue
			}
			for _, g := range skyStars {
				if runes[0] == g {
					t.Fatalf("star %q sat on the altitude axis at row %d", string(g), i)
				}
			}
		}
	})
}

func TestStarflight(t *testing.T) {
	t.Run("happy: near stars (✦) drift right-to-left one cell per tick", func(t *testing.T) {
		s := base()
		s.Tick = 0
		row := clearSkyRow(t, s)
		before := glyphCols(render(s)[row], '✦')
		if len(before) == 0 {
			t.Fatal("need at least one near star on an empty sky row to watch it fly")
		}
		s.Tick = 1
		after := glyphCols(render(s)[row], '✦')
		if equalInts(before, after) {
			t.Fatal("near stars must move as we fly")
		}
		if !equalInts(after, shiftCols(before, 1)) {
			t.Fatalf("✦ must shift left one cell per tick: tick0=%v tick1=%v", before, after)
		}
	})
	t.Run("happy: a near star wrapping off the left re-enters from the right", func(t *testing.T) {
		s := base()
		row := clearSkyRow(t, s)
		s.Tick = 0
		start := glyphCols(render(s)[row], '✦')
		if len(start) == 0 {
			t.Fatal("need a near star to wrap")
		}
		// fly until the leftmost ✦ of this row has wrapped
		left := start[0]
		for _, c := range start {
			if c < left {
				left = c
			}
		}
		s.Tick = left // that star was at `left`; after `left` ticks it wraps to Width-1
		got := glyphCols(render(s)[row], '✦')
		want := shiftCols(start, left)
		if !equalInts(got, want) {
			t.Fatalf("wrapping fly-by: want %v (with %d at the right edge), got %v", want, Width-1, got)
		}
		foundEdge := false
		for _, c := range got {
			if c == Width-1 {
				foundEdge = true
			}
		}
		if !foundEdge {
			t.Fatal("a star leaving the left edge must reappear at the right")
		}
	})
	t.Run("happy: far stars (·) parallax — they hold still while near stars fly", func(t *testing.T) {
		s := base()
		row := clearSkyRow(t, s)
		s.Tick = 0
		far0 := glyphCols(render(s)[row], '·')
		near0 := glyphCols(render(s)[row], '✦')
		if len(far0) == 0 || len(near0) == 0 {
			t.Fatal("need both a far and a near star on the empty sky row")
		}
		s.Tick = 1
		far1 := glyphCols(render(s)[row], '·')
		near1 := glyphCols(render(s)[row], '✦')
		if !equalInts(far0, far1) {
			t.Fatalf("far stars must hold still for a tick (parallax), %v -> %v", far0, far1)
		}
		if equalInts(near0, near1) {
			t.Fatal("near stars must already be flying on the first tick")
		}
	})
	t.Run("unhappy: a landed craft's starfield is frozen", func(t *testing.T) {
		s := base()
		s.AltFt = 0
		s.Attitude = Landed
		s.Tick = 0
		a := plain(Render(s))
		s.Tick = 16
		if plain(Render(s)) != a {
			t.Fatal("stars must not keep flying after touchdown")
		}
		for _, g := range skyStars {
			if !strings.ContainsRune(a, g) {
				t.Fatalf("the night sky remains after landing — missing %q", string(g))
			}
		}
	})
}

// clearSkyRow finds a mid-sky row at PDI that has no lander hull, so star
// motion can be observed without occlusion.
func clearSkyRow(t *testing.T, s State) int {
	t.Helper()
	for row := 12; row <= 22; row++ {
		line := plain(render(s)[row])
		if strings.ContainsAny(line, "█▓▙▟▛▜") {
			continue
		}
		hasNear, hasFar := false, false
		for _, r := range []rune(line) {
			if r == '✦' {
				hasNear = true
			}
			if r == '·' {
				hasFar = true
			}
		}
		if hasNear && hasFar {
			return row
		}
	}
	t.Fatal("no empty sky row showing both far (·) and near (✦) stars")
	return -1
}

func TestCaptions(t *testing.T) {
	t.Run("happy: phase, time, altitude, velocity, and event render", func(t *testing.T) {
		s := base()
		s.TimeSec = 316
		v := plain(Render(s))
		for _, want := range []string{"P63 BRAKING", "T+316s", "49971", "5560", "PDI"} {
			if !strings.Contains(v, want) {
				t.Fatalf("caption missing %q", want)
			}
		}
	})
	t.Run("happy: zero velocity still prints a zero", func(t *testing.T) {
		s := base()
		s.VelFps = 0
		if !strings.Contains(plain(Render(s)), "0ft/s") {
			t.Fatal("VEL must print 0 at touchdown, not blank")
		}
	})
	t.Run("unhappy: an empty event renders a blank caption, not garbage", func(t *testing.T) {
		s := base()
		s.Event = ""
		if got := len(render(s)); got != Height {
			t.Fatal("empty caption must keep the grid")
		}
	})
}
