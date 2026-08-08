-- Part 2: unfinished-SERVICER resource-leak tour.
-- Run from the repository root with :luafile memory_leak.lua
-- Navigate with :cnext / :cprev.

local items = {
  {
    filename = 'Luminary099/SERVICER.agc',
    lnum = 80,
    col = 1,
    text = '01 [MEMORY_LEAK1] GOREADAX: unconditionally schedule READACCS exactly 2.00 seconds later; demand never waits for old work.',
  },
  {
    filename = 'Luminary099/WAITLIST.agc',
    lnum = 388,
    col = 1,
    text = '02 [MEMORY_LEAK2] T3RUPT: hardware timer dispatches punctually even though radar time theft slowed ordinary jobs.',
  },
  {
    filename = 'Luminary099/SERVICER.agc',
    lnum = 95,
    col = 1,
    text = '03 [MEMORY_LEAK3] READACCS: capture accelerometer data and begin another guidance cycle without checking older SERVICER copies.',
  },
  {
    filename = 'Luminary099/SERVICER.agc',
    lnum = 121,
    col = 1,
    text = '04 [MEMORY_LEAK4] TC FINDVAC: allocate a brand-new priority-20 SERVICER with its own VAC area and core set.',
  },
  {
    filename = 'Luminary099/EXECUTIVE.agc',
    lnum = 170,
    col = 1,
    text = '05 [MEMORY_LEAK5] VACFOUND: mark one 44-word Vector Accumulator area busy for this new SERVICER copy.',
  },
  {
    filename = 'Luminary099/EXECUTIVE.agc',
    lnum = 202,
    col = 1,
    text = '06 [MEMORY_LEAK6] CORFOUND: write the job priority and mark one 12-word core set busy.',
  },
  {
    filename = 'Luminary099/SERVICER.agc',
    lnum = 206,
    col = 1,
    text = '07 [MEMORY_LEAK7] SERVICER begins: long priority-20 navigation/guidance job, repeatedly preempted by higher-priority work.',
  },
  {
    filename = 'Luminary099/SERVICER.agc',
    lnum = 482,
    col = 1,
    text = '08 [MEMORY_LEAK8] TCF ENDOFJOB: intended finish; an overrun copy has not reached this before the next copy is allocated.',
  },
  {
    filename = 'Luminary099/EXECUTIVE.agc',
    lnum = 420,
    col = 1,
    text = '09 [MEMORY_LEAK9] ENDJOB1: actual core/VAC release. If an old copy never gets here in time, its resources remain claimed.',
  },
}

vim.fn.setqflist({}, ' ', {
  items = items,
  title = 'Apollo 11 Part 2 - Unfinished-Job Memory Leak',
})

vim.cmd.copen()
