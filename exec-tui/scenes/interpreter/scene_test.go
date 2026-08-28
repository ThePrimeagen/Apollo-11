package interpreter

// Tests written FIRST: the Interpreter scene walks the virtual
// machine's code the way DANZIG sees it — a scrolling column of five
// fake interpretive instructions, VXV (vector cross vector) first,
// each block spelling the op, how its arguments arrive, what the op
// costs in milliseconds, and then the check to DANZIG — is a job of
// higher priority waiting? — dressed five different ways: the real
// assembly (CCS NEWJOB / TCF CHANG2), pseudocode, a fork diagram, a
// weighing of PRIORITY words, and a rubber stamp. The column wears
// Rose Pine over the Rose Pine base, behind a vertical vignette: the
// spotlit block is bright at its anchor, one block above and one
// below sit equally dimmed, the next one down is barely visible, and
// past that the code cannot be seen at all. An INTPRET prologue rides
// above the first instruction and an EXIT epilogue below the last, so
// the vignette never runs out of code. The spotlight rests HoldSeconds
// on each instruction, glides GlideSeconds to the next on an eased,
// exactly-landing camera, and holds forever on the fifth. A resize
// keeps the clock; Stop then Start replays from the top; a nil screen
// never panics.

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/danzig"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 100
	stageH = 30
)

// stock is the scene's default clock — the marks every test that
// plays the stock show steers by.
var stock = DefaultConfig()

func paint(sc screenplay.Scene) *screenplay.Screen {
	scr := screenplay.NewScreen(stageW, stageH)
	sc.Render(scr)
	return scr
}

func tick(sc screenplay.Scene, seconds float64) {
	const dt = 1.0 / 30
	for t := 0.0; t < seconds-dt/2; t += dt {
		sc.Update(dt)
	}
}

// opened is a scene past its curtain and first render, parked at t=0.
func opened(t *testing.T) *Show {
	t.Helper()
	s := New()
	s.Start()
	_ = paint(s)
	return s
}

// seek plays an opened scene forward by the given seconds.
func seek(t *testing.T, s *Show, by float64) *screenplay.Screen {
	t.Helper()
	tick(s, by)
	return paint(s)
}

func rowText(scr *screenplay.Screen, y int) string {
	w, _ := scr.Size()
	rs := make([]rune, 0, w)
	for x := 0; x < w; x++ {
		c := scr.Cell(x, y)
		if c == nil || c.Content == "" {
			rs = append(rs, ' ')
			continue
		}
		rs = append(rs, []rune(c.Content)[0])
	}
	return string(rs)
}

func findOn(scr *screenplay.Screen, text string) (x, y int, ok bool) {
	_, h := scr.Size()
	for row := 0; row < h; row++ {
		line := rowText(scr, row)
		if i := strings.Index(line, text); i >= 0 {
			return len([]rune(line[:i])), row, true
		}
	}
	return 0, 0, false
}

func mustSee(t *testing.T, scr *screenplay.Screen, text string) (x, y int) {
	t.Helper()
	x, y, ok := findOn(scr, text)
	if !ok {
		t.Fatalf("the stage must show %q", text)
	}
	return x, y
}

func mustNotSee(t *testing.T, scr *screenplay.Screen, text string) {
	t.Helper()
	if _, _, ok := findOn(scr, text); ok {
		t.Fatalf("the stage must not show %q", text)
	}
}

func fgAt(scr *screenplay.Screen, x, y int) int {
	c := scr.Cell(x, y)
	if c == nil {
		return -1
	}
	if ic, ok := c.Style.Fg.(ansi.IndexedColor); ok {
		return int(ic)
	}
	return -1
}

func bgAt(scr *screenplay.Screen, x, y int) int {
	c := scr.Cell(x, y)
	if c == nil {
		return -1
	}
	if ic, ok := c.Style.Bg.(ansi.IndexedColor); ok {
		return int(ic)
	}
	return -1
}

