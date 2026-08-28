package coreset2

// Tests written FIRST: Core Sets Two picks up exactly where the Core
// Set breakdown cut — the held bits frame, the parked PRIORITY word
// over six priority bits and nine VAC-address bits, every label and
// the caption in place — and teaches the scan. Act one, the pickup:
// the opening frame is identical, row for row, to scene one's hold.
// Act two, the roster: the word burns away while six real jobs land
// one per JobBeat, each wearing its own ink and its own priority —
// six different numbers. Act three, the sweep: the roster dissolves
// to an empty stage. Act four, the code: the EJSCAN loop reveals one
// line per CodeBeat — walk every core set, read the PRIORITY word,
// keep the highest — created by showing it. Act five, scan one: five
// core sets redraw with the full word math beside each box (priority
// plus VAC address; the NOVAC jobs carry 000), the scan speaks every
// comparison beat by beat, the arrow tracks the leader, and the third
// box down — RR READ at 32000 — is SELECTED. Act six, scan two: the
// redo with a duplicated job — three SERVICER copies at the same
// PRIO 20 but ascending VAC addresses (the real 400, 454, 530) — the
// equal-priority compares fall to the VAC address, the newest copy is
// always selected, and the old copies are tagged as the stubs they
// become. The scene holds there. A resize keeps the clock; Stop then
// Start replays; a nil screen never panics.

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/pools"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

const (
	stageW = 100
	stageH = 30
)

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

// stepOneAt is the clock time scan one's step i speaks.
func stepOneAt(i int) float64 { return ScanOneStart + BuildSeconds + float64(i)*CompareBeat }

// stepTwoAt is the clock time scan two's step i speaks.
func stepTwoAt(i int) float64 { return ScanTwoStart + BuildSeconds + float64(i)*CompareBeat }

func TestPickupAct(t *testing.T) {
	t.Run("happy: the opening frame is exactly scene one's held bits frame", func(t *testing.T) {
		one := coreset.New()
		one.Start()
		oneScr := screenplay.NewScreen(stageW, stageH)
		one.Render(oneScr)
		tick(one, one.Cfg.BitsStart()+1)
		oneScr = screenplay.NewScreen(stageW, stageH)
		one.Render(oneScr)

		two := opened(t)
		twoScr := seek(t, two, 0.2)
		for y := 0; y < stageH; y++ {
			if a, b := rowText(oneScr, y), rowText(twoScr, y); a != b {
				t.Fatalf("row %d differs — scene two must pick up scene one's exact frame:\n%q\n%q", y, a, b)
			}
		}
		px, py := mustSee(t, twoScr, "PRIORITY — OCT 20")
		if got, want := fgAt(twoScr, px, py), fgAt(oneScr, px, py); got != want {
			t.Fatalf("the priority label wears %d, scene one held it in %d", got, want)
		}
		mustSee(t, twoScr, "VAC ADDRESS — OCT 400")
		mustSee(t, twoScr, coreset.CaptionBits)
	})
	t.Run("unhappy: none of scene one's earlier acts leak onto the pickup", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, 0.2)
		mustNotSee(t, scr, "CORE SETS")
		mustNotSee(t, scr, "VAC AREAS")
		mustNotSee(t, scr, "MPAC")
		mustNotSee(t, scr, "BANKSET")
	})
	t.Run("unhappy: waiting before the first render never burns the act", func(t *testing.T) {
		s := New()
		s.Start()
		tick(s, HoldSeconds+2)
		scr := paint(s)
		mustSee(t, scr, "PRIORITY — OCT 20")
		mustSee(t, scr, "VAC ADDRESS — OCT 400")
	})
}

