package main

import "fmt"

type jobKind string

const (
	kindHW    jobKind = "HW"
	kindNOVAC jobKind = "NOVAC"
	kindVAC   jobKind = "VAC"
)

type scenario int

const (
	scenarioOverload scenario = iota
	scenarioHealthy
)

type landingJob struct {
	Name     string
	Prio     int
	Kind     jobKind
	Cadence  string
	Explain  string
	ColorHex string
	// CPUSec depends on scenario — set via withCPU
	CPUSec float64
}

func jobCatalog() []landingJob {
	return []landingJob{
		{
			Name: "RR ECDU", Prio: 0, Kind: kindHW, ColorHex: "#eb6f92",
			Cadence: "PINC/MINC on CDUT + CDUS",
			Explain: "Rendezvous-radar angle counters. Not a software job — each bogus pulse steals one memory cycle (~11.72 µs). In the overload story the switch is AUTO/SLEW and both CDUs spam at max rate (~15% CPU). Healthy: radar not stealing (LGC mode / zeroed).",
		},
		{
			Name: "LRHJOB", Prio: 32, Kind: kindNOVAC, ColorHex: "#c4a7e7",
			Cadence: "~1ms run, ~80ms sleep (LR altitude)",
			Explain: "Landing-radar altitude read. Highest Executive priority (32). Tiny bursts: run a millisecond, sleep while the radar syncs, wake, finish. Preempts SERVICER constantly during descent.",
		},
		{
			Name: "LRVJOB", Prio: 32, Kind: kindNOVAC, ColorHex: "#c4a7e7",
			Cadence: "short run / ~500ms sleep (LR velocity)",
			Explain: "Landing-radar velocity beams. Also prio 32. Same pattern as LRHJOB but for velocity. Below ~15,000 ft these fire regularly and cut ahead of guidance.",
		},
		{
			Name: "HIGATJOB", Prio: 32, Kind: kindVAC, ColorHex: "#ebbcba",
			Cadence: "once near High Gate",
			Explain: "Repositions the landing-radar antenna to position 2 around High Gate (~7,400 ft). VAC job. Brief; mostly idle in early braking (P63).",
		},
		{
			Name: "CHARIN", Prio: 30, Kind: kindNOVAC, ColorHex: "#f6c177",
			Cadence: "each DSKY keystroke",
			Explain: "Keyboard handler. Every time an astronaut presses a DSKY key, CHARIN runs at prio 30 — above SERVICER. Short, but it interrupts guidance mid-cycle.",
		},
		{
			Name: "MONDO", Prio: 30, Kind: kindNOVAC, ColorHex: "#f6c177",
			Cadence: "while V16 N68 is up",
			Explain: "Monitor-display job for Verb 16 Noun 68 (DELTA-H). Aldrin keyed this during descent. It ate into an already-thin margin (~10% left) — the tip, not the main theft. The rendezvous-radar ECDUs stole ~15%; MONDO alone did not equal that.",
		},
		{
			Name: "1/GYRO", Prio: 21, Kind: kindNOVAC, ColorHex: "#31748f",
			Cadence: "IMU gyro compensation",
			Explain: "Keeps the IMU gyros compensated. Priority 21 — still above SERVICER (20). Recurring, modest CPU, always in the cast.",
		},
		{
			Name: "MAKEPLAY", Prio: 20, Kind: kindVAC, ColorHex: "#9ccfd8",
			Cadence: "display job from SERVICER",
			Explain: "Display helper set up by SERVICER. Same priority band (20), VAC job. Competes for the same leftover time and VAC pool as SERVICER.",
		},
		{
			Name: "SERVICER", Prio: 20, Kind: kindVAC, ColorHex: "#9ccfd8",
			Cadence: "every 2.00s via READACCS",
			Explain: "The big guidance job: average-G nav → guidance → throttle/DAP → displays. Lowest priority of the landing set, longest runtime. Needs ~1.80s of CPU per cycle. Healthy: finishes and frees its vac+core. Overload: ~1.20s leftovers (radar took 0.30; MONDO tip 0.12; rest background) → misses end_of_job → leak.",
		},
	}
}

// CPU seconds inside one 2.00s period. Must sum to ~2.00.
func cpuFor(sc scenario) []float64 {
	switch sc {
	case scenarioHealthy:
		// No RR theft, no MONDO. SERVICER gets a full 1.80s and finishes.
		return []float64{
			0.00, // RR ECDU
			0.05, // LRHJOB
			0.03, // LRVJOB
			0.00, // HIGATJOB
			0.02, // CHARIN
			0.00, // MONDO
			0.06, // 1/GYRO
			0.04, // MAKEPLAY
			1.80, // SERVICER — exact need → success
		}
	default: // overload
		// RR ECDU is the unexpected ~15% thief (commanding).
		// MONDO (V16 N68) is only the tip that ate remaining margin — not equal to radar.
		// Other landing jobs are normal background traffic.
		return []float64{
			0.30, // RR ECDU  ← the problem
			0.08, // LRHJOB
			0.05, // LRVJOB
			0.01, // HIGATJOB
			0.04, // CHARIN
			0.12, // MONDO    ← tip (≪ radar)
			0.10, // 1/GYRO
			0.10, // MAKEPLAY
			1.20, // SERVICER got (needs 1.80 → still late)
		}
	}
}