func TestRoster(t *testing.T) {
	t.Run("happy: five fake instructions, VXV first, each op then args then time then the check", func(t *testing.T) {
		ins := Instructions()
		if len(ins) != 5 {
			t.Fatalf("the walkthrough covers %d instructions, want 5", len(ins))
		}
		if ins[0].Mnemonic != "VXV" || !strings.Contains(ins[0].Op, "vector cross vector") {
			t.Fatalf("the first instruction is V cross V, got %q / %q", ins[0].Mnemonic, ins[0].Op)
		}
		if !strings.Contains(ins[0].Time, "5.0 ms") {
			t.Fatalf("VXV costs the sourced ≈ 5.0 ms, got %q", ins[0].Time)
		}
		mn := map[string]bool{}
		for i, in := range ins {
			if in.Mnemonic == "" || mn[in.Mnemonic] {
				t.Fatalf("instruction %d mnemonic %q must be unique and non-empty", i, in.Mnemonic)
			}
			mn[in.Mnemonic] = true
			if !regexp.MustCompile(`^[A-Z0-9]{2,5}$`).MatchString(in.Mnemonic) {
				t.Fatalf("%q is not an AGC-flavored mnemonic", in.Mnemonic)
			}
			if !strings.HasPrefix(in.Op, in.Mnemonic) {
				t.Fatalf("the op line %q must lead with its mnemonic %q", in.Op, in.Mnemonic)
			}
			if len(in.Args) == 0 {
				t.Fatalf("%s carries no argument lines — the args are part of the lesson", in.Mnemonic)
			}
			if !strings.Contains(in.Time, "ms") || !strings.Contains(in.Time, "≈") {
				t.Fatalf("%s time %q must read as an honest ≈ figure in ms", in.Mnemonic, in.Time)
			}
			if len(in.Check) == 0 {
				t.Fatalf("%s has no DANZIG check — the check is the whole point", in.Mnemonic)
			}
			joined := strings.Join(in.Check, "\n")
			if !strings.Contains(joined, "NEWJOB") && !strings.Contains(joined, "DANZIG") {
				t.Fatalf("%s check %q never names the priority machinery", in.Mnemonic, joined)
			}
			if !strings.Contains(in.Caption, fmt.Sprintf("%d/5", i+1)) || !strings.Contains(in.Caption, in.Mnemonic) {
				t.Fatalf("%s caption %q must number the stop and name the op", in.Mnemonic, in.Caption)
			}
		}
	})
	t.Run("happy: the five checks are five different looks", func(t *testing.T) {
		ins := Instructions()
		styles := map[string]bool{}
		checks := map[string]bool{}
		for _, in := range ins {
			if in.Style == "" || styles[in.Style] {
				t.Fatalf("style %q must be unique and non-empty — five versions were asked for", in.Style)
			}
			styles[in.Style] = true
			joined := strings.Join(in.Check, "\n")
			if checks[joined] {
				t.Fatalf("%s repeats another instruction's check verbatim", in.Mnemonic)
			}
			checks[joined] = true
		}
		for _, want := range []string{"assembly", "pseudocode", "fork", "weighing", "stamp"} {
			if !styles[want] {
				t.Fatalf("the promised %q look is missing", want)
			}
		}
	})
	t.Run("happy: the roster is prologue, five instructions, epilogue — seven blocks so the vignette never runs dry", func(t *testing.T) {
		bs := Blocks()
		if len(bs) != 7 {
			t.Fatalf("the scroll holds %d blocks, want 7", len(bs))
		}
		if bs[0].Name != "INTPRET" || bs[6].Name != "EXIT" {
			t.Fatalf("the scroll must open on INTPRET and close on EXIT, got %q and %q", bs[0].Name, bs[6].Name)
		}
		ins := Instructions()
		for i, in := range ins {
			b := bs[i+1]
			if b.Name != in.Mnemonic {
				t.Fatalf("block %d is %q, want %q", i+1, b.Name, in.Mnemonic)
			}
			want := append([]string{in.Op}, in.Args...)
			want = append(want, in.Time)
			want = append(want, in.Check...)
			if len(b.Lines) != len(want) {
				t.Fatalf("%s block holds %d lines, want %d (op, args, time, check)", b.Name, len(b.Lines), len(want))
			}
			for j, line := range want {
				if b.Lines[j] != line {
					t.Fatalf("%s line %d is %q, want %q", b.Name, j, b.Lines[j], line)
				}
			}
		}
		for _, b := range bs {
			if len(b.Lines) == 0 {
				t.Fatalf("block %s is empty", b.Name)
			}
		}
	})
	t.Run("unhappy: no line outgrows the card and the stamp's borders align", func(t *testing.T) {
		for _, b := range Blocks() {
			for _, line := range b.Lines {
				if n := len([]rune(line)); n > 64 {
					t.Fatalf("%s line %q runs %d runes — past 64 the column no longer fits the stage", b.Name, line, n)
				}
			}
		}
		var stamp []string
		for _, in := range Instructions() {
			if in.Style == "stamp" {
				stamp = in.Check
			}
		}
		if len(stamp) < 3 {
			t.Fatalf("the stamp needs a top, a body, and a bottom, got %d lines", len(stamp))
		}
		w := len([]rune(stamp[0]))
		for _, line := range stamp {
			if len([]rune(line)) != w {
				t.Fatalf("the stamp's borders tear: %q is not %d runes wide", line, w)
			}
		}
	})
}