func TestJobsAct(t *testing.T) {
	t.Run("happy: six real jobs, six different priorities", func(t *testing.T) {
		jobs := Jobs()
		if len(jobs) != 6 {
			t.Fatalf("the roster lists %d jobs, want 6", len(jobs))
		}
		seenPrio := map[int]bool{}
		seenInk := map[int]bool{}
		for _, j := range jobs {
			if j.Name == "" {
				t.Fatal("every roster job carries a name")
			}
			if j.Prio <= 0 || seenPrio[j.Prio] {
				t.Fatalf("%s at PRIO %d — every job needs its own priority", j.Name, j.Prio)
			}
			if j.Ink <= 0 || seenInk[j.Ink] {
				t.Fatalf("%s wears ink %d — every job needs its own color", j.Name, j.Ink)
			}
			seenPrio[j.Prio] = true
			seenInk[j.Ink] = true
		}
	})
	t.Run("happy: the jobs land one per JobBeat, each wearing its own ink", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, JobsStart+2*JobBeat+0.05)
		mustSee(t, scr, Jobs()[0].Name)
		mustSee(t, scr, Jobs()[1].Name)
		mustNotSee(t, scr, Jobs()[3].Name)
		x, y := mustSee(t, scr, Jobs()[1].Name)
		if got := fgAt(scr, x, y); got != Jobs()[1].Ink {
			t.Fatalf("%s wears %d, want its ink %d", Jobs()[1].Name, got, Jobs()[1].Ink)
		}
		scr = seek(t, s, 4*JobBeat+0.5)
		for _, j := range Jobs() {
			nx, ny := mustSee(t, scr, j.Name)
			if got := fgAt(scr, nx, ny); got != j.Ink {
				t.Fatalf("%s wears %d, want its ink %d", j.Name, got, j.Ink)
			}
		}
		mustSee(t, scr, "PRIO 20")
		mustSee(t, scr, "PRIO 32")
		mustSee(t, scr, CaptionJobs)
	})
	t.Run("unhappy: the pickup's word burns away as the roster arrives", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, JobsStart+PickupFadeSeconds+0.1)
		mustNotSee(t, scr, "VAC ADDRESS — OCT 400")
		mustNotSee(t, scr, "PRIORITY — OCT 20")
		mustSee(t, scr, Jobs()[0].Name)
	})
	t.Run("unhappy: the roster never shows a seventh job", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, ClearStart-0.1)
		if got := countOn(scr, "PRIO"); got != 6 {
			t.Fatalf("the roster shows %d PRIO rows, want exactly 6", got)
		}
	})
}

func TestClearAct(t *testing.T) {
	t.Run("happy: the roster dissolves away to an empty stage", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, CodeStart-0.05)
		for _, j := range Jobs() {
			mustNotSee(t, scr, j.Name)
		}
		mustNotSee(t, scr, "PRIO 20")
	})
	t.Run("unhappy: the code never shows before its act", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, CodeStart-0.05)
		mustNotSee(t, scr, CodeLines()[0])
		mustNotSee(t, scr, "EJSCAN")
	})
}

func TestCodeAct(t *testing.T) {
	t.Run("happy: the scan loop is real code — walk the sets, compare the full word", func(t *testing.T) {
		lines := CodeLines()
		if len(lines) != 6 {
			t.Fatalf("the loop reads in %d lines, want 6", len(lines))
		}
		joined := strings.Join(lines, "\n")
		for _, want := range []string{"core set", "PRIORITY", "word > best", "EJSCAN", "VAC address"} {
			if !strings.Contains(joined, want) {
				t.Fatalf("the code must speak %q:\n%s", want, joined)
			}
		}
	})
	t.Run("happy: the code reveals one line per CodeBeat under the sourced caption", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, CodeStart+0.6*CodeBeat)
		mustSee(t, scr, CodeLines()[0])
		mustNotSee(t, scr, CodeLines()[1])
		scr = seek(t, s, 6*CodeBeat-0.6*CodeBeat+0.1)
		for _, line := range CodeLines() {
			mustSee(t, scr, line)
		}
		mustSee(t, scr, CaptionCode)
	})
	t.Run("unhappy: the code burns away as scan one opens", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, ScanOneStart+CodeFadeSeconds+0.15)
		mustNotSee(t, scr, CodeLines()[1])
		mustSee(t, scr, "CS1")
	})
}

