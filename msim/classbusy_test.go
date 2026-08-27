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
