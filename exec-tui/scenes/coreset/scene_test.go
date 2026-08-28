package coreset

// Tests written FIRST: the Core Set scene tears the Executive's
// memory open in five acts, every fact straight from Luminary099.
// Act one, the memory unit: the core set panel and the VAC panel side
// by side, tops aligned, living jobs in the boxes (SERVICER holding a
// core set AND a VAC, CHARIN a core set alone, MONITOR a pair). Act
// two, the drain: every box dissolves away one FadeBeat at a time —
// VACs first, then the core sets from the bottom of the stack — until
// only CS1 is left standing. Act three, the move: the survivor,
// relabeled plain CORE SET with no number, glides to the top center,
// LANDS EXACTLY on its parking spot, and rests there for the settle —
// only after the rest does the layout begin. The old cut moved the
// box one more cell on the very frame the words arrived; these tests
// pin the landing so the transition finishes first and the core set's
// contents display second. Act four, the anatomy: a long twelve-word
// bar builds under it one word per WordBeat — MPAC through MPAC+6,
// MODE, LOC, BANKSET, PUSHLOC, PRIORITY, the exact page-99 layout —
// each group wearing its own ink with its own sourced caption. Act
// five, the zoom: the other eleven words and the top box fade, PRIO
// glides to center stage — parked on its seat BEFORE the bits break
// open — and the 15-bit word splits: the top six bits the job's
// priority, the low nine its VAC area address — SERVICER at PRIO 20
// over VAC1 at 400, OCT 20400 in one word. The scene holds there.
// Every timing is a knob now: the acts run on the DefaultConfig
// clock, New plays the Active config, a replay picks up nudged knobs,
// and a mid-play nudge never warps the running show. A resize keeps
// the clock; Stop then Start replays from the top; a nil screen never
// panics.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/pools"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 100
	stageH = 30
)

// stock is the scene's default clock — the acts' marks in every test
// that plays the stock show.
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

// seek plays an opened scene forward to the given clock time.
func seek(t *testing.T, s *Show, at float64) *screenplay.Screen {
	t.Helper()
	tick(s, at)
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

func countOn(scr *screenplay.Screen, text string) int {
	_, h := scr.Size()
	n := 0
	for row := 0; row < h; row++ {
		n += strings.Count(rowText(scr, row), text)
	}
	return n
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
		t.Fatalf("the stage must no longer show %q", text)
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

// parkedLabel is where the CORE SET label sits once the box is parked
// at the top center: one cell inside the box's border.
func parkedLabel() (x, y int) {
	return (stageW-pools.BoxW)/2 + 1, 2
}

func TestUnitAct(t *testing.T) {
	t.Run("happy: the memory unit — both panels side by side, tops aligned, jobs alive", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, 0.5)
		_, coreY := mustSee(t, scr, "CORE SETS")
		_, vacY := mustSee(t, scr, "VAC AREAS")
		if coreY != vacY {
			t.Fatalf("the panel titles sit on rows %d and %d — nicely aligned means the same row", coreY, vacY)
		}
		coreX, _ := mustSee(t, scr, "CORE SETS")
		vacX, _ := mustSee(t, scr, "VAC AREAS")
		if coreX >= vacX {
			t.Fatalf("the core panel (x=%d) must sit left of the VAC panel (x=%d)", coreX, vacX)
		}
		if got := countOn(scr, "SERVICER·20"); got < 2 {
			t.Fatalf("SERVICER holds a core set AND a VAC — seen %d times, want at least 2", got)
		}
		mustSee(t, scr, "CHARIN·30")
		mustSee(t, scr, "MONITOR·26")
		mustSee(t, scr, CaptionUnit)
	})
	t.Run("unhappy: waiting before the first render never burns the act", func(t *testing.T) {
		s := New()
		s.Start()
		tick(s, stock.UnitSeconds+2)
		scr := paint(s)
		mustSee(t, scr, "VAC AREAS")
		mustSee(t, scr, "CORE SETS")
	})
}