func TestScanData(t *testing.T) {
	t.Run("happy: scan one is five busy sets, the third down holds the highest word", func(t *testing.T) {
		slots := ScanOne()
		if len(slots) != 5 {
			t.Fatalf("scan one draws %d core sets, want 5", len(slots))
		}
		wantWords := []int{0o20400, 0o21000, 0o32000, 0o26454, 0o30000}
		for i, sl := range slots {
			if sl.Free {
				t.Fatalf("scan one's %s is free — every set carries a job", sl.Label)
			}
			if got := sl.Word(); got != wantWords[i] {
				t.Fatalf("%s packs word %o, want %o", sl.Label, got, wantWords[i])
			}
		}
		if w := Winner(slots); w != 2 {
			t.Fatalf("the winner is slot %d, want 2 — the third one down", w)
		}
		if slots[2].Job.Name != "RR READ" {
			t.Fatalf("the highest word belongs to %s, want RR READ", slots[2].Job.Name)
		}
	})
	t.Run("happy: scan two duplicates SERVICER at PRIO 20 up the real VAC addresses — the newest wins", func(t *testing.T) {
		slots := ScanTwo()
		if len(slots) != 5 {
			t.Fatalf("scan two draws %d core sets, want 5", len(slots))
		}
		var copies []Slot
		for _, sl := range slots {
			if sl.Job.Name == "SERVICER" {
				copies = append(copies, sl)
			}
		}
		if len(copies) != 3 {
			t.Fatalf("scan two holds %d SERVICER copies, want 3", len(copies))
		}
		wantVACs := []int{0o400, 0o454, 0o530}
		for i, c := range copies {
			if c.Job.Prio != 20 {
				t.Fatalf("every copy schedules at PRIO 20, got %d", c.Job.Prio)
			}
			if c.VACAddr != wantVACs[i] {
				t.Fatalf("copy %d claims VAC %o, want the real address %o", i, c.VACAddr, wantVACs[i])
			}
		}
		if !slots[4].Free {
			t.Fatal("scan two's last set is free — the scan must skip it")
		}
		if w := Winner(slots); w != 3 {
			t.Fatalf("the winner is slot %d, want 3 — the newest copy", w)
		}
	})
	t.Run("happy: the steps speak every comparison — seed, greater, lesser, equal-priority, skip", func(t *testing.T) {
		one := Steps(ScanOne())
		if len(one) != 5 {
			t.Fatalf("scan one speaks %d steps, want 5", len(one))
		}
		wantOne := []string{
			"CS1 — word 20400 · the first busy set leads",
			"CS2 — 21000 > 20400 · takes the lead",
			"CS3 — 32000 > 21000 · takes the lead",
			"CS4 — 26454 < 32000 · CS3 keeps the lead",
			"CS5 — 30000 < 32000 · CS3 keeps the lead",
		}
		for i, w := range wantOne {
			if one[i].Text != w {
				t.Fatalf("scan one step %d says %q, want %q", i, one[i].Text, w)
			}
		}
		if one[4].Best != 2 {
			t.Fatalf("scan one ends with the lead at %d, want 2", one[4].Best)
		}
		two := Steps(ScanTwo())
		wantTwo := []string{
			"CS1 — word 20400 · the first busy set leads",
			"CS2 — 01000 < 20400 · CS1 keeps the lead",
			"CS3 — PRIO 20 = 20 · VAC 454 > 400 · the newer copy leads",
			"CS4 — PRIO 20 = 20 · VAC 530 > 454 · the newer copy leads",
			"CS5 — PRIORITY -0 · free · skipped",
		}
		for i, w := range wantTwo {
			if two[i].Text != w {
				t.Fatalf("scan two step %d says %q, want %q", i, two[i].Text, w)
			}
		}
		if two[4].Best != 3 {
			t.Fatalf("scan two ends with the lead at %d, want 3 — the newest copy", two[4].Best)
		}
	})
	t.Run("unhappy: an all-free pool finds nothing — DUMMYJOB idles", func(t *testing.T) {
		slots := []Slot{
			{Label: "CS1", Free: true},
			{Label: "CS2", Free: true},
		}
		steps := Steps(slots)
		if len(steps) != 2 {
			t.Fatalf("an all-free pool still walks every set, got %d steps", len(steps))
		}
		for _, st := range steps {
			if !strings.Contains(st.Text, "skipped") {
				t.Fatalf("a free set must be skipped, step says %q", st.Text)
			}
			if st.Best != -1 {
				t.Fatalf("nothing leads an empty pool, best=%d", st.Best)
			}
		}
		if w := Winner(slots); w != -1 {
			t.Fatalf("an all-free pool has no winner, got %d", w)
		}
		if got := Steps(nil); len(got) != 0 {
			t.Fatalf("no slots, no steps — got %d", len(got))
		}
		if w := Winner(nil); w != -1 {
			t.Fatalf("no slots, no winner — got %d", w)
		}
	})
	t.Run("unhappy: an exact tie keeps the earlier find — EJ1 proceeds with the search", func(t *testing.T) {
		slots := []Slot{
			{Label: "CS1", Job: pools.Job{Name: "A", Prio: 21}, VACAddr: 0},
			{Label: "CS2", Job: pools.Job{Name: "B", Prio: 21}, VACAddr: 0},
		}
		steps := Steps(slots)
		if steps[1].Best != 0 {
			t.Fatalf("an identical word must keep the earlier find, best=%d", steps[1].Best)
		}
		if !strings.Contains(steps[1].Text, "tie") || !strings.Contains(steps[1].Text, "earlier") {
			t.Fatalf("the tie must speak the rule, step says %q", steps[1].Text)
		}
		older := []Slot{
			{Label: "CS1", Job: pools.Job{Name: "S", Prio: 20}, VACAddr: 0o454},
			{Label: "CS2", Job: pools.Job{Name: "S", Prio: 20}, VACAddr: 0o400},
		}
		st := Steps(older)[1]
		if st.Best != 0 {
			t.Fatalf("the lower VAC copy must lose, best=%d", st.Best)
		}
		if !strings.Contains(st.Text, "VAC 400 < 454") {
			t.Fatalf("the equal-priority loss must show the VAC compare, step says %q", st.Text)
		}
	})
}

