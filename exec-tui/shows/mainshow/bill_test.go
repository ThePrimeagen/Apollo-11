package mainshow

// Tests written FIRST: MAIN is the one that puts everything together
// — a composable thirteen-scene bill that is every numbered show's
// bill added together, in shelf order: 01. Moon Orbit (the bare moon,
// the fly-in to orbit), 02. Walkthrough (pause, close-up, fire, fall,
// landing), 03. Mario (run, flagpole, board), then 04. Inverse
// Walkthrough (liftoff, engines on, engines off). Every entry carries
// the same performer its home show casts — the knobbed Shows keep
// their types so the editor can reach their knobs, and the bobble
// entries keep the bill's word on the engine. Each call builds fresh
// instances, and none of the old premiere's scenes ride along.

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/theprimeagen/apollo-11/exec-tui/components/caption"
	"github.com/theprimeagen/apollo-11/exec-tui/director"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/fall"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/lunarcloseup"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/moonshow"
)

var mainNames = []string{
	"the moon", "orbit",
	"pause", "Lunar Lander Close-Up", "fire", "fall", "landing",
	"run", "flagpole", "board",
	"liftoff", "engines on", "engines off",
}

func TestMainBill(t *testing.T) {
	t.Run("happy: the bill is every show's scenes in shelf order", func(t *testing.T) {
		b := Bill()
		if len(b) != len(mainNames) {
			t.Fatalf("MAIN holds %d scenes, want %d", len(b), len(mainNames))
		}
		for i, want := range mainNames {
			if b[i].Name != want {
				t.Fatalf("scene %d is %q, want %q", i+1, b[i].Name, want)
			}
			if b[i].Scene == nil {
				t.Fatalf("scene %q has no performer", want)
			}
		}
	})
	t.Run("happy: the show is called MAIN and its config lives beside the bill", func(t *testing.T) {
		if Title != "MAIN" {
			t.Fatalf("the show is called %q, want MAIN", Title)
		}
		if ConfigPath != "shows/mainshow/config.json" {
			t.Fatalf("MAIN's config lives at %q, want shows/mainshow/config.json", ConfigPath)
		}
	})
	t.Run("happy: every knobbed scene keeps its own type for the editor", func(t *testing.T) {
		b := Bill()
		byName := map[string]screenplay.Scene{}
		for _, e := range b {
			byName[e.Name] = e.Scene
		}
		if _, ok := byName["orbit"].(*moonshow.OrbitShow); !ok {
			t.Fatalf("orbit is %T, want the tunable orbit show", byName["orbit"])
		}
		if _, ok := byName["Lunar Lander Close-Up"].(*lunarcloseup.CloseupShow); !ok {
			t.Fatalf("the close-up is %T, want the tunable close-up show", byName["Lunar Lander Close-Up"])
		}
		if _, ok := byName["fire"].(*lunarcloseup.FireShow); !ok {
			t.Fatalf("fire is %T, want the tunable fire show", byName["fire"])
		}
		if _, ok := byName["fall"].(*fall.Show); !ok {
			t.Fatalf("fall is %T, want the fall show", byName["fall"])
		}
		if _, ok := byName["landing"].(*landing.Show); !ok {
			t.Fatalf("landing is %T, want the landing show", byName["landing"])
		}
		if _, ok := byName["liftoff"].(*liftoff.Show); !ok {
			t.Fatalf("liftoff is %T, want the liftoff show", byName["liftoff"])
		}
		beats := map[string]moonwalk.Beat{
			"run":      moonwalk.BeatRun,
			"flagpole": moonwalk.BeatPole,
			"board":    moonwalk.BeatBoard,
		}
		for name, beat := range beats {
			sc, ok := byName[name].(*moonwalk.Show)
			if !ok {
				t.Fatalf("%s is %T, want the moonwalk show", name, byName[name])
			}
			if sc.Beat() != beat {
				t.Fatalf("%s plays beat %v, want %v", name, sc.Beat(), beat)
			}
		}
	})
	t.Run("happy: the bobble entries keep the bill's word on the engine", func(t *testing.T) {
		b := Bill()
		lit, ok := b[11].Scene.(*bobble.Show)
		if !ok || b[11].Name != "engines on" {
			t.Fatalf("scene 12 is %q %T, want the lit bobble", b[11].Name, b[11].Scene)
		}
		if !lit.Cfg.Engine {
			t.Fatal("engines on must burn")
		}
		dark, ok := b[12].Scene.(*bobble.Show)
		if !ok || b[12].Name != "engines off" {
			t.Fatalf("scene 13 is %q %T, want the dark bobble", b[12].Name, b[12].Scene)
		}
		if dark.Cfg.Engine {
			t.Fatal("engines off must fly cold")
		}
	})
	t.Run("happy: every call builds a fresh cast", func(t *testing.T) {
		one, two := Bill(), Bill()
		for i := range one {
			if one[i].Scene == two[i].Scene {
				t.Fatalf("scene %q is shared between calls — the bills must be independent", one[i].Name)
			}
		}
	})
	t.Run("happy: the composed show walks all thirteen scenes and then has nothing left", func(t *testing.T) {
		p := screenplay.Compose(Bill())
		p.Start()
		defer p.Stop()
		if p.Len() != len(mainNames) || p.CurrentName() != "the moon" {
			t.Fatalf("the show opens on %d %q, want thirteen starting on the moon", p.Len(), p.CurrentName())
		}
		for i, want := range mainNames[1:] {
			if !p.Next() || p.CurrentName() != want {
				t.Fatalf("cut %d must land on %q, got %q", i+1, want, p.CurrentName())
			}
		}
		if p.Next() {
			t.Fatal("after engines off there is nothing left — the show ends")
		}
	})
	t.Run("unhappy: none of the old premiere rides along", func(t *testing.T) {
		for _, e := range Bill() {
			switch e.Name {
			case "arrival", "dsky", "descent orbit", "the end":
				t.Fatalf("MAIN must not carry old premiere scene %q", e.Name)
			}
		}
	})
	t.Run("unhappy: a scene stopped before its first render never panics", func(t *testing.T) {
		for _, e := range Bill() {
			e.Scene.Start()
			e.Scene.Update(1)
			e.Scene.Stop()
		}
	})
}

