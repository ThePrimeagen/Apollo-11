package main

// Demo harness tests, written first: the premiere plays a
// four-scene bill on the shared screen. Scene one, "arrival": three
// seconds of drifting sky, then a starfield that translates with the
// westbound craft as it slides in from the right wing — hull only, no
// booster fire — then parks and bobbles at center stage. Space cuts to scene two, "dsky": the craft
// parked, the right third of the sky wipes away one column at a time
// (~500ms), and the DSKY docks in that space. Space cuts to scene
// three, "descent orbit": the pixelated moon with the lone gold craft
// circling it eastward over the top — no line, the craft alone traces
// the path — where the craft was, and why it flies sideways. Space
// cuts to scene four, "the end": the height-5 banner card. Space on the final
// scene holds; q and ctrl+c close the house. The view is the rendered
// screen plus one status line, always exactly window-height lines.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
)

func frames(m model, n int) model {
	for i := 0; i < n; i++ {
		mm, _ := m.Update(frameMsg{})
		m = mm.(model)
	}
	return m
}

func press(m model, msg tea.Msg) model {
	mm, _ := m.Update(msg)
	return mm.(model)
}

func space() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "} }

func runeKey(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func hasStar(v string) bool {
	for _, g := range stars.Glyphs {
		if strings.ContainsRune(v, g) {
			return true
		}
	}
	return false
}

var ansiPat = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// markerCell locates the moon's gold marker in a rendered view, ANSI
// stripped so escape bytes never shift the column count.
func markerCell(v string) (row, col int, ok bool) {
	for r, line := range strings.Split(ansiPat.ReplaceAllString(v, ""), "\n") {
		for c, ch := range []rune(line) {
			if ch == moon.MarkerGlyph {
				return r, c, true
			}
		}
	}
	return 0, 0, false
}

// skyColumns is the leftmost n columns of every view row, ANSI
// stripped — on the orbit scene that strip is pure sky, west of the
// ring, so it watches the stars alone.
func skyColumns(v string, n int) string {
	var b strings.Builder
	for _, line := range strings.Split(ansiPat.ReplaceAllString(v, ""), "\n") {
		rs := []rune(line)
		if len(rs) > n {
			rs = rs[:n]
		}
		b.WriteString(string(rs))
		b.WriteString("\n")
	}
	return b.String()
}

func TestPremiere(t *testing.T) {
	t.Run("happy: the house opens on scene 1/4, arrival, under stars", func(t *testing.T) {
		m := newModel(0)
		v := m.View().Content
		for _, want := range []string{"1/4", "arrival", "space", "quit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("opening view is missing %q", want)
			}
		}
		if !hasStar(v) {
			t.Fatal("the opening scene must show the starfield")
		}
		if strings.ContainsRune(v, '▌') {
			t.Fatal("the craft is still off the right wing at t=0")
		}
	})
	t.Run("happy: frames fly the craft in with a cold engine — hull, no fire", func(t *testing.T) {
		m := newModel(0)
		_ = m.View() // the opening paint stages the cast, as bubbletea does
		m = frames(m, 60)
		if strings.ContainsRune(m.View().Content, '▌') {
			t.Fatal("the hold is still running — the craft must stay offstage")
		}
		m = frames(m, 120) // three more seconds: hold ends and the fly-in is well under way
		if m.elapsed < 5.9 || m.elapsed > 6.1 {
			t.Fatalf("elapsed %f after 180 frames, want ~6.0", m.elapsed)
		}
		v := m.View().Content
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("after the hold the hull must be on screen")
		}
		if strings.ContainsAny(v, "⠁⠒⠶▒") {
			t.Fatal("arrival must fly a dark engine — no booster fire yet")
		}
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear until the next scene")
		}
	})
	t.Run("happy: the arrival sky slides with the craft — same cells, same ease", func(t *testing.T) {
		hold := lander.FlyInHoldSeconds
		for _, w := range []int{40, 72, 120} {
			for _, tt := range []float64{0, 0.5, 1, 2, lander.FlyInSeconds, lander.FlyInSeconds + 3} {
				_, c0 := lander.FlightPath(w, 28, 0)
				_, c := lander.FlightPath(w, 28, tt)
				got := stars.SlideOffset(w, lander.BodyCols, tt, lander.FlyInSeconds)
				if c0-c != got {
					t.Fatalf("w=%d t=%.1f ship traveled %d, sky slide %d", w, tt, c0-c, got)
				}
			}
			for _, sceneT := range []float64{0, 2, hold, hold + 1, hold + lander.FlyInSeconds, hold + lander.FlyInSeconds + 3} {
				flyT := sceneT - hold
				_, c0 := lander.FlightPath(w, 28, 0)
				_, c := lander.FlightPath(w, 28, flyT)
				got := stars.SlideOffset(w, lander.BodyCols, flyT, lander.FlyInSeconds)
				if c0-c != got {
					t.Fatalf("w=%d scene t=%.1f (fly t=%.1f) ship traveled %d, sky slide %d", w, sceneT, flyT, c0-c, got)
				}
			}
		}
	})
	t.Run("happy: each frame schedules the next", func(t *testing.T) {
		m := newModel(0)
		_, cmd := m.Update(frameMsg{})
		if cmd == nil {
			t.Fatal("a frame must schedule the next tick")
		}
	})
	t.Run("happy: space cuts to scene 2/4 — DSKY docks after the wipe", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 90)
		m = press(m, space())
		opening := m.View().Content
		for _, want := range []string{"2/4", "dsky"} {
			if !strings.Contains(opening, want) {
				t.Fatalf("the dsky scene is missing %q", want)
			}
		}
		if strings.Contains(opening, "VERB") {
			t.Fatal("the opening frame of the dock must not yet show the DSKY")
		}
		m = frames(m, 15) // 500ms at 30 fps
		v := m.View().Content
		for _, want := range []string{"VERB", "NOUN", "PROG", "ENTR"} {
			if !strings.Contains(v, want) {
				t.Fatalf("after the wipe the DSKY is missing %q", want)
			}
		}
		if !strings.ContainsRune(v, '▌') {
			t.Fatal("the parked craft stays on stage beside the DSKY")
		}
		if strings.ContainsAny(v, "⠁⠒⠶▒") {
			t.Fatal("the engine stays dark through the dsky scene")
		}
		if strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("the moon's craft does not appear until the next scene")
		}
		if !hasStar(v) {
			t.Fatal("the left sky must keep drifting beside the dock")
		}
	})
	t.Run("happy: space cuts to scene 3/4 — the moon and its descent path", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 30)
		m = press(m, space())
		m = press(m, space())
		v := m.View().Content
		for _, want := range []string{"3/4", "descent orbit"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the orbit scene is missing %q", want)
			}
		}
		if !strings.ContainsRune(v, '▓') {
			t.Fatal("the moon must fill the middle of the stage")
		}
		if !strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("the gold craft must circle the moon")
		}
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY sits this scene out")
		}
		if strings.ContainsRune(v, '▌') {
			t.Fatal("the craft sits this scene out — the marker is the story")
		}
		if !hasStar(v) {
			t.Fatal("the orbit still plays under the stars")
		}
		r0, c0, ok := markerCell(v)
		if !ok {
			t.Fatal("no marker on the opening frame")
		}
		m = frames(m, 90) // three seconds: a quarter lap along the ring
		r1, c1, ok := markerCell(m.View().Content)
		if !ok {
			t.Fatal("the marker left the stage")
		}
		if r0 == r1 && c0 == c1 {
			t.Fatal("frames must fly the marker along the ring")
		}
	})
	t.Run("happy: the orbit sky holds still — no stars move behind the moon", func(t *testing.T) {
		m := newModel(0)
		_ = m.View()
		m = frames(m, 30)
		m = press(m, space())
		m = press(m, space())
		before := skyColumns(m.View().Content, 12)
		if !hasStar(before) {
			t.Fatal("test premise: the strip west of the ring must hold stars")
		}
		m = frames(m, 90)
		if got := skyColumns(m.View().Content, 12); got != before {
			t.Fatal("the orbit scene's stars must hold perfectly still")
		}
	})
	t.Run("happy: space cuts to scene 4/4 — THE END, centered", func(t *testing.T) {
		m := frames(newModel(0), 30)
		m = press(m, space())
		m = press(m, space())
		m = press(m, space())
		v := m.View().Content
		for _, want := range []string{"4/4", "the end", "___"} {
			if !strings.Contains(v, want) {
				t.Fatalf("the end card is missing %q", want)
			}
		}
		if !hasStar(v) {
			t.Fatal("the end card still plays under the stars")
		}
		if strings.ContainsRune(v, '▌') {
			t.Fatal("the craft does not appear in the end card")
		}
		if strings.Contains(v, "VERB") {
			t.Fatal("the DSKY does not appear in the end card")
		}
		if strings.ContainsRune(v, moon.MarkerGlyph) {
			t.Fatal("the moon sets before the end card")
		}
	})
	t.Run("happy: the cut restarts the clock for the new scene's cast", func(t *testing.T) {
		m := frames(newModel(0), 30)
		m = press(m, space())
		m = press(m, space())
		m = press(m, space())
		before := m.View().Content
		m = frames(m, 30)
		if m.View().Content == before {
			t.Fatal("the end card's sky must drift on after the cut")
		}
	})
	t.Run("unhappy: space on the final scene holds the card", func(t *testing.T) {
		m := press(newModel(0), space())
		m = press(m, space())
		m = press(m, space())
		m = press(m, space())
		m = press(m, runeKey(' '))
		v := m.View().Content
		if !strings.Contains(v, "4/4") || !strings.Contains(v, "the end") {
			t.Fatal("extra spaces must hold on the final scene")
		}
	})
	t.Run("unhappy: q and ctrl+c close the house", func(t *testing.T) {
		for _, msg := range []tea.Msg{
			runeKey('q'),
			tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl},
		} {
			_, cmd := newModel(0).Update(msg)
			if cmd == nil {
				t.Fatalf("%v must quit", msg)
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Fatalf("%v must issue tea.Quit", msg)
			}
		}
	})
	t.Run("happy: -seconds brings the curtain down on time", func(t *testing.T) {
		m := newModel(0.05)
		mm, cmd := m.Update(frameMsg{})
		m = mm.(model)
		if _, ok := cmd().(tea.QuitMsg); ok {
			t.Fatal("one frame is 0.033s — too early for a 0.05s curtain")
		}
		_, cmd = m.Update(frameMsg{})
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("two frames pass 0.05s — the curtain must fall")
		}
	})
	t.Run("happy: the view fills the window even when the sky runs short", func(t *testing.T) {
		m := newModel(0)
		mm, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 20 {
			t.Fatalf("view has %d lines for a 20-line window", got)
		}
		mm, _ = m.Update(tea.WindowSizeMsg{Width: 90, Height: 32})
		m = mm.(model)
		if got := len(strings.Split(m.View().Content, "\n")); got != 32 {
			t.Fatalf("view has %d lines for a 32-line window", got)
		}
	})
	t.Run("happy: Init schedules the first frame", func(t *testing.T) {
		if newModel(0).Init() == nil {
			t.Fatal("Init must start the clock")
		}
	})
}

