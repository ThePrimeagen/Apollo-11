package scrollcode

// Tests written FIRST: the scrollable code component is the moving
// part — you hand it many code cards in reading order, flag which of
// them are stops, and it does everything the walkthrough needs: it
// stacks the cards into one column with a blank row between them,
// parks the spotlit card's first row on the anchor, and wears the
// vignette — the focused card bright, one card out equally dimmed on
// both sides, two out faint, three out barely there, and past that
// nothing paints. On its own clock it rests HoldSeconds on each stop
// and glides GlideSeconds to the next on an eased camera that lands
// exactly before the hold begins, then holds forever on the last
// stop. Next cuts the current rest short for callers that want to
// drive it by hand. The cards themselves never move — this component
// moves them.

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/code"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

const (
	stageW = 60
	stageH = 40
)

// card is a two-line pseudo card whose label makes it findable.
func card(label string) *code.Code {
	return code.New(code.LangPseudo, []string{
		label + ":",
		"    run " + strings.ToLower(label),
	})
}

// roster is six cards: a prologue, three stops, and a two-card tail.
//
//	AA (prologue), BB*, CC*, DD*, EE, FF
func roster() []Block {
	return []Block{
		{Code: card("AA")},
		{Code: card("BB"), Stop: true},
		{Code: card("CC"), Stop: true},
		{Code: card("DD"), Stop: true},
		{Code: card("EE")},
		{Code: card("FF")},
	}
}

func started(blocks []Block) *Scroll {
	s := New(blocks...).Tune(1.0, 0.5)
	s.Start(stageW, stageH)
	return s
}

func tick(s *Scroll, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		s.Update(dt)
	}
}

func artRow(sp sprite.Sprite, r int) string {
	rs := make([]rune, 0, sp.Width)
	for c := 0; c < sp.Width; c++ {
		ch := sp.At(r, c).Ch
		if ch == 0 {
			ch = ' '
		}
		rs = append(rs, ch)
	}
	return string(rs)
}

func findArt(sp sprite.Sprite, text string) (x, y int, ok bool) {
	for r := 0; r < sp.Height; r++ {
		if i := strings.Index(artRow(sp, r), text); i >= 0 {
			return len([]rune(artRow(sp, r)[:i])), r, true
		}
	}
	return 0, 0, false
}

func mustFind(t *testing.T, sp sprite.Sprite, text string) (x, y int) {
	t.Helper()
	x, y, ok := findArt(sp, text)
	if !ok {
		t.Fatalf("the column must show %q", text)
	}
	return x, y
}

func mustNotFind(t *testing.T, sp sprite.Sprite, text string) {
	t.Helper()
	if _, _, ok := findArt(sp, text); ok {
		t.Fatalf("the column must not show %q", text)
	}
}

func fgOf(t *testing.T, sp sprite.Sprite, text string) int {
	t.Helper()
	x, y := mustFind(t, sp, text)
	return sp.At(y, x).FG
}

func TestVignette(t *testing.T) {
	t.Run("happy: the opening stop is bright on the anchor with the fade running out both sides", func(t *testing.T) {
		s := started(roster())
		defer s.Stop()
		sp := s.Render()
		_, by := mustFind(t, sp, "BB:")
		if want := AnchorY(stageH); by != want {
			t.Fatalf("the spotlit card's first row sits at %d, want the anchor %d", by, want)
		}
		_, ay := mustFind(t, sp, "AA:")
		_, cy := mustFind(t, sp, "CC:")
		if ay >= by || cy <= by {
			t.Fatalf("AA rides above (%d) and CC below (%d) the spotlight (%d)", ay, cy, by)
		}
		lit, dim, faint, ghost := code.Dim(code.Foam, 0), code.Dim(code.Foam, 1), code.Dim(code.Foam, 2), code.Dim(code.Foam, 3)
		if got := fgOf(t, sp, "BB:"); got != lit {
			t.Fatalf("the spotlit label wears %d, want %d", got, lit)
		}
		above, below := fgOf(t, sp, "AA:"), fgOf(t, sp, "CC:")
		if above != dim || below != dim {
			t.Fatalf("one card out dims equally both sides: %d and %d, want %d", above, below, dim)
		}
		if got := fgOf(t, sp, "DD:"); got != faint {
			t.Fatalf("two cards out wears %d, want the faint %d", got, faint)
		}
		if got := fgOf(t, sp, "EE:"); got != ghost {
			t.Fatalf("three cards out wears %d, want the barely-there %d", got, ghost)
		}
		mustNotFind(t, sp, "FF:")
	})
	t.Run("happy: cards stack with one blank row between and share a left edge", func(t *testing.T) {
		s := started(roster())
		defer s.Stop()
		sp := s.Render()
		bx, by := mustFind(t, sp, "BB:")
		cx, cy := mustFind(t, sp, "CC:")
		if cx != bx {
			t.Fatalf("the column shares one left edge: %d vs %d", cx, bx)
		}
		if cy != by+3 {
			t.Fatalf("a two-line card plus one blank row: CC at %d, want %d", cy, by+3)
		}
		if strings.TrimSpace(artRow(sp, by+2)) != "" {
			t.Fatalf("the row between cards is blank, got %q", artRow(sp, by+2))
		}
	})
	t.Run("unhappy: a roster with no stops paints nothing, and nil is quiet", func(t *testing.T) {
		s := New(Block{Code: card("AA")}, Block{Code: card("BB")})
		s.Start(stageW, stageH)
		defer s.Stop()
		sp := s.Render()
		mustNotFind(t, sp, "AA:")
		mustNotFind(t, sp, "BB:")
		if s.Stops() != 0 {
			t.Fatalf("no stops means no stops, got %d", s.Stops())
		}
		var ghost *Scroll
		ghost.Start(10, 10)
		ghost.Update(1)
		_ = ghost.Render()
		ghost.Stop()
		if ghost.Next() {
			t.Fatal("a nil scroll goes nowhere")
		}
	})
}

