package menu

import (
	"strings"
	"testing"
)

func TestCatalogCarriesTheInterpreterScene(t *testing.T) {
	// happy: the Interpreter walkthrough is a portable scene on the
	// Scenes shelf, right after Bobble, with its own tuner
	interp, bobble := -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "interpreter":
			interp = i
		case "bobble":
			bobble = i
		}
	}
	if interp < 0 {
		t.Fatal("catalog missing the interpreter scene")
	}
	e := c[interp]
	if e.Section != "Scenes" {
		t.Fatalf("interpreter section = %q, want Scenes", e.Section)
	}
	if e.Title != "Interpreter" {
		t.Fatalf("interpreter title = %q, want Interpreter", e.Title)
	}
	if e.Pkg != "./cmd/interpreter" {
		t.Fatalf("interpreter pkg = %q, want ./cmd/interpreter", e.Pkg)
	}
	if !strings.Contains(e.Desc, "DANZIG") {
		t.Fatalf("interpreter desc %q must name the DANZIG check — that is the lesson", e.Desc)
	}
	if !strings.Contains(e.Desc, "knobs") {
		t.Fatalf("interpreter desc %q must sell its live knobs", e.Desc)
	}
	if bobble < 0 || interp != bobble+1 {
		t.Fatalf("interpreter sits at %d, want right after bobble at %d", interp, bobble)
	}
}

func TestInterpreterShelving(t *testing.T) {
	// unhappy: the scene appears exactly once, never as a screenplay,
	// and the catalog's head and tail stay put
	c := Catalog()
	seen := 0
	for _, e := range c {
		if e.ID == "interpreter" {
			seen++
			if e.Section != "Scenes" {
				t.Fatalf("the interpreter is a scene, not a %s entry", e.Section)
			}
		}
		if e.Section == "Screenplays" && strings.Contains(e.Title, "Interpreter") {
			t.Fatalf("a Screenplays entry %q carries the scene", e.Title)
		}
	}
	if seen != 1 {
		t.Fatalf("the catalog holds %d interpreter entries, want exactly 1", seen)
	}
	if c[0].ID != "screenplay" {
		t.Fatalf("catalog head disturbed: %s", c[0].ID)
	}
	if c[len(c)-2].ID != "agctop" || c[len(c)-1].ID != "agcgraph" {
		t.Fatalf("catalog tail = %s, %s; want agctop then agcgraph", c[len(c)-2].ID, c[len(c)-1].ID)
	}
}
