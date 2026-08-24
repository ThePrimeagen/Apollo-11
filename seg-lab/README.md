# seg-lab

Standalone segmented letter viewer. **Not wired into the DSKY** — run it
by itself.

```bash
cd seg-lab
go run .
```

Pass a string and a height unit 1–5. Height 1 is the terminal's default
font. Heights 2–5 are constructed 14-seg. Anything outside 1–5 returns
`ErrHeight`.

```go
font.Render("HELLO WORLD", 1) // default font
font.Render("HELLO WORLD", 2)
font.Render("HELLO WORLD", 3)
font.Render("HELLO WORLD", 4)
font.Render("HELLO WORLD", 5)
```

`tab` cycles 1→2→3→4→5→1. Digits `1`–`5` set the unit. Type to edit.
`esc` clears to the A–Z catalog. `ctrl-c` quits. `q` is a letter.

| height | what |
|--------|------|
| 1      | terminal default font (1×1) |
| 2      | constructed 14-seg, 7×5 |
| 3      | constructed 14-seg, 10×7 |
| 4      | constructed 14-seg, 13×9 |
| 5      | constructed 14-seg, 16×11 |
