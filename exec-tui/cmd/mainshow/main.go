// mainshow: 05. Main — the whole show inside the screenplay editor,
// wearing MAIN's own numbers. The bill is every numbered show's bill
// added together (moon orbit, walkthrough, mario, inverse — thirteen
// scenes) and the director opens on scene one, browsing: the marquee,
// one hold row trimmed with h/l, and the help. ctrl+n and ctrl+p (or
// plain n/p) scroll the scenes; e opens the MAIN CONFIG panel — the
// hold, then every one of the scene's own knobs, j/k picking and h/l
// turning, never clamped. Space plays the show through, cutting on
// each scene's hold; f drops the chrome and premieres the whole thing
// fullscreen from the top (f or esc comes back); r replays the scene;
// s saves one file — MAIN's own config beside the bill — and never
// touches a scene's own config.json.
//
//	ctrl+n / ctrl+p   next / previous scene (n/p work too)
//	h / l             trim how long this scene lasts in play mode
//	e                 edit the scene: MAIN CONFIG, every knob (e/esc done)
//	j/k h/l           pick a knob, turn a knob (editing)
//	space             play / pause (cuts itself on each scene's hold)
//	f                 fullscreen premiere from the top (f/esc returns)
//	r                 replay the scene from its top
//	s                 save MAIN's config
//	q                 quit
//
//	go run ./cmd/mainshow
//	go run ./cmd/mainshow -seconds 30
package main

import (
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/colorprofile"

	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/director"
	"github.com/theprimeagen/apollo-11/exec-tui/menu"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/fall"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/moonwalk"
	"github.com/theprimeagen/apollo-11/exec-tui/shows/mainshow"
	"github.com/theprimeagen/apollo-11/exec-tui/termreset"
)

// applySky loads a tuned sky config and makes it the active sky. A
// missing file quietly keeps the stock sky; a broken file is an error
// worth stopping for.
func applySky(path string) error {
	c, err := stars.LoadSky(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return stars.UseSky(c)
}

// applyPuff loads a tuned dust config and makes it the active kick.
// A missing file quietly keeps the stock puff; a broken file is an
// error worth stopping for.
func applyPuff(path string) error {
	c, err := dust.LoadPuff(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return dust.UsePuff(c)
}

func forcedColorProfile() (colorprofile.Profile, bool) {
	if os.Getenv("CLICOLOR_FORCE") != "" {
		return colorprofile.ANSI256, true
	}
	return 0, false
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "mainshow:", err)
	os.Exit(1)
}

func main() {
	seconds := flag.Float64("seconds", 0, "auto-quit after N seconds (0 = interactive)")
	skyPath := flag.String("stars", "components/stars/config.json",
		"sky config JSON (adjuststars); a missing file keeps the stock sky")
	puffPath := flag.String("dust", "components/dust/config.json",
		"dust puff JSON (adjustdust); a missing file keeps the stock kick")
	landPath := flag.String("landing", landing.DefaultConfigPath,
		"landing timing JSON; a missing file keeps the stock knobs")
	fallPath := flag.String("fall", fall.DefaultConfigPath,
		"fall timing JSON; a missing file keeps the stock knobs")
	walkPath := flag.String("moonwalk", moonwalk.DefaultConfigPath,
		"moonwalk timing JSON; a missing file keeps the stock knobs")
	liftPath := flag.String("liftoff", liftoff.DefaultConfigPath,
		"liftoff timing JSON; a missing file keeps the stock knobs")
	bobPath := flag.String("bobble", bobble.DefaultConfigPath,
		"bobble ride JSON; a missing file keeps the stock knobs")
	cfgPath := flag.String("config", mainshow.ConfigPath,
		"MAIN's own config JSON — every scene's hold and knobs (the editor saves it); a missing file plays the stock show")
	flag.Parse()
	if err := applySky(menu.Resolve(*skyPath)); err != nil {
		fail(err)
	}
	if err := applyPuff(menu.Resolve(*puffPath)); err != nil {
		fail(err)
	}
	lc, err := landing.LoadOrDefault(menu.Resolve(*landPath))
	if err != nil {
		fail(err)
	}
	if err := landing.Use(lc); err != nil {
		fail(err)
	}
	fc, err := fall.LoadOrDefault(menu.Resolve(*fallPath))
	if err != nil {
		fail(err)
	}
	if err := fall.Use(fc); err != nil {
		fail(err)
	}
	wc, err := moonwalk.LoadOrDefault(menu.Resolve(*walkPath))
	if err != nil {
		fail(err)
	}
	if err := moonwalk.Use(wc); err != nil {
		fail(err)
	}
	tc, err := liftoff.LoadOrDefault(menu.Resolve(*liftPath))
	if err != nil {
		fail(err)
	}
	if err := liftoff.Use(tc); err != nil {
		fail(err)
	}
	bc, err := bobble.LoadOrDefault(menu.Resolve(*bobPath))
	if err != nil {
		fail(err)
	}
	if err := bobble.Use(bc); err != nil {
		fail(err)
	}
	cfg, err := director.LoadOrDefault(menu.Resolve(*cfgPath))
	if err != nil {
		fail(err)
	}
	m := director.New(mainshow.Title, mainshow.Bill(), cfg, *cfgPath, *seconds)
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fail(err)
	}
}
