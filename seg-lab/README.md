# seg-lab

Standalone segmented letter viewer. **Not wired into the DSKY** — run it
by itself.

```bash
cd seg-lab
go run .
```

`tab` cycles styles. Type to edit. `esc` clears to the A–Z catalog.
`ctrl-c` quits. `q` is a letter.

## What Unicode actually has

Unicode 13 added **ten segmented digits** in Symbols for Legacy Computing,
and nothing else:

| Codepoints | Glyphs |
| --- | --- |
| `U+1FBF0`–`U+1FBF9` | 🯰🯱🯲🯳🯴🯵🯶🯷🯸🯹 |

There are no segmented letter codepoints. A 7-segment or 14-segment "A"
has to be composed from `_` / `|` / `─` / `│` / `╱` / `╲`.

## Styles

| `tab` | How it draws | Letters |
| --- | --- | --- |
| `unicode` | Official `U+1FBF0`–`U+1FBF9` cells | Blank — no codepoints |
| `7-seg` | 3×3 `_`/`\|` strokes (same language as the DSKY) | A b C d E F … — not K/M/V/W/X |
| `14-seg` | 5×5 box-drawing | Full A–Z |

The component is `seg.Render(text, style)`. Pure. No TUI imports.
