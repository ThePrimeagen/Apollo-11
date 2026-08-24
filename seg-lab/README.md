# seg-lab

Standalone segmented letter viewer. **Not wired into the DSKY** — run it
by itself.

```bash
cd seg-lab
go run .
```

The writing is the `font` package. Pass a string and a size:

```go
font.Render("HELLO WORLD", font.Small)
font.Render("HELLO WORLD", font.Large)
```

`tab` flips small ↔ large. Type to edit. `esc` clears to the A–Z catalog.
`ctrl-c` quits. `q` is a letter.

There is no Python and no TTF. Unicode only has segmented digits
(`U+1FBF0`–`U+1FBF9`); letters have to be drawn. `font` stamps the same
14-segment outlines the old TTF used onto a character grid (filled `█▀▄`
bars, gapped joints). Small is 7×5; large is 13×9. A terminal cell is
taller than it is wide, so the grid is wider than it is tall — a square
`█` grid is what made the first Go port look like skinny sticks.