func TestVignette(t *testing.T) {
	t.Run("happy: distance rounds to the shade level — 0 lit, 1 dim, 2 faint, 3 gone", func(t *testing.T) {
		cases := []struct {
			d    float64
			want int
		}{
			{0, 0}, {0.4, 0}, {0.6, 1}, {1, 1}, {1.4, 1}, {1.6, 2}, {2, 2}, {2.6, 3}, {3, 3},
		}
		for _, tc := range cases {
			if got := vigLevel(tc.d); got != tc.want {
				t.Fatalf("distance %v shades at level %d, want %d", tc.d, got, tc.want)
			}
		}
		if vigLevel(-1) != vigLevel(1) {
			t.Fatal("the vignette is symmetric — a block above dims like a block below")
		}
	})
	t.Run("unhappy: far and broken distances are invisible, never a panic", func(t *testing.T) {
		if got := vigLevel(9); got < 3 {
			t.Fatalf("distance 9 must be past seeing, level %d", got)
		}
		if got := vigLevel(math.NaN()); got < 3 {
			t.Fatalf("a NaN distance must be invisible, level %d", got)
		}
		if got := vigLevel(math.Inf(1)); got < 3 {
			t.Fatalf("an infinite distance must be invisible, level %d", got)
		}
	})
	t.Run("happy: level 0 is the danzig card's Rose Pine, levels dim without repeating, level 3 does not paint", func(t *testing.T) {
		brights := map[ink]int{
			inkText:  danzig.Text256,
			inkMuted: danzig.Muted256,
			inkGold:  danzig.Gold256,
			inkFoam:  danzig.Foam256,
			inkIris:  danzig.Iris256,
			inkRose:  danzig.Rose256,
		}
		for in, want := range brights {
			if got := shade(in, 0); got != want {
				t.Fatalf("ink %d at level 0 wears %d, want the danzig Rose Pine %d", in, got, want)
			}
			l1, l2 := shade(in, 1), shade(in, 2)
			if l1 == want || l1 < 0 {
				t.Fatalf("ink %d level 1 must dim off the bright %d, got %d", in, want, l1)
			}
			if l2 == l1 || l2 == want || l2 < 0 {
				t.Fatalf("ink %d level 2 must sink further: bright %d, dim %d, faint %d", in, want, l1, l2)
			}
			if got := shade(in, 3); got != -1 {
				t.Fatalf("ink %d at level 3 must not paint, got %d", in, got)
			}
			if got := shade(in, 7); got != -1 {
				t.Fatalf("ink %d far past the vignette must not paint, got %d", in, got)
			}
		}
	})
	t.Run("unhappy: a negative level clamps to lit and a ghost ink still paints readably", func(t *testing.T) {
		if got := shade(inkText, -2); got != shade(inkText, 0) {
			t.Fatalf("a negative level is the spotlight, got %d", got)
		}
		if got := shade(ink(99), 0); got < 0 {
			t.Fatalf("a ghost ink must fall back to something visible, got %d", got)
		}
	})
	t.Run("happy: the tokenizer's kinds land on their Rose Pine inks", func(t *testing.T) {
		cases := []struct {
			kind danzig.Kind
			want ink
		}{
			{danzig.KindComment, inkMuted},
			{danzig.KindKeyword, inkIris},
			{danzig.KindLabel, inkFoam},
			{danzig.KindNumber, inkGold},
			{danzig.KindOp, inkRose},
			{danzig.KindIdent, inkText},
			{danzig.KindSpace, inkText},
		}
		for _, tc := range cases {
			if got := kindInk(tc.kind); got != tc.want {
				t.Fatalf("kind %d paints ink %d, want %d", tc.kind, got, tc.want)
			}
		}
	})
}

