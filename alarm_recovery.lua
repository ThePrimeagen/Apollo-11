-- Part 3: resource-exhaustion detection, alarm reporting, restart, recovery.
-- Run from the repository root with :luafile alarm_recovery.lua
-- Navigate with :cnext / :cprev.
--
-- Steps 1-2 and 3-4 are alternatives:
--   VAC exhaustion: 1 -> 2 -> 5
--   Core exhaustion: 3 -> 4 -> 5
-- Both branches share steps 5-12.

local items = {
  {
    filename = 'Luminary099/EXECUTIVE.agc',
    lnum = 141,
    col = 1,
    text = '01 [ALARM_RECOVERY1 / VAC branch] FINDVAC2 scans all five VAC use-words; positive means free.',
  },
  {
    filename = 'Luminary099/EXECUTIVE.agc',
    lnum = 161,
    col = 1,
    text = '02 [ALARM_RECOVERY2 / VAC branch] OCT 1201: all five VAC areas are busy; call BAILOUT1 with inline NO VAC AREAS code.',
  },
  {
    filename = 'Luminary099/EXECUTIVE.agc',
    lnum = 183,
    col = 1,
    text = '03 [ALARM_RECOVERY3 / core branch] NOVAC2/NOVAC3 scans eight core-set PRIORITY words; minus zero means free.',
  },
  {
    filename = 'Luminary099/EXECUTIVE.agc',
    lnum = 249,
    col = 1,
    text = '04 [ALARM_RECOVERY4 / core branch] OCT 1202: all eight core sets are busy; call BAILOUT1 with inline NO CORE SETS code.',
  },
  {
    filename = 'Luminary099/ALARM_AND_ABORT.agc',
    lnum = 221,
    col = 1,
    text = '05 [ALARM_RECOVERY5 / shared] BAILOUT1 reads the caller inline OCT 1201 or OCT 1202 word and passes it to CHKFAIL1.',
  },
  {
    filename = 'Luminary099/ALARM_AND_ABORT.agc',
    lnum = 74,
    col = 1,
    text = '06 [ALARM_RECOVERY6] CHKFAIL1 stores the alarm in the first free FAILREG slot for crew display/diagnosis.',
  },
  {
    filename = 'Luminary099/ALARM_AND_ABORT.agc',
    lnum = 100,
    col = 1,
    text = '07 [ALARM_RECOVERY7] PROGLARM sets the DSKY lamp-table bit that lights the yellow PROG warning lamp.',
  },
  {
    filename = 'Luminary099/ALARM_AND_ABORT.agc',
    lnum = 167,
    col = 1,
    text = '08 [ALARM_RECOVERY8] WHIMPER transfers to ENEMA and deliberately starts a software restart.',
  },
  {
    filename = 'Luminary099/FRESH_START_AND_RESTART.agc',
    lnum = 290,
    col = 1,
    text = '09 [ALARM_RECOVERY9] ENEMA enters restart code while preserving engine, IMU, and gyro flight state.',
  },
  {
    filename = 'Luminary099/FRESH_START_AND_RESTART.agc',
    lnum = 525,
    col = 1,
    text = '10 [ALARM_RECOVERY10] Clear scheduler queues, all eight core sets, and all five VAC areas; accumulated stubs vanish.',
  },
  {
    filename = 'Luminary099/RESTART_TABLES.agc',
    lnum = 267,
    col = 1,
    text = '11 [ALARM_RECOVERY11] 5.4SPOT rebuilds exactly one REREADAC timer task and one current SERVICER job.',
  },
  {
    filename = 'Luminary099/SERVICER.agc',
    lnum = 639,
    col = 1,
    text = '12 [ALARM_RECOVERY12] REREADAC captures PIPA counts accumulated through restart; navigation continues with no sensor-data loss.',
  },
}

vim.fn.setqflist({}, ' ', {
  items = items,
  title = 'Apollo 11 Part 3 - Detect, Report, Restart, Recover',
})

vim.cmd.copen()