func TestMainFireSink(t *testing.T) {
	t.Run("happy: MAIN's fire knobs turn on a sink and the lit hull eases down", func(t *testing.T) {
		cfg, err := director.Load("config.json")
		if err != nil {
			t.Fatalf("MAIN config: %v", err)
		}
		raw := cfg.KnobsFor("fire")
		if !bytes.Contains(raw, []byte(`"sinkSeconds"`)) {
			t.Fatal("MAIN's fire knobs must name sinkSeconds — that is how MAIN turns the fall on")
		}
		var face lunarcloseup.FireConfig
		if err := json.Unmarshal(raw, &face); err != nil {
			t.Fatalf("fire knobs: %v", err)
		}
		if face.SinkSeconds == 0 {
			t.Fatal("MAIN's sink must be a non-zero window")
		}

		sc, ok := Bill()[4].Scene.(*lunarcloseup.FireShow)
		if !ok {
			t.Fatalf("MAIN scene 5 is %T, want the fire show", Bill()[4].Scene)
		}
		next := sc.Cfg
		if err := json.Unmarshal(raw, &next); err != nil {
			t.Fatalf("apply fire knobs: %v", err)
		}
		sc.Cfg = next
		if sc.Cfg.SinkSeconds != face.SinkSeconds {
			t.Fatalf("applied sink is %v, want MAIN's %v", sc.Cfg.SinkSeconds, face.SinkSeconds)
		}
		sc.Start()
		defer sc.Stop()
		openScr := screenplay.NewScreen(72, 27)
		sc.Render(openScr)
		open := westHullTop(openScr)
		if open < 0 {
			t.Fatal("test premise: MAIN fire must open on the west hull")
		}
		const dt = 1.0 / 30
		for tck := 0.0; tck < 7-dt/2; tck += dt {
			sc.Update(dt)
		}
		mid := screenplay.NewScreen(72, 27)
		sc.Render(mid)
		got := westHullTop(mid)
		if got < 0 {
			t.Fatal("mid-sink the west hull must still be on stage")
		}
		if got <= open {
			t.Fatalf("MAIN fire hull top %d, want below the opening %d (cfg %+v) — the booster is on, the hull must sink", got, open, sc.Cfg)
		}
		if !strings.ContainsRune(mid.Render(), '▌') {
			t.Fatal("the sinking craft must stay west-facing")
		}
	})
	t.Run("unhappy: stock MAIN fire (no saved knobs) stays parked, and a zero or odd sink never panics", func(t *testing.T) {
		stock, ok := Bill()[4].Scene.(*lunarcloseup.FireShow)
		if !ok {
			t.Fatalf("MAIN scene 5 is %T, want the fire show", Bill()[4].Scene)
		}
		if stock.Cfg.SinkSeconds != 0 {
			t.Fatalf("stock fire sink is %v, want 0 — only MAIN's saved knobs turn it on", stock.Cfg.SinkSeconds)
		}
		stock.Start()
		openScr := screenplay.NewScreen(72, 27)
		stock.Render(openScr)
		open := westHullTop(openScr)
		const dt = 1.0 / 30
		for tck := 0.0; tck < 4-dt/2; tck += dt {
			stock.Update(dt)
		}
		mid := screenplay.NewScreen(72, 27)
		stock.Render(mid)
		got := westHullTop(mid)
		if got < open-1 || got > open+1 {
			t.Fatalf("stock MAIN fire hull top %d, want the park around %d", got, open)
		}
		stock.Stop()

		odd := lunarcloseup.NewFireShow(nil)
		odd.Cfg.SinkSeconds = -3
		odd.Start()
		oddScr := screenplay.NewScreen(72, 27)
		odd.Render(oddScr)
		odd.Update(1)
		odd.Stop()

		gone := lunarcloseup.NewFireShow(nil)
		gone.Cfg.SinkSeconds = 0
		gone.Start()
		gone.Update(1)
		gone.Stop()
	})
}

