# timeline-tui

Rose Pine TUI for the Apollo 11 MEMORY_LEAK story (`timeline.md` / `timeline.c`).

- **500ms fade** from black into Rose Pine
- **Left:** erasable board (cores + VACs) + one 2.00s cycle time bars
- **Right:** simplified C for the current leak step
- **Step** through events that claim unreclaimed core sets / VACs
- **Play** one guidance cycle: SERVICER needs **1.80s**, only gets **~1.20s**, with radar as the big unexpected steal and MONDO as a smaller tip

## Run

```bash
cd timeline-tui
go run .
# or
go build -o timeline-tui . && ./timeline-tui
```

## Keys

| Key | Action |
| :--- | :--- |
| `h` | Toggle **HEALTHY** ↔ **OVERLOAD** scenario |
| `n` / `→` / `Space` / `Enter` | Next step (mutates the board) |
| `b` / `←` | Previous step |
| `p` | Play the 2.00s cycle for the current scenario |
| `j` / `k` | Highlight each job; right pane shows a short explanation |
| `esc` | Leave job browse → back to step C code |
| `r` | Reset current scenario |
| `q` | Quit |

## Scenarios

**HEALTHY** (`h`): no RR theft, no V16 N68. SERVICER gets the full **1.80s**, reaches `end_of_job()`, pool stays free.

**OVERLOAD** (default): RR ~**0.30s** (the unexpected steal) + MONDO tip ~0.12s + LR/… leave SERVICER ~**1.20s** of a 1.80s need → miss finish → leak → 1201.
