package dsky

// Tests written FIRST: the DSKY panel carries its own compact keypad,
// pressed through exposed functions — no clocks, no animation, every
// press repaints instantly. VERB and NOUN open a two-digit entry: the
// field blanks and typed digits fill it left to right. An entry closes
// however you leave it: complete (two digits) it commits — by ENTR or
// by opening the next entry — incomplete it falls back to the value the
// field held before. CLR always cancels back to that value. RSET wipes
// the caution lights. Digits without an open entry go nowhere.

import (
	"testing"

	lab "github.com/theprimeagen/apollo-11/dsky-lab/dsky"
)

func TestKeyFromRune(t *testing.T) {
	t.Run("happy: letters map case-insensitively and digits map to themselves", func(t *testing.T) {
		want := map[rune]Key{
			'v': KeyVerb, 'V': KeyVerb,
			'n': KeyNoun, 'N': KeyNoun,
			'e': KeyEnter, 'E': KeyEnter,
			'c': KeyClear, 'C': KeyClear,
			'r': KeyReset, 'R': KeyReset,
		}
		for r, k := range want {
			got, ok := KeyFromRune(r)
			if !ok || got != k {
				t.Fatalf("KeyFromRune(%q) = %v/%v, want %v", r, got, ok, k)
			}
		}
		for r := '0'; r <= '9'; r++ {
			got, ok := KeyFromRune(r)
			if !ok || got != Key(r) {
				t.Fatalf("KeyFromRune(%q) = %v/%v, want the digit itself", r, got, ok)
			}
		}
	})
	t.Run("unhappy: runes the DSKY has no key for are refused", func(t *testing.T) {
		for _, r := range []rune{'x', 'q', '+', '-', ' ', '#', 'é'} {
			if k, ok := KeyFromRune(r); ok {
				t.Fatalf("KeyFromRune(%q) = %v, want a refusal", r, k)
			}
		}
	})
}

// pressAll feeds a string of keyboard runes through KeyFromRune,
// skipping any the DSKY has no key for — the way a host would.
func pressAll(p *Panel, keys string) {
	for _, r := range keys {
		if k, ok := KeyFromRune(r); ok {
			p.Press(k)
		}
	}
}

func TestVerbNounEntry(t *testing.T) {
	t.Run("happy: VERB opens a two-digit entry that fills left to right and ENTR commits", func(t *testing.T) {
		p := NewPanel(lab.State{Prog: "63"})
		p.Press(KeyVerb)
		if p.State.Verb != "" {
			t.Fatalf("VERB must blank the verb for entry, got %q", p.State.Verb)
		}
		p.Press(Key('1'))
		if p.State.Verb != "1 " {
			t.Fatalf("the first digit fills the left slot, got %q", p.State.Verb)
		}
		p.Press(Key('6'))
		if p.State.Verb != "16" {
			t.Fatalf("the second digit completes the pair, got %q", p.State.Verb)
		}
		p.Press(KeyEnter)
		if p.State.Verb != "16" {
			t.Fatalf("ENTR must commit the verb, got %q", p.State.Verb)
		}
		p.Press(Key('9'))
		if p.State.Verb != "16" {
			t.Fatalf("digits after the commit go nowhere, got %q", p.State.Verb)
		}
	})
	t.Run("happy: NOUN runs its own entry the same way, registers and PROG untouched", func(t *testing.T) {
		p := NewPanel(MonitorState())
		pressAll(p, "n33")
		if p.State.Noun != "33" {
			t.Fatalf("noun entry landed %q, want 33", p.State.Noun)
		}
		p.Press(KeyEnter)
		if p.State.Noun != "33" || p.State.Verb != "16" {
			t.Fatalf("commit holds noun 33 beside verb 16, got %q/%q", p.State.Verb, p.State.Noun)
		}
		mon := MonitorState()
		if p.State.R1 != mon.R1 || p.State.R2 != mon.R2 || p.State.R3 != mon.R3 || p.State.Prog != mon.Prog {
			t.Fatal("an entry must never disturb the registers or PROG")
		}
	})
	t.Run("unhappy: digits with no entry open go nowhere", func(t *testing.T) {
		p := NewPanel(MonitorState())
		before := p.State
		pressAll(p, "42")
		if p.State != before {
			t.Fatalf("stray digits changed the state: %+v -> %+v", before, p.State)
		}
	})
	t.Run("unhappy: a third digit mid-entry is ignored", func(t *testing.T) {
		p := NewPanel(lab.State{})
		pressAll(p, "v169")
		if p.State.Verb != "16" {
			t.Fatalf("the pair is full — got %q, want 16", p.State.Verb)
		}
		p.Press(KeyEnter)
		if p.State.Verb != "16" {
			t.Fatalf("the ignored digit must not poison the commit, got %q", p.State.Verb)
		}
	})
}

