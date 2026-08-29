// mainshow: 05. Main — the whole show inside the screenplay editor.
// The bill is every numbered show's bill added together (moon orbit,
// walkthrough, mario, inverse — thirteen scenes) and the director
// opens on scene one. n and p walk the cuts both ways; j/k pick a
// knob on the panel — the editor's own hold first, then the scene's
// live knobs — and h/l turn it, never clamped. Space plays the show
// through, cutting on each scene's hold; f drops the chrome and
// premieres the whole thing fullscreen from the top (f or esc comes
// back); r replays the scene; s saves the holds beside the bill and
// the scene's knobs to their own config files.
//
//	n / p       next / previous scene
//	space       play / pause (cuts itself on each scene's hold)
//	f           fullscreen premiere from the top (f/esc returns)
//	j/k h/l     pick a knob, turn a knob
//	r           replay the scene from its top
//	s           save holds + the scene's knobs
//	q           quit
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
	holdsPath := flag.String("holds", mainshow.HoldsPath,
		"scene holds JSON (the editor saves it); a missing file plays the stock holds")
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
	holds, err := director.LoadOrDefault(menu.Resolve(*holdsPath))
	if err != nil {
		fail(err)
	}
	m := director.New(mainshow.Title, mainshow.Bill(), holds, *holdsPath, *seconds)
	var opts []tea.ProgramOption
	if p, ok := forcedColorProfile(); ok {
		opts = append(opts, tea.WithColorProfile(p))
	}
	if _, err := termreset.Run(m, opts...); err != nil {
		fail(err)
	}
}
