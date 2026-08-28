// Package msim is an instruction-level, single-CPU simulator of the Apollo 11
// LGC Executive during P63 powered descent. The clock advances 100 µs at a
// time; within each tick the CPU is divided, in order, between the
// rendezvous-radar counter theft (hardware, invisible), due interrupts and
// waitlist tasks (which pause the running job mid-instruction), and the
// highest-priority ready job. Jobs are preempted only between instructions —
// the DANZIG boundary (INTERPRETER.agc L74-L82, where NEWJOB is tested).
package msim

// Nanos is AGC time in nanoseconds since the start of the run.
type Nanos = int64

const (
	Microsecond Nanos = 1_000
	Millisecond Nanos = 1_000_000
	Second      Nanos = 1_000_000_000

	// MCT is the AGC memory cycle time: 11.72 us.
	MCT Nanos = 11_720

	// TheftMaxPerMs is the RR CDU bug's theoretical ceiling per wall
	// millisecond: 12,800 PINC/MINC per second x 11.72 us = 150,016 ns
	// (6,400 pulses/s per angle, two angles: CDUS and CDUT).
	// TheftMinPerMs is Grumman's measured floor (~12.8%). The actual skim
	// sweeps deterministically between them (see Engine.theftAtMs).
	TheftMaxPerMs Nanos = 150_016
	TheftMinPerMs Nanos = 128_000

	// SleepUntilWake marks an instruction whose JOBSLEEP has no hardware
	// timer — only an explicit JOBWAKE releases it.
	SleepUntilWake Nanos = -1
)

// Instr is one interpretive instruction (or one short basic block): the unit
// between two DANZIG visits. Cost is pure CPU time; interrupts stretch it.
type Instr struct {
	Section string // e.g. "MUNRVG"
	Ref     string // e.g. "SERVICER.agc:1086"
	Op      string // e.g. "VLOAD"
	Cost    Nanos
	// SleepNs > 0: after this instruction the job calls JOBSLEEP and a
	// hardware event wakes it SleepNs later. SleepUntilWake: only Wake().
	SleepNs Nanos
	// Then runs at the instruction's completion (the DANZIG boundary) —
	// used for job spawns (DISPEXIT, 1/PIPA's gyro gate) and probes.
	Then func(*Engine)
}

// Script is a job body: instructions executed in order.
type Script []Instr

// Total is the script's pure-CPU cost.
func (s Script) Total() Nanos {
	var t Nanos
	for _, in := range s {
		t += in.Cost
	}
	return t
}

// JobSpec describes a job to be entered into the Executive.
type JobSpec struct {
	Name   string
	Prio   int
	VAC    bool // FINDVAC (core set + VAC area) vs NOVAC (core set only)
	Script Script
}

// Alarm is a failed allocation: 1201 (no VAC areas) or 1202 (no core sets).
// The pool counts are the snapshot at the failing request's entry.
type Alarm struct {
	Code      int
	At        Nanos
	Requester string
	CoresHeld int
	VACsHeld  int
}

// Event is one timeline entry.
type Event struct {
	At     Nanos
	Kind   string // "spawn", "sleep", "wake", "alarm", "restart", "note"
	Job    string
	Prio   int
	Detail string
}

// Sample is the per-millisecond occupancy record. The three class fields
// split that millisecond's software CPU by what the consumer held: a VAC
// area (VacNs), a core set only (CoreNs), or nothing — the task/interrupt
// layer (OpsNs). Theft belongs to no class.
type Sample struct {
	AtMs    int
	Cores   int
	VACs    int
	Running string
	VacNs   Nanos
	CoreNs  Nanos
	OpsNs   Nanos
	// ByName splits the same millisecond's software CPU by consumer name —
	// jobs via the runner, tasks and interrupts via their activity. Nil on
	// a millisecond nobody ran in; the theft is nameless hardware.
	ByName map[string]Nanos
}

// JobState is the scheduler-visible state of a named job.
type JobState int

const (
	JobUnknown  JobState = iota
	JobWaiting           // allocated, never yet run
	JobRunning           // in slot 0, executing
	JobParked            // preempted mid-run, holding its memory
	JobSleeping          // dormant (negative priority), holding its memory
	JobDone              // reached ENDOFJOB
)

func (s JobState) String() string {
	switch s {
	case JobWaiting:
		return "waiting"
	case JobRunning:
		return "running"
	case JobParked:
		return "parked"
	case JobSleeping:
		return "sleeping"
	case JobDone:
		return "done"
	}
	return "unknown"
}

// Phase selects a SERVICER script variant.
type Phase int

const (
	P63Prelock  Phase = iota // braking, before landing-radar lock
	P63Locked                // braking, radar locked: + the nav-frame conversion
	P64Approach              // approach: + the REDESIG landing-site perturbations
)

// Config wires an Engine.
type Config struct {
	RadarBug    bool // the RR CDU counter theft
	Interrupts  bool // DAP / T4RUPT / DOWNRUPT cadences
	AutoRestart bool // BAILOUT triggers the software restart
	// RestartHook mirrors the restart tables (RESTART_TABLES.agc 5.4SPOT):
	// invoked after the flush to rebuild the protected chains.
	RestartHook func(*Engine)
	// TheftPhaseMS offsets the theft sweep — the waveform's phase is one of
	// the run's free parameters (msim/RESEARCH.md). Zero is the flight
	// window: a floor dwell over the monitor keyings.
	TheftPhaseMS int
}
