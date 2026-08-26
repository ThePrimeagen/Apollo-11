package screenplay

// Tests written FIRST: a Bill is one screenplay's worth of scenes —
// the composable unit. Compose flattens bills, in order, into one big
// screenplay, so the final product is every show's bill added
// together. An empty composition is an empty show that starts nothing
// and never advances.

import "testing"

func TestCompose(t *testing.T) {
	t.Run("happy: bills add together, in order, into one big screenplay", func(t *testing.T) {
		journal := &[]string{}
		a := &door{name: "a", journal: journal}
		b := &door{name: "b", journal: journal}
		c := &door{name: "c", journal: journal}
		moonBill := Bill{
			Entry{Name: "the moon", Scene: a},
			Entry{Name: "orbit", Scene: b},
		}
		endBill := Bill{
			Entry{Name: "the end", Scene: c},
		}
		p := Compose(moonBill, endBill)
		if p.Len() != 3 {
			t.Fatalf("composed bill holds %d scenes, want 3", p.Len())
		}
		for i, want := range []string{"the moon", "orbit", "the end"} {
			if p.SceneIndex() != i || p.CurrentName() != want {
				t.Fatalf("scene %d is %q, want %q", p.SceneIndex(), p.CurrentName(), want)
			}
			if i == 0 {
				p.Start()
			} else if !p.Next() {
				t.Fatalf("the cut into %q must move", want)
			}
		}
		if !equalLog(*journal, "start:a", "stop:a", "start:b", "stop:b", "start:c") {
			t.Fatalf("the seam between bills must cut like any other: %v", *journal)
		}
		if p.Next() {
			t.Fatal("after the last bill there is nothing left")
		}
	})
	t.Run("happy: one bill composes to exactly its own show", func(t *testing.T) {
		journal := &[]string{}
		a := &door{name: "solo", journal: journal}
		p := Compose(Bill{Entry{Name: "solo", Scene: a}})
		if p.Len() != 1 || p.CurrentName() != "solo" {
			t.Fatalf("solo bill reads %d %q", p.Len(), p.CurrentName())
		}
	})
	t.Run("unhappy: composing nothing is an empty show that never moves", func(t *testing.T) {
		for _, p := range []*Screenplay{Compose(), Compose(Bill{}, nil, Bill{})} {
			if p.Len() != 0 || p.CurrentName() != "" {
				t.Fatalf("empty composition reads %d %q", p.Len(), p.CurrentName())
			}
			p.Start()
			if p.Next() {
				t.Fatal("an empty show has nowhere to go")
			}
			p.Update(1)
			p.Render(NewScreen(4, 2))
			p.Stop()
		}
	})
}
