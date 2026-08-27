package menu

import "testing"

func TestCatalogCarriesTheCommandScreen(t *testing.T) {
	// happy: the command screen is launchable from the menu under its own
	// EXECUTIVE section, as a cmd/ program
	var found *Entry
	for i := range Catalog() {
		if Catalog()[i].ID == "agctop" {
			e := Catalog()[i]
			found = &e
		}
	}
	if found == nil {
		t.Fatalf("catalog missing the agctop command screen")
	}
	if found.Section != "EXECUTIVE" {
		t.Fatalf("agctop section = %q, want EXECUTIVE", found.Section)
	}
	if found.Pkg != "./cmd/agctop" {
		t.Fatalf("agctop pkg = %q, want ./cmd/agctop", found.Pkg)
	}
}

func TestCommandScreenAppendsWithoutReordering(t *testing.T) {
	// unhappy: the new entry must not disturb the existing selection order
	// (the screenplay entries stay at the top of the list)
	c := Catalog()
	if c[0].ID != "screenplay" || c[1].ID != "moon" || c[2].ID != "closeup" {
		t.Fatalf("existing catalog head disturbed: %s, %s, %s", c[0].ID, c[1].ID, c[2].ID)
	}
	if c[len(c)-1].ID != "agctop" {
		t.Fatalf("agctop must sit at the end of the catalog, got %s", c[len(c)-1].ID)
	}
}
