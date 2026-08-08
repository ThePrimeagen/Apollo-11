-- Part 1: rendezvous-radar time-theft tour.
-- Run from the repository root with :luafile radar_problem.lua
-- Navigate with :cnext / :cprev.

local items = {
  {
    filename = 'Luminary099/INPUT_OUTPUT_CHANNEL_BIT_DESCRIPTIONS.agc',
    lnum = 84,
    col = 1,
    text = '01 [RADAR_PROBLEM1] Channel 12 bits: software can ZERO the RR CDUs or ENABLE their error counters; individual pulses are external hardware.',
  },
  {
    filename = 'Luminary099/T4RUPT_PROGRAM.agc',
    lnum = 1121,
    col = 1,
    text = '02 [RADAR_PROBLEM2] RRAUTCHK sees RR AUTO/POWER state, but cannot see the dangerous phase difference between the two 800-Hz references.',
  },
  {
    filename = 'Luminary099/P20-P25.agc',
    lnum = 1693,
    col = 1,
    text = '03 [RADAR_PROBLEM3] RRZEROSB performs normal startup and removes CDU zeroing, allowing external ECDUs to update the angle counters.',
  },
  {
    filename = 'Luminary099/ERASABLE_ASSIGNMENTS.agc',
    lnum = 131,
    col = 1,
    text = '04 [RADAR_PROBLEM4] CDUT/CDUS: the hardware counters hit by up to 12,800 PINC/MINC requests/s, stealing about 15% of CPU time.',
  },
}

vim.fn.setqflist({}, ' ', {
  items = items,
  title = 'Apollo 11 Part 1 - Rendezvous-Radar Time Theft',
})

vim.cmd.copen()
