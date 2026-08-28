package menu

import (
	"strings"
	"testing"
)

func TestCatalogCarriesTheSpacelanderScenes(t *testing.T) {
	// happy: Fall, Climb, and Prog sit on the Scenes shelf after Big E
	// (itself after the shooting star), each with its own tuner — the
	// talk's three spacelander beats (top to bottom, bottom to top,
	// then 1202 / 1202 / 1201).
	fall, climb, prog, explorer := -1, -1, -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "fall":
			fall = i
		case "climb":
			climb = i
		case "prog":
			prog = i
		case "explorer":
			explorer = i
		}
	}
	if fall < 0 || climb < 0 || prog < 0 {
		t.Fatal("catalog missing a spacelander scene (fall / climb / prog)")
	}
	if explorer < 0 || fall != explorer+1 || climb != fall+1 || prog != climb+1 {
		t.Fatalf("fall/climb/prog sit at %d/%d/%d, want right after Big E at %d", fall, climb, prog, explorer)
	}
	if e := c[fall]; e.Title != "Spacelander Fall" || e.Pkg != "./cmd/fall" || e.Section != "Scenes" {
		t.Fatalf("fall entry = %+v, want Spacelander Fall out of ./cmd/fall", e)
	}
	if !strings.Contains(c[fall].Desc, "twinkle") {
		t.Fatalf("fall desc %q must name the twinkling sky", c[fall].Desc)
	}
	if e := c[climb]; e.Title != "Spacelander Climb" || e.Pkg != "./cmd/climb" || e.Section != "Scenes" {
		t.Fatalf("climb entry = %+v, want Spacelander Climb out of ./cmd/climb", e)
	}
	if !strings.Contains(c[climb].Desc, "bottom") {
		t.Fatalf("climb desc %q must say it rises from the bottom", c[climb].Desc)
	}
	if e := c[prog]; e.Title != "Program Alarms" || e.Pkg != "./cmd/prog" || e.Section != "Scenes" {
		t.Fatalf("prog entry = %+v, want Program Alarms out of ./cmd/prog", e)
	}
	if !strings.Contains(c[prog].Desc, "1202") || !strings.Contains(c[prog].Desc, "1201") {
		t.Fatalf("prog desc %q must name 1202 then 1201", c[prog].Desc)
	}
}

func TestSpacelanderShelving(t *testing.T) {
	// unhappy: each scene appears exactly once, never as a screenplay
	c := Catalog()
	seen := map[string]int{}
	for _, e := range c {
		switch e.ID {
		case "fall", "climb", "prog":
			seen[e.ID]++
			if e.Section != "Scenes" {
				t.Fatalf("%s is a scene, not a %s entry", e.ID, e.Section)
			}
		}
		if e.Section == "Screenplays" && strings.Contains(strings.ToLower(e.Title), "spacelander") {
			t.Fatalf("a Screenplays entry %q carries a spacelander scene", e.Title)
		}
	}
	for _, id := range []string{"fall", "climb", "prog"} {
		if seen[id] != 1 {
			t.Fatalf("the catalog holds %d %s entries, want exactly 1", seen[id], id)
		}
	}
}