func TestScanOneAct(t *testing.T) {
	t.Run("happy: the core sets redraw one per SlotBeat with the word math beside each", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, ScanOneStart+2*SlotBeat+0.1)
		mustSee(t, scr, "CS1")
		mustSee(t, scr, "CS3")
		mustNotSee(t, scr, "CS4")
		mustSee(t, scr, "20 + 400 = 20400")
		scr = seek(t, s, BuildSeconds-2*SlotBeat)
		for _, want := range []string{
			"SERVICER·20", "1/GYRO·21", "RR READ·32", "MONITOR·26", "CHARIN·30",
			"21 + 000 = 21000", "32 + 000 = 32000", "26 + 454 = 26454", "30 + 000 = 30000",
		} {
			mustSee(t, scr, want)
		}
		mustSee(t, scr, CaptionScanOne)
	})
	t.Run("happy: the NOVAC rows wear empty low bits — 000 in the dim ink, real addresses in the VAC ink", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, ScanOneStart+BuildSeconds+0.1)
		x, y := mustSee(t, scr, "20 + 400 = 20400")
		vacX := x + len([]rune("20 + "))
		if got := fgAt(scr, vacX, y); got != coreset.VACAddrInk {
			t.Fatalf("SERVICER's VAC address wears %d, want the VAC ink %d", got, coreset.VACAddrInk)
		}
		x2, y2 := mustSee(t, scr, "21 + 000 = 21000")
		zeroX := x2 + len([]rune("21 + "))
		if got := fgAt(scr, zeroX, y2); got != pools.DimInk {
			t.Fatalf("a NOVAC address wears %d, want the dim ink %d — the low bits are empty", got, pools.DimInk)
		}
	})
	t.Run("happy: the scan speaks each comparison and the arrow tracks the leader", func(t *testing.T) {
		s := opened(t)
		steps := Steps(ScanOne())
		scr := seek(t, s, stepOneAt(0)+0.3)
		mustSee(t, scr, steps[0].Text)
		_, lead0 := mustSee(t, scr, "◀ best")
		_, cs1 := mustSee(t, scr, "20 + 400 = 20400")
		if lead0 != cs1 {
			t.Fatalf("after the seed the arrow sits on row %d, want CS1's row %d", lead0, cs1)
		}
		scr = seek(t, s, CompareBeat)
		mustSee(t, scr, steps[1].Text)
		mustNotSee(t, scr, steps[0].Text)
		_, lead1 := mustSee(t, scr, "◀ best")
		_, cs2 := mustSee(t, scr, "21 + 000 = 21000")
		if lead1 != cs2 {
			t.Fatalf("after step one the arrow sits on row %d, want CS2's row %d", lead1, cs2)
		}
		scr = seek(t, s, 2*CompareBeat)
		mustSee(t, scr, steps[3].Text)
		_, lead3 := mustSee(t, scr, "◀ best")
		_, cs3 := mustSee(t, scr, "32 + 000 = 32000")
		if lead3 != cs3 {
			t.Fatalf("losing compares must not move the arrow: row %d, want CS3's row %d", lead3, cs3)
		}
	})
	t.Run("happy: the third box down is SELECTED", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, SelectOneStart+0.3)
		_, selY := mustSee(t, scr, "◀ SELECTED")
		_, winY := mustSee(t, scr, "32 + 000 = 32000")
		if selY != winY {
			t.Fatalf("SELECTED sits on row %d, want the winner's row %d", selY, winY)
		}
		mustSee(t, scr, CaptionWinnerOne)
		mustNotSee(t, scr, "◀ best")
	})
	t.Run("unhappy: no comparison speaks before the build completes", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, ScanOneStart+BuildSeconds-0.1)
		mustNotSee(t, scr, "◀")
		mustNotSee(t, scr, Steps(ScanOne())[0].Text)
	})
	t.Run("unhappy: nothing is SELECTED before the last compare lands", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, SelectOneStart-0.2)
		mustNotSee(t, scr, "SELECTED")
	})
}