func westHullTop(scr *screenplay.Screen) int {
	w, h := scr.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := scr.Cell(x, y)
			if c != nil && strings.ContainsRune(c.Content, '▌') {
				return y
			}
		}
	}
	return -1
}

func TestMainFallAlarms(t *testing.T) {
	t.Run("happy: MAIN's fall knobs arm the 1202 / 1202 / 1201 holds", func(t *testing.T) {
		cfg, err := director.Load("config.json")
		if err != nil {
			t.Fatalf("MAIN config: %v", err)
		}
		raw := cfg.KnobsFor("fall")
		if !bytes.Contains(raw, []byte(`"hold1"`)) {
			t.Fatal("MAIN's fall knobs must name hold1 — that is how MAIN turns the alarms on")
		}
		bill := Bill()
		_ = director.New(Title, bill, cfg, ConfigPath, 0)
		sc, ok := bill[5].Scene.(*fall.Show)
		if !ok || bill[5].Name != "fall" {
			t.Fatalf("scene 6 is %q %T, want fall", bill[5].Name, bill[5].Scene)
		}
		if sc.Cfg.Hold1 <= 0 || sc.Cfg.Hold2 <= 0 || sc.Cfg.Hold3 <= 0 {
			t.Fatalf("MAIN fall holds are %v/%v/%v, want positive so the cards fire",
				sc.Cfg.Hold1, sc.Cfg.Hold2, sc.Cfg.Hold3)
		}
		sc.Start()
		defer sc.Stop()
		scr := screenplay.NewScreen(72, 27)
		sc.Render(scr)
		const dt = 1.0 / 30
		var saw1202, saw1201 bool
		for i := 0; i < 240; i++ {
			sc.Update(dt)
			scr.Clear()
			sc.Render(scr)
			if caption.Painted(scr, "1202") {
				saw1202 = true
			}
			if caption.Painted(scr, "1201") {
				saw1201 = true
			}
		}
		if !saw1202 {
			t.Fatal("MAIN scene 6 must paint 1202 — the first P63 alarm")
		}
		if !saw1201 {
			t.Fatal("MAIN scene 6 must reach 1201 — 1202, 1202, then 1201")
		}
	})
	t.Run("unhappy: stock walkthrough fall stays dark — MAIN does not rewrite the scene", func(t *testing.T) {
		stock := fall.DefaultConfig()
		if stock.Hold1 != 0 || stock.Hold2 != 0 || stock.Hold3 != 0 {
			t.Fatalf("stock fall holds are %v/%v/%v, want zero so walkthrough stays a plain drop",
				stock.Hold1, stock.Hold2, stock.Hold3)
		}
		sc := fall.New(nil)
		sc.Start()
		defer sc.Stop()
		scr := screenplay.NewScreen(72, 27)
		sc.Render(scr)
		const dt = 1.0 / 30
		for i := 0; i < 90; i++ {
			sc.Update(dt)
			sc.Render(scr)
			if caption.Painted(scr, "1202") || caption.Painted(scr, "1201") {
				t.Fatalf("stock fall must not flash a card at t=%.2f — MAIN only arms its own knobs", float64(i+1)*dt)
			}
		}
	})
}

