# seg-lab

Standalone segmented letter viewer. **Not wired into the DSKY** — run it
by itself.

```bash
cd seg-lab
./run.sh          # loads the 14-seg alpha font, then the TUI
go run .          # same TUI; letters tofu unless the font is installed
```

`tab` cycles styles. Type to edit. `esc` clears to the A–Z catalog.
`ctrl-c` quits. `q` is a letter.

## What Unicode actually has

Unicode 13 added **ten segmented digits** and nothing else:

| Codepoints | Glyphs |
| --- | --- |
| `U+1FBF0`–`U+1FBF9` | 🯰🯱🯲🯳🯴🯵🯶🯷🯸🯹 |

There are no official segmented letter codepoints. Alpha is a 14-segment
font we ship (`font/SegmentedAlpha.ttf`) mapped onto the Private Use Area
`U+E000`–`U+E019` (A–Z). One cell per letter, same idea as the digits.

Regenerate the font:

```bash
python3 font/genfont.py
```

## Styles

| `tab` | How it draws | Letters |
| --- | --- | --- |
| `alpha` | One cell, 14-seg font at `U+E000`–`U+E019` | Full A–Z |
| `unicode` | Official `U+1FBF0`–`U+1FBF9` cells | Blank — no codepoints |
| `7-seg` | 3×3 `_`/`\|` strokes (same language as the DSKY) | A b C d E F … — not K/M/V/W/X |
| `14-seg` | 5×5 box-drawing, no font required | Full A–Z |

The component is `seg.Render(text, style)`. Pure. No TUI imports.
