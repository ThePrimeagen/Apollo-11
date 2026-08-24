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

There is no Python and no TTF. Letters are 14-segment box-drawing in Go.
Unicode only has segmented digits (`U+1FBF0`–`U+1FBF9`); that is why this
package exists.