func TestFadeAct(t *testing.T) {
	t.Run("happy: the boxes drain away beat by beat — VACs first, CS1 last standing", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.FadeStart()+stock.DissolveSeconds+0.06)
		mustNotSee(t, scr, "VC5")
		mustSee(t, scr, "VC1")
		mustSee(t, scr, "CS2")
		scr = seek(t, s, stock.MoveStart()-stock.FadeStart()-stock.DissolveSeconds-0.06-0.02)
		_ = scr
		scr = paint(s)
		mustNotSee(t, scr, "VC1")
		mustNotSee(t, scr, "VAC AREAS")
		mustNotSee(t, scr, "CS2")
		mustNotSee(t, scr, "CORE SETS")
		mustSee(t, scr, "CS1")
		mustSee(t, scr, "SERVICER·20")
		mustSee(t, scr, CaptionFade)
	})
	t.Run("unhappy: CS1 never dissolves at any point of the drain", func(t *testing.T) {
		s := opened(t)
		for at := 0.5; at < stock.MoveStart()-0.05; at += 0.5 {
			tick(s, 0.5)
			scr := paint(s)
			if _, _, ok := findOn(scr, "CS1"); !ok {
				t.Fatalf("CS1 vanished at t=%.2f — the survivor must stand through the whole drain", at)
			}
		}
	})
}

func TestMoveAct(t *testing.T) {
	t.Run("happy: the survivor drops its number and glides to the top center", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.MoveStart()+0.05)
		mustSee(t, scr, "CORE SET")
		mustNotSee(t, scr, "CS1")
		_, y1 := mustSee(t, scr, "CORE SET")
		scr = seek(t, s, stock.MoveSeconds/2)
		_, y2 := mustSee(t, scr, "CORE SET")
		if y2 > y1 {
			t.Fatalf("the box must glide upward: row %d then %d", y1, y2)
		}
		scr = seek(t, s, stock.MoveSeconds/2-0.02)
		x, y := mustSee(t, scr, "CORE SET")
		wantX, wantY := parkedLabel()
		if x != wantX || y != wantY {
			t.Fatalf("at the move's end the box label sits at (%d,%d), want exactly (%d,%d)", x, y, wantX, wantY)
		}
	})
	t.Run("happy: the survivor lands and rests through the settle before the first word", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.SettleStart()+0.02)
		x0, y0 := mustSee(t, scr, "CORE SET")
		mustNotSee(t, scr, "MPAC")
		wantX, wantY := parkedLabel()
		if x0 != wantX || y0 != wantY {
			t.Fatalf("the landed box label sits at (%d,%d), want exactly (%d,%d) — the glide finishes before the layout", x0, y0, wantX, wantY)
		}
		scr = seek(t, s, stock.SettleSeconds-0.04)
		x1, y1 := mustSee(t, scr, "CORE SET")
		mustNotSee(t, scr, "MPAC")
		mustSee(t, scr, CaptionMove)
		if x1 != x0 || y1 != y0 {
			t.Fatalf("the parked box drifted (%d,%d)→(%d,%d) during the settle", x0, y0, x1, y1)
		}
	})
	t.Run("unhappy: the landing never shares a frame with the word bar", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.WordsStart()-0.05)
		bx, by := mustSee(t, scr, "CORE SET")
		mustNotSee(t, scr, "MPAC")
		scr = seek(t, s, 0.1)
		mustSee(t, scr, "MPAC")
		x, y := mustSee(t, scr, "CORE SET")
		if x != bx || y != by {
			t.Fatalf("the box hopped (%d,%d)→(%d,%d) on the very frame the words arrived — first the transition, then the layout", bx, by, x, y)
		}
	})
	t.Run("unhappy: the glide's last frame already rests on the landing spot — no hop into the settle", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.SettleStart()-0.02)
		x, y := mustSee(t, scr, "CORE SET")
		wantX, wantY := parkedLabel()
		if x != wantX || y != wantY {
			t.Fatalf("one frame before the settle the box label sits at (%d,%d), want (%d,%d)", x, y, wantX, wantY)
		}
	})
	t.Run("unhappy: the move carries no panel leftovers and no job text", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.MoveStart()+stock.MoveSeconds/2)
		mustNotSee(t, scr, "SERVICER")
		mustNotSee(t, scr, "VAC")
		mustNotSee(t, scr, "CS1")
		mustNotSee(t, scr, "free")
	})
}

