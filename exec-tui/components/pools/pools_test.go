package pools

// Tests written FIRST: the pools package is the Executive's job memory
// as two scene components — the core set view (eight 12-word register
// blocks, CS1…CS8, two stacks of four, alarm 1202 when the eighth
// fills) and the VAC view (five 44-word vector areas, VC1…VC5, one
// stack, alarm 1201). Add parks a job in the lowest free slot and
// reports which; a full pool refuses. Remove frees the lowest slot
// wearing that job's name; an unknown name is refused. Every job
// carries an ink — the xterm-256 color that relates it to the same
// job's lanes on the other graphs — and its box wears it: the name·prio
// glyphs and the border both, after an arrival flash of FlashSeconds
// in FlashInk. A job with no ink falls back to DefaultInk, the sim's
// job green. Free boxes sit dim with their label and the word free.
// The title row counts busy/cap, turns red one slot from full, and a
// full pool raises its program alarm chip white-on-red. Jobs survive
// a resize (Stop then Start); before Start and after Stop the stage
// is empty; dt <= 0 never advances the flash; a nil view never
// panics.

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

var _ screenplay.Component = (*View)(nil)

const (
	stageW = 60
	stageH = 20
)

// rowGlyphs is one rendered row as a plain string of runes.
func rowGlyphs(sp sprite.Sprite, r int) string {
	rs := make([]rune, sp.Width)
	for c := 0; c < sp.Width; c++ {
		ch := sp.At(r, c).Ch
		if ch == 0 {
			ch = ' '
		}
		rs[c] = ch
	}
	return string(rs)
}

// findText locates text on the stage and returns its row and rune column.
func findText(sp sprite.Sprite, text string) (r, c int, ok bool) {
	for row := 0; row < sp.Height; row++ {
		line := rowGlyphs(sp, row)
		if i := strings.Index(line, text); i >= 0 {
			return row, utf8.RuneCountInString(line[:i]), true
		}
	}
	return 0, 0, false
}

// countText tallies every occurrence of text across the stage.
func countText(sp sprite.Sprite, text string) int {
	n := 0
	for row := 0; row < sp.Height; row++ {
		n += strings.Count(rowGlyphs(sp, row), text)
	}
	return n
}

// mustFind fails the test unless text is on the stage.
func mustFind(t *testing.T, sp sprite.Sprite, text string) (r, c int) {
	t.Helper()
	r, c, ok := findText(sp, text)
	if !ok {
		t.Fatalf("stage must show %q:\n%s", text, sprite.Render(sp))
	}
	return r, c
}

// borderAbove is the top-border cell over the label at (r, c): the box
// wears its ink there.
func borderAbove(t *testing.T, sp sprite.Sprite, r, c int) sprite.Cell {
	t.Helper()
	cell := sp.At(r-1, c)
	if cell.Ch != '─' {
		t.Fatalf("cell above the label at (%d,%d) is %q, want the top border ─", r-1, c, cell.Ch)
	}
	return cell
}

func TestNewPools(t *testing.T) {
	t.Run("happy: the core set view holds eight free slots CS1…CS8 under CORE SETS 0/8", func(t *testing.T) {
		v := NewCoreSets()
		if v.Cap() != 8 || v.Busy() != 0 || v.Full() {
			t.Fatalf("a fresh core set view must be 0 of 8 and not full, got busy %d cap %d full %v", v.Busy(), v.Cap(), v.Full())
		}
		v.Start(stageW, stageH)
		sp := v.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("render is %dx%d, want the %dx%d stage", sp.Width, sp.Height, stageW, stageH)
		}
		mustFind(t, sp, "CORE SETS")
		mustFind(t, sp, "0/8")
		for _, label := range []string{"CS1", "CS2", "CS3", "CS4", "CS5", "CS6", "CS7", "CS8"} {
			mustFind(t, sp, label)
		}
		if got := countText(sp, "free"); got != 8 {
			t.Fatalf("a fresh pool shows %d free boxes, want 8", got)
		}
		if _, _, ok := findText(sp, "VC1"); ok {
			t.Fatal("the core set view must not carry VAC labels")
		}
	})
	t.Run("happy: the VAC view holds five free slots VC1…VC5 under VAC AREAS 0/5", func(t *testing.T) {
		v := NewVACs()
		if v.Cap() != 5 || v.Busy() != 0 || v.Full() {
			t.Fatalf("a fresh VAC view must be 0 of 5 and not full, got busy %d cap %d full %v", v.Busy(), v.Cap(), v.Full())
		}
		v.Start(stageW, stageH)
		sp := v.Render()
		mustFind(t, sp, "VAC AREAS")
		mustFind(t, sp, "0/5")
		for _, label := range []string{"VC1", "VC2", "VC3", "VC4", "VC5"} {
			mustFind(t, sp, label)
		}
		if got := countText(sp, "free"); got != 5 {
			t.Fatalf("a fresh pool shows %d free boxes, want 5", got)
		}
		if _, _, ok := findText(sp, "CS1"); ok {
			t.Fatal("the VAC view must not carry core set labels")
		}
	})
	t.Run("unhappy: JobAt refuses free slots and slots off the pool", func(t *testing.T) {
		v := NewCoreSets()
		for _, slot := range []int{-1, 0, 3, 8, 100} {
			if _, ok := v.JobAt(slot); ok {
				t.Fatalf("JobAt(%d) on a fresh pool must be refused", slot)
			}
		}
	})
}

