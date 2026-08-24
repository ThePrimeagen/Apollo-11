# Terminal Fonts

Multi-height banner fonts for the TUI, segment-display flavored — something
near a 14-segment readout drawn in plain ASCII. Pick a height from 1 to 5
and get back a byte buffer you can blit anywhere inside your own frame data.

```bash
cd terminal-fonts
go run .                     # A-Z + seven-segment digit catalogs, every height
go run . -height 3           # one height only
go run . -seven              # seven-segment digits one through zero
go run . -text "APOLLO 11"   # custom banner text
go run . -text 1969 -seven   # custom seven-segment text
```

## The contract

```go
import "github.com/theprimeagen/apollo-11/terminal-fonts/termfont"

buf, width, err := termfont.Render(3, "APOLLO 11")
```

`Render(height, text)` returns:

- **`buf []byte`** — exactly `height*width` bytes, row-major, space padded,
  **no newlines**. Row `r` is `buf[r*width : (r+1)*width]`, so you can copy
  each row straight into any destination grid at any offset.
- **`width int`** — the width of each row in bytes (all output is printable
  ASCII, so bytes == terminal columns).
- **`err error`** — `nil`, or one of the sentinels below.

`Lines(height, text)` is the convenience view: the same buffer split into
`height` strings.

## Heights

| Height | What you get |
| --- | --- |
| 1 | The terminal font itself — the characters pass through untouched |
| 2 | Two-row ASCII art (compact, `~` overlines stand in for top bars) |
| 3 | Classic segment look: top bar, mid bar, bottom bar |
| 4 | Stretched segments, crossbars centered |
| 5 | Fully stretched 14-segment-style letterforms |
| anything else | `ErrInvalidHeight` — nothing beyond five |

Height 3, the reference cut:

```text
 _   _   _   _   _   _   _      ___  __                    _   _   _   _   _  ___                      ___
|_| |_) |   | \ |_  |_  | _ |_|  |    | |_/ |   |\/| |\ | | | |_) | | |_) |_   |  | | \ / |  | \ / \ /  / 
| | |_) |_  |_/ |_  |   |_| | | _|_ |_| | \ |_  |  | | \| |_| |   |_\ | \  _|  |  |_|  V  |/\| / \  |  /__
```

## Seven-segment numbers

`RenderSeven(height, text)` and `LinesSeven(height, text)` carry the exact
same buffer contract, but draw true seven-segment shapes: straight `_` and
`|` segments only — no diagonals, no slashed zero, no slanted one — and
all ten digits share one display cell width. The charset is what a real
display can show, `termfont.SevenCharset`: `0-9`, space, and `. : -`.
Height 1 prints the characters plainly, but letters are rejected at every
height — a numeric display shows no alphabet. Height 2 has no room for a
top-bar row, so 0 and 7 carry a `~` overline to stay unambiguous.

```text
     _   _       _   _   _   _   _   _
  |  _|  _| |_| |_  |_    | |_| |_| | |
  | |_   _|   |  _| |_|   | |_|  _| |_|
```

## Charset and errors

Heights 2-5 draw `termfont.Charset`: `A-Z`, `0-9`, space, and
`. , : ; ! ? ' " - + = ( ) / \ _`. Lowercase folds onto the uppercase
glyphs. Zero is slashed so it never reads as the letter O. Height 1 is
broader — any printable ASCII passes through, case preserved.

Failures are sentinel-wrapped, so plan for them:

- `errors.Is(err, termfont.ErrInvalidHeight)` — height outside 1..5.
- `errors.Is(err, termfont.ErrUnsupportedRune)` — the message names the
  offending rune and its index. On error the buffer is `nil`, width `0`;
  nothing partial ever comes back.

Empty text is not an error: a zero-width, zero-length buffer.

## How to integrate (exec-tui, anything)

The package is pure and dependency-free — no Bubble Tea imports, no ANSI
inside the buffer. In the consumer module:

```text
require github.com/theprimeagen/apollo-11/terminal-fonts v0.0.0
replace github.com/theprimeagen/apollo-11/terminal-fonts => ../terminal-fonts
```

Then blit rows into your cell grid:

```go
buf, width, err := termfont.Render(4, "GO")
if err != nil {
    // handle: bad height or a rune the font cannot draw
}
for r := 0; r < 4; r++ {
    copy(grid[y+r][x:], buf[r*width:(r+1)*width])
}
```

Not wired into `exec-tui` today — that is intentional; this lab stands
alone like the others.