func TestAnatomyAct(t *testing.T) {
	t.Run("happy: the six groups are the page-99 core set — twelve words exactly", func(t *testing.T) {
		gs := Groups()
		if len(gs) != 6 {
			t.Fatalf("a core set breaks into %d groups, want 6", len(gs))
		}
		wantNames := []string{"MPAC", "MODE", "LOC", "BANKSET", "PUSHLOC", "PRIORITY"}
		wantWords := []int{7, 1, 1, 1, 1, 1}
		sum := 0
		seenInks := map[int]bool{}
		for i, g := range gs {
			if g.Name != wantNames[i] {
				t.Fatalf("group %d is %q, want %q", i, g.Name, wantNames[i])
			}
			if g.Words != wantWords[i] {
				t.Fatalf("%s spans %d words, want %d", g.Name, g.Words, wantWords[i])
			}
			if g.Ink <= 0 || seenInks[g.Ink] {
				t.Fatalf("%s wears ink %d — every group needs its own color", g.Name, g.Ink)
			}
			seenInks[g.Ink] = true
			if g.Caption == "" {
				t.Fatalf("%s carries no caption — every group explains itself", g.Name)
			}
			sum += g.Words
		}
		if sum != 12 {
			t.Fatalf("the groups span %d words — a core set is 12 registers, never 9 or 11", sum)
		}
	})
	t.Run("happy: the bar builds one word per beat under the parked box", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.WordsStart()+2.5*stock.WordBeat)
		mustSee(t, scr, "MPAC")
		mustSee(t, scr, "+1")
		mustNotSee(t, scr, "PRIO")
		scr = seek(t, s, 12*stock.WordBeat-2.5*stock.WordBeat+0.1)
		for _, label := range []string{"MPAC", "+1", "+2", "+3", "+4", "+5", "+6", "MODE", "LOC", "BANK", "PUSH", "PRIO"} {
			mustSee(t, scr, label)
		}
		mustSee(t, scr, "CORE SET")
		mustSee(t, scr, CaptionWords)
		for _, g := range Groups() {
			mustSee(t, scr, g.Caption)
		}
		x, y := mustSee(t, scr, "PRIO")
		if got := fgAt(scr, x, y); got != Groups()[5].Ink {
			t.Fatalf("the PRIO word wears %d, want the PRIORITY group ink %d", got, Groups()[5].Ink)
		}
	})
	t.Run("unhappy: the full bar never spills past twelve word labels", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.ZoomStart()-0.1)
		if got := countOn(scr, "+6"); got != 1 {
			t.Fatalf("MPAC ends at +6 — seen %d times, want exactly 1", got)
		}
		mustNotSee(t, scr, "+7")
	})
}

func TestZoomAct(t *testing.T) {
	t.Run("happy: the rest fades out while PRIO holds its slot, then it glides to center", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.ZoomStart()-0.05)
		bx, by := mustSee(t, scr, "PRIO")
		scr = seek(t, s, 0.05+stock.ZoomFadeSeconds*0.6)
		x, y := mustSee(t, scr, "PRIO")
		if x != bx || y != by {
			t.Fatalf("mid-fade PRIO moved from (%d,%d) to (%d,%d) — the glide must wait for the fade", bx, by, x, y)
		}
		mustSee(t, scr, CaptionZoom)
		scr = seek(t, s, stock.ZoomFadeSeconds*0.4+0.07)
		mustNotSee(t, scr, "MPAC")
		mustNotSee(t, scr, "MODE")
		mustNotSee(t, scr, "BANK")
		mustNotSee(t, scr, "PUSH")
		mustNotSee(t, scr, "CORE SET")
		scr = seek(t, s, stock.ZoomGlideSeconds-0.07-0.02-0.05)
		x2, _ := mustSee(t, scr, "PRIO")
		center := stageW / 2
		if x2 < center-12 || x2 > center+12 {
			t.Fatalf("PRIO sits at column %d, want near the center %d", x2, center)
		}
	})
	t.Run("happy: the priority word is parked on its seat before the bits break open", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.BitsStart()-0.05)
		bx, by := mustSee(t, scr, "PRIO")
		mustNotSee(t, scr, "VAC ADDRESS")
		scr = seek(t, s, 0.1)
		mustSee(t, scr, "VAC ADDRESS")
		x, y := mustSee(t, scr, "PRIO")
		if x != bx || y != by {
			t.Fatalf("PRIO hopped (%d,%d)→(%d,%d) on the very frame the bits arrived — the glide must land first", bx, by, x, y)
		}
	})
	t.Run("unhappy: the glide never begins before the fade-out completes", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.ZoomStart()+0.02)
		bx, by := mustSee(t, scr, "PRIO")
		for _, dt := range []float64{0.1, 0.1} {
			scr = seek(t, s, dt)
			x, y := mustSee(t, scr, "PRIO")
			if x != bx || y != by {
				t.Fatalf("PRIO left (%d,%d) for (%d,%d) inside the quarter-second fade", bx, by, x, y)
			}
		}
	})
	t.Run("unhappy: the glide's last frames never wobble the parked word", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.BitsStart()-0.09)
		x0, y0 := mustSee(t, scr, "PRIO")
		scr = seek(t, s, 0.05)
		x1, y1 := mustSee(t, scr, "PRIO")
		mustNotSee(t, scr, "VAC ADDRESS")
		scr = seek(t, s, 0.1)
		mustSee(t, scr, "VAC ADDRESS")
		x2, y2 := mustSee(t, scr, "PRIO")
		if x0 != x1 || y0 != y1 || x1 != x2 || y1 != y2 {
			t.Fatalf("PRIO wobbled at the end of the glide: (%d,%d) (%d,%d) (%d,%d)", x0, y0, x1, y1, x2, y2)
		}
	})
	t.Run("unhappy: the parked CORE SET box fades with the rest", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.BitsStart()-0.05)
		mustNotSee(t, scr, "CORE SET")
	})
}