func TestMainLandingCaptions(t *testing.T) {
	t.Run("happy: MAIN's landing knobs keep the 1202s and LAND dark", func(t *testing.T) {
		cfg, err := director.Load("config.json")
		if err != nil {
			t.Fatalf("MAIN config: %v", err)
		}
		bill := Bill()
		_ = director.New(Title, bill, cfg, ConfigPath, 0)
		sc, ok := bill[6].Scene.(*landing.Show)
		if !ok || bill[6].Name != "landing" {
			t.Fatalf("scene 7 is %q %T, want landing", bill[6].Name, bill[6].Scene)
		}
		if sc.Cfg.Code1Hold != 0 || sc.Cfg.Code2Hold != 0 || sc.Cfg.LandCaptionHold != 0 {
			t.Fatalf("MAIN landing holds are 1202a=%v 1202b=%v LAND=%v, want zero so the cards stay dark",
				sc.Cfg.Code1Hold, sc.Cfg.Code2Hold, sc.Cfg.LandCaptionHold)
		}
		if !sc.Cfg.Star.Dust {
			t.Fatal("MAIN's star knobs must arm the persist trail — that is the dust")
		}
		if sc.Cfg.Star.Delay <= 0 {
			t.Fatal("MAIN's star must wait before it flies")
		}
		if sc.Cfg.Star.StartY == 0 || sc.Cfg.Star.StartY >= 0.08 {
			t.Fatalf("MAIN star startY %v, want a higher top-right start than the stock 0.08 path", sc.Cfg.Star.StartY)
		}
		sc.Start()
		defer sc.Stop()
		scr := screenplay.NewScreen(72, 27)
		sc.Render(scr)
		const dt = 1.0 / 30
		for i := 0; i < 180; i++ {
			sc.Update(dt)
			sc.Render(scr)
			if caption.Painted(scr, "1202") || caption.Painted(scr, "LAND") {
				t.Fatalf("MAIN scene 7 must not flash 1202 or LAND at t=%.2f", float64(i+1)*dt)
			}
		}
	})
	t.Run("unhappy: stock landing still paints the cards — MAIN does not strip the scene", func(t *testing.T) {
		stock := landing.DefaultConfig()
		if stock.Code1Hold <= 0 || stock.Code2Hold <= 0 || stock.LandCaptionHold <= 0 {
			t.Fatalf("stock landing holds are 1202a=%v 1202b=%v LAND=%v, want the walkthrough's cards still timed",
				stock.Code1Hold, stock.Code2Hold, stock.LandCaptionHold)
		}
		sc := landing.New(nil)
		sc.Cfg.Code1At, sc.Cfg.Code1Hold = 0.1, 0.4
		sc.Cfg.Code2At, sc.Cfg.Code2Hold = 0.6, 0.4
		sc.Cfg.LandCaptionAt, sc.Cfg.LandCaptionHold = 1.2, 0.4
		sc.Start()
		defer sc.Stop()
		scr := screenplay.NewScreen(72, 27)
		sc.Render(scr)
		const dt = 1.0 / 30
		for i := 0; i < 10; i++ {
			sc.Update(dt)
			sc.Render(scr)
		}
		if !caption.Painted(scr, "1202") {
			t.Fatal("a landing that still holds the cards must paint 1202 — MAIN only zeros its own knobs")
		}
	})
}