func TestOpening(t *testing.T) {
	t.Run("happy: VXV spotlit at the anchor — INTPRET dim above, DOT dim below, MXV faint, VXSC unseen", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		vx, vy := mustSee(t, scr, "VXV")
		if want := anchorY(stageH); vy != want {
			t.Fatalf("the spotlit op sits on row %d, want the anchor %d", vy, want)
		}
		ix, iy := mustSee(t, scr, "INTPRET")
		if iy >= vy {
			t.Fatalf("INTPRET (row %d) must ride above the spotlight (row %d)", iy, vy)
		}
		dx, dy := mustSee(t, scr, "DOT")
		if dy <= vy {
			t.Fatalf("DOT (row %d) must wait below the spotlight (row %d)", dy, vy)
		}
		mx, my := mustSee(t, scr, "MXV")
		if my <= dy {
			t.Fatalf("MXV (row %d) must sit below DOT (row %d)", my, dy)
		}
		mustNotSee(t, scr, "VXSC")
		mustNotSee(t, scr, "DAD")
		mustNotSee(t, scr, "EXIT")

		lit := shade(inkFoam, 0)
		dim := shade(inkFoam, 1)
		faint := shade(inkFoam, 2)
		if got := fgAt(scr, vx, vy); got != lit {
			t.Fatalf("the spotlit VXV wears %d, want the lit foam %d", got, lit)
		}
		if got := fgAt(scr, ix, iy); got != dim {
			t.Fatalf("INTPRET wears %d, want the vignette's dim foam %d", got, dim)
		}
		if got := fgAt(scr, dx, dy); got != dim {
			t.Fatalf("DOT wears %d, want the same dim foam %d as the block above — equally shaded", got, dim)
		}
		if got := fgAt(scr, mx, my); got != faint {
			t.Fatalf("MXV wears %d, want the barely-visible foam %d", got, faint)
		}
	})
	t.Run("happy: the stage floor is the Rose Pine base and the column carries its octal gutter", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		if got := bgAt(scr, 0, 0); got != danzig.Base256 {
			t.Fatalf("the stage floor wears %d, want the Rose Pine base %d", got, danzig.Base256)
		}
		if got := bgAt(scr, stageW-1, stageH-1); got != danzig.Base256 {
			t.Fatalf("the floor must reach the far corner, got %d", got)
		}
		_, vy := mustSee(t, scr, "VXV")
		line := rowText(scr, vy)
		addr := regexp.MustCompile(`[0-7]{5}`).FindString(line)
		if addr == "" {
			t.Fatalf("the op row %q must carry a five-digit octal address — scrolling code has a gutter", line)
		}
		if strings.Index(line, addr) > strings.Index(line, "VXV") {
			t.Fatal("the address gutter leads the line, the code follows")
		}
	})
	t.Run("happy: the first caption names the first stop", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		mustSee(t, scr, Instructions()[0].Caption)
	})
	t.Run("unhappy: a tiny stage still opens on VXV without a panic", func(t *testing.T) {
		s := New()
		s.Start()
		defer s.Stop()
		scr := screenplay.NewScreen(12, 4)
		s.Render(scr)
		if _, _, ok := findOn(scr, "VXV"); !ok {
			t.Fatal("even a tiny stage opens on the spotlit op")
		}
	})
	t.Run("unhappy: a nil screen and a stop before the first render never panic", func(t *testing.T) {
		s := New()
		s.Start()
		s.Render(nil)
		s.Stop()
		s2 := New()
		s2.Start()
		s2.Update(1)
		s2.Stop()
	})
}