func jobsFor(sc scenario) []landingJob {
	base := jobCatalog()
	cpus := cpuFor(sc)
	for i := range base {
		base[i].CPUSec = cpus[i]
	}
	return base
}

type burst struct {
	JobIndex int
	Start    float64
	End      float64
}

func buildBursts(sc scenario) []burst {
	jobs := jobsFor(sc)
	idx := func(name string) int {
		for i, j := range jobs {
			if j.Name == name {
				return i
			}
		}
		return -1
	}
	var out []burst

	if sc == scenarioHealthy {
		// Quiet radar — no continuous steal lane (or empty).
		for t := 0.0; t < periodS; t += 0.12 {
			out = append(out, burst{idx("LRHJOB"), t, minf(t+0.002, periodS)})
		}
		for t := 0.20; t < periodS; t += 0.70 {
			out = append(out, burst{idx("LRVJOB"), t, minf(t+0.003, periodS)})
		}
		out = append(out, burst{idx("CHARIN"), 0.50, 0.52})
		out = append(out, burst{idx("1/GYRO"), 0.40, 0.46}, burst{idx("1/GYRO"), 1.40, 1.46})
		out = append(out, burst{idx("MAKEPLAY"), 0.95, 1.00})
		// SERVICER owns most of the timeline — long contiguous windows.
		svc := [][2]float64{
			{0.00, 0.40}, {0.46, 0.50}, {0.52, 0.95},
			{1.00, 1.40}, {1.46, 2.00},
		}
		for _, w := range svc {
			out = append(out, burst{idx("SERVICER"), w[0], w[1]})
		}
		return out
	}

	// Overload schedule
	out = append(out, burst{idx("RR ECDU"), 0, periodS})
	for t := 0.0; t < periodS; t += 0.082 {
		out = append(out, burst{idx("LRHJOB"), t, minf(t+0.002, periodS)})
	}
	for t := 0.05; t < periodS; t += 0.50 {
		out = append(out, burst{idx("LRVJOB"), t, minf(t+0.003, periodS)})
	}
	out = append(out, burst{idx("HIGATJOB"), 0.40, 0.42})
	for _, t := range []float64{0.15, 0.55, 1.10, 1.70} {
		out = append(out, burst{idx("CHARIN"), t, t + 0.02})
	}
	// MONDO: shorter refresh slices — tip load, not radar-sized.
	for _, seg := range [][2]float64{{0.20, 0.28}, {0.70, 0.78}, {1.30, 1.38}, {1.75, 1.82}} {
		out = append(out, burst{idx("MONDO"), seg[0], seg[1]})
	}
	out = append(out,
		burst{idx("1/GYRO"), 0.30, 0.38},
		burst{idx("1/GYRO"), 1.30, 1.38},
	)
	out = append(out, burst{idx("MAKEPLAY"), 0.90, 1.05})
	// SERVICER: leftovers between higher work (got ~1.20s, needs 1.80s).
	for _, w := range [][2]float64{
		{0.00, 0.15}, {0.28, 0.30}, {0.42, 0.55}, {0.78, 0.90},
		{1.05, 1.20}, {1.38, 1.65}, {1.82, 2.00},
	} {
		out = append(out, burst{idx("SERVICER"), w[0], w[1]})
	}
	return out
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func cpuSpentBy(jobs []landingJob, t float64) []float64 {
	p := t / periodS
	if p < 0 {
		p = 0
	}
	if p > 1 {
		p = 1
	}
	out := make([]float64, len(jobs))
	for i, j := range jobs {
		out[i] = j.CPUSec * p
	}
	return out
}

func activeJobsAt(t float64, bursts []burst) []int {
	seen := map[int]bool{}
	var ids []int
	for _, b := range bursts {
		if t >= b.Start && t < b.End {
			if !seen[b.JobIndex] {
				seen[b.JobIndex] = true
				ids = append(ids, b.JobIndex)
			}
		}
	}
	return ids
}

func scenarioName(sc scenario) string {
	if sc == scenarioHealthy {
		return "HEALTHY"
	}
	return "OVERLOAD"
}

func jobLegendLine(j landingJob) string {
	prio := "  —"
	if j.Prio > 0 {
		prio = fmt.Sprintf("%3d", j.Prio)
	}
	return fmt.Sprintf("%-8s prio %s %-5s  %4.2fs  %s",
		j.Name, prio, j.Kind, j.CPUSec, j.Cadence)
}