func TestBitsAct(t *testing.T) {
	t.Run("happy: the priority word is the sourced OCT 20400 — PRIO 20 over VAC1 at 400", func(t *testing.T) {
		if PriorityWord != PrioOctal<<9|VACAddrOctal {
			t.Fatalf("PriorityWord %o must be the priority shifted over the VAC address", PriorityWord)
		}
		if PriorityWord != 0o20400 {
			t.Fatalf("the worked example is SERVICER: OCT 20400, got %o", PriorityWord)
		}
		if PrioBitCount != 6 || VACBitCount != 9 || PrioBitCount+VACBitCount != 15 {
			t.Fatalf("the split is 6 priority bits over 9 VAC bits — one 15-bit word, got %d+%d", PrioBitCount, VACBitCount)
		}
	})
	t.Run("happy: fifteen bits on stage, the top six in the priority ink, the low nine in the VAC ink", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.BitsStart()+1)
		wantBits := ""
		for i := 14; i >= 0; i-- {
			wantBits += string(rune('0' + (PriorityWord>>i)&1))
		}
		_, h := scr.Size()
		bitRow := -1
		for y := 0; y < h; y++ {
			digits := ""
			for _, ch := range rowText(scr, y) {
				if ch == '0' || ch == '1' {
					digits += string(ch)
				}
			}
			if digits == wantBits {
				bitRow = y
				break
			}
		}
		if bitRow < 0 {
			t.Fatalf("no row spells the fifteen bits %s", wantBits)
		}
		line := []rune(rowText(scr, bitRow))
		seen := 0
		for x, ch := range line {
			if ch != '0' && ch != '1' {
				continue
			}
			want := PrioInk
			if seen >= PrioBitCount {
				want = VACAddrInk
			}
			if got := fgAt(scr, x, bitRow); got != want {
				t.Fatalf("bit %d wears %d, want %d — six priority bits, then nine VAC bits", seen, got, want)
			}
			seen++
		}
		if seen != 15 {
			t.Fatalf("the bit row holds %d bits, want 15", seen)
		}
		mustSee(t, scr, "PRIORITY")
		mustSee(t, scr, "VAC ADDRESS")
		mustSee(t, scr, "OCT 20")
		mustSee(t, scr, "OCT 400")
		mustSee(t, scr, CaptionBits)
	})
	t.Run("happy: the field labels share one row, spaced under their own fields", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.BitsStart()+1)
		px, py := mustSee(t, scr, "PRIORITY — OCT 20")
		vx, vy := mustSee(t, scr, "VAC ADDRESS — OCT 400")
		if py != vy {
			t.Fatalf("the labels sit on rows %d and %d — they must share one line", py, vy)
		}
		prioEnd := px + len([]rune("PRIORITY — OCT 20"))
		if prioEnd+2 > vx {
			t.Fatalf("the labels collide: priority ends at %d, VAC begins at %d — want daylight between them", prioEnd, vx)
		}
	})
	t.Run("happy: the bits sit wide enough to carry both labels beneath them", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.BitsStart()+1)
		wantBits := ""
		for i := 14; i >= 0; i-- {
			wantBits += string(rune('0' + (PriorityWord>>i)&1))
		}
		_, h := scr.Size()
		var cols []int
		for y := 0; y < h && cols == nil; y++ {
			line := []rune(rowText(scr, y))
			digits := ""
			var xs []int
			for x, ch := range line {
				if ch == '0' || ch == '1' {
					digits += string(ch)
					xs = append(xs, x)
				}
			}
			if digits == wantBits {
				cols = xs
			}
		}
		if cols == nil {
			t.Fatalf("no row spells the fifteen bits %s", wantBits)
		}
		for i := 1; i < len(cols); i++ {
			if gap := cols[i] - cols[i-1]; gap < 3 {
				t.Fatalf("bits %d and %d sit %d apart — the row must breathe so the labels fit beneath", i-1, i, gap)
			}
		}
		if span := cols[len(cols)-1] - cols[0]; span < 50 {
			t.Fatalf("the bit row spans %d columns — too narrow to seat both labels on one line", span)
		}
	})
	t.Run("happy: the scene holds on the bits — a long wait changes nothing", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stock.BitsStart()+1)
		before := rowText(scr, stageH/2)
		scr = seek(t, s, 10)
		if got := rowText(scr, stageH/2); got != before {
			t.Fatalf("the hold drifted:\n%q\n%q", before, got)
		}
	})
	t.Run("unhappy: the octal caption never contradicts the constants", func(t *testing.T) {
		if !strings.Contains(CaptionBits, "20400") {
			t.Fatalf("the caption %q must name OCT 20400", CaptionBits)
		}
		if PrioOctal != 0o20 || VACAddrOctal != 0o400 {
			t.Fatalf("the sourced example is PRIO 20 over VAC1 at 400 — got %o and %o", PrioOctal, VACAddrOctal)
		}
	})
}