func TestScroll(t *testing.T) {
	t.Run("happy: the second stop spotlights DOT with VXV dimmed above and VXSC barely below", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, stock.StopStart(1)+0.05)
		dx, dy := mustSee(t, scr, "DOT")
		if want := anchorY(stageH); dy != want {
			t.Fatalf("DOT sits on row %d, want the anchor %d", dy, want)
		}
		if got := fgAt(scr, dx, dy); got != shade(inkFoam, 0) {
			t.Fatalf("the spotlight must hand over to DOT: fg %d, want %d", got, shade(inkFoam, 0))
		}
		vx, vy := mustSee(t, scr, "VXV")
		if vy >= dy {
			t.Fatalf("VXV (row %d) must have scrolled above DOT (row %d)", vy, dy)
		}
		if got := fgAt(scr, vx, vy); got != shade(inkFoam, 1) {
			t.Fatalf("VXV steps back into the vignette: fg %d, want %d", got, shade(inkFoam, 1))
		}
		sx, sy := mustSee(t, scr, "VXSC")
		if got := fgAt(scr, sx, sy); got != shade(inkFoam, 2) {
			t.Fatalf("VXSC surfaces barely visible: fg %d, want %d", got, shade(inkFoam, 2))
		}
		mustNotSee(t, scr, "DAD")
		mustNotSee(t, scr, "INTPRET")
		mustSee(t, scr, Instructions()[1].Caption)
	})
	t.Run("happy: the glide carries the code upward and lands exactly on the anchor", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, stock.GlideStart(0)+0.3)
		_, mid := mustSee(t, scr, "DOT")
		if want := anchorY(stageH); mid <= want {
			t.Fatalf("mid-glide DOT (row %d) must still ride below the anchor %d", mid, want)
		}
		scr = seek(t, s, 0.2)
		_, later := mustSee(t, scr, "DOT")
		if later > mid {
			t.Fatalf("the code must scroll upward: DOT row %d then %d", mid, later)
		}
		scr = seek(t, s, stock.StopStart(1)-stock.GlideStart(0)-0.5-0.02)
		_, before := mustSee(t, scr, "DOT")
		if want := anchorY(stageH); before != want {
			t.Fatalf("one frame before the hold DOT sits on row %d, want the anchor %d — the glide lands first", before, want)
		}
		scr = seek(t, s, 0.1)
		_, after := mustSee(t, scr, "DOT")
		if after != before {
			t.Fatalf("DOT hopped %d→%d on the very frame the hold began", before, after)
		}
	})
	t.Run("unhappy: the glide never overshoots and never wobbles back", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		pitch := len(Blocks()[1].Lines) + 1
		_ = seek(t, s, stock.GlideStart(0)+0.02)
		last := math.MaxInt32
		anchor := anchorY(stageH)
		for i := 0; i < 18; i++ {
			scr := seek(t, s, 0.05)
			_, y := mustSee(t, scr, "DOT")
			if y > last {
				t.Fatalf("DOT wobbled back down: row %d after %d", y, last)
			}
			if y < anchor || y > anchor+pitch {
				t.Fatalf("DOT overshot its rails: row %d, want within [%d, %d]", y, anchor, anchor+pitch)
			}
			last = y
		}
		if last != anchor {
			t.Fatalf("the sweep must end landed on the anchor %d, got %d", anchor, last)
		}
	})
	t.Run("happy: the last stop spotlights DAD's stamp with EXIT dim below, and holds forever", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, stock.StopStart(4)+0.1)
		ax, ay := mustSee(t, scr, "DAD")
		if want := anchorY(stageH); ay != want {
			t.Fatalf("DAD sits on row %d, want the anchor %d", ay, want)
		}
		if got := fgAt(scr, ax, ay); got != shade(inkFoam, 0) {
			t.Fatalf("DAD must hold the spotlight: fg %d, want %d", got, shade(inkFoam, 0))
		}
		mustSee(t, scr, "carry on")
		ex, ey := mustSee(t, scr, "EXIT")
		if ey <= ay {
			t.Fatalf("EXIT (row %d) rides below the spotlight (row %d)", ey, ay)
		}
		if got := fgAt(scr, ex, ey); got != shade(inkFoam, 1) {
			t.Fatalf("EXIT is never spotlit — fg %d, want the dim %d", got, shade(inkFoam, 1))
		}
		mustSee(t, scr, "VXSC")
		mustNotSee(t, scr, "DOT")
		mustSee(t, scr, Instructions()[4].Caption)

		before := rowText(scr, ay)
		scr = seek(t, s, 30)
		if got := rowText(scr, ay); got != before {
			t.Fatalf("the final hold drifted:\n%q\n%q", before, got)
		}
	})
	t.Run("unhappy: there is no sixth stop — a long wait never spotlights EXIT", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, stock.StopStart(4)+100)
		ex, ey := mustSee(t, scr, "EXIT")
		if got := fgAt(scr, ex, ey); got == shade(inkFoam, 0) {
			t.Fatal("EXIT stole the spotlight — the walkthrough ends on DAD")
		}
		_, ay := mustSee(t, scr, "DAD")
		if want := anchorY(stageH); ay != want {
			t.Fatalf("DAD must still hold the anchor %d, got row %d", want, ay)
		}
	})
}