func TestScanTwoAct(t *testing.T) {
	t.Run("happy: the redo redraws the duplicated job at ascending VAC addresses", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, ScanTwoStart+BuildSeconds+0.1)
		if got := countOn(scr, "SERVICER·20"); got != 3 {
			t.Fatalf("the redo shows %d SERVICER copies, want 3", got)
		}
		mustSee(t, scr, "20 + 400 = 20400")
		mustSee(t, scr, "20 + 454 = 20454")
		mustSee(t, scr, "20 + 530 = 20530")
		mustSee(t, scr, "SELFCHK·1")
		mustSee(t, scr, CaptionScanTwo)
	})
	t.Run("happy: the equal-priority compares fall to the VAC address and the newest copy wins", func(t *testing.T) {
		s := opened(t)
		steps := Steps(ScanTwo())
		scr := seek(t, s, stepTwoAt(2)+0.3)
		mustSee(t, scr, steps[2].Text)
		_, leadY := mustSee(t, scr, "◀ best")
		_, cs3 := mustSee(t, scr, "20 + 454 = 20454")
		if leadY != cs3 {
			t.Fatalf("the newer copy must take the lead: arrow on row %d, want row %d", leadY, cs3)
		}
		scr = seek(t, s, SelectTwoStart-stepTwoAt(2)-0.3+0.3)
		_, selY := mustSee(t, scr, "◀ SELECTED")
		_, newest := mustSee(t, scr, "20 + 530 = 20530")
		if selY != newest {
			t.Fatalf("SELECTED sits on row %d, want the newest copy's row %d", selY, newest)
		}
		mustSee(t, scr, CaptionWinnerTwo)
	})
	t.Run("happy: the passed-over copies are the stubs — tagged, starving", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, SelectTwoStart+0.5)
		if got := countOn(scr, StubTag); got != 2 {
			t.Fatalf("the hold tags %d stubs, want 2 — the two older copies", got)
		}
		_, oldY := mustSee(t, scr, "20 + 400 = 20400")
		_, tagY := mustSee(t, scr, StubTag)
		if tagY != oldY {
			t.Fatalf("the first stub tag sits on row %d, want the oldest copy's row %d", tagY, oldY)
		}
	})
	t.Run("happy: the scene holds on the lesson — a long wait changes nothing", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, SelectTwoStart+1)
		before := rowText(scr, stageH/2)
		scr = seek(t, s, 10)
		if got := rowText(scr, stageH/2); got != before {
			t.Fatalf("the hold drifted:\n%q\n%q", before, got)
		}
	})
	t.Run("unhappy: the free set is skipped — never a job, spoken as free", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, stepTwoAt(4)+0.3)
		mustSee(t, scr, Steps(ScanTwo())[4].Text)
		mustSee(t, scr, "free")
		if got := countOn(scr, "SERVICER"); got != 3 {
			t.Fatalf("the free set must stay empty — %d SERVICERs on stage, want 3", got)
		}
	})
	t.Run("unhappy: scan one's cast is gone from the redo", func(t *testing.T) {
		s := opened(t)
		scr := seek(t, s, ScanTwoStart+BuildSeconds+0.1)
		mustNotSee(t, scr, "RR READ")
		mustNotSee(t, scr, "MONITOR")
		mustNotSee(t, scr, "CHARIN")
		mustNotSee(t, scr, "1/GYRO")
	})
}

func TestSceneLifecycle(t *testing.T) {
	t.Run("happy: a resize keeps the clock — no fall back to the pickup", func(t *testing.T) {
		s := opened(t)
		_ = seek(t, s, ScanOneStart+BuildSeconds+0.5)
		big := screenplay.NewScreen(110, 32)
		s.Render(big)
		if _, _, ok := findOn(big, "RR READ·32"); !ok {
			t.Fatal("after a resize the scan must still be on stage")
		}
		if _, _, ok := findOn(big, "VAC ADDRESS — OCT 400"); ok {
			t.Fatal("a resize must not rewind to the pickup")
		}
	})
	t.Run("happy: Stop then Start replays from the top", func(t *testing.T) {
		s := opened(t)
		_ = seek(t, s, SelectTwoStart+1)
		s.Stop()
		s.Start()
		scr := paint(s)
		mustSee(t, scr, "PRIORITY — OCT 20")
		mustSee(t, scr, "VAC ADDRESS — OCT 400")
	})
	t.Run("happy: the bill is one scene named Core Sets Two", func(t *testing.T) {
		b := Bill()
		if len(b) != 1 {
			t.Fatalf("the bill holds %d scenes, want 1", len(b))
		}
		if b[0].Name != "Core Sets Two" || b[0].Scene == nil {
			t.Fatalf("the bill must carry the Core Sets Two scene, got %+v", b[0])
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
