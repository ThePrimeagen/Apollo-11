Every change requires a video attached to it if there's any sort of TUI change.

# Agent instructions

- The video must show the changed TUI actually running — record the real
  terminal in real time (no time-compressed captures), walk every scene or
  state the change touches, and attach it to the change (the PR and the
  summary) before calling the work done.
- Tests are written first, committed red, then made green. Every test
  covers a happy and an unhappy path. See the git history for the pattern.
- Never clamp knob or config values, for any reason. If the operator
  wants a negative size, a speed of 400, or a count of zero, that is
  the value. Nudge steps; it does not enforce floors or ceilings. Do
  not rewrite a stored value to a "sane" default.
- No classes. State is a nicely typed plain object made by a small
  constructor function; behavior is exported functions that take that
  object. Guard preconditions with `assert(value, message)` — it
  validates a truthy or throws. Internal callback plumbing (wiring a
  response back to its request) stays module-private and is not
  tested directly.