func TestAdd(t *testing.T) {
	t.Run("happy: Add fills the lowest free slot and hands back its number", func(t *testing.T) {
		v := NewCoreSets()
		jobs := []Job{
			{Name: "SERVICER", Prio: 20, Ink: 83},
			{Name: "CHARIN", Prio: 30, Ink: 213},
			{Name: "MONITOR", Prio: 26, Ink: 220},
		}
		for i, j := range jobs {
			slot, ok := v.Add(j)
			if !ok || slot != i {
				t.Fatalf("Add %q landed in slot %d ok=%v, want slot %d", j.Name, slot, ok, i)
			}
		}
		if v.Busy() != 3 {
			t.Fatalf("three adds leave %d busy, want 3", v.Busy())
		}
		got, ok := v.JobAt(1)
		if !ok || got != jobs[1] {
			t.Fatalf("JobAt(1) = %+v ok=%v, want %+v", got, ok, jobs[1])
		}
	})
	t.Run("happy: a freed slot is reused lowest-first", func(t *testing.T) {
		v := NewCoreSets()
		v.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		v.Add(Job{Name: "CHARIN", Prio: 30, Ink: 213})
		v.Add(Job{Name: "MONITOR", Prio: 26, Ink: 220})
		if !v.Remove("CHARIN") {
			t.Fatal("Remove(CHARIN) must free slot 1")
		}
		slot, ok := v.Add(Job{Name: "DAP", Prio: 31, Ink: 214})
		if !ok || slot != 1 {
			t.Fatalf("Add after a free must reuse slot 1, got %d ok=%v", slot, ok)
		}
		got, _ := v.JobAt(1)
		if got.Name != "DAP" {
			t.Fatalf("slot 1 holds %q, want DAP", got.Name)
		}
	})
	t.Run("unhappy: a full pool refuses the add and nothing changes", func(t *testing.T) {
		v := NewVACs()
		names := []string{"SERVICER", "CHARIN", "MONITOR", "RR READ", "LR READ"}
		for _, n := range names {
			if _, ok := v.Add(Job{Name: n, Prio: 20, Ink: 83}); !ok {
				t.Fatalf("filling the pool must accept %q", n)
			}
		}
		if !v.Full() {
			t.Fatal("five adds must fill the five VACs")
		}
		if slot, ok := v.Add(Job{Name: "GYRO COMP", Prio: 22, Ink: 255}); ok {
			t.Fatalf("a full pool accepted GYRO COMP into slot %d — 1201 means no room", slot)
		}
		if v.Busy() != 5 {
			t.Fatalf("the refused add changed busy to %d, want 5", v.Busy())
		}
		for i, n := range names {
			got, ok := v.JobAt(i)
			if !ok || got.Name != n {
				t.Fatalf("slot %d holds %q ok=%v, want %q untouched", i, got.Name, ok, n)
			}
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("happy: Remove frees the lowest slot wearing the name and leaves later copies", func(t *testing.T) {
		v := NewCoreSets()
		v.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		v.Add(Job{Name: "CHARIN", Prio: 30, Ink: 213})
		v.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		if !v.Remove("SERVICER") {
			t.Fatal("Remove(SERVICER) must free a slot")
		}
		if v.Busy() != 2 {
			t.Fatalf("one remove leaves %d busy, want 2", v.Busy())
		}
		if _, ok := v.JobAt(0); ok {
			t.Fatal("the lowest SERVICER copy in slot 0 must be the one freed")
		}
		got, ok := v.JobAt(2)
		if !ok || got.Name != "SERVICER" {
			t.Fatalf("the later SERVICER copy must stay in slot 2, got %+v ok=%v", got, ok)
		}
	})
	t.Run("unhappy: a name not in the pool is refused and nothing changes", func(t *testing.T) {
		v := NewVACs()
		if v.Remove("SERVICER") {
			t.Fatal("removing from an empty pool must be refused")
		}
		v.Add(Job{Name: "CHARIN", Prio: 30, Ink: 213})
		if v.Remove("MONITOR") {
			t.Fatal("removing a job the pool never held must be refused")
		}
		if v.Busy() != 1 {
			t.Fatalf("the refused remove changed busy to %d, want 1", v.Busy())
		}
	})
}

func TestRenderBoxes(t *testing.T) {
	t.Run("happy: a busy box wears the job's ink on the name·prio and the border, a free box sits dim", func(t *testing.T) {
		v := NewCoreSets()
		v.Start(stageW, stageH)
		v.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		v.Update(FlashSeconds + 0.05)
		sp := v.Render()
		r, c := mustFind(t, sp, "SERVICER·20")
		if got := sp.At(r, c).FG; got != 83 {
			t.Fatalf("the job name wears ink %d, want the job's 83", got)
		}
		lr, lc := mustFind(t, sp, "CS1")
		if got := sp.At(lr, lc).FG; got != LabelInk {
			t.Fatalf("a busy label wears %d, want LabelInk %d", got, LabelInk)
		}
		if got := borderAbove(t, sp, lr, lc).FG; got != 83 {
			t.Fatalf("the busy border wears %d, want the job's 83", got)
		}
		fr, fc := mustFind(t, sp, "free")
		if got := sp.At(fr, fc).FG; got != DimInk {
			t.Fatalf("a free box's word wears %d, want DimInk %d", got, DimInk)
		}
		lr2, lc2 := mustFind(t, sp, "CS2")
		if got := borderAbove(t, sp, lr2, lc2).FG; got != DimInk {
			t.Fatalf("a free border wears %d, want DimInk %d", got, DimInk)
		}
		mustFind(t, sp, "1/8")
	})
	t.Run("happy: the title wears TitleInk and the calm count wears DimInk", func(t *testing.T) {
		v := NewVACs()
		v.Start(stageW, stageH)
		sp := v.Render()
		tr, tc := mustFind(t, sp, "VAC AREAS")
		if got := sp.At(tr, tc).FG; got != TitleInk {
			t.Fatalf("the title wears %d, want TitleInk %d", got, TitleInk)
		}
		cr, cc := mustFind(t, sp, "0/5")
		if got := sp.At(cr, cc).FG; got != DimInk {
			t.Fatalf("a calm count wears %d, want DimInk %d", got, DimInk)
		}
	})
	t.Run("happy: a job with no prio shows its name alone", func(t *testing.T) {
		v := NewVACs()
		v.Start(stageW, stageH)
		v.Add(Job{Name: "IDLE", Ink: 245})
		sp := v.Render()
		mustFind(t, sp, "IDLE")
		if _, _, ok := findText(sp, "IDLE·"); ok {
			t.Fatal("a prio-less job must not print a · separator")
		}
	})
	t.Run("unhappy: a long name is truncated inside its box, never over the border", func(t *testing.T) {
		v := NewVACs()
		v.Start(stageW, stageH)
		v.Add(Job{Name: "GUIDANCE AND NAV", Prio: 5, Ink: 87})
		sp := v.Render()
		mustFind(t, sp, "GUIDANCE A·5")
		if _, _, ok := findText(sp, "GUIDANCE AN"); ok {
			t.Fatal("the name must be cut to keep ·prio inside the box")
		}
		lr, lc := mustFind(t, sp, "VC1")
		if got := sp.At(lr, lc+BoxW-2).Ch; got != '│' {
			t.Fatalf("the right border after the text is %q, want │", got)
		}
	})
	t.Run("unhappy: a job with no ink falls back to the sim's job green", func(t *testing.T) {
		v := NewCoreSets()
		v.Start(stageW, stageH)
		v.Add(Job{Name: "EXEC", Prio: 7})
		v.Update(FlashSeconds + 0.05)
		sp := v.Render()
		r, c := mustFind(t, sp, "EXEC·7")
		if got := sp.At(r, c).FG; got != DefaultInk {
			t.Fatalf("an inkless job wears %d, want DefaultInk %d", got, DefaultInk)
		}
		lr, lc := mustFind(t, sp, "CS1")
		if got := borderAbove(t, sp, lr, lc).FG; got != DefaultInk {
			t.Fatalf("an inkless border wears %d, want DefaultInk %d", got, DefaultInk)
		}
	})
}

func TestFlash(t *testing.T) {
	t.Run("happy: a fresh add flashes the border, then settles into the job's ink", func(t *testing.T) {
		v := NewVACs()
		v.Start(stageW, stageH)
		v.Add(Job{Name: "RR READ", Prio: 32, Ink: 87})
		sp := v.Render()
		lr, lc := mustFind(t, sp, "VC1")
		if got := borderAbove(t, sp, lr, lc).FG; got != FlashInk {
			t.Fatalf("a fresh border wears %d, want the FlashInk %d", got, FlashInk)
		}
		v.Update(FlashSeconds + 0.05)
		sp = v.Render()
		if got := borderAbove(t, sp, lr, lc).FG; got != 87 {
			t.Fatalf("a settled border wears %d, want the job's 87", got)
		}
	})
	t.Run("happy: only the newest box flashes — settled neighbors keep their ink", func(t *testing.T) {
		v := NewVACs()
		v.Start(stageW, stageH)
		v.Add(Job{Name: "RR READ", Prio: 32, Ink: 87})
		v.Update(FlashSeconds + 0.05)
		v.Add(Job{Name: "LR READ", Prio: 32, Ink: 75})
		sp := v.Render()
		r1, c1 := mustFind(t, sp, "VC1")
		if got := borderAbove(t, sp, r1, c1).FG; got != 87 {
			t.Fatalf("the settled box re-flashed to %d, want 87", got)
		}
		r2, c2 := mustFind(t, sp, "VC2")
		if got := borderAbove(t, sp, r2, c2).FG; got != FlashInk {
			t.Fatalf("the new box wears %d, want the FlashInk %d", got, FlashInk)
		}
	})
	t.Run("unhappy: dt <= 0 never burns the flash down", func(t *testing.T) {
		v := NewVACs()
		v.Start(stageW, stageH)
		v.Add(Job{Name: "RR READ", Prio: 32, Ink: 87})
		v.Update(0)
		v.Update(-5)
		sp := v.Render()
		lr, lc := mustFind(t, sp, "VC1")
		if got := borderAbove(t, sp, lr, lc).FG; got != FlashInk {
			t.Fatalf("dt <= 0 moved the flash to %d — time never runs backwards", got)
		}
	})
}

func TestFullAlarm(t *testing.T) {
	fill := func(v *View, n int) {
		names := []string{"SERVICER", "CHARIN", "MONITOR", "RR READ", "LR READ", "GYRO COMP", "DAP", "SELFCHK"}
		for i := 0; i < n; i++ {
			v.Add(Job{Name: names[i], Prio: 20 + i, Ink: 83})
		}
	}
	t.Run("happy: one slot from full the count turns red; full raises the alarm chip", func(t *testing.T) {
		v := NewVACs()
		v.Start(stageW, stageH)
		fill(v, 4)
		sp := v.Render()
		cr, cc := mustFind(t, sp, "4/5")
		if got := sp.At(cr, cc).FG; got != RedInk {
			t.Fatalf("one from full the count wears %d, want RedInk %d", got, RedInk)
		}
		if _, _, ok := findText(sp, "→ 1201"); ok {
			t.Fatal("the alarm chip must wait for the last slot")
		}
		v.Add(Job{Name: "SELFCHK", Prio: 1, Ink: 245})
		sp = v.Render()
		mustFind(t, sp, "5/5")
		ar, ac := mustFind(t, sp, "→ 1201")
		cell := sp.At(ar, ac)
		if cell.FG != AlarmFG || cell.BG != AlarmBG {
			t.Fatalf("the alarm chip wears %d on %d, want %d on %d — white on red", cell.FG, cell.BG, AlarmFG, AlarmBG)
		}
		if _, _, ok := findText(sp, "→ 1202"); ok {
			t.Fatal("a full VAC pool raises 1201, never the core sets' 1202")
		}
	})
	t.Run("happy: the core sets raise 1202 and a remove clears the chip", func(t *testing.T) {
		v := NewCoreSets()
		v.Start(stageW, stageH)
		fill(v, 8)
		sp := v.Render()
		mustFind(t, sp, "→ 1202")
		if !v.Remove("SERVICER") {
			t.Fatal("draining one core set must work")
		}
		sp = v.Render()
		if _, _, ok := findText(sp, "→ 1202"); ok {
			t.Fatal("freeing a slot must clear the alarm chip")
		}
		cr, cc := mustFind(t, sp, "7/8")
		if got := sp.At(cr, cc).FG; got != RedInk {
			t.Fatalf("seven of eight still wears %d, want the red warning %d", got, RedInk)
		}
	})
	t.Run("unhappy: a calm pool never shows red or the alarm", func(t *testing.T) {
		v := NewCoreSets()
		v.Start(stageW, stageH)
		fill(v, 6)
		sp := v.Render()
		cr, cc := mustFind(t, sp, "6/8")
		if got := sp.At(cr, cc).FG; got != DimInk {
			t.Fatalf("six of eight wears %d, want the calm DimInk %d", got, DimInk)
		}
		if _, _, ok := findText(sp, "→ 1202"); ok {
			t.Fatal("no alarm below a full pool")
		}
	})
}

func TestLifecycle(t *testing.T) {
	t.Run("happy: jobs survive a resize — Stop then Start keeps every slot", func(t *testing.T) {
		v := NewCoreSets()
		v.Start(stageW, stageH)
		v.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		v.Add(Job{Name: "CHARIN", Prio: 30, Ink: 213})
		v.Stop()
		v.Start(76, 24)
		if v.Busy() != 2 {
			t.Fatalf("the resize dropped jobs — busy %d, want 2", v.Busy())
		}
		sp := v.Render()
		if sp.Width != 76 || sp.Height != 24 {
			t.Fatalf("render is %dx%d, want the new 76x24 stage", sp.Width, sp.Height)
		}
		mustFind(t, sp, "SERVICER·20")
		mustFind(t, sp, "CHARIN·30")
	})
	t.Run("unhappy: before Start and after Stop the stage is empty", func(t *testing.T) {
		v := NewVACs()
		if sp := v.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("before Start the view renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		v.Start(stageW, stageH)
		v.Stop()
		if sp := v.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("after Stop the view renders %dx%d, want nothing", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a zero stage renders empty and a tiny stage clips, without panic", func(t *testing.T) {
		v := NewCoreSets()
		v.Start(0, 0)
		v.Update(1)
		if sp := v.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a 0x0 stage renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		v.Stop()
		v.Start(7, 4)
		v.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		sp := v.Render()
		if sp.Width != 7 || sp.Height != 4 {
			t.Fatalf("a tiny stage renders %dx%d, want 7x4", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a nil view refuses every call without panic", func(t *testing.T) {
		var v *View
		v.Start(10, 5)
		v.Update(1)
		if sp := v.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatal("a nil view must render nothing")
		}
		v.Stop()
		if _, ok := v.Add(Job{Name: "SERVICER"}); ok {
			t.Fatal("a nil view must refuse Add")
		}
		if v.Remove("SERVICER") {
			t.Fatal("a nil view must refuse Remove")
		}
		if v.Busy() != 0 || v.Cap() != 0 || v.Full() {
			t.Fatal("a nil view holds nothing and is never full")
		}
		if _, ok := v.JobAt(0); ok {
			t.Fatal("a nil view has no jobs")
		}
	})
}
