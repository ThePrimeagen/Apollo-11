package menu

import (
	"strings"
	"testing"
)

func TestCatalogCarriesTheCodeScenes(t *testing.T) {
	// happy: Check Priority and Alarms are portable scenes on the
	// Scenes shelf, right after the Interpreter they grew out of
	interp, prio, alarms := -1, -1, -1
	c := Catalog()
	for i := range c {
		switch c[i].ID {
		case "interpreter":
			interp = i
		case "checkprio":
			prio = i
		case "alarms":
			alarms = i
		}
	}
	if prio < 0 {
		t.Fatal("catalog missing the Check Priority scene")
	}
	if alarms < 0 {
		t.Fatal("catalog missing the Alarms scene")
	}
	p := c[prio]
	if p.Section != "Scenes" || p.Title != "Check Priority" || p.Pkg != "./cmd/checkprio" {
		t.Fatalf("checkprio entry misfiled: %+v", p)
	}
	if !strings.Contains(p.Desc, "data[11]") || !strings.Contains(p.Desc, "priority") {
		t.Fatalf("checkprio desc %q must sell the twelfth-word read and the priority compare", p.Desc)
	}
	a := c[alarms]
	if a.Section != "Scenes" || a.Title != "Alarms" || a.Pkg != "./cmd/alarms" {
		t.Fatalf("alarms entry misfiled: %+v", a)
	}
	if !strings.Contains(a.Desc, "1202") || !strings.Contains(a.Desc, "1201") {
		t.Fatalf("alarms desc %q must name both codes — that is the lesson", a.Desc)
	}
	if interp < 0 || prio != interp+1 || alarms != prio+1 {
		t.Fatalf("shelf order interpreter=%d checkprio=%d alarms=%d — the code scenes follow the Interpreter", interp, prio, alarms)
	}
}

func TestCodeScenesShelving(t *testing.T) {
	// unhappy: each scene appears exactly once, never as a
	// screenplay, and the catalog's head and tail stay put
	c := Catalog()
	seen := map[string]int{}
	for _, e := range c {
		switch e.ID {
		case "checkprio", "alarms":
			seen[e.ID]++
			if e.Section != "Scenes" {
				t.Fatalf("%s is a scene, not a %s entry", e.ID, e.Section)
			}
		}
		if e.Section == "Screenplays" && (strings.Contains(e.Title, "Check Priority") || strings.Contains(e.Title, "Alarms")) {
			t.Fatalf("a Screenplays entry %q carries a code scene", e.Title)
		}
	}
	for _, id := range []string{"checkprio", "alarms"} {
		if seen[id] != 1 {
			t.Fatalf("the catalog holds %d %s entries, want exactly 1", seen[id], id)
		}
	}
	if c[0].ID != "moon" {
		t.Fatalf("catalog head disturbed: %s", c[0].ID)
	}
	if c[len(c)-2].ID != "agctop" || c[len(c)-1].ID != "agcgraph" {
		t.Fatalf("catalog tail = %s, %s; want agctop then agcgraph", c[len(c)-2].ID, c[len(c)-1].ID)
	}
}