func TestApplySky(t *testing.T) {
	t.Run("happy: a tuned stars.json is applied as the active sky", func(t *testing.T) {
		t.Cleanup(stars.ResetSky)
		path := filepath.Join(t.TempDir(), "stars.json")
		cfg := stars.SkyConfig{Delay: []int{1, 1, 1, 1}, Density: []int{99, 99, 99, 99}}
		if err := cfg.Save(path); err != nil {
			t.Fatalf("seed save: %v", err)
		}
		used, err := applySky(path)
		if err != nil || !used {
			t.Fatalf("applySky = %v/%v, want used and no error", used, err)
		}
		if stars.ActiveSky().DensityLayers() != [4]int{99, 99, 99, 99} {
			t.Fatal("the premiere must fly the tuned sky")
		}
	})
	t.Run("happy: a missing file quietly keeps the stock sky", func(t *testing.T) {
		used, err := applySky(filepath.Join(t.TempDir(), "nowhere.json"))
		if err != nil || used {
			t.Fatalf("applySky = %v/%v, want quietly unused", used, err)
		}
	})
	t.Run("unhappy: a broken file is an error worth stopping for", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "stars.json")
		if err := os.WriteFile(path, []byte("{broken"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := applySky(path); err == nil {
			t.Fatal("a broken sky file must surface its error")
		}
	})
}

func TestForcedColorProfile(t *testing.T) {
	t.Run("happy: CLICOLOR_FORCE forces ANSI256 for tapes and CI", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "1")
		p, ok := forcedColorProfile()
		if !ok || p != colorprofile.ANSI256 {
			t.Fatalf("got %v/%v, want ANSI256 forced", p, ok)
		}
	})
	t.Run("unhappy: without the flag, detection is left alone", func(t *testing.T) {
		t.Setenv("CLICOLOR_FORCE", "")
		if _, ok := forcedColorProfile(); ok {
			t.Fatal("an empty CLICOLOR_FORCE must not force a profile")
		}
	})
}
