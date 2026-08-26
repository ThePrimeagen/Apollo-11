package danzig

// Tests written FIRST: the DANZIG card is the Executive's job-picker
// written as a handful of lines of pseudocode, syntax-highlighted in
// Rose Pine. FINDVAC packs class|VAC into PRIORITY; DANZIG swaps to
// NEWJOB between opcodes; EJSCAN picks the largest PRIORITY word —
// which is why two "priority 20" SERVICERs are not equal and the
// newest (higher VAC address) always runs.

import (
	"regexp"
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripAnsi(s string) string { return ansiPat.ReplaceAllString(s, "") }

func kindsOf(line string) []Kind {
	var out []Kind
	for _, tok := range TokenizeLine(line) {
		if tok.Text == "" || tok.Kind == KindSpace {
			continue
		}
		out = append(out, tok.Kind)
	}
	return out
}

func TestSourceTellsThePackedPriorityStory(t *testing.T) {
	t.Run("happy: the source shows the core-set scan and the packed word", func(t *testing.T) {
		for _, want := range []string{
			"FINDVAC", "SETLOC", "HASNEWJOB", "DANZIG", "EJSCAN",
			"NEWJOB", "PRIORITY", "20401", "20455",
			"12,24", "continue",
		} {
			if !strings.Contains(Source, want) {
				t.Fatalf("Source must contain %q:\n%s", want, Source)
			}
		}
	})
	t.Run("unhappy: the source is not a dump of the AGC listing", func(t *testing.T) {
		for _, gone := range []string{"CCS", "TCF", "BZMF", "INDEX", "2CADR"} {
			if strings.Contains(Source, gone) {
				t.Fatalf("pseudocode must not leak assembly mnemonic %q", gone)
			}
		}
	})
}

func TestTokenize(t *testing.T) {
	t.Run("happy: a comment, a label, a keyword, and a packed number", func(t *testing.T) {
		got := kindsOf("# packed")
		if len(got) != 1 || got[0] != KindComment {
			t.Fatalf("comment line kinds %v, want [Comment]", got)
		}
		got = kindsOf("FINDVAC(job, class):")
		if len(got) < 2 || got[0] != KindLabel {
			t.Fatalf("FINDVAC must start as a Label, got %v", got)
		}
		got = kindsOf("    if NEWJOB:")
		if len(got) < 2 || got[0] != KindKeyword || got[1] != KindLabel {
			t.Fatalf("if NEWJOB must be Keyword then Label, got %v", got)
		}
		got = kindsOf("20455")
		if len(got) != 1 || got[0] != KindNumber {
			t.Fatalf("20455 must be a Number, got %v", got)
		}
	})
	t.Run("unhappy: punctuation and unknown words still tokenize, never drop text", func(t *testing.T) {
		line := "        if w <= 0: continue      # free / asleep"
		joined := ""
		for _, tok := range TokenizeLine(line) {
			joined += tok.Text
		}
		if joined != line {
			t.Fatalf("tokenize must be lossless\n got %q\nwant %q", joined, line)
		}
		if kindsOf("???")[0] == KindKeyword {
			t.Fatal("unknown text must not be classified as a keyword")
		}
	})
}

func TestHighlightRosePine(t *testing.T) {
	t.Run("happy: numbers paint gold, comments muted, labels foam", func(t *testing.T) {
		out := Highlight(Source)
		plain := stripAnsi(out)
		if !strings.Contains(plain, "FINDVAC") || !strings.Contains(plain, "20455") {
			t.Fatalf("highlighted output must still be readable, got %q", plain)
		}
		gold := "38;2;246;193;119"  // #f6c177
		muted := "38;2;110;106;134" // #6e6a86
		foam := "38;2;156;207;216"  // #9ccfd8
		if !strings.Contains(out, gold) {
			t.Fatalf("numbers must use Rose Pine gold, missing %s in\n%s", gold, out)
		}
		if !strings.Contains(out, muted) {
			t.Fatalf("comments must use Rose Pine muted, missing %s", muted)
		}
		if !strings.Contains(out, foam) {
			t.Fatalf("labels must use Rose Pine foam, missing %s", foam)
		}
		if !strings.Contains(out, "48;2;25;23;36") { // #191724 base
			t.Fatal("the card must sit on Rose Pine base")
		}
	})
	t.Run("unhappy: empty source still produces a titled card, no panic", func(t *testing.T) {
		out := Highlight("")
		if stripAnsi(out) == "" {
			t.Fatal("an empty source must still render the title chrome")
		}
		if !strings.Contains(stripAnsi(out), "HOW THE EXEC PICKS A JOB") {
			t.Fatalf("empty source must keep the title, got %q", stripAnsi(out))
		}
	})
}

// The compile-time pin: a Card plays as a screenplay component.
var _ screenplay.Component = (*Card)(nil)

func stageText(sp sprite.Sprite) string {
	rows := make([]string, sp.Height)
	for r := 0; r < sp.Height; r++ {
		row := make([]rune, sp.Width)
		for c := 0; c < sp.Width; c++ {
			ch := sp.At(r, c).Ch
			if ch == 0 {
				ch = ' '
			}
			row[c] = ch
		}
		rows[r] = string(row)
	}
	return strings.Join(rows, "\n")
}

func TestCardOnStage(t *testing.T) {
	t.Run("happy: a started card paints FINDVAC in foam on Rose Pine base", func(t *testing.T) {
		card := New()
		card.Start(72, 28)
		stage := card.Render()
		if stage.Width != 72 || stage.Height != 28 {
			t.Fatalf("stage %dx%d, want 72x28", stage.Width, stage.Height)
		}
		text := stageText(stage)
		if !strings.Contains(text, "FINDVAC") || !strings.Contains(text, "DANZIG") {
			t.Fatalf("stage is missing the picker source:\n%s", text)
		}
		// foam 256 ≈ 152, base 256 ≈ 235
		foundFoam, foundBase := false, false
		for r := 0; r < stage.Height; r++ {
			for c := 0; c < stage.Width; c++ {
				cell := stage.At(r, c)
				if cell.FG == Foam256 {
					foundFoam = true
				}
				if cell.BG == Base256 {
					foundBase = true
				}
			}
		}
		if !foundFoam {
			t.Fatal("FINDVAC (a label) must be inked in Rose Pine foam")
		}
		if !foundBase {
			t.Fatal("the card must fill Rose Pine base behind the code")
		}
	})
	t.Run("unhappy: rendering before Start is an empty stage", func(t *testing.T) {
		if sp := New().Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("an unstarted card rendered %dx%d", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a stage smaller than the card clips instead of panicking", func(t *testing.T) {
		card := New()
		card.Start(10, 3)
		sp := card.Render()
		if sp.Width != 10 || sp.Height != 3 {
			t.Fatalf("tiny stage %dx%d, want 10x3", sp.Width, sp.Height)
		}
		n := 0
		for r := 0; r < sp.Height; r++ {
			for c := 0; c < sp.Width; c++ {
				if !sp.At(r, c).Transparent() || sp.At(r, c).BG >= 0 {
					n++
				}
			}
		}
		if n > 10*3 {
			t.Fatalf("tiny stage lit %d cells, has only %d", n, 10*3)
		}
	})
	t.Run("unhappy: Stop clears the staging; a nil card skips every cue", func(t *testing.T) {
		card := New()
		card.Start(40, 20)
		card.Stop()
		if sp := card.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a stopped card rendered %dx%d", sp.Width, sp.Height)
		}
		var ghost *Card
		ghost.Start(4, 2)
		ghost.Update(1)
		ghost.Render()
		ghost.Stop()
	})
}

func TestCardSize(t *testing.T) {
	t.Run("happy: the card reports a positive footprint that fits the source", func(t *testing.T) {
		if CardWidth() < 40 || CardHeight() < 12 {
			t.Fatalf("card %dx%d is too small to hold the picker", CardWidth(), CardHeight())
		}
	})
	t.Run("unhappy: no source line is wider than the inner card", func(t *testing.T) {
		inner := CardWidth() - 4 // border + pad
		for i, line := range strings.Split(Source, "\n") {
			if len(line) > inner {
				t.Fatalf("line %d is %d runes, inner width %d: %q", i, len(line), inner, line)
			}
		}
	})
}
