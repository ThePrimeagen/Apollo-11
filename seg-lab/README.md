# seg-lab

Standalone segmented letter viewer. **Not wired into the DSKY** — run it
by itself.

```bash
cd seg-lab
go run .
```

Pass a string and a height. Height is the number of terminal rows.
Height 1 is the terminal's default font. Heights 3–5 are constructed
14-seg. Height 2 is not possible (a 14-seg letter needs a top, mid,
and bottom) and returns `ErrHeight`, as does anything else outside
1, 3, 4, 5.

```go
font.Render("HELLO WORLD", 1) // default font
font.Render("HELLO WORLD", 3)
font.Render("HELLO WORLD", 4)
font.Render("HELLO WORLD", 5)
```

`tab` cycles 1→3→4→5→1. Digits `1`, `3`, `4`, `5` set the unit. Type
to edit. `esc` clears to the A–Z catalog. `ctrl-c` quits. `q` is a
letter.

| height | what |
|--------|------|
| 1      | terminal default font (1 row) |
| 2      | not supported |
| 3      | constructed 14-seg, 5×3 |
| 4      | constructed 14-seg, 7×4 |
| 5      | constructed 14-seg, 7×5 |
