package menu

import (
	"strings"
	"testing"
)

func TestCatalogCarriesTheMarioScreenplay(t *testing.T) {
	// happy: 03. Mario is a screenplay — it launches from the
	// Screenplays shelf, right after 02. Walkthrough
	mario, closeup := -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "mario":
			mario = i
		case "closeup":
			closeup = i
		}
	}
	if mario < 0 {
		t.Fatal("catalog missing the mario screenplay")
	}
	e := c[mario]
	if e.Section != "Screenplays" {
		t.Fatalf("mario section = %q, want Screenplays", e.Section)
	}
	if e.Title != "03. Mario" {
		t.Fatalf("mario title = %q, want 03. Mario", e.Title)
	}
	if e.Pkg != "./cmd/mario" {
		t.Fatalf("mario pkg = %q, want ./cmd/mario", e.Pkg)
	}
	if !strings.Contains(strings.ToLower(e.Desc), "flagpole") {
		t.Fatalf("mario desc %q must sell the flagpole run", e.Desc)
	}
	if closeup < 0 || mario != closeup+1 {
		t.Fatalf("mario sits at %d, want right after 02. Walkthrough at %d", mario, closeup)
	}
}

func TestMarioScreenplayShelving(t *testing.T) {
	// unhappy: the screenplay is not a scene, the moonwalk tuner is
	// not a screenplay, nothing repeats, and MAIN is gone
	c := Catalog()
	seen := map[string]int{}
	for _, e := range c {
		switch e.ID {
		case "mario", "moonwalk", "screenplay":
			seen[e.ID]++
		}
		if e.ID == "mario" && e.Section != "Screenplays" {
			t.Fatal("mario is a screenplay, not a scene")
		}
		if e.ID == "moonwalk" && e.Section != "Scenes" {
			t.Fatalf("the moonwalk tuner is a scene, not a %s entry", e.Section)
		}
		if e.Section == "Screenplays" && e.Title == "Moonwalk" {
			t.Fatalf("a Screenplays entry %q carries the moonwalk tuner", e.Title)
		}
		if e.ID == "screenplay" || e.Title == "MAIN" {
			t.Fatalf("MAIN is gone, found %+v", e)
		}
	}
	if seen["mario"] != 1 {
		t.Fatalf("the catalog holds %d mario entries, want exactly 1", seen["mario"])
	}
	if seen["moonwalk"] != 1 {
		t.Fatalf("the moonwalk tuner must stay listed exactly once, saw %d", seen["moonwalk"])
	}
	if seen["screenplay"] != 0 {
		t.Fatal("MAIN must not be in the catalog")
	}
	if c[0].ID != "moon" {
		t.Fatalf("catalog head disturbed: %s", c[0].ID)
	}
	if c[len(c)-2].ID != "agctop" || c[len(c)-1].ID != "agcgraph" {
		t.Fatalf("catalog tail = %s, %s; want agctop then agcgraph", c[len(c)-2].ID, c[len(c)-1].ID)
	}
}
