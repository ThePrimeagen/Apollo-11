package menu

import "testing"

func TestCatalogCarriesTheGraphsScreen(t *testing.T) {
	// happy: the graphs screen launches from the EXECUTIVE section
	var found *Entry
	for i := range Catalog() {
		if Catalog()[i].ID == "agcgraph" {
			e := Catalog()[i]
			found = &e
		}
	}
	if found == nil {
		t.Fatalf("catalog missing the agcgraph graphs screen")
	}
	if found.Section != "EXECUTIVE" {
		t.Fatalf("agcgraph section = %q, want EXECUTIVE", found.Section)
	}
	if found.Pkg != "./cmd/agcgraph" {
		t.Fatalf("agcgraph pkg = %q, want ./cmd/agcgraph", found.Pkg)
	}
}

func TestGraphsScreenKeepsExecutiveContiguous(t *testing.T) {
	// unhappy: the EXECUTIVE section must stay contiguous (agctop then
	// agcgraph at the catalog tail) and the head order untouched
	c := Catalog()
	if c[0].ID != "screenplay" {
		t.Fatalf("catalog head disturbed: %s", c[0].ID)
	}
	if c[len(c)-2].ID != "agctop" || c[len(c)-1].ID != "agcgraph" {
		t.Fatalf("catalog tail = %s, %s; want agctop then agcgraph", c[len(c)-2].ID, c[len(c)-1].ID)
	}
}
