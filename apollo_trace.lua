-- Apollo 11 guidance-computer 1201/1202 alarm trace, as a Vim quickfix "tour".
--
-- Open Neovim at the root of this repo and `:luafile apollo_trace.lua`
-- (or `:source` it). Step forward with :cnext, backward with :cprev, jump to
-- an entry from the quickfix window (:copen) with <CR>.
--
-- Line numbers match the annotated .agc files on this branch. Each step is a
-- SINGLE location, in true order of execution. The matching in-source comments
-- are tagged NOTE(...) (full trace) and MEMORY_LEAK... (leak-only trace); find
-- them with:  grep -rn "NOTE(" Luminary099/*.agc
--             grep -rn "MEMORY_LEAK[0-9]" Luminary099/*.agc
--
-- Companion docs: walkthrough.md (plain English), definitions.md (glossary),
-- errorcodes.markdown (deep code trace), timeline.markdown (minute-by-minute).

-- ========================================================================
-- TRACE 1 (default): the full story, radar theft -> alarm -> restart -> recovery.
-- 14 steps, one location each.
-- ========================================================================
local alarm_trace = {
  { filename = 'Luminary099/ERASABLE_ASSIGNMENTS.agc', lnum = 132, col = 1,
    text = '01 [ERAS1] Radar CDU counters CDUT/CDUS: hardware counters the rendezvous radar spams, stealing ~15% of the CPU. Root cause.' },
  { filename = 'Luminary099/WAITLIST.agc', lnum = 388, col = 1,
    text = '02 [WAIT2] T3RUPT: the 10ms hardware clock. Immune to the theft, so it keeps launching work punctually while jobs fall behind.' },
  { filename = 'Luminary099/SERVICER.agc', lnum = 97, col = 1,
    text = '03 [SERV3] READACCS: the 2-second task. Reads accelerometers, then schedules the big SERVICER job.' },
  { filename = 'Luminary099/SERVICER.agc', lnum = 134, col = 1,
    text = '04 [SERV4] TC FINDVAC: asks the Executive for a BRAND-NEW SERVICER every 2s, even if the last one has not finished. The leak.' },
  { filename = 'Luminary099/EXECUTIVE.agc', lnum = 167, col = 1,
    text = '05 [EXEC5] OCT 1201: FINDVAC2 scanned all 5 VAC areas, none free -> raises alarm 1201 (NO VAC AREAS).' },
  { filename = 'Luminary099/EXECUTIVE.agc', lnum = 253, col = 1,
    text = '06 [EXEC6] OCT 1202: NEXTCORE scanned all 8 core sets, none free -> raises alarm 1202 (NO CORE SETS).' },
  { filename = 'Luminary099/ALARM_AND_ABORT.agc', lnum = 221, col = 1,
    text = '07 [ALARM7] BAILOUT1: catches the alarm code (the OCT word after the caller TC), leaves it in L, jumps to CHKFAIL1.' },
  { filename = 'Luminary099/ALARM_AND_ABORT.agc', lnum = 74, col = 1,
    text = '08 [ALARM8] CHKFAIL1: stores the code in the first free FAILREG slot (up to 3 alarms remembered, read by V05 N09).' },
  { filename = 'Luminary099/ALARM_AND_ABORT.agc', lnum = 100, col = 1,
    text = '09 [ALARM9] PROGLARM: sets the alarm bit in DSPTAB+11D. THIS lights the yellow PROG lamp in front of Aldrin.' },
  { filename = 'Luminary099/ALARM_AND_ABORT.agc', lnum = 167, col = 1,
    text = '10 [ALARM10] WHIMPER: pulls the ripcord -- jumps (via POSTJUMP) to ENEMA to force a software restart.' },
  { filename = 'Luminary099/FRESH_START_AND_RESTART.agc', lnum = 290, col = 1,
    text = '11 [START11] ENEMA: software-restart entry. Re-inits the machine but leaves engine, IMU and gyros flying.' },
  { filename = 'Luminary099/FRESH_START_AND_RESTART.agc', lnum = 525, col = 1,
    text = '12 [START12] Free all register sets: wipes the queues and frees all 8 core sets + 5 VAC areas. Every leaked stub dies here.' },
  { filename = 'Luminary099/RESTART_TABLES.agc', lnum = 267, col = 1,
    text = '13 [RSTAB13] 5.4SPOT: the rebuild recipe. Re-creates exactly ONE REREADAC task + ONE SERVICER job. The backlog is gone.' },
  { filename = 'Luminary099/SERVICER.agc', lnum = 640, col = 1,
    text = '14 [SERV14] REREADAC: re-reads the accelerometer HW counters (which kept counting through the restart) -- no data lost. Eagle flies on.' },
}

-- ========================================================================
-- TRACE 2 (optional): ONLY the resource leak (jobs not finishing). 5 steps.
-- To follow this one instead, comment out the setqflist/copen for alarm_trace
-- below and uncomment the memory_leak block at the very bottom.
-- ========================================================================
local memory_leak = {
  { filename = 'Luminary099/SERVICER.agc', lnum = 81, col = 1,
    text = '01 [MEMORY_LEAK1] CA 2SECS / VARDELAY: re-arms READACCS every 2.00s unconditionally. Demand never slows down.' },
  { filename = 'Luminary099/SERVICER.agc', lnum = 134, col = 1,
    text = '02 [MEMORY_LEAK2] TC FINDVAC: allocates a fresh SERVICER (its own VAC area + core set) each cycle. Nothing reuses the old copy.' },
  { filename = 'Luminary099/EXECUTIVE.agc', lnum = 176, col = 1,
    text = '03 [MEMORY_LEAK3] VACFOUND: claims a VAC area (writes 0 into its use-flag). Held until ENDOFJOB.' },
  { filename = 'Luminary099/EXECUTIVE.agc', lnum = 212, col = 1,
    text = '04 [MEMORY_LEAK4] CORFOUND: claims a core set (writes a positive priority). Now the copy holds both resources.' },
  { filename = 'Luminary099/EXECUTIVE.agc', lnum = 127, col = 1,
    text = '05 [MEMORY_LEAK5] ENDOFJOB: the ONLY release (ENDJOB1 XCH PRIORITY). An unfinished SERVICER never gets here -> its memory leaks. Pool empties -> 1201/1202.' },
}

vim.fn.setqflist({}, ' ', {
  items = alarm_trace,
  title = 'Apollo 11 - 1201/1202 Alarm Trace (14 steps)',
})

vim.cmd.copen()

-- To follow ONLY the memory leak, comment the two lines above (setqflist/copen)
-- and uncomment these instead:
--
-- vim.fn.setqflist({}, ' ', {
--   items = memory_leak,
--   title = 'Apollo 11 - Memory Leak Trace (5 steps)',
-- })
-- vim.cmd.copen()