func TestClock(t *testing.T) {
	t.Run("happy: the spotlight rests, glides upward, and lands exactly before the next rest", func(t *testing.T) {
		s := started(roster())
		defer s.Stop()
		if s.Stops() != 3 || s.FocusStop() != 0 {
			t.Fatalf("the show opens on stop 0 of 3, got %d of %d", s.FocusStop(), s.Stops())
		}
		tick(s, 1.2)
		sp := s.Render()
		_, mid := mustFind(t, sp, "CC:")
		if want := AnchorY(stageH); mid <= want {
			t.Fatalf("mid-glide the next card still rides below the anchor: %d", mid)
		}
		tick(s, 0.28)
		sp = s.Render()
		_, before := mustFind(t, sp, "CC:")
		if want := AnchorY(stageH); before != want {
			t.Fatalf("one frame before the rest the card sits at %d, want the anchor %d — the glide lands first", before, want)
		}
		tick(s, 0.1)
		sp = s.Render()
		if _, after := mustFind(t, sp, "CC:"); after != before {
			t.Fatalf("the card hopped %d→%d on the very frame its rest began", before, after)
		}
		if s.FocusStop() != 1 {
			t.Fatalf("the spotlight must have handed over to stop 1, got %d", s.FocusStop())
		}
	})
	t.Run("happy: the last stop holds forever", func(t *testing.T) {
		s := started(roster())
		defer s.Stop()
		tick(s, 2*1.5+1.2)
		sp := s.Render()
		_, dy := mustFind(t, sp, "DD:")
		if want := AnchorY(stageH); dy != want {
			t.Fatalf("the last stop parks on the anchor: %d, want %d", dy, want)
		}
		row := artRow(sp, dy)
		tick(s, 30)
		sp = s.Render()
		if got := artRow(sp, dy); got != row {
			t.Fatalf("the final hold drifted:\n%q\n%q", row, got)
		}
		if s.FocusStop() != 2 {
			t.Fatalf("the spotlight ends on stop 2, got %d", s.FocusStop())
		}
	})
	t.Run("happy: Next cuts the rest short and glides now", func(t *testing.T) {
		s := started(roster())
		defer s.Stop()
		tick(s, 0.2)
		if !s.Next() {
			t.Fatal("Next during a rest must move")
		}
		tick(s, 0.55)
		sp := s.Render()
		_, cy := mustFind(t, sp, "CC:")
		if want := AnchorY(stageH); cy != want {
			t.Fatalf("after Next and one glide the next stop is parked: %d, want %d", cy, want)
		}
		if s.FocusStop() != 1 {
			t.Fatalf("Next must hand the spotlight over, got stop %d", s.FocusStop())
		}
	})
	t.Run("unhappy: Next mid-glide and Next at the last stop both refuse", func(t *testing.T) {
		s := started(roster())
		defer s.Stop()
		tick(s, 1.2)
		if s.Next() {
			t.Fatal("a glide already underway cannot be cut shorter")
		}
		tick(s, 2*1.5)
		if s.FocusStop() != 2 {
			t.Fatal("test premise: the show must be on its last stop")
		}
		if s.Next() {
			t.Fatal("after the last stop there is nowhere to go")
		}
	})
	t.Run("unhappy: dt<=0 holds the clock and a resize keeps it", func(t *testing.T) {
		s := started(roster())
		defer s.Stop()
		tick(s, 1.5+0.2)
		s.Update(0)
		s.Update(-5)
		if s.FocusStop() != 1 {
			t.Fatalf("time never runs backwards: stop %d", s.FocusStop())
		}
		s.Stop()
		s.Start(80, 30)
		sp := s.Render()
		_, cy := mustFind(t, sp, "CC:")
		if want := AnchorY(30); cy != want {
			t.Fatalf("a resize keeps the clock: CC at %d, want the new anchor %d", cy, want)
		}
	})
}
