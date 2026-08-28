# components/stars

The one-cell starfield component. Paint it first; everything else lands
on top.

```bash
cd exec-tui
go run ./cmd/stars                       # the demo: dust-rush (flying)
go run ./cmd/stars -strategy still       # stars hold still
```

`n` / `p` cycle styles. `space` pauses. `q` quits.

## Styles

| `-strategy` | Motion |
| --- | --- |
| `dust-rush` (default) | · and ˚ streak right→left; * and ✦ almost hang |
| `still` | no motion — a static night sky |
| `twinkle` | no motion — some stars fade in and out while the rest hold |
| `far-fast` / `near-fast` / `uniform` / `uniform-slow` / `hyperspace` / `drift` | other fly takes |

Glyphs: `·` dust, `˚` spark, `*` mid, `✦` near. Colors are white with a little
blue-white and red-white. `*` / `✦` are sparse (~25% of the dust layers).

## The component

`Starfield` plays as a `screenplay.Component`: `Start(w, h)` scatters and
**caches** the star catalog for that stage (`NewCatalog` — every star's
home cell, laid out once), `Update(dt)` runs the fly clock, `Render()`
paints the cached catalog into a stage-sized sprite, and `Stop()` deletes
the array — a stopped sky holds no allocation, and the next `Start`
re-scatters. `NewTunedStarfield()` samples the active sky settings at
`Start`, so a tuned config just works in any scene.

`Field` is the one-shot shape underneath: `Field.Paint` scatters and
paints in one breath (the demo and the tuner use it), and `Field.Density`
is the per-layer frequency knob — stars per 1000 cells for dust, spark,
mid, near. The zero value paints the stock sky (`DefaultDensity`,
`{56, 33, 6, 4}`); larger numbers thicken a layer, capped at `MaxDensity`
(400). Speed stays on `Strategy.Delay` — ticks per cell of travel, lower
is faster, and **0 parks the layer**: set every movement to zero and the
sky simply holds. `Starfield.Still()` is the same idea as a modifier —
a second star scene that never moves, whatever the strategy or the
tuned config say (the moon's descent orbit plays under it).

## The config

`config.json` in this folder is the component's own tuning file — one
`delay` and one `density` per layer. `SkyConfig` carries it: `LoadSky` /
`Save` / `Validate`, plus the package-active setting — `UseSky` /
`ActiveSky` / `ResetSky` — that consumers (the premiere, the
`adjuststars` tuner) read so a tuned file just works.

```bash
go run ./cmd/adjuststars/main            # tunes components/stars/config.json
```

## Twinkle

The `twinkle` mode parks the sky and lets about a third of the stars
breathe — fade in over a ramp, hold bright, fade out, hold dark, and
around again. Each breather deterministically draws its cycle from
`[MinCycleSeconds, MaxCycleSeconds]` and its ramps from
`[MinFadeSeconds, MaxFadeSeconds]` (clamped to half its cycle), dims
through its layer's grays, and vanishes at zero while the steady stars
hold the frame. `TwinkleConfig` carries the four knobs between the
rails (`MinTwinkleCycle`/`MaxTwinkleCycle`, `MinTwinkleFade`/
`MaxTwinkleFade`); `UseTwinkle` / `ActiveTwinkle` / `ResetTwinkle` are
the package-active setting the paint reads every frame, so a tuner
(the explorer scene's `cmd/explorer`) retunes the breathing live.
