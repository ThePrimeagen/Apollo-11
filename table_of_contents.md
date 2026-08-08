# Apollo 11 Alarm Investigation — Reading Order

Start here. The investigation is deliberately split into three causal chains so you can learn
one mechanism at a time instead of jumping among unrelated files.

## Required reading

### 1. Definitions first

Read [`definitions.md`](definitions.md).

It defines:

- VAC areas and core sets;
- jobs vs. tasks;
- the Executive and WAITLIST schedulers;
- the radar/CDU hardware;
- every AGC instruction and assembler directive encountered;
- important fixed addresses and constants;
- octal notation and ones'-complement minus zero.

Do not skip this file if AGC assembly is new to you.

### 2. The initiating cause: rendezvous-radar time theft

Read [`radar_problem.md`](radar_problem.md).

Then run:

```vim
:luafile radar_problem.lua
```

This four-location tour explains:

1. the hardware control bits;
2. the radar-mode monitor that could not see the electrical phase fault;
3. normal radar zeroing being removed;
4. the two hardware counters that received up to 12,800 bogus requests per second.

This is the **rendezvous radar**, not the voice radio. The later Houston conversation reports
the consequences; it does not cause them.

### 3. The amplification: unfinished-job memory leak

Read [`memory_leak.md`](memory_leak.md).

Then run:

```vim
:luafile memory_leak.lua
```

This nine-location tour explains:

1. the fixed 2-second schedule;
2. the timer dispatch;
3. `READACCS`;
4. allocation of another `SERVICER`;
5. VAC claim;
6. core-set claim;
7. the long, low-priority job;
8. its intended finish;
9. the release it fails to reach before another copy is created.

### 4. The response: detect exhaustion, report, restart, recover

Read [`alarm_recovery.md`](alarm_recovery.md).

Then run:

```vim
:luafile alarm_recovery.lua
```

This twelve-location tour explains:

1. the alternative 1201 and 1202 resource scans;
2. proof that no compatible memory remains;
3. selection and storage of the alarm code;
4. lighting the DSKY PROG lamp;
5. software restart;
6. queue/resource cleanup;
7. phase-table rebuild;
8. sensor re-read and continuation.

The first four markers include **two alternatives**. A failed allocation follows either the
1201 VAC branch or the 1202 core-set branch, then both join at `BAILOUT1`.

## Recommended supporting reading

- [`walkthrough.md`](walkthrough.md) — the entire story in plain English.
- [`timeline.markdown`](timeline.markdown) — exact mission times, altitudes, and voice calls.
- [`errorcodes.markdown`](errorcodes.markdown) — the older single-document deep technical
  treatment; useful as a consolidated reference after the three focused parts.
- [`outline.markdown`](outline.markdown) — orientation to the whole Apollo-11 repository.

## Source marker commands

```bash
grep -rn "RADAR_PROBLEM[0-9]" Luminary099/*.agc
grep -rn "MEMORY_LEAK[0-9]" Luminary099/*.agc
grep -rn "ALARM_RECOVERY[0-9]" Luminary099/*.agc
```

Each marker family has its own numbering and its own Vim quickfix file. There is no combined
tour, so one quickfix entry always represents one precise source location.
