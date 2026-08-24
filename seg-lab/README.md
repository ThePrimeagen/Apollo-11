# seg-lab

Standalone segmented letter viewer. **Not wired into the DSKY** — run it
by itself.

```bash
cd seg-lab
go run .
```

The writing is the `font` package. Pass a string and a height unit 1–5:

```go
font.Render("HELLO WORLD", 1)
font.Render("HELLO WORLD", 3)
font.Render("HELLO WORLD", 5)
```

`tab` cycles 1→2→3→4→5→1. Digits `1`–`5` set the unit. Type to edit.
`esc` clears to the A–Z catalog. `ctrl-c` quits. `q` is a letter.

There is no Python and no TTF. Unicode only has segmented digits
(`U+1FBF0`–`U+1FBF9`); letters have to be drawn. `font` stamps the same
14-segment outlines the old TTF used onto a character grid (filled `█▀▄`
bars, gapped joints). A terminal cell is taller than it is wide, so each
unit is wider than it is tall.

| height | cell (cols × rows) |
|--------|--------------------|
| 1      | 5 × 3              |
| 2      | 7 × 5              |
| 3      | 10 × 7             |
| 4      | 13 × 9             |
| 5      | 16 × 11            |
