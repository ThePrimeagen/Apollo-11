# screenplay

A screenplay is scenes in order; a scene is a cast of components playing
over time. `space` cuts to the next scene. The numbered bills live under
`shows/` and launch from the Screenplays shelf:

```bash
cd exec-tui
go run ./cmd/moon                    # 01. Moon Orbit
go run ./cmd/lunarcloseup            # 02. Walkthrough
go run ./cmd/mario                   # 03. Mario
go run ./cmd/inverse                 # 04. Inverse Walkthrough
go run ./cmd/mainshow                # 05. Main — everything, in the editor
```

`space` cuts to the next scene (past the last one, the show ends). `q` / `ctrl+c` quit.

**05. Main** is the one that puts everything together: its bill is the
other four bills added together — thirteen scenes — and it runs inside
`director`, the screenplay editor, wearing **MAIN's own numbers**.
`ctrl+n`/`ctrl+p` (or plain `n`/`p`) scroll the scenes both ways (the
screenplay's `Next` has a `Prev` mirror, and `Rewind` cuts to the top).
Browsing is the quiet face: one **hold** row — how many seconds the
scene plays in play mode before the cut — trimmed directly with
`h`/`l`. `e` opens the **MAIN CONFIG** panel for the scene now
playing: the hold first, then every one of the scene's own knobs —
the orbit's arrive and lap, the close-up's fly-in, the fire's brake,
the fall's drop, the landing's twenty-four, the moonwalk's eleven,
the liftoff's nine, each bobble's three — `j`/`k` pick a row, `h`/`l`
turn it, `e`/`esc` hand the quiet face back, and nothing is ever
clamped: a zero or negative hold cuts at once. The panel wears the
MAIN CONFIG name because these numbers are the show's copy: `s`
writes one file, `shows/mainshow/config.json` — every scene in bill
order with its hold and its knobs — and never touches a scene
package's config.json or its Active, so tuning MAIN never retunes a
scene launched by itself. Saving from a moonwalk beat syncs its
sibling beats, since the three are one performance, while the two
bobble entries keep the bill's word on the engine. `space` plays the
show through on the holds; `f` drops the chrome and premieres the
whole show fullscreen from the top (`f`/`esc` returns, and the end of
the show returns on its own); `r` replays the scene.

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

Shows compose. A `Bill` is one screenplay's worth of scenes — the
composable unit — and `Compose(bills...)` flattens bills, in order,
into one big screenplay, so the final product is every show's bill
added together. `shows/moonshow` is the first packaged bill: scene
one, the bare moon alone under a parked sky; scene two, a spaceship
streaks in fast off the left wing, brakes onto its orbit, and circles
the moon until the next cut (`go run ./cmd/moon` plays it; space past
the last scene ends the show). `shows/lunarcloseup` is **02. Walkthrough**:
five scenes — the pause (the drifting sky alone, held for as long as
the audience likes), then the zoomed-in lander slides in from the
right the moment space is pressed, then the parked craft lights its
booster while the stars slow by 60% over five seconds, then a
north-facing fall from the top of the stage to the bottom, then a
huge moon horizon (five rows high in the middle, one row at the
edges) that the lander comes down onto
(`go run ./cmd/lunarcloseup`; space past the last scene ends the show).
One `stars.Continuity` seeds every scene's sky — each new starfield
opens on the exact frame the last one left on screen, so no cut ever
jumps or skips a star.

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
| `components/stars` | `Starfield` — the cached-catalog sky, with an optional right-edge dock wipe, a Slow brake, and a Continuity `Seed` that carries the sky across scene cuts |
| `components/particle` | `Engine` — live config get/set (count, life, max distance) |
| `components/lander` | `Ship` — atlas hull + optional booster plume; west fly-in, north drop, north landing (plume throttles ¾, ½, ¼, off) |
| `components/dsky` | `Panel` — the DSKY docking on the right third, column-by-column |
| `components/moon` | `Moon` — the pixelated disc alone; `Orbit` — the lone gold craft circling it; `Horizon` — a huge moon's surface as a colored floor along the bottom |
| `components/title` | `Title` — banner cards set in terminal-fonts |

The `screenplay` package knows nothing about landers, stars, or fonts —
it speaks sprites and lip gloss cells only.

## adjuststars: the sky tuner

Another adjusting scene, in the adjustflame mold: the whole starfield
plays behind a panel of eight numbers — a fly **delay** (ticks per cell,
lower is faster, 0 parks the layer still) and a **density** (stars per
1000 cells) for each of the four star layers (`·` dust, `˚` spark, `*`
mid, `✦` near). The sky reacts live as you turn the knobs — zero every
delay and the whole sky holds.

```bash
go run ./cmd/adjuststars/main               # edits components/stars/config.json
go run ./cmd/adjuststars/main -seconds 15   # auto-quit, handy for tapes
```

`j`/`k` (or arrows) pick a number, `h`/`l` change it, `s` **saves the
config file and quits**, `q` quits without saving. The tuner is itself a
`screenplay.Scene` on a one-scene bill — the same lifecycle that runs
any screenplay runs the tool.

The numbered screenplays read the same file at boot (`-stars`, default
`components/stars/config.json`) and their scenes cast `NewTunedStarfield()`,
which samples the active sky when it starts — so a tuned sky just works
in any scene, and a missing file just means the stock sky.

## Reuse, not rewrites

The ship is composed entirely from existing parts: the size-4 `W` frame
(`lander.DefaultAtlas()`), its baked tilde plume stripped, over a `fire`/
`particle` booster aimed `{X:1, Y:0}` — left-to-right fire out the tail
while the craft flies west. Every part lives in `components/`, ready to
be lifted into another project whole.
