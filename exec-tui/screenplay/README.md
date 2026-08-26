# screenplay

A screenplay is scenes in order; a scene is a cast of components playing
over time. `space` cuts to the next scene. The premiere plays a
four-scene bill:

- **Scene 1 — arrival.** Three seconds of drifting sky, then a starfield
  that translates with the craft as it comes in: every star shifts left
  on the same ease-out cubic the hull flies, so the whole scene rushes
  past during the fly-in, then the sky's own parallax drift takes over
  once the craft parks. The full zoomed-in craft (the size-4, 26×10
  west-facing frame from the lander atlas) slides in from the right
  wing with a cold engine — no booster fire — parks at center stage,
  then bobbles one full cell up and down on a sine with a 10-second
  period.
- **Scene 2 — dsky.** The parked craft stays. Over ~500ms the right
  third of the sky blanks out one column at a time from the right edge,
  and the DSKY (V16 N68 on P63) docks in that space.
- **Scene 3 — descent orbit.** The explainer: a pixelated moon at
  center stage under a wide dotted ring — the descent path — and a
  lone gold marker rides that ring eastward over the top. This is
  where the craft was, and why it flies sideways. All circle math
  runs in half-cell pixels (a terminal cell is ~2× taller than wide),
  so the moon and the ring read round on a real terminal.
- **Scene 4 — the end.** `THE END` in the height-5 terminal-fonts banner,
  centered under the same sky.

```bash
cd exec-tui
go run ./cmd/premiere                # interactive
go run ./cmd/premiere -seconds 30    # auto-quit, handy for tapes
```

`space` cuts to the next scene (the final scene holds). `q` / `ctrl+c` quit.

## The shape

Two interfaces carry the whole show. A scene is anything that can start,
update, render onto the shared screen, and stop:

```go
type Scene interface {
    Start()             // allocate: the curtain rises
    Update(dt float64)  // advance internal clocks; no render data
    Render(scr *Screen) // write this instant's cells into the screen
    Stop()              // deallocate: the curtain falls
}
```

A component is one performer inside a scene. It never touches the
screen — every frame it hands back a sprite, and the scene composites
the cast in its own order:

```go
type Component interface {
    Start(width, height int) // allocate everything for a w×h stage
    Update(dt float64)       // advance internal clocks; no render data
    Render() sprite.Sprite   // this instant's pixels, stage-sized
    Stop()                   // free what Start built; Start may come again
}
```

The lifecycle loops — Start, update, render, update, render, …, Stop —
and a later Start re-allocates from scratch, so a stopped component
holds nothing. The stars are the model citizen: `Start` scatters and
caches the star catalog, `Render` flies it by tick, `Stop` deletes the
array, and the next `Start` scatters again.

`Ensemble` is the common scene shape: `Assemble` builds the cast at the
curtain, the components start on the first render (the moment the stage
size is known), a resize is a stop-and-restart at the new size, and
render order is cast order — later sprites land on top, transparent
cells sparing whatever is beneath. There is no z-index: every scene
decides ordering by how it lines up its cast.

The `Screen` is the shared render target: one Lip Gloss v2 canvas — a
`uv.Cell` of content plus style per terminal cell — plus `Resize` /
`Resized` bookkeeping. `Screen.PutCell` and `Screen.Blit` speak the
components' xterm-256 color language (-1 for "no color"), so the whole
canvas goes to lip gloss in one `screen.Render()`. The screenplay owns
the frame loop: it clears the screen, updates only the scene now playing
(so each scene's clocks start at its cut), renders it, and `Next()`
stops the old scene before starting the new.

| Package | What it owns |
| --- | --- |
| `screenplay` | `Screen` (the lip gloss cell canvas + xterm blit bridge), `Scene` and `Component` (the four-verb interfaces), `Ensemble` (the common scene shape), `Screenplay` (the lifecycle cursor) |
| `components/stars` | `Starfield` — the cached-catalog sky, with an optional right-edge dock wipe |
| `components/lander` | `Ship` — atlas hull + optional booster plume + `FlightPath` choreography |
| `components/dsky` | `Panel` — the DSKY docking on the right third, column-by-column |
| `components/moon` | `Moon` — the pixelated moon under the dotted descent path, gold marker riding it |
| `components/title` | `Title` — banner cards set in terminal-fonts |

The `screenplay` package knows nothing about landers, stars, or fonts —
it speaks sprites and lip gloss cells only.

## adjuststars: the sky tuner

Another adjusting scene, in the adjustflame mold: the whole starfield
plays behind a panel of eight numbers — a fly **delay** (ticks per cell,
lower is faster) and a **density** (stars per 1000 cells) for each of the
four star layers (`·` dust, `˚` spark, `*` mid, `✦` near). The sky reacts
live as you turn the knobs.

```bash
go run ./cmd/adjuststars/main               # edits components/stars/config.json
go run ./cmd/adjuststars/main -seconds 15   # auto-quit, handy for tapes
```

`j`/`k` (or arrows) pick a number, `h`/`l` change it, `s` **saves the
config file and quits**, `q` quits without saving. The tuner is itself a
`screenplay.Scene` on a one-scene bill — the same lifecycle that runs the
premiere runs the tool.

The premiere reads the same file at boot (`-stars`, default
`components/stars/config.json`) and its scenes cast `NewTunedStarfield()`,
which samples the active sky when it starts — so a tuned sky just works
in any scene, and a missing file just means the stock sky.

## Reuse, not rewrites

The ship is composed entirely from existing parts: the size-4 `W` frame
(`lander.DefaultAtlas()`), its baked tilde plume stripped, over a `fire`/
`particle` booster aimed `{X:1, Y:0}` — left-to-right fire out the tail
while the craft flies west. Every part lives in `components/`, ready to
be lifted into another project whole.
