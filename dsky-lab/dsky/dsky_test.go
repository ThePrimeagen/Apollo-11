package dsky

// Tests written FIRST. The component contract: a semi-accurate, vertical
// terminal DSKY — warning-light panel on top (including the LM's ALT/VEL
// landing-radar lights), then COMP ACTY + PROG, VERB + NOUN, and three
// signed 5-digit registers in seven-segment digits. Raw ANSI 256-color, a
// fixed footprint in every state, pure Render. Happy + unhappy throughout.

import (
	"regexp"
	"strings"
	"testing"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string { return ansiRE.ReplaceAllString(s, "") }

func lines(s string) []string { return strings.Split(s, "\n") }

// ---------------------------------------------------------------------------
// geometry & purity
// ---------------------------------------------------------------------------

func TestGeometry(t *testing.T) {
	t.Run("happy: the panel renders exactly Width x Height in every state", func(t *testing.T) {
		states := []State{
			{},
			{Prog: "63", Verb: "06", Noun: "63", R1: "+05559", R2: "-00002", R3: "+49971", CompActy: true},
			{Verb: "16", Noun: "68", R1: "+01405", R2: "+00335", R3: "-02900", Flash: true,
				Lights: Lights{Prog: true, Restart: true, Alt: true, Vel: true}},
		}
		for i, s := range states {
			out := Render(s, true)
			ls := lines(out)
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
	t.Run("unhappy: Render never mutates its input", func(t *testing.T) {
		s := State{Verb: "16", Noun: "68", R3: "-02900", Lights: Lights{Prog: true}}
		before := s
		_ = Render(s, true)
		_ = Render(s, false)
		if s != before {
			t.Fatal("Render must be pure")
		}
	})
}

// ---------------------------------------------------------------------------
// seven-segment digits
// ---------------------------------------------------------------------------

func TestSevenSegmentDigits(t *testing.T) {
	t.Run("happy: distinctive digit shapes render", func(t *testing.T) {
		// 8 lights every segment; 1 only the right verticals; 7 top + right.
		rows8 := SegRows('8')
		if rows8[0] != " _ " || rows8[1] != "|_|" || rows8[2] != "|_|" {
			t.Fatalf("digit 8 wrong: %q", rows8)
		}
		rows1 := SegRows('1')
		if strings.Contains(rows1[0]+rows1[1]+rows1[2], "_") {
			t.Fatalf("digit 1 must have no horizontal segments: %q", rows1)
		}
		rows7 := SegRows('7')
		if rows7[0] != " _ " || rows7[1] != "  |" {
			t.Fatalf("digit 7 wrong: %q", rows7)
		}
	})
	t.Run("happy: a register renders its digits and sign", func(t *testing.T) {
		out := plain(Render(State{R3: "-02900"}, true))
		if !strings.Contains(out, "_") {
			t.Fatal("register digits must render segments")
		}
		if !strings.Contains(out, "−") && !strings.Contains(out, "-") {
			t.Fatal("the minus sign must render")
		}
	})
	t.Run("happy: blank fields render no segments in their rows", func(t *testing.T) {
		empty := Render(State{}, true)
		if strings.Contains(plain(empty), "_") || strings.Contains(plain(empty), "|") {
			t.Fatal("an all-blank DSKY must show no lit segments")
		}
	})
	t.Run("unhappy: malformed register strings degrade to blanks, not panics", func(t *testing.T) {
		for _, bad := range []string{"12", "+123456789", "abc", "+1a2b3"} {
			out := Render(State{R1: bad}, true)
			if len(lines(out)) != Height {
				t.Fatalf("malformed %q broke the grid", bad)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// PROG / VERB / NOUN and flashing
// ---------------------------------------------------------------------------

func TestVerbNounFlash(t *testing.T) {
	t.Run("happy: flashing verb/noun disappears on the off phase", func(t *testing.T) {
		s := State{Verb: "16", Noun: "68", Flash: true}
		on := plain(Render(s, true))
		off := plain(Render(s, false))
		if strings.Count(on, "_") <= strings.Count(off, "_") {
			t.Fatal("the off blink phase must hide the verb/noun segments")
		}
	})
	t.Run("happy: PROG never flashes", func(t *testing.T) {
		s := State{Prog: "63", Verb: "16", Noun: "68", Flash: true}
		off := plain(Render(s, false))
		if !strings.Contains(off, "_") {
			t.Fatal("PROG digits must stay lit while verb/noun blink")
		}
	})
	t.Run("unhappy: without Flash, both phases render identically", func(t *testing.T) {
		s := State{Verb: "06", Noun: "63"}
		if Render(s, true) != Render(s, false) {
			t.Fatal("a steady display must not depend on the blink phase")
		}
	})
}

// ---------------------------------------------------------------------------
// lights
// ---------------------------------------------------------------------------

func TestLights(t *testing.T) {
	t.Run("happy: a lit warning light renders as an amber block", func(t *testing.T) {
		out := Render(State{Lights: Lights{Prog: true}}, true)
		if !strings.Contains(out, "48;5;220") {
			t.Fatal("lit PROG light must have the amber background")
		}
		if !strings.Contains(plain(out), "PROG") {
			t.Fatal("the PROG label must render")
		}
	})
	t.Run("happy: the LM radar lights ALT and VEL exist", func(t *testing.T) {
		out := plain(Render(State{}, true))
		if !strings.Contains(out, "ALT") || !strings.Contains(out, "VEL") {
			t.Fatal("the LM DSKY carries ALT and VEL landing-radar lights")
		}
	})
	t.Run("happy: COMP ACTY lights green only when active", func(t *testing.T) {
		lit := Render(State{CompActy: true}, true)
		dark := Render(State{}, true)
		if !strings.Contains(lit, "48;5;40") {
			t.Fatal("COMP ACTY must light green")
		}
		if strings.Contains(dark, "48;5;40") {
			t.Fatal("COMP ACTY must be dark when idle")
		}
	})
	t.Run("unhappy: unlit lights carry no amber background anywhere", func(t *testing.T) {
		out := Render(State{}, true)
		if strings.Contains(out, "48;5;220") {
			t.Fatal("no amber blocks may render while every light is off")
		}
	})
}
