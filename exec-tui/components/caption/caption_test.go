package caption

// Tests written FIRST: Board is a timed side banner. Each Cue paints
// its text in the terminal-fonts face on the right of the stage from
// At until At+Hold. Alarm codes wear the PROG red; LAND wears the
// mission gold. Cues do not stack — at most one card is up, and a
// later cue wins an overlap. Before Start and after Stop the board
// is off. This is the 1202 / 1202 / 1201 (and 1202 / 1202 / LAND)
// talk sitting beside the spacelander.

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/terminal-fonts/termfont"
)

const (
	stageW = 72
	stageH = 28
)

var _ screenplay.Component = (*Board)(nil)

func stageText(sp sprite.Sprite) string {
	var b strings.Builder
	for r := 0; r < sp.Height; r++ {
		for c := 0; c < sp.Width; c++ {
			ch := sp.At(r, c).Ch
			if ch == 0 {
				ch = ' '
			}
			b.WriteRune(ch)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func hasCard(sp sprite.Sprite, text string) bool {
	lines, err := termfont.Lines(Height, text)
	if err != nil {
		return false
	}
	body := stageText(sp)
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if !strings.Contains(body, trim) {
			return false
		}
	}
	return true
}

func cardInk(sp sprite.Sprite, text string) int {
	lines, err := termfont.Lines(Height, text)
	if err != nil {
		return -1
	}
	body := stageText(sp)
	var row int
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "" {
			continue
		}
		if idx := strings.Index(body, trim); idx >= 0 {
			row = strings.Count(body[:idx], "\n")
			// find a painted cell on that row
			for c := 0; c < sp.Width; c++ {
				cell := sp.At(row, c)
				if cell.Ch != 0 && cell.Ch != ' ' {
					_ = i
					return cell.FG
				}
			}
		}
	}
	return -1
}

func TestBoardCues(t *testing.T) {
	cues := []Cue{
		{Text: "1202", At: 1.0, Hold: 0.8},
		{Text: "1202", At: 2.5, Hold: 0.8},
		{Text: "1201", At: 4.0, Hold: 0.8},
	}
	t.Run("happy: each cue paints on the right for its hold and then clears", func(t *testing.T) {
		b := New(cues...)
		b.Start(stageW, stageH)
		if hasCard(b.Render(), "1202") {
			t.Fatal("before the first cue the board must be blank")
		}
		b.Update(1.1)
		if !hasCard(b.Render(), "1202") {
			t.Fatal("the first 1202 must be up during its hold")
		}
		if cardInk(b.Render(), "1202") != AlarmInk {
			t.Fatalf("1202 ink %d, want the PROG red %d", cardInk(b.Render(), "1202"), AlarmInk)
		}
		b.Update(0.9)
		if hasCard(b.Render(), "1202") {
			t.Fatal("after the first hold the 1202 must clear")
		}
		b.Update(0.6)
		if !hasCard(b.Render(), "1202") {
			t.Fatal("the second 1202 must come up")
		}
		b.Update(0.9)
		b.Update(0.6)
		if !hasCard(b.Render(), "1201") {
			t.Fatal("the 1201 must be the third card — 1202, 1202, then 1201")
		}
		if hasCard(b.Render(), "LAND") {
			t.Fatal("the drop board never says LAND")
		}
	})
	t.Run("happy: LAND wears the mission gold on the right side", func(t *testing.T) {
		b := New(Cue{Text: "LAND", At: 0, Hold: 2})
		b.Start(stageW, stageH)
		sp := b.Render()
		if !hasCard(sp, "LAND") {
			t.Fatal("LAND must paint from t=0")
		}
		if cardInk(sp, "LAND") != LandInk {
			t.Fatalf("LAND ink %d, want the mission gold %d", cardInk(sp, "LAND"), LandInk)
		}
		// The card sits on the right half, not centered like a title.
		lines, err := termfont.Lines(Height, "LAND")
		if err != nil {
			t.Fatal(err)
		}
		body := stageText(sp)
		idx := strings.Index(body, strings.TrimSpace(lines[0]))
		if idx < 0 {
			t.Fatal("test premise: LAND is on stage")
		}
		col := idx % (stageW + 1)
		if col < stageW/2 {
			t.Fatalf("LAND starts at col %d, want the right half", col)
		}
	})
	t.Run("unhappy: a cue with no hold never paints, a bad rune is refused, and a nil board skips every cue", func(t *testing.T) {
		b := New(Cue{Text: "1202", At: 0, Hold: 0})
		b.Start(stageW, stageH)
		if hasCard(b.Render(), "1202") {
			t.Fatal("Hold 0 must stay blank")
		}
		if got := New(Cue{Text: "thé", At: 0, Hold: 1}); got != nil {
			t.Fatal("a rune off the charset must not build a board")
		}
		var ghost *Board
		ghost.Start(4, 2)
		ghost.Update(1)
		if sp := ghost.Render(); sp.Width != 0 {
			t.Fatalf("a nil board rendered %dx%d", sp.Width, sp.Height)
		}
		ghost.Stop()
	})
}

func TestBoardLifecycle(t *testing.T) {
	t.Run("happy: Stop clears the staging; a fresh Start shows the opening cue again", func(t *testing.T) {
		b := New(Cue{Text: "1202", At: 0, Hold: 5})
		b.Start(stageW, stageH)
		if !hasCard(b.Render(), "1202") {
			t.Fatal("test premise: a started board must show")
		}
		b.Stop()
		if sp := b.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped board rendered %dx%d", sp.Width, sp.Height)
		}
		b.Start(stageW, stageH)
		if !hasCard(b.Render(), "1202") {
			t.Fatal("a restaged board must show its opening cue")
		}
	})
	t.Run("unhappy: rendering before the first start is empty, and dt<=0 holds", func(t *testing.T) {
		b := New(Cue{Text: "1201", At: 0.5, Hold: 1})
		if sp := b.Render(); sp.Width != 0 {
			t.Fatalf("an unstarted board rendered %dx%d", sp.Width, sp.Height)
		}
		b.Start(stageW, stageH)
		b.Update(0)
		b.Update(-1)
		if hasCard(b.Render(), "1201") {
			t.Fatal("dt<=0 must not walk into the cue")
		}
	})
}
