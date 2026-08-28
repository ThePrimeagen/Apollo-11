package pools

// Tests written FIRST: the pools package is the Executive's job memory
// split into two layers of component. The Box is one memory slot on
// its own — the bordered pill that turns on and off, wears a job's
// ink on its text and border, flashes FlashInk on arrival, and
// carries a little label ("CS1", "VC3", or the unnumbered "CORE SET").
// The Panel is the composite: NewCoreSetPanel is eight core set boxes
// (CS1…CS8, two stacks of four, alarm 1202), NewVACPanel is five VAC
// boxes (VC1…VC5, one stack, alarm 1201), each a real Box component
// the panel starts at BoxW×BoxH and blits onto its grid under a
// title-and-count row. Add parks a job in the lowest free box and
// reports which; a full panel refuses. Remove frees the lowest box
// wearing the name. Box(i) hands out the live component and Origin(i)
// / Size() give the grid geometry, so scenes can choreograph the
// boxes themselves. Jobs survive a resize; a nil box or panel never
// panics.

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/theprimeagen/apollo-11/exec-tui/components/sprite"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

var (
	_ screenplay.Component = (*Box)(nil)
	_ screenplay.Component = (*Panel)(nil)
)

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

func TestBox(t *testing.T) {
	t.Run("happy: a box carries its label, turns on with a job, and off again", func(t *testing.T) {
		b := NewBox("CS1")
		if b.Label() != "CS1" {
			t.Fatalf("label %q, want CS1", b.Label())
		}
		if b.Busy() {
			t.Fatal("a fresh box must be free")
		}
		j := Job{Name: "SERVICER", Prio: 20, Ink: 83}
		b.Set(j)
		if !b.Busy() {
			t.Fatal("Set must turn the box on")
		}
		got, ok := b.Job()
		if !ok || got != j {
			t.Fatalf("Job() = %+v ok=%v, want %+v", got, ok, j)
		}
		b.Clear()
		if b.Busy() {
			t.Fatal("Clear must turn the box off")
		}
		if _, ok := b.Job(); ok {
			t.Fatal("a cleared box holds no job")
		}
	})
	t.Run("happy: the named constructors are the unnumbered core set and VAC", func(t *testing.T) {
		if got := NewCoreSet().Label(); got != "CORE SET" {
			t.Fatalf("NewCoreSet label %q, want CORE SET — no number", got)
		}
		if got := NewVAC().Label(); got != "VAC" {
			t.Fatalf("NewVAC label %q, want VAC", got)
		}
	})
	t.Run("happy: Set on a busy box swaps the job in place", func(t *testing.T) {
		b := NewBox("VC1")
		b.Set(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		b.Set(Job{Name: "CHARIN", Prio: 30, Ink: 213})
		got, ok := b.Job()
		if !ok || got.Name != "CHARIN" {
			t.Fatalf("the second Set must win, got %+v ok=%v", got, ok)
		}
	})
	t.Run("unhappy: a nil box refuses every call without panic", func(t *testing.T) {
		var b *Box
		b.Start(10, 5)
		b.Update(1)
		if sp := b.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatal("a nil box must render nothing")
		}
		b.Stop()
		b.Set(Job{Name: "SERVICER"})
		b.Clear()
		if b.Busy() {
			t.Fatal("a nil box is never busy")
		}
		if _, ok := b.Job(); ok {
			t.Fatal("a nil box holds no job")
		}
		if b.Label() != "" {
			t.Fatal("a nil box has no label")
		}
	})
}

