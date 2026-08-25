# screenplay-lab

A screenplay is scenes in order; a scene is a cast of sprites playing over
time. `space` cuts to the next scene. The lab premieres a two-scene bill:

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

## The pieces

| Package | What it owns |
| --- | --- |
| `screenplay` | `Stage` (compose sprites, clip at edges), `Actor` (Advance/Paint), `Scene` (a named cast; cast order is paint order), `Screenplay` (the scene cursor; `Next()` cuts, time reaches only the current scene) |
| `cast` | The troupe: `Ship` (atlas hull + slimmed booster plume + `FlightPath` choreography), `Starfield` (wraps `stars.Field`), `Title` (wraps `termfont`) |

The `screenplay` package knows nothing about the lander, stars, or fonts —
actors adapt those labs to the two-method `Actor` face. New scenes are new
casts; new performers are two methods.

## Reuse, not rewrites

The ship is composed entirely from existing parts: the size-4 `W` frame
(`sprite.Default()`), with its baked tilde plume stripped the same way
`components/rocket` does, over a `fire`/`particle` booster aimed `{X:1, Y:0}`
— left-to-right fire out the tail while the craft flies west. Nothing in
`lander-lab`, `stars-lab`, or `terminal-fonts` changed to put this on stage.
