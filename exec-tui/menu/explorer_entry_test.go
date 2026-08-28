package menu

import (
	"strings"
	"testing"
)

func TestCatalogCarriesTheBigEScene(t *testing.T) {
	// happy: Big E is a portable scene on the Scenes shelf, right
	// after Shooting Star, with its own tuner
	explorer, shoot := -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "explorer":
			explorer = i
		case "shootingstar":
			shoot = i
		}
	}
	if explorer < 0 {
		t.Fatal("catalog missing the Big E scene")
	}
	e := c[explorer]
	if e.Section != "Scenes" || e.Title != "Big E" || e.Pkg != "./cmd/explorer" {
		t.Fatalf("explorer entry = %+v, want the Big E scene out of ./cmd/explorer", e)
	}
	if !strings.Contains(strings.ToLower(e.Desc), "shooting star") {
		t.Fatalf("Big E desc %q must mention the shooting star", e.Desc)
	}
	if !strings.Contains(e.Desc, "knobs") {
		t.Fatalf("Big E desc %q must sell its live knobs", e.Desc)
	}
	if shoot < 0 || explorer != shoot+1 {
		t.Fatalf("Big E sits at %d, want right after shootingstar at %d", explorer, shoot)
	}
}

func TestBigEShelving(t *testing.T) {
	// unhappy: the scene appears exactly once, never as a screenplay,
	// and it is titled Big E — not Explorer, not a component entry
	c := Catalog()
	seen := 0
	for _, e := range c {
		if e.ID == "explorer" {
			seen++
			if e.Section != "Scenes" {
				t.Fatalf("the Big E is a scene, not a %s entry", e.Section)
			}
			if e.Title != "Big E" {
				t.Fatalf("the scene is titled %q, want Big E", e.Title)
			}
		}
		if e.Section == "Screenplays" && strings.Contains(e.Title, "Big E") {
			t.Fatalf("a Screenplays entry %q carries the scene", e.Title)
		}
	}
	if seen != 1 {
		t.Fatalf("the catalog holds %d explorer entries, want exactly 1", seen)
	}
	if c[0].ID != "screenplay" {
		t.Fatalf("catalog head disturbed: %s", c[0].ID)
	}
	if c[len(c)-2].ID != "agctop" || c[len(c)-1].ID != "agcgraph" {
		t.Fatalf("catalog tail = %s, %s; want agctop then agcgraph", c[len(c)-2].ID, c[len(c)-1].ID)
	}
}
