package menu

import (
	"strings"
	"testing"
)

func TestCatalogCarriesTheInverseWalkthroughScreenplay(t *testing.T) {
	// happy: 03. Inverse Walkthrough is a screenplay — it launches
	// from the Screenplays shelf, right after 02. Walkthrough
	inverse, closeup := -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "inverse":
			inverse = i
		case "closeup":
			closeup = i
		}
	}
	if inverse < 0 {
		t.Fatal("catalog missing the inverse walkthrough screenplay")
	}
	e := c[inverse]
	if e.Section != "Screenplays" {
		t.Fatalf("inverse section = %q, want Screenplays", e.Section)
	}
	if e.Title != "03. Inverse Walkthrough" {
		t.Fatalf("inverse title = %q, want 03. Inverse Walkthrough", e.Title)
	}
	if e.Pkg != "./cmd/inverse" {
		t.Fatalf("inverse pkg = %q, want ./cmd/inverse", e.Pkg)
	}
	if closeup < 0 || inverse != closeup+1 {
		t.Fatalf("inverse sits at %d, want right after 02. Walkthrough at %d", inverse, closeup)
	}
}

func TestCatalogCarriesTheLiftoffAndBobbleScenes(t *testing.T) {
	// happy: Liftoff and Bobble are portable scenes on the Scenes
	// shelf, right after the Core Set, each with its own tuner
	liftoff, bobble, coreset := -1, -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "liftoff":
			liftoff = i
		case "bobble":
			bobble = i
		case "coreset":
			coreset = i
		}
	}
	if liftoff < 0 || bobble < 0 {
		t.Fatal("catalog missing the liftoff or bobble scene")
	}
	lo := c[liftoff]
	if lo.Section != "Scenes" || lo.Title != "Liftoff" || lo.Pkg != "./cmd/liftoff" {
		t.Fatalf("liftoff entry = %+v, want the Liftoff scene out of ./cmd/liftoff", lo)
	}
	if !strings.Contains(lo.Desc, "knobs") {
		t.Fatalf("liftoff desc %q must sell its live knobs", lo.Desc)
	}
	bo := c[bobble]
	if bo.Section != "Scenes" || bo.Title != "Bobble" || bo.Pkg != "./cmd/bobble" {
		t.Fatalf("bobble entry = %+v, want the Bobble scene out of ./cmd/bobble", bo)
	}
	if !strings.Contains(bo.Desc, "knobs") {
		t.Fatalf("bobble desc %q must sell its live knobs", bo.Desc)
	}
	if coreset < 0 || liftoff != coreset+1 || bobble != liftoff+1 {
		t.Fatalf("scenes sit at coreset=%d liftoff=%d bobble=%d, want the core set then liftoff then bobble", coreset, liftoff, bobble)
	}
}

func TestInverseWalkthroughShelving(t *testing.T) {
	// unhappy: the screenplay is not a scene, the scenes are not
	// screenplays, nothing repeats, and the catalog's head and tail
	// stay put
	c := Catalog()
	seen := map[string]int{}
	for _, e := range c {
		switch e.ID {
		case "inverse", "liftoff", "bobble":
			seen[e.ID]++
		}
		if e.ID == "inverse" && e.Section != "Screenplays" {
			t.Fatal("the inverse walkthrough is a screenplay, not a scene")
		}
		if (e.ID == "liftoff" || e.ID == "bobble") && e.Section != "Scenes" {
			t.Fatalf("%s is a scene, not a %s entry", e.ID, e.Section)
		}
		if e.Section == "Screenplays" && (strings.Contains(e.Title, "Liftoff") || strings.Contains(e.Title, "Bobble")) {
			t.Fatalf("a Screenplays entry %q carries a scene", e.Title)
		}
	}
	for _, id := range []string{"inverse", "liftoff", "bobble"} {
		if seen[id] != 1 {
			t.Fatalf("the catalog holds %d %s entries, want exactly 1", seen[id], id)
		}
	}
	if c[0].ID != "screenplay" {
		t.Fatalf("catalog head disturbed: %s", c[0].ID)
	}
	if c[len(c)-2].ID != "agctop" || c[len(c)-1].ID != "agcgraph" {
		t.Fatalf("catalog tail = %s, %s; want agctop then agcgraph", c[len(c)-2].ID, c[len(c)-1].ID)
	}
}
