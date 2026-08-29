package menu

import (
	"strings"
	"testing"
)

func TestCatalogCarriesTheMainScreenplay(t *testing.T) {
	// happy: 05. Main is the screenplay that puts everything together
	// — it launches from the Screenplays shelf, right after 04.
	// Inverse Walkthrough, and it opens the editor
	main, inverse := -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "main":
			main = i
		case "inverse":
			inverse = i
		}
	}
	if main < 0 {
		t.Fatal("catalog missing the main screenplay")
	}
	e := c[main]
	if e.Section != "Screenplays" {
		t.Fatalf("main section = %q, want Screenplays", e.Section)
	}
	if e.Title != "05. Main" {
		t.Fatalf("main title = %q, want 05. Main", e.Title)
	}
	if e.Pkg != "./cmd/mainshow" {
		t.Fatalf("main pkg = %q, want ./cmd/mainshow", e.Pkg)
	}
	if !strings.Contains(strings.ToLower(e.Desc), "editor") {
		t.Fatalf("main desc %q must sell the screenplay editor", e.Desc)
	}
	if !strings.Contains(strings.ToLower(e.Desc), "e edit") {
		t.Fatalf("main desc %q must sell e editing each scene into MAIN's own config", e.Desc)
	}
	if inverse < 0 || main != inverse+1 {
		t.Fatalf("main sits at %d, want right after 04. Inverse Walkthrough at %d", main, inverse)
	}
}

func TestMainScreenplayShelving(t *testing.T) {
	// unhappy: the editor is a screenplay, not a scene; it appears
	// exactly once; the old premiere MAIN stays gone; and the
	// catalog's head and tail stay put
	c := Catalog()
	seen := 0
	for _, e := range c {
		if e.ID == "main" {
			seen++
			if e.Section != "Screenplays" {
				t.Fatal("main is a screenplay, not a scene")
			}
		}
		if e.ID == "screenplay" || e.Title == "MAIN" {
			t.Fatalf("the old premiere MAIN stays gone, found %+v", e)
		}
	}
	if seen != 1 {
		t.Fatalf("the catalog holds %d main entries, want exactly 1", seen)
	}
	if c[0].ID != "moon" {
		t.Fatalf("catalog head disturbed: %s", c[0].ID)
	}
	if c[len(c)-2].ID != "agctop" || c[len(c)-1].ID != "agcgraph" {
		t.Fatalf("catalog tail = %s, %s; want agctop then agcgraph", c[len(c)-2].ID, c[len(c)-1].ID)
	}
}
