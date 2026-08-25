# screenplay-lab

A screenplay is scenes in order; a scene is anything that can start,
update, render, and stop. `space` cuts to the next scene. The lab
premieres a two-scene bill:

- **Scene 1 — arrival.** A drifting starfield. The full zoomed-in craft
  (the size-4, 26×10 west-facing frame from the lander-lab atlas) slides in
  from the right wing with live left-to-right booster fire trailing off its
  tail, parks at center stage, then bobbles one full cell up and down on a
  sine with a 10-second period. (Half a cell needs half-shifted art the
  atlas doesn't have — the bobble rides whole cells.)
- **Scene 2 — the end.** `THE END` in the height-5 terminal-fonts banner,
  centered under the same sky.

```bash
cd screenplay-lab
go run .                # interactive
go run . -seconds 30    # auto-quit, handy for tapes
```

`space` cuts to the next scene (the final scene holds). `q` / `ctrl+c` quit.

## The shape

Every scene implements the same four verbs:

```go
type Scene interface {
    Start()             // allocate: the curtain rises
    Update(dt float64)  // advance internal clocks; no render data
    Render(scr *Screen) // write this instant's cells into the screen
    Stop()              // deallocate: the curtain falls
}
```

The `Screen` is the shared render target: one Lip Gloss v2 canvas — a
`uv.Cell` of content plus style (foreground, background, attributes) per
terminal cell, everything lip gloss needs to draw it — plus `Resize` /
`Resized` bookkeeping so a terminal change repaints everything. The
screenplay owns the frame loop: it hands the same screen pointer to the
current scene's `Render` after clearing it, updates only the scene now
playing (so each scene's clocks start at its cut), and `Next()` stops the
old scene before starting the new. The TUI then puts `screen.Render()` on
the terminal.

| Package | What it owns |
| --- | --- |
| `screenplay` | `Screen` (the lip gloss cell canvas + frame bookkeeping), `Scene` (the four-verb interface), `Ensemble` (the common scene shape: a cast of `Actor`s assembled at curtain, dropped at Stop), `Screenplay` (the lifecycle cursor) |
| `cast` | The troupe: `Ship` (atlas hull + slimmed booster plume + `FlightPath` choreography), `Starfield` (wraps `stars.Field`), `Title` (wraps `termfont`), and the blit bridge from the labs' xterm-256 sprites onto screen cells |

The `screenplay` package knows nothing about the lander, stars, or fonts —
it speaks lip gloss cells only. The labs' art crosses over in one place,
`cast/blit.go`.

## Reuse, not rewrites

The ship is composed entirely from existing parts: the size-4 `W` frame
(`sprite.Default()`), its baked tilde plume stripped, over a `fire`/
`particle` booster aimed `{X:1, Y:0}` — left-to-right fire out the tail
while the craft flies west. Nothing in `lander-lab`, `stars-lab`, or
`terminal-fonts` changed to put this on stage.
