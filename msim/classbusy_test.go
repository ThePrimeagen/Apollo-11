package msim

import (
	"testing"
)

// ---------- per-millisecond class-attributed busy time ----------
//
// The graphs screen needs to know, for every millisecond, how much CPU went
// to VAC-holding jobs, to coreset-only jobs, and to the no-priority layer
// (tasks & interrupts). The three fields ride on the per-ms Sample.

func TestClassBusySplitsAndSums(t *testing.T) {
	// happy: a VAC job, a NOVAC job and the hardware cadences each land in
	// their own bucket, and the buckets sum to the software ledger exactly
	e := NewEngine(Config{Interrupts: true})
	e.Spawn(JobSpec{Name: "V", Prio: 20, VAC: true, Script: Script{
		{Section: "V", Op: "VXV", Cost: 30 * Millisecond}}})
	e.ScheduleTask(40*Millisecond, "SP", 0, func(en *Engine) {
		en.Spawn(JobSpec{Name: "C", Prio: 20, VAC: false, Script: Script{
			{Section: "C", Op: "BASIC", Cost: 10 * Millisecond}}})
	})
	e.RunMS(100)

	var vac, core, ops Nanos
	for _, s := range e.Samples() {
		vac += s.VacNs
		core += s.CoreNs
		ops += s.OpsNs
		if s.VacNs < 0 || s.CoreNs < 0 || s.OpsNs < 0 {
			t.Fatalf("negative class time at ms %d: %+v", s.AtMs, s)
		}
		if s.VacNs+s.CoreNs+s.OpsNs > Millisecond {
			t.Fatalf("ms %d class time exceeds the millisecond: %+v", s.AtMs, s)
		}
	}
	if vac != 30*Millisecond {
		t.Fatalf("VAC class total = %d, want exactly 30 ms (job V)", vac)
	}
	if core != 10*Millisecond {
		t.Fatalf("CORE class total = %d, want exactly 10 ms (job C)", core)
	}
	if ops <= 0 {
		t.Fatalf("OPS class total = %d, want > 0 (the cadences)", ops)
	}
	if got := vac + core + ops; got != e.SoftwareBusyNs() {
		t.Fatalf("class totals %d != software ledger %d — every busy nanosecond has a class", got, e.SoftwareBusyNs())
	}
	// the VAC job's work must show in the VAC lane within its first few
	// milliseconds (ms 0 itself belongs to the t=0 interrupt costs)
	early := false
	for _, s := range e.Samples()[:5] {
		if s.VacNs > 0 {
			early = true
			break
		}
	}
	if !early {
		t.Fatalf("no VAC time in the first 5 ms while V was running: %+v", e.Samples()[:5])
	}
}

func TestBusyTotalsByName(t *testing.T) {
	// happy: every busy nanosecond lands on its consumer's name — jobs via
	// the runner, tasks/interrupts via the activity
	e := NewEngine(Config{Interrupts: true})
	e.Spawn(JobSpec{Name: "V", Prio: 20, VAC: true, Script: Script{
		{Section: "V", Op: "VXV", Cost: 30 * Millisecond}}})
	e.RunMS(100)
	if got := e.BusyNs("V"); got != 30*Millisecond {
		t.Fatalf("BusyNs(V) = %d, want exactly 30 ms", got)
	}
	if got, want := e.BusyNs("DAP"), Nanos(e.InterruptFires("DAP"))*12*Millisecond; got != want {
		t.Fatalf("BusyNs(DAP) = %d, want fires x 12 ms = %d", got, want)
	}
	// unhappy: unknown names report zero
	if got := e.BusyNs("NOBODY"); got != 0 {
		t.Fatalf("BusyNs(NOBODY) = %d, want 0", got)
	}
}

func TestClassBusyIdleAndTheftUncounted(t *testing.T) {
	// unhappy: an idle machine with the RR bug reports zero class time on
	// every millisecond — the theft is hardware, not software
	e := NewEngine(Config{RadarBug: true})
	e.RunMS(50)
	for _, s := range e.Samples() {
		if s.VacNs != 0 || s.CoreNs != 0 || s.OpsNs != 0 {
			t.Fatalf("idle ms %d carries class time: %+v", s.AtMs, s)
		}
	}
	if e.TheftNs() == 0 {
		t.Fatalf("theft should still accumulate on the idle machine")
	}
}

// ---------- per-millisecond per-consumer busy time ----------
//
// The graphs screen draws one lane per process, so every millisecond's
// Sample also carries WHO consumed it: Sample.ByName maps each consumer
// (jobs via the runner, tasks and interrupts via their activity) to its
// share of that millisecond.

func TestSampleByNameAttribution(t *testing.T) {
	// happy: a job, a waitlist task and the hardware cadences each land
	// per-millisecond under their own names; the per-name series sums to
	// the cumulative BusyNs ledger and to the class fields exactly
	e := NewEngine(Config{Interrupts: true})
	e.Spawn(JobSpec{Name: "V", Prio: 20, VAC: true, Script: Script{
		{Section: "V", Op: "VXV", Cost: 30 * Millisecond}}})
	e.ScheduleTask(40*Millisecond, "PING", 2*Millisecond, func(*Engine) {})
	e.RunMS(100)

	sums := map[string]Nanos{}
	for _, s := range e.Samples() {
		var total Nanos
		for name, ns := range s.ByName {
			if ns <= 0 {
				t.Fatalf("ms %d carries a non-positive ByName entry %q = %d", s.AtMs, name, ns)
			}
			sums[name] += ns
			total += ns
		}
		if total != s.VacNs+s.CoreNs+s.OpsNs {
			t.Fatalf("ms %d ByName sums to %d, class fields sum to %d — every named nanosecond has a class",
				s.AtMs, total, s.VacNs+s.CoreNs+s.OpsNs)
		}
	}
	for _, name := range []string{"V", "PING", "DAP", "T4RUPT", "DOWNRUPT"} {
		if sums[name] != e.BusyNs(name) {
			t.Fatalf("ByName total for %s = %d, BusyNs = %d — the series must telescope to the ledger",
				name, sums[name], e.BusyNs(name))
		}
	}
	if sums["V"] != 30*Millisecond {
		t.Fatalf("ByName total for V = %d, want exactly 30 ms", sums["V"])
	}
	if sums["PING"] != 2*Millisecond {
		t.Fatalf("ByName total for PING = %d, want exactly 2 ms", sums["PING"])
	}
}

func TestSampleByNameIdleAndUnknown(t *testing.T) {
	// unhappy: an idle machine's samples carry no ByName entries at all —
	// the theft is nameless hardware — and reading an unknown name is zero
	e := NewEngine(Config{RadarBug: true})
	e.RunMS(50)
	for _, s := range e.Samples() {
		if len(s.ByName) != 0 {
			t.Fatalf("idle ms %d carries ByName entries: %+v", s.AtMs, s.ByName)
		}
	}
	if got := e.Samples()[10].ByName["NOBODY"]; got != 0 {
		t.Fatalf("unknown name reads %d, want 0", got)
	}
}
