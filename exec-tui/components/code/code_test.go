package code

// Tests written FIRST: the code component just displays code. You
// hand it lines and the language they are written in, and it paints
// them as a Rose Pine card — the coloring is private to the
// component, keyed by the language, so a language it does not know
// stays plain text instead of guessing weirdly. Tabs expand to the
// 8-column stops of the AGC listings. An optional gutter numbers the
// non-empty lines in five-digit octal. Marks are the highlighting:
// the caller says which span of which line to highlight and in what
// color, and the mark wins over the syntax ink. Dim is the vignette
// ramp the scroller leans on: level 0 is the bright palette, three
// deepening levels sink every hue toward the base, and past level 3
// nothing paints at all. The component itself never moves — it is a
// still card, and it also plays as a centered screenplay Component.

import (
	"strings"
	"testing"

	"github.com/theprimeagen/apollo-11/exec-tui/components/danzig"
	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
)

// artRow reads row r of a sprite as plain text.
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

// findArt locates text on a sprite.
func findArt(sp sprite.Sprite, text string) (x, y int, ok bool) {
	for r := 0; r < sp.Height; r++ {
		if i := strings.Index(artRow(sp, r), text); i >= 0 {
			return len([]rune(artRow(sp, r)[:i])), r, true
		}
	}
	return 0, 0, false
}

func fgArt(t *testing.T, sp sprite.Sprite, text string) int {
	t.Helper()
	x, y, ok := findArt(sp, text)
	if !ok {
		t.Fatalf("the card must show %q", text)
	}
	return sp.At(y, x).FG
}

func TestDisplay(t *testing.T) {
	t.Run("happy: the card is the lines, tabs expanded to the listing's 8-column stops", func(t *testing.T) {
		c := New(LangAGC, []string{
			"MUNRVG\t\tVLOAD\tVXSC",
			"\t\t\tDELV",
		})
		lines := c.Lines()
		if len(lines) != 2 {
			t.Fatalf("the card holds %d lines, want 2", len(lines))
		}
		if want := "MUNRVG" + strings.Repeat(" ", 10) + "VLOAD" + strings.Repeat(" ", 3) + "VXSC"; lines[0] != want {
			t.Fatalf("tabs must expand on the 8-column grid, got %q want %q", lines[0], want)
		}
		if want := strings.Repeat(" ", 24) + "DELV"; lines[1] != want {
			t.Fatalf("an operand line lands on column 24, got %q", lines[1])
		}
		w, h := c.Size()
		if h != 2 || w != len([]rune(lines[0])) {
			t.Fatalf("Size is the widest line by the count: %dx%d", w, h)
		}
		art := c.Art()
		if art.Width != w || art.Height != h {
			t.Fatalf("Art is Size: %dx%d, want %dx%d", art.Width, art.Height, w, h)
		}
		if _, _, ok := findArt(art, "VLOAD"); !ok {
			t.Fatal("the art must carry the code")
		}
	})
	t.Run("happy: every cell of the card sits on the Rose Pine base", func(t *testing.T) {
		c := New(LangAGC, []string{"\t\tVXSC", "\t\t\tR"})
		art := c.Art()
		for r := 0; r < art.Height; r++ {
			for col := 0; col < art.Width; col++ {
				if got := art.At(r, col).BG; got != Base {
					t.Fatalf("cell (%d,%d) floor is %d, want the base %d", r, col, got, Base)
				}
			}
		}
	})
	t.Run("happy: the card plays as a centered screenplay component", func(t *testing.T) {
		c := New(LangAGC, []string{"\t\tVLOAD\tVXSC"})
		c.Start(60, 9)
		defer c.Stop()
		stage := c.Render()
		if stage.Width != 60 || stage.Height != 9 {
			t.Fatalf("the component paints its stage: %dx%d", stage.Width, stage.Height)
		}
		_, y, ok := findArt(stage, "VLOAD")
		if !ok {
			t.Fatal("the staged card must show its code")
		}
		if y != 4 {
			t.Fatalf("one line centers on row 4 of a 9-row stage, got %d", y)
		}
	})
	t.Run("unhappy: empty code, a stopped card, and nil are quiet, never a panic", func(t *testing.T) {
		empty := New(LangAGC, nil)
		if w, h := empty.Size(); w != 0 || h != 0 {
			t.Fatalf("no lines is no card: %dx%d", w, h)
		}
		if art := empty.Art(); art.Width != 0 || art.Height != 0 {
			t.Fatal("no lines renders no art")
		}
		c := New(LangAGC, []string{"\t\tVLOAD"})
		if sp := c.Render(); sp.Width != 0 {
			t.Fatal("before Start the stage is empty")
		}
		c.Start(20, 5)
		c.Update(1)
		c.Stop()
		if sp := c.Render(); sp.Width != 0 {
			t.Fatal("after Stop the stage is empty")
		}
		var ghost *Code
		ghost.Start(10, 10)
		ghost.Update(1)
		_ = ghost.Render()
		ghost.Stop()
	})
}