func TestScenePlaysConfig(t *testing.T) {
	t.Cleanup(Reset)
	t.Run("happy: New plays the Active knobs on the first curtain", func(t *testing.T) {
		t.Cleanup(Reset)
		fast := DefaultConfig()
		fast.UnitSeconds = 0.5
		if err := Use(fast); err != nil {
			t.Fatalf("Use: %v", err)
		}
		s := opened(t)
		scr := seek(t, s, 0.7)
		mustSee(t, scr, CaptionFade)
		mustNotSee(t, scr, CaptionUnit)
	})
	t.Run("happy: a nudged knob is what the replay plays", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, 0.7)
		mustSee(t, scr, CaptionUnit)
		s.Stop()
		s.Cfg.UnitSeconds = 0.5
		s.Start()
		_ = paint(s)
		scr = seek(t, s, 0.7)
		mustSee(t, scr, CaptionFade)
		mustNotSee(t, scr, CaptionUnit)
	})
	t.Run("unhappy: changing knobs mid-flight never retimes the running show", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, 0.3)
		mustSee(t, scr, CaptionUnit)
		s.Cfg.UnitSeconds = 0.1
		scr = seek(t, s, 0.5)
		mustSee(t, scr, CaptionUnit)
		mustNotSee(t, scr, CaptionFade)
	})
}

func TestSceneLifecycle(t *testing.T) {
	t.Run("happy: a resize keeps the clock — no fall back to the first act", func(t *testing.T) {
		s := opened(t)
		_ = seek(t, s, stock.WordsStart()+12*stock.WordBeat+0.2)
		big := screenplay.NewScreen(110, 32)
		s.Render(big)
		if _, _, ok := findOn(big, "MPAC"); !ok {
			t.Fatal("after a resize the anatomy must still be on stage")
		}
		if _, _, ok := findOn(big, "VAC AREAS"); ok {
			t.Fatal("a resize must not rewind to the memory unit")
		}
	})
	t.Run("happy: Stop then Start replays from the top", func(t *testing.T) {
		s := opened(t)
		_ = seek(t, s, stock.BitsStart()+1)
		s.Stop()
		s.Start()
		scr := paint(s)
		mustSee(t, scr, "VAC AREAS")
		mustSee(t, scr, "CORE SETS")
	})
	t.Run("happy: the bill is one scene named Core Set", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "Core Set" || b[0].Scene == nil {
			t.Fatalf("the bill must carry the Core Set scene, got %+v", b[0])
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