func TestBoxRender(t *testing.T) {
	t.Run("happy: a busy box wears the job's ink on the name·prio and, settled, on the border", func(t *testing.T) {
		b := NewBox("CS1")
		b.Start(stageW, stageH)
		b.Set(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		b.Update(FlashSeconds + 0.05)
		sp := b.Render()
		if sp.Width != stageW || sp.Height != stageH {
			t.Fatalf("render is %dx%d, want the %dx%d stage", sp.Width, sp.Height, stageW, stageH)
		}
		r, c := mustFind(t, sp, "SERVICER·20")
		if got := sp.At(r, c).FG; got != 83 {
			t.Fatalf("the job name wears ink %d, want 83", got)
		}
		lr, lc := mustFind(t, sp, "CS1")
		if got := sp.At(lr, lc).FG; got != LabelInk {
			t.Fatalf("a busy label wears %d, want LabelInk %d", got, LabelInk)
		}
		if got := borderAbove(t, sp, lr, lc).FG; got != 83 {
			t.Fatalf("the settled border wears %d, want 83", got)
		}
	})
	t.Run("happy: a fresh Set flashes the border in FlashInk before it settles", func(t *testing.T) {
		b := NewBox("VC1")
		b.Start(stageW, stageH)
		b.Set(Job{Name: "RR READ", Prio: 32, Ink: 87})
		sp := b.Render()
		lr, lc := mustFind(t, sp, "VC1")
		if got := borderAbove(t, sp, lr, lc).FG; got != FlashInk {
			t.Fatalf("a fresh border wears %d, want FlashInk %d", got, FlashInk)
		}
		b.Update(FlashSeconds + 0.05)
		sp = b.Render()
		lr, lc = mustFind(t, sp, "VC1")
		if got := borderAbove(t, sp, lr, lc).FG; got != 87 {
			t.Fatalf("the settled border wears %d, want 87", got)
		}
	})
	t.Run("happy: a free box sits dim with its label and the word free", func(t *testing.T) {
		b := NewBox("CS2")
		b.Start(stageW, stageH)
		sp := b.Render()
		lr, lc := mustFind(t, sp, "CS2")
		if got := sp.At(lr, lc).FG; got != DimInk {
			t.Fatalf("a free label wears %d, want DimInk %d", got, DimInk)
		}
		fr, fc := mustFind(t, sp, "free")
		if got := sp.At(fr, fc).FG; got != DimInk {
			t.Fatalf("the word free wears %d, want DimInk %d", got, DimInk)
		}
		if got := borderAbove(t, sp, lr, lc).FG; got != DimInk {
			t.Fatalf("a free border wears %d, want DimInk %d", got, DimInk)
		}
	})
	t.Run("happy: a long label leaves the leftover room to the text — the unnumbered CORE SET box says only its name", func(t *testing.T) {
		b := NewCoreSet()
		b.Start(stageW, stageH)
		b.Set(Job{Ink: 83})
		b.Update(FlashSeconds + 0.05)
		sp := b.Render()
		lr, lc := mustFind(t, sp, "CORE SET")
		if got := sp.At(lr, lc).FG; got != LabelInk {
			t.Fatalf("the lone label wears %d, want LabelInk %d", got, LabelInk)
		}
		if got := borderAbove(t, sp, lr, lc).FG; got != 83 {
			t.Fatalf("the lit border wears %d, want the job's 83", got)
		}
		if _, _, ok := findText(sp, "free"); ok {
			t.Fatal("a lit box never says free")
		}
	})
	t.Run("unhappy: a long name is truncated inside the box, never over the border", func(t *testing.T) {
		b := NewBox("VC1")
		b.Start(stageW, stageH)
		b.Set(Job{Name: "GUIDANCE AND NAV", Prio: 5, Ink: 87})
		sp := b.Render()
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
		b := NewBox("CS1")
		b.Start(stageW, stageH)
		b.Set(Job{Name: "EXEC", Prio: 7})
		b.Update(FlashSeconds + 0.05)
		sp := b.Render()
		r, c := mustFind(t, sp, "EXEC·7")
		if got := sp.At(r, c).FG; got != DefaultInk {
			t.Fatalf("an inkless job wears %d, want DefaultInk %d", got, DefaultInk)
		}
	})
	t.Run("unhappy: dt <= 0 never burns the flash down", func(t *testing.T) {
		b := NewBox("VC1")
		b.Start(stageW, stageH)
		b.Set(Job{Name: "RR READ", Prio: 32, Ink: 87})
		b.Update(0)
		b.Update(-5)
		sp := b.Render()
		lr, lc := mustFind(t, sp, "VC1")
		if got := borderAbove(t, sp, lr, lc).FG; got != FlashInk {
			t.Fatalf("dt <= 0 moved the flash to %d — time never runs backwards", got)
		}
	})
	t.Run("unhappy: before Start and after Stop the stage is empty; the job stays", func(t *testing.T) {
		b := NewBox("CS1")
		b.Set(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		if sp := b.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("before Start the box renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		b.Start(stageW, stageH)
		b.Stop()
		if sp := b.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("after Stop the box renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		if !b.Busy() {
			t.Fatal("the job is the box's identity and survives Stop")
		}
	})
}

func TestPanelComposite(t *testing.T) {
	t.Run("happy: Add fills the lowest free box and the box component shares the state", func(t *testing.T) {
		p := NewCoreSetPanel()
		if p.Cap() != 8 || p.Busy() != 0 || p.Full() {
			t.Fatalf("a fresh panel must be 0 of 8, got busy %d cap %d full %v", p.Busy(), p.Cap(), p.Full())
		}
		jobs := []Job{
			{Name: "SERVICER", Prio: 20, Ink: 83},
			{Name: "CHARIN", Prio: 30, Ink: 213},
			{Name: "MONITOR", Prio: 26, Ink: 220},
		}
		for i, j := range jobs {
			slot, ok := p.Add(j)
			if !ok || slot != i {
				t.Fatalf("Add %q landed in slot %d ok=%v, want %d", j.Name, slot, ok, i)
			}
		}
		box := p.Box(1)
		if box == nil {
			t.Fatal("Box(1) must hand out the live component")
		}
		got, ok := box.Job()
		if !ok || got != jobs[1] {
			t.Fatalf("the box component holds %+v ok=%v, want %+v", got, ok, jobs[1])
		}
		if box.Label() != "CS2" {
			t.Fatalf("slot 1 wears label %q, want CS2", box.Label())
		}
		box.Clear()
		if p.Busy() != 2 {
			t.Fatalf("clearing through the box must show on the panel, busy %d want 2", p.Busy())
		}
		if slot, ok := p.Add(Job{Name: "DAP", Prio: 31, Ink: 214}); !ok || slot != 1 {
			t.Fatalf("the freed box must be reused lowest-first, got %d ok=%v", slot, ok)
		}
	})
	t.Run("happy: Remove frees the lowest box wearing the name and leaves later copies", func(t *testing.T) {
		p := NewVACPanel()
		p.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		p.Add(Job{Name: "CHARIN", Prio: 30, Ink: 213})
		p.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		if !p.Remove("SERVICER") {
			t.Fatal("Remove(SERVICER) must free a slot")
		}
		if _, ok := p.JobAt(0); ok {
			t.Fatal("the lowest SERVICER copy in slot 0 must be the one freed")
		}
		got, ok := p.JobAt(2)
		if !ok || got.Name != "SERVICER" {
			t.Fatalf("the later copy must stay in slot 2, got %+v ok=%v", got, ok)
		}
	})
	t.Run("unhappy: a full panel refuses the add and nothing changes", func(t *testing.T) {
		p := NewVACPanel()
		names := []string{"SERVICER", "CHARIN", "MONITOR", "RR READ", "LR READ"}
		for _, n := range names {
			if _, ok := p.Add(Job{Name: n, Prio: 20, Ink: 83}); !ok {
				t.Fatalf("filling the panel must accept %q", n)
			}
		}
		if !p.Full() {
			t.Fatal("five adds must fill the five VACs")
		}
		if slot, ok := p.Add(Job{Name: "GYRO COMP", Prio: 22, Ink: 255}); ok {
			t.Fatalf("a full panel accepted GYRO COMP into slot %d — 1201 means no room", slot)
		}
		for i, n := range names {
			got, ok := p.JobAt(i)
			if !ok || got.Name != n {
				t.Fatalf("slot %d holds %q ok=%v, want %q untouched", i, got.Name, ok, n)
			}
		}
	})
	t.Run("unhappy: unknown names, off-range boxes and slots are refused", func(t *testing.T) {
		p := NewCoreSetPanel()
		if p.Remove("NOPE") {
			t.Fatal("removing a job the panel never held must be refused")
		}
		for _, i := range []int{-1, 8, 100} {
			if p.Box(i) != nil {
				t.Fatalf("Box(%d) must be nil off the grid", i)
			}
			if _, ok := p.JobAt(i); ok {
				t.Fatalf("JobAt(%d) must be refused", i)
			}
		}
		if _, ok := p.JobAt(0); ok {
			t.Fatal("JobAt on a free slot must be refused")
		}
	})
}

func TestPanelGeometry(t *testing.T) {
	t.Run("happy: the core panel is two stacks of four, the VAC panel one stack of five", func(t *testing.T) {
		p := NewCoreSetPanel()
		w, h := p.Size()
		if w != 2*BoxW || h != 1+4*BoxH {
			t.Fatalf("core panel size %dx%d, want %dx%d — a title row over a 2x4 grid", w, h, 2*BoxW, 1+4*BoxH)
		}
		for i := 0; i < 4; i++ {
			if x, y := p.Origin(i); x != 0 || y != 1+i*BoxH {
				t.Fatalf("slot %d at (%d,%d), want the left stack (0,%d)", i, x, y, 1+i*BoxH)
			}
		}
		for i := 4; i < 8; i++ {
			if x, y := p.Origin(i); x != BoxW || y != 1+(i-4)*BoxH {
				t.Fatalf("slot %d at (%d,%d), want the right stack (%d,%d)", i, x, y, BoxW, 1+(i-4)*BoxH)
			}
		}
		v := NewVACPanel()
		vw, vh := v.Size()
		if vh != 1+5*BoxH {
			t.Fatalf("VAC panel height %d, want %d — a title row over five boxes", vh, 1+5*BoxH)
		}
		if vw < BoxW {
			t.Fatalf("VAC panel width %d must hold a box of %d", vw, BoxW)
		}
		gx, _ := v.Origin(0)
		for i := 0; i < 5; i++ {
			if x, y := v.Origin(i); x != gx || y != 1+i*BoxH {
				t.Fatalf("VAC slot %d at (%d,%d), want the single stack (%d,%d)", i, x, y, gx, 1+i*BoxH)
			}
		}
	})
	t.Run("happy: a panel started at its own size paints each box exactly at its origin", func(t *testing.T) {
		p := NewCoreSetPanel()
		p.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		w, h := p.Size()
		p.Start(w, h)
		sp := p.Render()
		if sp.Width != w || sp.Height != h {
			t.Fatalf("render is %dx%d, want the exact art %dx%d", sp.Width, sp.Height, w, h)
		}
		x, y := p.Origin(0)
		if got := sp.At(y, x).Ch; got != '╭' {
			t.Fatalf("slot 0's corner at (%d,%d) is %q, want ╭", y, x, got)
		}
		lr, lc := mustFind(t, sp, "CS1")
		if lr != y+1 || lc != x+1 {
			t.Fatalf("CS1's label at (%d,%d), want inside its origin (%d,%d)", lr, lc, y+1, x+1)
		}
	})
	t.Run("unhappy: origins off the grid are refused", func(t *testing.T) {
		p := NewVACPanel()
		for _, i := range []int{-1, 5, 50} {
			if x, y := p.Origin(i); x != 0 || y != 0 {
				t.Fatalf("Origin(%d) = (%d,%d), want the zero refusal", i, x, y)
			}
		}
	})
}

func TestPanelRender(t *testing.T) {
	fill := func(p *Panel, n int) {
		names := []string{"SERVICER", "CHARIN", "MONITOR", "RR READ", "LR READ", "GYRO COMP", "DAP", "SELFCHK"}
		for i := 0; i < n; i++ {
			p.Add(Job{Name: names[i], Prio: 20 + i, Ink: 83})
		}
	}
	t.Run("happy: the panel shows its title, count, labels and free boxes", func(t *testing.T) {
		p := NewCoreSetPanel()
		p.Start(stageW, stageH)
		sp := p.Render()
		mustFind(t, sp, "CORE SETS")
		mustFind(t, sp, "0/8")
		for _, label := range []string{"CS1", "CS2", "CS3", "CS4", "CS5", "CS6", "CS7", "CS8"} {
			mustFind(t, sp, label)
		}
		if got := countText(sp, "free"); got != 8 {
			t.Fatalf("a fresh panel shows %d free boxes, want 8", got)
		}
		v := NewVACPanel()
		v.Start(stageW, stageH)
		vp := v.Render()
		mustFind(t, vp, "VAC AREAS")
		mustFind(t, vp, "0/5")
		for _, label := range []string{"VC1", "VC2", "VC3", "VC4", "VC5"} {
			mustFind(t, vp, label)
		}
	})
	t.Run("happy: one slot from full the count turns red; full raises the alarm chip", func(t *testing.T) {
		p := NewVACPanel()
		p.Start(stageW, stageH)
		fill(p, 4)
		sp := p.Render()
		cr, cc := mustFind(t, sp, "4/5")
		if got := sp.At(cr, cc).FG; got != RedInk {
			t.Fatalf("one from full the count wears %d, want RedInk %d", got, RedInk)
		}
		if _, _, ok := findText(sp, "→ 1201"); ok {
			t.Fatal("the alarm chip must wait for the last slot")
		}
		p.Add(Job{Name: "SELFCHK", Prio: 1, Ink: 245})
		sp = p.Render()
		ar, ac := mustFind(t, sp, "→ 1201")
		cell := sp.At(ar, ac)
		if cell.FG != AlarmFG || cell.BG != AlarmBG {
			t.Fatalf("the alarm chip wears %d on %d, want %d on %d", cell.FG, cell.BG, AlarmFG, AlarmBG)
		}
		if _, _, ok := findText(sp, "→ 1202"); ok {
			t.Fatal("a full VAC panel raises 1201, never the core sets' 1202")
		}
	})
	t.Run("happy: the core panel raises 1202 and a remove clears the chip", func(t *testing.T) {
		p := NewCoreSetPanel()
		p.Start(stageW, stageH)
		fill(p, 8)
		sp := p.Render()
		mustFind(t, sp, "→ 1202")
		if !p.Remove("SERVICER") {
			t.Fatal("draining one core set must work")
		}
		sp = p.Render()
		if _, _, ok := findText(sp, "→ 1202"); ok {
			t.Fatal("freeing a slot must clear the alarm chip")
		}
		cr, cc := mustFind(t, sp, "7/8")
		if got := sp.At(cr, cc).FG; got != RedInk {
			t.Fatalf("seven of eight still wears %d, want the red warning %d", got, RedInk)
		}
	})
	t.Run("unhappy: a calm panel never shows red or the alarm", func(t *testing.T) {
		p := NewCoreSetPanel()
		p.Start(stageW, stageH)
		fill(p, 6)
		sp := p.Render()
		cr, cc := mustFind(t, sp, "6/8")
		if got := sp.At(cr, cc).FG; got != DimInk {
			t.Fatalf("six of eight wears %d, want the calm DimInk %d", got, DimInk)
		}
		if _, _, ok := findText(sp, "→ 1202"); ok {
			t.Fatal("no alarm below a full panel")
		}
	})
}

func TestLifecycle(t *testing.T) {
	t.Run("happy: jobs survive a resize — Stop then Start keeps every box", func(t *testing.T) {
		p := NewCoreSetPanel()
		p.Start(stageW, stageH)
		p.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		p.Add(Job{Name: "CHARIN", Prio: 30, Ink: 213})
		p.Stop()
		p.Start(76, 24)
		if p.Busy() != 2 {
			t.Fatalf("the resize dropped jobs — busy %d, want 2", p.Busy())
		}
		sp := p.Render()
		if sp.Width != 76 || sp.Height != 24 {
			t.Fatalf("render is %dx%d, want the new 76x24 stage", sp.Width, sp.Height)
		}
		mustFind(t, sp, "SERVICER·20")
		mustFind(t, sp, "CHARIN·30")
	})
	t.Run("unhappy: before Start and after Stop the stage is empty", func(t *testing.T) {
		p := NewVACPanel()
		if sp := p.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("before Start the panel renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		p.Start(stageW, stageH)
		p.Stop()
		if sp := p.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("after Stop the panel renders %dx%d, want nothing", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a zero stage renders empty and a tiny stage clips, without panic", func(t *testing.T) {
		p := NewCoreSetPanel()
		p.Start(0, 0)
		p.Update(1)
		if sp := p.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatalf("a 0x0 stage renders %dx%d, want nothing", sp.Width, sp.Height)
		}
		p.Stop()
		p.Start(7, 4)
		p.Add(Job{Name: "SERVICER", Prio: 20, Ink: 83})
		sp := p.Render()
		if sp.Width != 7 || sp.Height != 4 {
			t.Fatalf("a tiny stage renders %dx%d, want 7x4", sp.Width, sp.Height)
		}
	})
	t.Run("unhappy: a nil panel refuses every call without panic", func(t *testing.T) {
		var p *Panel
		p.Start(10, 5)
		p.Update(1)
		if sp := p.Render(); sp.Width != 0 || sp.Height != 0 {
			t.Fatal("a nil panel must render nothing")
		}
		p.Stop()
		if _, ok := p.Add(Job{Name: "SERVICER"}); ok {
			t.Fatal("a nil panel must refuse Add")
		}
		if p.Remove("SERVICER") {
			t.Fatal("a nil panel must refuse Remove")
		}
		if p.Busy() != 0 || p.Cap() != 0 || p.Full() {
			t.Fatal("a nil panel holds nothing and is never full")
		}
		if p.Box(0) != nil {
			t.Fatal("a nil panel has no boxes")
		}
		if _, ok := p.JobAt(0); ok {
			t.Fatal("a nil panel has no jobs")
		}
		if x, y := p.Origin(0); x != 0 || y != 0 {
			t.Fatal("a nil panel has no geometry")
		}
		if w, h := p.Size(); w != 0 || h != 0 {
			t.Fatal("a nil panel has no size")
		}
	})
}
