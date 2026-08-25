package screenplay

// Tests written FIRST: a screenplay is scenes in order with a cursor.
// It opens on the first scene, Next cuts forward and reports whether it
// moved, and time only ever reaches the scene now playing. The final
// scene holds — no wrap, no walk off the end.

import "testing"

func twoScenes() (*Screenplay, *probe, *probe) {
	one, two := &probe{glyph: '1'}, &probe{glyph: '2'}
	p := New(
		&Scene{Name: "arrival", Cast: []Actor{one}},
		&Scene{Name: "the end", Cast: []Actor{two}},
	)
	return p, one, two
}

func TestScreenplayCursor(t *testing.T) {
	t.Run("happy: opens on scene one and Next cuts to scene two", func(t *testing.T) {
		p, _, _ := twoScenes()
		if p.Len() != 2 {
			t.Fatalf("len %d, want 2", p.Len())
		}
		if p.SceneIndex() != 0 || p.Current().Name != "arrival" {
			t.Fatalf("opened on %d %q, want 0 arrival", p.SceneIndex(), p.Current().Name)
		}
		if !p.Next() {
			t.Fatal("Next off scene one must report a cut")
		}
		if p.SceneIndex() != 1 || p.Current().Name != "the end" {
			t.Fatalf("after Next: %d %q, want 1 the end", p.SceneIndex(), p.Current().Name)
		}
	})
	t.Run("unhappy: the final scene holds — Next is false and stays", func(t *testing.T) {
		p, _, _ := twoScenes()
		p.Next()
		if p.Next() {
			t.Fatal("Next on the final scene must report no cut")
		}
		if p.SceneIndex() != 1 || p.Current().Name != "the end" {
			t.Fatalf("final scene drifted to %d %q", p.SceneIndex(), p.Current().Name)
		}
	})
	t.Run("unhappy: an empty screenplay is inert", func(t *testing.T) {
		p := New()
		if p.Len() != 0 || p.SceneIndex() != 0 {
			t.Fatalf("empty play len=%d idx=%d", p.Len(), p.SceneIndex())
		}
		if p.Current() != nil {
			t.Fatal("empty play must have no current scene")
		}
		if p.Next() {
			t.Fatal("empty play must not cut")
		}
		p.Advance(0.5) // must not panic
	})
	t.Run("unhappy: a nil screenplay ignores every call", func(t *testing.T) {
		var p *Screenplay
		if p.Len() != 0 || p.SceneIndex() != 0 || p.Current() != nil || p.Next() {
			t.Fatal("nil screenplay must be inert")
		}
		p.Advance(0.5)
	})
}

func TestScreenplayAdvance(t *testing.T) {
	t.Run("happy: time reaches only the scene now playing", func(t *testing.T) {
		p, one, two := twoScenes()
		p.Advance(0.3)
		if len(one.advanced) != 1 || one.advanced[0] != 0.3 {
			t.Fatalf("scene one saw %v, want [0.3]", one.advanced)
		}
		if len(two.advanced) != 0 {
			t.Fatalf("scene two ticked before its cut: %v", two.advanced)
		}
		p.Next()
		p.Advance(0.2)
		if len(one.advanced) != 1 {
			t.Fatalf("scene one kept ticking after the cut: %v", one.advanced)
		}
		if len(two.advanced) != 1 || two.advanced[0] != 0.2 {
			t.Fatalf("scene two saw %v, want [0.2]", two.advanced)
		}
	})
	t.Run("unhappy: dt<=0 holds the whole play", func(t *testing.T) {
		p, one, _ := twoScenes()
		p.Advance(0)
		p.Advance(-2)
		if len(one.advanced) != 0 {
			t.Fatalf("dt<=0 reached the cast: %v", one.advanced)
		}
	})
}