func TestEntryRecovery(t *testing.T) {
	t.Run("happy: CLR cancels the entry and restores the old value", func(t *testing.T) {
		p := NewPanel(MonitorState())
		pressAll(p, "n4")
		if p.State.Noun != "4 " {
			t.Fatalf("test premise: mid-entry noun %q, want '4 '", p.State.Noun)
		}
		p.Press(KeyClear)
		if p.State.Noun != "68" {
			t.Fatalf("CLR must bring back the old noun, got %q", p.State.Noun)
		}
	})
	t.Run("happy: opening the next entry commits a complete field — v16n68 types naturally", func(t *testing.T) {
		p := NewPanel(lab.State{Prog: "63"})
		pressAll(p, "v16n68")
		if p.State.Verb != "16" {
			t.Fatalf("opening NOUN must commit the finished verb, got %q", p.State.Verb)
		}
		if p.State.Noun != "68" {
			t.Fatalf("noun entry landed %q, want 68", p.State.Noun)
		}
		p.Press(KeyEnter)
		if p.State.Verb != "16" || p.State.Noun != "68" {
			t.Fatalf("the commit holds V16 N68, got %q/%q", p.State.Verb, p.State.Noun)
		}
	})
	t.Run("happy: pressing VERB again restarts the entry without losing the original", func(t *testing.T) {
		p := NewPanel(MonitorState())
		pressAll(p, "v9")
		p.Press(KeyVerb)
		if p.State.Verb != "" {
			t.Fatalf("a fresh VERB must blank the field again, got %q", p.State.Verb)
		}
		p.Press(KeyEnter)
		if p.State.Verb != "16" {
			t.Fatalf("abandoning the restarted entry must restore the original, got %q", p.State.Verb)
		}
	})
	t.Run("unhappy: ENTR on an incomplete entry rejects it back to the old value", func(t *testing.T) {
		p := NewPanel(MonitorState())
		pressAll(p, "v9")
		if p.State.Verb != "9 " {
			t.Fatalf("test premise: mid-entry verb %q, want '9 '", p.State.Verb)
		}
		p.Press(KeyEnter)
		if p.State.Verb != "16" {
			t.Fatalf("an incomplete commit must fall back to the old verb, got %q", p.State.Verb)
		}
	})
	t.Run("unhappy: switching entries with an incomplete field restores it", func(t *testing.T) {
		p := NewPanel(MonitorState())
		pressAll(p, "v2n")
		if p.State.Verb != "16" {
			t.Fatalf("the abandoned half-entry must restore the verb, got %q", p.State.Verb)
		}
		if p.State.Noun != "" {
			t.Fatalf("the noun entry must be open and blank, got %q", p.State.Noun)
		}
		pressAll(p, "42")
		if p.State.Noun != "42" {
			t.Fatalf("the new entry still takes digits, got %q", p.State.Noun)
		}
	})
	t.Run("unhappy: CLR and ENTR with no entry open are no-ops", func(t *testing.T) {
		p := NewPanel(MonitorState())
		before := p.State
		p.Press(KeyClear)
		p.Press(KeyEnter)
		if p.State != before {
			t.Fatalf("idle CLR/ENTR changed the state: %+v -> %+v", before, p.State)
		}
	})
}

func TestResetAndSafety(t *testing.T) {
	t.Run("happy: RSET wipes the caution lights and touches nothing else", func(t *testing.T) {
		st := MonitorState()
		st.Lights = lab.Lights{Prog: true, Restart: true, Alt: true, Vel: true}
		p := NewPanel(st)
		p.Press(KeyReset)
		if p.State.Lights != (lab.Lights{}) {
			t.Fatalf("RSET left lamps burning: %+v", p.State.Lights)
		}
		if p.State.Verb != "16" || p.State.Noun != "68" || p.State.R1 != st.R1 {
			t.Fatal("RSET must not touch the display")
		}
	})
	t.Run("unhappy: unknown keys are ignored and a nil panel skips every press", func(t *testing.T) {
		p := NewPanel(MonitorState())
		before := p.State
		for _, k := range []Key{Key('x'), Key('+'), Key(0), Key('Z')} {
			p.Press(k)
		}
		if p.State != before {
			t.Fatalf("unknown keys changed the state: %+v -> %+v", before, p.State)
		}
		var ghost *Panel
		ghost.Press(KeyVerb)
		ghost.Press(Key('7'))
		ghost.Press(KeyEnter)
	})
}
