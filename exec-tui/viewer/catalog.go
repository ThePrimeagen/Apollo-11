package viewer

import (
	"path/filepath"

	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustarmed"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustcloud"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustdust"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustflame"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustgunfire"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustparticle"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/adjustsky"
	"github.com/theprimeagen/apollo-11/exec-tui/cmd/editor"
	"github.com/theprimeagen/apollo-11/exec-tui/components/armed"
	"github.com/theprimeagen/apollo-11/exec-tui/components/astro"
	"github.com/theprimeagen/apollo-11/exec-tui/components/dsky"
	"github.com/theprimeagen/apollo-11/exec-tui/components/dust"
	"github.com/theprimeagen/apollo-11/exec-tui/components/eagle"
	"github.com/theprimeagen/apollo-11/exec-tui/components/flag"
	"github.com/theprimeagen/apollo-11/exec-tui/components/lander"
	"github.com/theprimeagen/apollo-11/exec-tui/components/moon"
	"github.com/theprimeagen/apollo-11/exec-tui/components/nyan"
	"github.com/theprimeagen/apollo-11/exec-tui/components/pools"
	"github.com/theprimeagen/apollo-11/exec-tui/components/shotgun"
	"github.com/theprimeagen/apollo-11/exec-tui/components/stars"
	"github.com/theprimeagen/apollo-11/exec-tui/components/title"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/america"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/bobble"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/coreset2"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/landing"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/liftoff"
	"github.com/theprimeagen/apollo-11/exec-tui/scenes/skies"
	"github.com/theprimeagen/apollo-11/exec-tui/screenplay"
)

// Kind is the type-line word under the title.
type Kind string

const (
	KindComponent Kind = "component"
	KindParticle  Kind = "particle"
	KindScene     Kind = "scene"
)

func (k Kind) String() string { return string(k) }

// Item is one entry the viewer can show and edit.
type Item struct {
	ID      string
	Title   string
	Kind    Kind
	Path    string
	Program string
	spawn   func() screenplay.Component
}

// Edit is what e chose: the kind, the file the editor should open, and
// the program a particle/scene tuner launches.
type Edit struct {
	Kind    Kind
	Path    string
	Program string
}

