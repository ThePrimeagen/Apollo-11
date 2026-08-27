package menu

import (
	"strings"
	"testing"
)

func TestCatalogCarriesTheInverseWalkthrough(t *testing.T) {
	// happy: 03. Inverse Walkthrough launches from the Scenes section,
	// right after the Skies scene, out of its own cmd folder
	liftoff, skies := -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "liftoff":
			liftoff = i
		case "skies":
			skies = i
		}
	}
	if liftoff < 0 {
		t.Fatal("catalog missing the inverse walkthrough scene")
	}
	e := c[liftoff]
	if e.Section != "Scenes" {
		t.Fatalf("liftoff section = %q, want Scenes", e.Section)
	}
	if e.Title != "03. Inverse Walkthrough" {
		t.Fatalf("liftoff title = %q, want 03. Inverse Walkthrough", e.Title)
	}
	if e.Pkg != "./cmd/liftoff" {
		t.Fatalf("liftoff pkg = %q, want ./cmd/liftoff", e.Pkg)
	}
	if !strings.Contains(e.Desc, "knobs") {
		t.Fatalf("liftoff desc %q must sell its live knobs", e.Desc)
	}
	if skies < 0 || liftoff != skies+1 {
		t.Fatalf("liftoff sits at %d, want right after skies at %d", liftoff, skies)
	}
}

func TestInverseWalkthroughIsASceneNotAScreenplay(t *testing.T) {
	// unhappy: the operator asked for a scene — it must not slip onto
	// the Screenplays shelf, must not repeat, and must not disturb the
	// catalog's head or tail
	c := Catalog()
	seen := 0
	for _, e := range c {
		if e.ID == "liftoff" {
			seen++
			if e.Section == "Screenplays" {
				t.Fatal("the inverse walkthrough is a scene, not a screenplay")
			}
		}
		if e.Section == "Screenplays" && strings.Contains(e.Title, "Inverse") {
			t.Fatalf("a Screenplays entry %q carries the inverse walkthrough", e.Title)
		}
	}
	if seen != 1 {
		t.Fatalf("the catalog holds %d liftoff entries, want exactly 1", seen)
	}
	if c[0].ID != "screenplay" {
		t.Fatalf("catalog head disturbed: %s", c[0].ID)
	}
	if c[len(c)-2].ID != "agctop" || c[len(c)-1].ID != "agcgraph" {
		t.Fatalf("catalog tail = %s, %s; want agctop then agcgraph", c[len(c)-2].ID, c[len(c)-1].ID)
	}
}