func TestColoring(t *testing.T) {
	t.Run("happy: the AGC language — labels foam, opcodes iris, operands foam, numbers gold, comments muted", func(t *testing.T) {
		c := New(LangAGC, []string{
			"MUNRVG\t\tVLOAD\tVXSC",
			"\t\t\tKPIP2",
			"\t\t\t36D",
			"\t\tSTODL\tDELVS\t\t# LUNAR ROTATION CORRECTION",
			"# Page 883",
		})
		art := c.Art()
		if got := fgArt(t, art, "MUNRVG"); got != Foam {
			t.Fatalf("a column-0 label wears %d, want foam %d", got, Foam)
		}
		if got := fgArt(t, art, "VLOAD"); got != Iris {
			t.Fatalf("the opcode field wears %d, want iris %d", got, Iris)
		}
		if got := fgArt(t, art, "VXSC"); got != Foam {
			t.Fatalf("the pair's far field wears %d, want foam %d", got, Foam)
		}
		if got := fgArt(t, art, "KPIP2"); got != Foam {
			t.Fatalf("an operand symbol wears %d, want foam %d", got, Foam)
		}
		if got := fgArt(t, art, "36D"); got != Gold {
			t.Fatalf("a numeric operand wears %d, want gold %d", got, Gold)
		}
		if got := fgArt(t, art, "STODL"); got != Iris {
			t.Fatalf("a store opcode wears %d, want iris %d", got, Iris)
		}
		if got := fgArt(t, art, "# LUNAR"); got != Muted {
			t.Fatalf("a comment wears %d, want muted %d", got, Muted)
		}
		if got := fgArt(t, art, "# Page"); got != Muted {
			t.Fatalf("a page comment wears %d, want muted %d", got, Muted)
		}
	})
	t.Run("happy: the pseudo language — keywords iris, labels foam, numbers gold, operators rose", func(t *testing.T) {
		c := New(LangPseudo, []string{
			"if NEWJOB != 0:   # DANZIG",
			"    swap cores[0], cores[NEWJOB]",
		})
		art := c.Art()
		if got := fgArt(t, art, "if"); got != Iris {
			t.Fatalf("a keyword wears %d, want iris %d", got, Iris)
		}
		if got := fgArt(t, art, "NEWJOB"); got != Foam {
			t.Fatalf("a label wears %d, want foam %d", got, Foam)
		}
		if got := fgArt(t, art, "0:"); got != Gold {
			t.Fatalf("a number wears %d, want gold %d", got, Gold)
		}
		if got := fgArt(t, art, "!"); got != Rose {
			t.Fatalf("an operator wears %d, want rose %d", got, Rose)
		}
		if got := fgArt(t, art, "# DANZIG"); got != Muted {
			t.Fatalf("a comment wears %d, want muted %d", got, Muted)
		}
		if got := fgArt(t, art, "cores"); got != Text {
			t.Fatalf("an ident wears %d, want text %d", got, Text)
		}
	})
	t.Run("happy: the palette is the danzig card's Rose Pine", func(t *testing.T) {
		pairs := [][2]int{
			{Base, danzig.Base256}, {Text, danzig.Text256}, {Muted, danzig.Muted256},
			{Gold, danzig.Gold256}, {Foam, danzig.Foam256}, {Iris, danzig.Iris256},
			{Rose, danzig.Rose256},
		}
		for _, p := range pairs {
			if p[0] != p[1] {
				t.Fatalf("the palettes drifted: %d vs the danzig %d", p[0], p[1])
			}
		}
	})
	t.Run("unhappy: a language the component does not know stays plain text — no weird coloring", func(t *testing.T) {
		c := New(Lang("brainfuck"), []string{"MUNRVG\t\tVLOAD\t36D\t# comment"})
		art := c.Art()
		for _, tok := range []string{"MUNRVG", "VLOAD", "36D", "# comment"} {
			if got := fgArt(t, art, tok); got != Text {
				t.Fatalf("%q wears %d under an unknown language, want plain text %d", tok, got, Text)
			}
		}
	})
}