// Catalog is every component, particle effect, and scene, shotgun first.
func Catalog() []Item {
	assets := editor.DefaultAssetsDir
	return []Item{
		{ID: "shotgun", Title: "SHOTGUN", Kind: KindComponent, Path: shotgun.FindAtlas(), spawn: func() screenplay.Component { return newCyclingGun() }},
		{ID: "stars", Title: "STARS", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return stars.NewTunedStarfield() }},
		{ID: "sky", Title: "SKY", Kind: KindComponent, Path: adjustsky.DefaultConfigPath, Program: "./cmd/adjustsky/main", spawn: func() screenplay.Component { return newSkyPreview() }},
		{ID: "cloud", Title: "CLOUD", Kind: KindComponent, Path: adjustcloud.DefaultConfigPath, Program: "./cmd/adjustcloud/main", spawn: func() screenplay.Component { return newCloudPreview() }},
		{ID: "lander", Title: "LANDER", Kind: KindComponent, Path: filepath.Join(lander.FindArtDir(), "lm.json"), spawn: func() screenplay.Component { return lander.NewShip(11) }},
		{ID: "flag", Title: "FLAG", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return flag.New(4) }},
		{ID: "transition", Title: "TRANSITION", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return newTransitionPreview() }},
		{ID: "eagle", Title: "EAGLE", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return eagle.New() }},
		{ID: "armed", Title: "ARMED EAGLE", Kind: KindComponent, Path: adjustarmed.DefaultConfigPath, Program: "./cmd/adjustarmed/main", spawn: func() screenplay.Component { return armed.New() }},
		{ID: "moon", Title: "MOON", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return moon.New() }},
		{ID: "dsky", Title: "DSKY", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return dsky.NewPanel(dsky.MonitorState()) }},
		{ID: "coreset", Title: "CORE SET", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return newBoxDemo(pools.NewCoreSet()) }},
		{ID: "coresets", Title: "CORE SET PANEL", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return newPoolDemo(pools.NewCoreSetPanel()) }},
		{ID: "vac", Title: "VAC", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return newBoxDemo(pools.NewVAC()) }},
		{ID: "vacs", Title: "VAC PANEL", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return newPoolDemo(pools.NewVACPanel()) }},
		{ID: "title", Title: "TITLE", Kind: KindComponent, Path: assets, spawn: mustTitle},
		{ID: "astronaut", Title: "ASTRONAUT", Kind: KindComponent, Path: astro.FindAtlas(), spawn: func() screenplay.Component { return newAstroRun() }},
		{ID: "rocket", Title: "ROCKET", Kind: KindComponent, Path: assets, spawn: func() screenplay.Component { return newRocketPreview() }},
		{ID: "gunfire", Title: "GUNFIRE", Kind: KindParticle, Path: adjustgunfire.DefaultConfigPath, Program: "./cmd/adjustgunfire/main", spawn: func() screenplay.Component { return newBurstingBlast() }},
		{ID: "flame", Title: "FLAME", Kind: KindParticle, Path: adjustflame.DefaultConfigPath, Program: "./cmd/adjustflame/main", spawn: func() screenplay.Component { return newFlamePreview() }},
		{ID: "dust", Title: "DUST", Kind: KindParticle, Path: adjustdust.DefaultConfigPath, Program: "./cmd/adjustdust/main", spawn: func() screenplay.Component { return dust.NewCloud(7) }},
		{ID: "nyan", Title: "NYAN", Kind: KindParticle, Path: adjustparticle.DefaultConfigPath, Program: "./cmd/adjustparticle/main", spawn: func() screenplay.Component { return nyan.NewParked(7) }},
		{ID: "landing", Title: "LANDING", Kind: KindScene, Path: landing.DefaultConfigPath, Program: "./cmd/landing", spawn: func() screenplay.Component { return wrapScene(landing.New(nil)) }},
		{ID: "america", Title: "AMERICA", Kind: KindScene, Path: america.DefaultConfigPath, Program: "./cmd/america", spawn: func() screenplay.Component { return wrapScene(america.New()) }},
		{ID: "moonwalk", Title: "MOONWALK", Kind: KindScene, Path: "scenes/moonwalk/config.json", Program: "./cmd/astronaut", spawn: func() screenplay.Component { return newMoonwalkPreview() }},
		{ID: "skies", Title: "SKIES", Kind: KindScene, Path: skies.DefaultConfigPath, Program: "./cmd/skies", spawn: func() screenplay.Component { return wrapScene(skies.New()) }},
		{ID: "breakdown", Title: "BREAKDOWN", Kind: KindScene, Path: coreset.DefaultConfigPath, Program: "./cmd/coreset", spawn: func() screenplay.Component { return wrapScene(coreset.New()) }},
		{ID: "scan", Title: "SCAN", Kind: KindScene, Path: "scenes/coreset2", Program: "./cmd/coreset2", spawn: func() screenplay.Component { return wrapScene(coreset2.New()) }},
		{ID: "liftoff", Title: "LIFTOFF", Kind: KindScene, Path: liftoff.DefaultConfigPath, Program: "./cmd/liftoff", spawn: func() screenplay.Component { return wrapScene(liftoff.New(nil)) }},
		{ID: "bobble", Title: "BOBBLE", Kind: KindScene, Path: bobble.DefaultConfigPath, Program: "./cmd/bobble", spawn: func() screenplay.Component { return wrapScene(bobble.New(nil)) }},
	}
}

func mustTitle() screenplay.Component {
	t, err := title.New("APOLLO", 5)
	if err != nil {
		return &emptyPreview{}
	}
	return t
}