func TestScenePlaysConfig(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: New plays the Active knobs on the first curtain", func(t *testing.T) {
		t.Cleanup(Reset)
		fast := DefaultConfig()
		fast.HoldSeconds = 0.5
		if err := Use(fast); err != nil {
			t.Fatalf("Use: %v", err)
		}
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, fast.StopStart(1)+0.1)
		dx, dy := mustSee(t, scr, "DOT")
		if got := fgAt(scr, dx, dy); got != shade(inkFoam, 0) {
			t.Fatalf("a fast hold must already spotlight DOT: fg %d, want %d", got, shade(inkFoam, 0))
		}
	})
	t.Run("happy: a nudged knob is what the replay plays", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		scr := seek(t, s, 0.3)
		vx, vy := mustSee(t, scr, "VXV")
		if got := fgAt(scr, vx, vy); got != shade(inkFoam, 0) {
			t.Fatal("test premise: the stock show opens on VXV")
		}
		s.Stop()
		s.Cfg.HoldSeconds = 0.5
		s.Start()
		_ = paint(s)
		scr = seek(t, s, s.Cfg.StopStart(1)+0.1)
		dx, dy := mustSee(t, scr, "DOT")
		if got := fgAt(scr, dx, dy); got != shade(inkFoam, 0) {
			t.Fatalf("the replay must ride the nudged hold: fg %d, want %d", got, shade(inkFoam, 0))
		}
	})
	t.Run("unhappy: changing knobs mid-flight never retimes the running show", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		_ = seek(t, s, 0.3)
		s.Cfg.HoldSeconds = 0.1
		s.Cfg.GlideSeconds = StepSeconds
		scr := seek(t, s, 1.7)
		vx, vy := mustSee(t, scr, "VXV")
		if got := fgAt(scr, vx, vy); got != shade(inkFoam, 0) {
			t.Fatalf("a ride in the air keeps the knobs it launched with: fg %d, want %d", got, shade(inkFoam, 0))
		}
	})
}

func TestSceneLifecycle(t *testing.T) {
	t.Run("happy: a resize keeps the clock — no fall back to the first stop", func(t *testing.T) {
		s := opened(t)
		defer s.Stop()
		_ = seek(t, s, stock.StopStart(1)+0.2)
		big := screenplay.NewScreen(110, 32)
		s.Render(big)
		dx, dy := mustSee(t, big, "DOT")
		if want := anchorY(32); dy != want {
			t.Fatalf("after the resize DOT sits on row %d, want the new anchor %d", dy, want)
		}
		if got := fgAt(big, dx, dy); got != shade(inkFoam, 0) {
			t.Fatalf("the resize must keep the spotlight on DOT: fg %d", got)
		}
	})
	t.Run("happy: Stop then Start replays from the top", func(t *testing.T) {
		s := opened(t)
		_ = seek(t, s, stock.StopStart(4)+1)
		s.Stop()
		s.Start()
		scr := paint(s)
		vx, vy := mustSee(t, scr, "VXV")
		if got := fgAt(scr, vx, vy); got != shade(inkFoam, 0) {
			t.Fatalf("the replay must open back on VXV: fg %d", got)
		}
		mustSee(t, scr, "INTPRET")
		s.Stop()
	})
	t.Run("happy: the bill is one scene named Interpreter", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "Interpreter" || b[0].Scene == nil {
			t.Fatalf("the bill must carry the Interpreter scene, got %+v", b[0])
		}
	})
	t.Run("unhappy: after the one scene there is nothing left, and dt<=0 holds the clock", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != 1 || p.CurrentName() != "Interpreter" {
			t.Fatalf("the show opens on %d %q, want one Interpreter", p.Len(), p.CurrentName())
		}
		if p.Next() {
			t.Fatal("after the walkthrough there is nothing left")
		}

		s := opened(t)
		defer s.Stop()
		scr := paint(s)
		_, before := mustSee(t, scr, "VXV")
		s.Update(0)
		s.Update(-3)
		scr = paint(s)
		if _, after := mustSee(t, scr, "VXV"); after != before {
			t.Fatal("dt<=0 must hold the spotlight still")
		}
	})
}
