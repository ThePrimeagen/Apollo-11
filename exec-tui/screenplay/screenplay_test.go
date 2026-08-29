package screenplay

// Tests written FIRST: a screenplay is scenes in order with a cursor
// and a lifecycle. New binds the bill but starts nothing. Start raises
// the first curtain. Every frame is Update then Render on the current
// scene only — Render clears the screen first and consumes its resized
// flag. Next cuts: the old scene stops before the new one starts. Stop
// ends the run. The final scene holds; sad paths never panic.

import (
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

// door records every lifecycle call in a shared journal.
type door struct {
	name       string
	journal    *[]string
	updated    []float64
	rendered   int
	sawResized []bool
}

func (d *door) Start() { *d.journal = append(*d.journal, "start:"+d.name) }
func (d *door) Stop()  { *d.journal = append(*d.journal, "stop:"+d.name) }

func (d *door) Update(dt float64) { d.updated = append(d.updated, dt) }

func (d *door) Render(scr *Screen) {
	d.rendered++
	d.sawResized = append(d.sawResized, scr.Resized())
}

func twoSceneBill() (*Screenplay, *door, *door, *[]string) {
	journal := &[]string{}
	a := &door{name: "a", journal: journal}
	b := &door{name: "b", journal: journal}
	p := New(
		Entry{Name: "arrival", Scene: a},
		Entry{Name: "the end", Scene: b},
	)
	return p, a, b, journal
}

func equalLog(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestScreenplayLifecycle(t *testing.T) {
	t.Run("happy: New binds the bill but raises no curtain", func(t *testing.T) {
		p, _, _, journal := twoSceneBill()
		if len(*journal) != 0 {
			t.Fatalf("New must start nothing, journal %v", *journal)
		}
		if p.Len() != 2 || p.SceneIndex() != 0 || p.CurrentName() != "arrival" {
			t.Fatalf("bill reads %d/%d %q", p.SceneIndex(), p.Len(), p.CurrentName())
		}
	})
	t.Run("happy: Start raises the first curtain exactly once", func(t *testing.T) {
		p, _, _, journal := twoSceneBill()
		p.Start()
		p.Start()
		if !equalLog(*journal, "start:a") {
			t.Fatalf("journal %v, want one start:a", *journal)
		}
	})
	t.Run("happy: frames reach only the scene now playing", func(t *testing.T) {
		p, a, b, _ := twoSceneBill()
		p.Start()
		p.Update(0.3)
		p.Render(NewScreen(4, 2))
		if len(a.updated) != 1 || a.updated[0] != 0.3 || a.rendered != 1 {
			t.Fatalf("scene a saw updates %v renders %d", a.updated, a.rendered)
		}
		if len(b.updated) != 0 || b.rendered != 0 {
			t.Fatal("scene b played before its cut")
		}
	})
	t.Run("happy: a cut stops the old scene before the new one starts", func(t *testing.T) {
		p, a, b, journal := twoSceneBill()
		p.Start()
		if !p.Next() {
			t.Fatal("Next off scene one must report a cut")
		}
		if !equalLog(*journal, "start:a", "stop:a", "start:b") {
			t.Fatalf("journal %v, want stop:a before start:b", *journal)
		}
		if p.SceneIndex() != 1 || p.CurrentName() != "the end" {
			t.Fatalf("after the cut: %d %q", p.SceneIndex(), p.CurrentName())
		}
		p.Update(0.2)
		if len(a.updated) != 0 || len(b.updated) != 1 {
			t.Fatalf("time went to the wrong scene: a=%v b=%v", a.updated, b.updated)
		}
	})
	t.Run("happy: Render clears the screen and consumes the resized flag", func(t *testing.T) {
		p, a, _, _ := twoSceneBill()
		p.Start()
		scr := NewScreen(4, 2)
		scr.Put(0, 0, 'X', uv.Style{})
		scr.Resize(5, 3)
		p.Render(scr)
		if c := scr.Cell(0, 0); c != nil && c.Content == "X" {
			t.Fatal("Render must clear the previous frame")
		}
		if len(a.sawResized) != 1 || !a.sawResized[0] {
			t.Fatalf("the scene must see the resize on the next render, saw %v", a.sawResized)
		}
		if scr.Resized() {
			t.Fatal("Render must consume the resized flag")
		}
		p.Render(scr)
		if a.sawResized[1] {
			t.Fatal("the frame after next must not still read as resized")
		}
	})
	t.Run("unhappy: before Start the play is inert", func(t *testing.T) {
		p, a, _, journal := twoSceneBill()
		p.Update(0.5)
		p.Render(NewScreen(4, 2))
		if p.Next() {
			t.Fatal("Next before Start must not cut")
		}
		p.Stop()
		if len(*journal) != 0 || len(a.updated) != 0 || a.rendered != 0 {
			t.Fatalf("inert play still moved: journal %v", *journal)
		}
	})
	t.Run("unhappy: the final scene holds — no cut, no lifecycle calls", func(t *testing.T) {
		p, _, _, journal := twoSceneBill()
		p.Start()
		p.Next()
		before := len(*journal)
		if p.Next() {
			t.Fatal("Next on the final scene must report no cut")
		}
		if len(*journal) != before {
			t.Fatalf("a held cut must not stop or start anything: %v", *journal)
		}
		if p.SceneIndex() != 1 || p.CurrentName() != "the end" {
			t.Fatal("the final scene drifted")
		}
	})
	t.Run("unhappy: Stop ends the run once; later calls are no-ops", func(t *testing.T) {
		p, a, _, journal := twoSceneBill()
		p.Start()
		p.Stop()
		p.Stop()
		if !equalLog(*journal, "start:a", "stop:a") {
			t.Fatalf("journal %v, want one stop", *journal)
		}
		p.Update(0.5)
		p.Render(NewScreen(4, 2))
		if p.Next() {
			t.Fatal("a stopped play must not cut")
		}
		if len(a.updated) != 0 || a.rendered != 0 {
			t.Fatal("a stopped play still played")
		}
	})
	t.Run("unhappy: dt<=0 holds the whole play", func(t *testing.T) {
		p, a, _, _ := twoSceneBill()
		p.Start()
		p.Update(0)
		p.Update(-2)
		if len(a.updated) != 0 {
			t.Fatalf("dt<=0 reached the scene: %v", a.updated)
		}
	})
	t.Run("unhappy: a nil screen renders nothing and calls no one", func(t *testing.T) {
		p, a, _, _ := twoSceneBill()
		p.Start()
		p.Render(nil)
		if a.rendered != 0 {
			t.Fatal("a nil screen must not reach the scene")
		}
	})
	t.Run("unhappy: an empty bill is inert everywhere", func(t *testing.T) {
		p := New()
		p.Start()
		p.Update(0.5)
		p.Render(NewScreen(2, 2))
		p.Stop()
		if p.Len() != 0 || p.SceneIndex() != 0 || p.CurrentName() != "" || p.Next() {
			t.Fatal("an empty bill must be inert")
		}
	})
	t.Run("unhappy: a nil screenplay ignores every call", func(t *testing.T) {
		var p *Screenplay
		p.Start()
		p.Update(0.5)
		p.Render(NewScreen(2, 2))
		p.Stop()
		if p.Len() != 0 || p.SceneIndex() != 0 || p.CurrentName() != "" || p.Next() {
			t.Fatal("a nil screenplay must be inert")
		}
	})
	t.Run("unhappy: a nil scene on the bill is tolerated", func(t *testing.T) {
		journal := &[]string{}
		a := &door{name: "a", journal: journal}
		b := &door{name: "b", journal: journal}
		p := New(
			Entry{Name: "a", Scene: a},
			Entry{Name: "ghost", Scene: nil},
			Entry{Name: "b", Scene: b},
		)
		p.Start()
		p.Update(0.1)
		if !p.Next() {
			t.Fatal("cutting onto a ghost scene still moves the cursor")
		}
		p.Update(0.2)
		p.Render(NewScreen(2, 2))
		if !p.Next() {
			t.Fatal("cutting off a ghost scene still moves the cursor")
		}
		if !equalLog(*journal, "start:a", "stop:a", "start:b") {
			t.Fatalf("journal %v — the ghost must be silent, the rest intact", *journal)
		}
		p.Update(0.3)
		if len(b.updated) != 1 || b.updated[0] != 0.3 {
			t.Fatalf("scene b saw %v after the ghost", b.updated)
		}
	})
}

// Tests written FIRST: Prev is the cut played backwards — the scene
// now playing stops, the one before it starts — so an editor can walk
// the bill both ways. Rewind is the premiere cut: whatever is playing
// stops and the first scene starts again, even when the first scene
// is the one on stage, so a show always opens fresh from the top.
// Both hold (and report false) before Start, after Stop, on an empty
// bill, and on a nil screenplay; Prev also holds on the first scene.

func TestScreenplayPrev(t *testing.T) {
	t.Run("happy: Prev cuts back — the current stops, the earlier one starts", func(t *testing.T) {
		p, a, b, journal := twoSceneBill()
		p.Start()
		p.Next()
		if !p.Prev() {
			t.Fatal("Prev off scene two must report a cut")
		}
		if !equalLog(*journal, "start:a", "stop:a", "start:b", "stop:b", "start:a") {
			t.Fatalf("journal %v, want stop:b before the second start:a", *journal)
		}
		if p.SceneIndex() != 0 || p.CurrentName() != "arrival" {
			t.Fatalf("after the cut back: %d %q", p.SceneIndex(), p.CurrentName())
		}
		p.Update(0.2)
		if len(a.updated) != 1 || len(b.updated) != 0 {
			t.Fatalf("time went to the wrong scene: a=%v b=%v", a.updated, b.updated)
		}
	})
	t.Run("unhappy: the first scene holds — no cut, no lifecycle calls", func(t *testing.T) {
		p, _, _, journal := twoSceneBill()
		p.Start()
		before := len(*journal)
		if p.Prev() {
			t.Fatal("Prev on the first scene must report no cut")
		}
		if len(*journal) != before {
			t.Fatalf("a held cut must not stop or start anything: %v", *journal)
		}
		if p.SceneIndex() != 0 || p.CurrentName() != "arrival" {
			t.Fatal("the first scene drifted")
		}
	})
	t.Run("unhappy: before Start and after Stop Prev is inert", func(t *testing.T) {
		p, _, _, journal := twoSceneBill()
		if p.Prev() {
			t.Fatal("Prev before Start must not cut")
		}
		p.Start()
		p.Next()
		p.Stop()
		before := len(*journal)
		if p.Prev() {
			t.Fatal("a stopped play must not cut back")
		}
		if len(*journal) != before {
			t.Fatalf("a stopped play still moved: %v", *journal)
		}
	})
	t.Run("unhappy: empty and nil screenplays hold", func(t *testing.T) {
		p := New()
		p.Start()
		if p.Prev() {
			t.Fatal("an empty bill must not cut back")
		}
		var nilPlay *Screenplay
		if nilPlay.Prev() {
			t.Fatal("a nil screenplay must not cut back")
		}
	})
	t.Run("unhappy: a ghost scene on the way back stays silent", func(t *testing.T) {
		journal := &[]string{}
		a := &door{name: "a", journal: journal}
		b := &door{name: "b", journal: journal}
		p := New(
			Entry{Name: "a", Scene: a},
			Entry{Name: "ghost", Scene: nil},
			Entry{Name: "b", Scene: b},
		)
		p.Start()
		p.Next()
		p.Next()
		if !p.Prev() {
			t.Fatal("cutting back onto a ghost scene still moves the cursor")
		}
		if !p.Prev() {
			t.Fatal("cutting back off a ghost scene still moves the cursor")
		}
		if !equalLog(*journal, "start:a", "stop:a", "start:b", "stop:b", "start:a") {
			t.Fatalf("journal %v — the ghost must be silent, the rest intact", *journal)
		}
		if p.SceneIndex() != 0 {
			t.Fatalf("two cuts back must land on the first scene, got %d", p.SceneIndex())
		}
	})
}

func TestScreenplayRewind(t *testing.T) {
	threeSceneBill := func() (*Screenplay, *[]string) {
		journal := &[]string{}
		p := New(
			Entry{Name: "one", Scene: &door{name: "a", journal: journal}},
			Entry{Name: "two", Scene: &door{name: "b", journal: journal}},
			Entry{Name: "three", Scene: &door{name: "c", journal: journal}},
		)
		return p, journal
	}
	t.Run("happy: Rewind cuts from anywhere to the top of the bill", func(t *testing.T) {
		p, journal := threeSceneBill()
		p.Start()
		p.Next()
		p.Next()
		if !p.Rewind() {
			t.Fatal("Rewind off scene three must report a cut")
		}
		if !equalLog(*journal, "start:a", "stop:a", "start:b", "stop:b", "start:c", "stop:c", "start:a") {
			t.Fatalf("journal %v, want stop:c then a fresh start:a", *journal)
		}
		if p.SceneIndex() != 0 || p.CurrentName() != "one" {
			t.Fatalf("after the rewind: %d %q", p.SceneIndex(), p.CurrentName())
		}
	})
	t.Run("happy: Rewind on the first scene restarts it from the top", func(t *testing.T) {
		p, journal := threeSceneBill()
		p.Start()
		if !p.Rewind() {
			t.Fatal("Rewind on the first scene must restart it")
		}
		if !equalLog(*journal, "start:a", "stop:a", "start:a") {
			t.Fatalf("journal %v, want the first scene stopped and started again", *journal)
		}
		if p.SceneIndex() != 0 {
			t.Fatal("the rewound show must sit on the first scene")
		}
	})
	t.Run("unhappy: before Start, after Stop, empty, and nil hold", func(t *testing.T) {
		p, journal := threeSceneBill()
		if p.Rewind() {
			t.Fatal("Rewind before Start must not cut")
		}
		p.Start()
		p.Stop()
		before := len(*journal)
		if p.Rewind() {
			t.Fatal("a stopped play must not rewind")
		}
		if len(*journal) != before {
			t.Fatalf("a stopped play still moved: %v", *journal)
		}
		empty := New()
		empty.Start()
		if empty.Rewind() {
			t.Fatal("an empty bill must not rewind")
		}
		var nilPlay *Screenplay
		if nilPlay.Rewind() {
			t.Fatal("a nil screenplay must not rewind")
		}
	})
}
