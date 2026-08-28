package menu

import (
	"strings"
	"testing"
)

func TestCatalogCarriesTheShootingStarSceneAndTrail(t *testing.T) {
	// happy: Shooting Star is a portable scene on the Scenes shelf,
	// right after Interpreter, and STAR TRAIL CONFIG is the persist-trail
	// tuner on the Particles shelf
	shoot, interp, trail := -1, -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "shootingstar":
			shoot = i
		case "interpreter":
			interp = i
		case "startrail-config":
			trail = i
		}
	}
	if shoot < 0 {
		t.Fatal("catalog missing the shooting star scene")
	}
	e := c[shoot]
	if e.Section != "Scenes" || e.Title != "Shooting Star" || e.Pkg != "./cmd/shootingstar" {
		t.Fatalf("shootingstar entry = %+v, want the Shooting Star scene out of ./cmd/shootingstar", e)
	}
	if !strings.Contains(e.Desc, "knobs") {
		t.Fatalf("shootingstar desc %q must sell its live knobs", e.Desc)
	}
	if interp < 0 || shoot != interp+1 {
		t.Fatalf("shootingstar sits at %d, want right after interpreter at %d", shoot, interp)
	}
	if trail < 0 {
		t.Fatal("catalog missing the star trail tuner")
	}
	tr := c[trail]
	if tr.Section != "Particles" || tr.Title != "STAR TRAIL CONFIG" || tr.Pkg != "./cmd/shootingstar" {
		t.Fatalf("startrail-config entry = %+v, want STAR TRAIL CONFIG out of ./cmd/shootingstar", tr)
	}
}

func TestShootingStarShelving(t *testing.T) {
	// unhappy: the scene is not a particle, the trail tuner is not a
	// scene, nothing repeats
	c := Catalog()
	seen := map[string]int{}
	for _, e := range c {
		switch e.ID {
		case "shootingstar", "startrail-config":
			seen[e.ID]++
		}
		if e.ID == "shootingstar" && e.Section != "Scenes" {
			t.Fatal("the shooting star is a scene, not a particle")
		}
		if e.ID == "startrail-config" && e.Section != "Particles" {
			t.Fatal("the star trail tuner is a particle config, not a scene")
		}
	}
	for _, id := range []string{"shootingstar", "startrail-config"} {
		if seen[id] != 1 {
			t.Fatalf("the catalog holds %d %s entries, want exactly 1", seen[id], id)
		}
	}
}