func TestGutterAndMarks(t *testing.T) {
	t.Run("happy: the gutter numbers non-empty lines in five-digit octal, muted, and blanks stay blank", func(t *testing.T) {
		c := New(LangAGC, []string{"\t\tVLOAD", "", "\t\tVXSC"}).Gutter(0o4007)
		art := c.Art()
		x, y, ok := findArt(art, "04007")
		if !ok {
			t.Fatal("the first line must carry its octal address")
		}
		if y != 0 || x != 0 {
			t.Fatalf("the gutter leads the line: address at (%d,%d)", x, y)
		}
		if got := art.At(0, 0).FG; got != Muted {
			t.Fatalf("the gutter wears %d, want muted %d", got, Muted)
		}
		if _, y2, ok := findArt(art, "04010"); !ok || y2 != 2 {
			t.Fatalf("the blank consumes no address — 04010 sits on row 2, got %v %d", ok, y2)
		}
		if strings.TrimSpace(artRow(art, 1)) != "" {
			t.Fatalf("the blank row stays blank, got %q", artRow(art, 1))
		}
		cx, _, _ := findArt(art, "VLOAD")
		if cx != 7+16 {
			t.Fatalf("the code keeps its own indent after a 7-cell gutter: VLOAD at %d, want %d", cx, 7+16)
		}
	})
	t.Run("unhappy: without a gutter there are no addresses", func(t *testing.T) {
		c := New(LangAGC, []string{"\t\tVLOAD"})
		if _, _, ok := findArt(c.Art(), "04000"); ok {
			t.Fatal("no gutter was asked for")
		}
	})
	t.Run("happy: a mark highlights exactly its span in its color, over the syntax ink", func(t *testing.T) {
		c := New(LangAGC, []string{"\t\tCCS\tNEWJOB\t# CHECK"})
		line := c.Lines()[0]
		start := strings.Index(line, "NEWJOB")
		c.Mark(0, start, start+6, Love)
		art := c.Art()
		x, y, _ := findArt(art, "NEWJOB")
		for i := 0; i < 6; i++ {
			if got := art.At(y, x+i).FG; got != Love {
				t.Fatalf("marked cell %d wears %d, want love %d", i, got, Love)
			}
		}
		if got := art.At(y, x+6).FG; got == Love {
			t.Fatal("the mark must end where its span ends")
		}
		if got := fgArt(t, art, "CCS"); got != Iris {
			t.Fatal("the rest of the line keeps its syntax ink")
		}
	})
	t.Run("unhappy: marks off the card clamp quietly and never panic", func(t *testing.T) {
		c := New(LangAGC, []string{"\t\tCCS"})
		c.Mark(9, 0, 4, Love)
		c.Mark(-1, 0, 4, Love)
		c.Mark(0, -5, 500, Gold)
		art := c.Art()
		if got := fgArt(t, art, "CCS"); got != Gold {
			t.Fatal("a clamped span still paints the line it can reach")
		}
	})
	t.Run("happy: Dim sinks every hue level by level and stops painting past three", func(t *testing.T) {
		for _, ink := range []int{Text, Muted, Gold, Foam, Iris, Rose, Love} {
			if got := Dim(ink, 0); got != ink {
				t.Fatalf("level 0 is the bright ink itself: %d became %d", ink, got)
			}
			seen := map[int]bool{ink: true}
			for lvl := 1; lvl <= 3; lvl++ {
				got := Dim(ink, lvl)
				if got < 0 {
					t.Fatalf("ink %d still paints at level %d", ink, lvl)
				}
				if seen[got] {
					t.Fatalf("ink %d level %d repeats an earlier rung (%d) — the fade must keep sinking", ink, lvl, got)
				}
				seen[got] = true
			}
			if got := Dim(ink, 4); got != -1 {
				t.Fatalf("past the vignette ink %d must not paint, got %d", ink, got)
			}
		}
	})
	t.Run("unhappy: Dim of no-color stays no-color, unknown inks fall to the text ramp, negatives stay bright", func(t *testing.T) {
		if got := Dim(-1, 2); got != -1 {
			t.Fatalf("no color dims to no color, got %d", got)
		}
		if got := Dim(999, 2); got != Dim(Text, 2) {
			t.Fatalf("an unknown ink follows the text ramp, got %d", got)
		}
		if got := Dim(Foam, -3); got != Foam {
			t.Fatalf("a negative level is the spotlight, got %d", got)
		}
	})
}
