Every change requires a video attached to it if there's any sort of TUI change.

# Agent instructions

- The video must show the changed TUI actually running — record the real
  terminal in real time (no time-compressed captures), walk every scene or
  state the change touches, and attach it to the change (the PR and the
  summary) before calling the work done.
- Tests are written first, committed red, then made green. Every test
  covers a happy and an unhappy path. See the git history for the pattern.