func TestMainLiftoff(t *testing.T) {
	t.Run("happy: MAIN's liftoff knobs fly the white north cabin and keep the pad dustless", func(t *testing.T) {
		cfg, err := director.Load("config.json")
		if err != nil {
			t.Fatalf("MAIN config: %v", err)
		}
		raw := cfg.KnobsFor("liftoff")
		if !bytes.Contains(raw, []byte(`"whiteOnly"`)) {
			t.Fatal("MAIN's liftoff knobs must name whiteOnly — that is how MAIN flies just the ascent cabin")
		}
		bill := Bill()
		_ = director.New(Title, bill, cfg, ConfigPath, 0)
		sc, ok := bill[10].Scene.(*liftoff.Show)
		if !ok || bill[10].Name != "liftoff" {
			t.Fatalf("scene 11 is %q %T, want liftoff", bill[10].Name, bill[10].Scene)
		}
		if !sc.Cfg.WhiteOnly {
			t.Fatal("MAIN liftoff must arm whiteOnly so only the white top leaves the pad")
		}
		if sc.Cfg.DustRun != 0 {
			t.Fatalf("MAIN dust run is %v, want 0 so the pad stays clear", sc.Cfg.DustRun)
		}
		sc.Start()
		defer sc.Stop()
		scr := screenplay.NewScreen(72, 27)
		sc.Render(scr)
		const (
			silver = 252
			gold   = 178
		)
		open := firstMainInk(scr, silver)
		if open < 0 {
			t.Fatal("MAIN scene 11 must open on the white north cabin")
		}
		if n := countMainInk(scr, gold); n != 0 {
			t.Fatalf("MAIN opening still paints %d gold cells — only the white top should be on stage", n)
		}
		const dt = 1.0 / 30
		for i := 0; i < 66; i++ {
			sc.Update(dt)
		}
		scr.Clear()
		sc.Render(scr)
		got := firstMainInk(scr, silver)
		if got < 0 {
			t.Fatal("mid-climb the white cabin must still be on stage")
		}
		if got >= open {
			t.Fatalf("MAIN white cabin row %d, want above the opening %d", got, open)
		}
		if n := countMainInk(scr, gold); n != 0 {
			t.Fatalf("mid-climb gold is on stage (%d) — the full hull must not rise", n)
		}
		if mainPadSmoke(scr) {
			t.Fatal("MAIN scene 11 must stay dustless after ignition")
		}
	})
	t.Run("unhappy: stock inverse/standalone liftoff is still the full craft — MAIN does not rewrite the scene", func(t *testing.T) {
		stock := liftoff.DefaultConfig()
		if stock.WhiteOnly {
			t.Fatal("stock WhiteOnly is true — MAIN must not change the scene default")
		}
		if stock.DustRun <= 0 {
			t.Fatalf("stock dust run is %v, want the pad cloud still scheduled", stock.DustRun)
		}
		sc := liftoff.New(nil)
		sc.Start()
		defer sc.Stop()
		scr := screenplay.NewScreen(72, 27)
		sc.Render(scr)
		if countMainInk(scr, 178) == 0 {
			t.Fatal("stock liftoff must still wear the gold descent — MAIN only arms its own knobs")
		}
	})
}

const mainCabinGlyphs = "▗▀▜▛▖▟█▙░▞▚"

func firstMainInk(scr *screenplay.Screen, fg int) int {
	w, h := scr.Size()
	lo := (w - 26) / 2
	hi := lo + 26
	for y := 0; y < h; y++ {
		for x := lo; x < hi && x < w; x++ {
			c := scr.Cell(x, y)
			if c == nil {
				continue
			}
			ic, ok := c.Style.Fg.(ansi.IndexedColor)
			if !ok || int(ic) != fg {
				continue
			}
			if fg == 252 && !strings.ContainsAny(c.Content, mainCabinGlyphs) {
				continue
			}
			return y
		}
	}
	return -1
}

func countMainInk(scr *screenplay.Screen, fg int) int {
	n := 0
	w, h := scr.Size()
	lo := (w - 26) / 2
	hi := lo + 26
	for y := 0; y < h; y++ {
		for x := lo; x < hi && x < w; x++ {
			c := scr.Cell(x, y)
			if c == nil {
				continue
			}
			ic, ok := c.Style.Fg.(ansi.IndexedColor)
			if !ok || int(ic) != fg {
				continue
			}
			if fg == 252 && !strings.ContainsAny(c.Content, mainCabinGlyphs) {
				continue
			}
			n++
		}
	}
	return n
}

func mainPadSmoke(scr *screenplay.Screen) bool {
	w, h := scr.Size()
	hullCol := (w - 26) / 2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x >= hullCol && x < hullCol+26 {
				continue
			}
			c := scr.Cell(x, y)
			if c == nil {
				continue
			}
			for _, r := range c.Content {
				if r == '░' || r == '▒' {
					return true
				}
			}
		}
	}
	return false
}
