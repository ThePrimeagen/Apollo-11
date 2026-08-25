# stars-lab

Standalone one-cell starfield. **Not wired into the lander or exec-tui** — run
it by itself, then paint it behind whatever you want.

```bash
cd stars-lab
go run .                         # dust-rush (flying)
go run . -strategy still         # stars hold still
```

`n` / `p` cycle styles. `space` pauses. `q` quits.

## Styles

| `-strategy` | Motion |
| --- | --- |
| `dust-rush` (default) | · and ˚ streak right→left; * and ✦ almost hang |
| `still` | no motion — a static night sky |
| `far-fast` / `near-fast` / `uniform` / `uniform-slow` / `hyperspace` / `drift` | other fly takes |

Glyphs: `·` dust, `˚` spark, `*` mid, `✦` near. Colors are white with a little
blue-white and red-white. `*` / `✦` are sparse (~25% of the dust layers).

`Field.Density` is the per-layer frequency knob: stars per 1000 cells for
dust, spark, mid, near. The zero value paints the stock sky
(`DefaultDensity`, `{56, 33, 6, 4}`); larger numbers thicken a layer,
capped at `MaxDensity` (400). Speed stays on `Strategy.Delay` — ticks per
cell of travel, lower is faster.

`SkyConfig` is the file config, the same shape the fire's heat ladder
uses: a JSON of the eight knobs (`delay` + `density` per layer) with
`LoadSky` / `Save` / `Validate`, and a package-active setting —
`UseSky` / `ActiveSky` / `ResetSky` — that consumers (like the
screenplay-lab premiere and its `adjuststars` tuner) read so a tuned file
just works.

## How to integrate (lander, exec-tui, anything)

The component has **no** lander/DSKY imports. Paint it **first** so everything
else overwrites the sky.

1. In the consumer module (`lander-lab/go.mod`, `exec-tui/go.mod`):

```
require github.com/theprimeagen/apollo-11/stars-lab v0.0.0
replace github.com/theprimeagen/apollo-11/stars-lab => ../stars-lab
```

2. Import `"github.com/theprimeagen/apollo-11/stars-lab/stars"`.

3. After you allocate an empty cell grid, **before** header / craft / surface:

```go
stars.Field{
    Width:    width,   // same as the destination grid
    Height:   height,
    Tick:     frame,   // ignored when Strategy is stars.Still
    Strategy: stars.DustRush, // or stars.Still
    Frozen:   false,   // true also freezes any flying style
}.Paint(func(row, col int, ch rune, fg int) {
    grid[row][col] = cell{ch, fg} // your cell type
})
```

4. Then draw the lander, DSKY, captions on top as usual. Occupied cells
   replace stars. Empty cells keep the sky.

`Paint` is pure. `Render()` is the ANSI string the standalone TUI uses — you
don't need it if you already have a grid.

Not wired today: `lander-lab/lander.Render` and `exec-tui` do not import this
package. That is intentional.
