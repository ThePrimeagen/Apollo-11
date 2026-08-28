package dsky

// The compact DSKY keypad, pressed through Press. VERB and NOUN open a
// two-digit entry: the field blanks and digits fill it left to right.
// An entry closes however it is left — complete it commits, whether by
// ENTR or by opening the next entry; incomplete it falls back to the
// value the field held before. CLR always cancels back to that value,
// RSET wipes the caution lights, and digits without an open entry go
// nowhere. The keypad is pure state — no clocks — so every press shows
// on the very next render.

import lab "github.com/theprimeagen/apollo-11/dsky-lab/dsky"

// Key is one key of the DSKY keypad. The digits are their own runes:
// Key('0') through Key('9').
type Key rune

const (
	KeyVerb  Key = 'V'
	KeyNoun  Key = 'N'
	KeyEnter Key = 'E'
	KeyClear Key = 'C'
	KeyReset Key = 'R'
)

// KeyFromRune maps a keyboard rune onto the keypad, case-insensitive:
// v, n, e, c, r and the digits. The bool is false for any rune the
// DSKY has no key for — hosts skip those.
func KeyFromRune(r rune) (Key, bool) {
	if r >= '0' && r <= '9' {
		return Key(r), true
	}
	switch r {
	case 'v', 'V':
		return KeyVerb, true
	case 'n', 'N':
		return KeyNoun, true
	case 'e', 'E':
		return KeyEnter, true
	case 'c', 'C':
		return KeyClear, true
	case 'r', 'R':
		return KeyReset, true
	}
	return 0, false
}

// Press feeds one key to the panel. Unknown keys are ignored.
// Nil-safe.
func (p *Panel) Press(k Key) {
	if p == nil {
		return
	}
	if k >= '0' && k <= '9' {
		p.pressDigit(rune(k))
		return
	}
	switch k {
	case KeyVerb, KeyNoun:
		p.closeEntry()
		p.entry = k
		p.prev = *p.entryField()
		*p.entryField() = ""
	case KeyEnter:
		p.closeEntry()
	case KeyClear:
		p.cancelEntry()
	case KeyReset:
		p.State.Lights = lab.Lights{}
	}
}

// entryField is the display field the open entry types into.
func (p *Panel) entryField() *string {
	if p.entry == KeyNoun {
		return &p.State.Noun
	}
	return &p.State.Verb
}

// pressDigit fills the open entry left to right. The third digit — and
// any digit without an open entry — goes nowhere.
func (p *Panel) pressDigit(d rune) {
	if p.entry == 0 || len(p.buf) >= 2 {
		return
	}
	p.buf += string(d)
	if len(p.buf) < 2 {
		*p.entryField() = p.buf + " "
		return
	}
	*p.entryField() = p.buf
}

// closeEntry ends the open entry the way it was left: a complete pair
// stays committed, an incomplete one falls back to the old value.
func (p *Panel) closeEntry() {
	if p.entry == 0 {
		return
	}
	if len(p.buf) < 2 {
		*p.entryField() = p.prev
	}
	p.entry, p.buf, p.prev = 0, "", ""
}

// cancelEntry is CLR: whatever was typed, the old value comes back.
func (p *Panel) cancelEntry() {
	if p.entry == 0 {
		return
	}
	*p.entryField() = p.prev
	p.entry, p.buf, p.prev = 0, "", ""
}
