package msim

import (
	"fmt"
	"sort"
	"strings"
)

// t0GETSeconds is GET 102:37:55 in seconds.
const t0GETSeconds = 102*3600 + 37*60 + 55

func getStamp(ns Nanos) string {
	s := t0GETSeconds + int(ns/Second)
	return fmt.Sprintf("%d:%02d:%02d", s/3600, (s%3600)/60, s%60)
}

func bar(n, max int) string {
	if n < 0 {
		n = 0
	}
	if n > max {
		n = max
	}
	return strings.Repeat("#", n) + strings.Repeat(".", max-n)
}

// RenderTimeline renders one scenario: a per-second occupancy strip (cores
// 0-8, VACs 0-5, the running job at the sample instant), then the event log
// (keys, notes, restarts) with every alarm's pool snapshot, then the demand
// accounting.
func RenderTimeline(res *Result, title string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "Window: t=0 is GET %s (PDI+290 s), %d ms simulated, 1 ms per tick.\n\n",
		T0GET, res.DurationMS)

	// --- occupancy strip
	durSec := res.DurationMS / 1000
	if durSec > 0 && len(res.Samples) > 0 {
		b.WriteString("```text\n")
		b.WriteString("  t(s)  GET        cores 0-8    vacs 0-5   running at sample\n")
		for s := 0; s < durSec; s++ {
			idx := s*1000 + 999
			if idx >= len(res.Samples) {
				break
			}
			sm := res.Samples[idx]
			running := sm.Running
			if running == "" {
				running = "-"
			}
			fmt.Fprintf(&b, "  %4d  %s  |%s| %d/8  |%s| %d/5  %s\n",
				s, getStamp(Nanos(s)*Second),
				bar(sm.Cores, 8), sm.Cores,
				bar(sm.VACs, 5), sm.VACs,
				running)
		}
		b.WriteString("```\n\n")
	}

	// --- events: keys, notes, restarts, alarms (with pool snapshots)
	type line struct {
		at   Nanos
		text string
	}
	var lines []line
	for _, ev := range res.Events {
		switch ev.Kind {
		case "key":
			lines = append(lines, line{ev.At, fmt.Sprintf("key      %s", ev.Detail)})
		case "note":
			lines = append(lines, line{ev.At, fmt.Sprintf("crew     %s", ev.Detail)})
		case "restart":
			lines = append(lines, line{ev.At, "RESTART  software restart — " + ev.Detail})
		}
	}
	for _, a := range res.Alarms {
		code := "1202 NO CORE SETS"
		if a.Code == 1201 {
			code = "1201 NO VAC AREAS"
		}
		lines = append(lines, line{a.At, fmt.Sprintf(
			"ALARM    %s — request %q denied; cores %d/8, vacs %d/5 at the request",
			code, a.Requester, a.CoresHeld, a.VACsHeld)})
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].at < lines[j].at })
	if len(lines) > 0 {
		b.WriteString("## Events\n\n```text\n")
		for _, l := range lines {
			fmt.Fprintf(&b, "  t=%8.3fs  GET %s  %s\n", float64(l.at)/1e9, getStamp(l.at), l.text)
		}
		b.WriteString("```\n\n")
	}

	// --- demand accounting
	if res.ElapsedNs > 0 {
		soft := 100 * float64(res.SoftwareNs) / float64(res.ElapsedNs)
		theft := 100 * float64(res.TheftNs) / float64(res.ElapsedNs)
		idle := 100 * float64(res.IdleNs) / float64(res.ElapsedNs)
		fmt.Fprintf(&b, "## Accounting\n\n")
		fmt.Fprintf(&b, "- software (jobs + tasks + interrupts): %.2f%%\n", soft)
		fmt.Fprintf(&b, "- RR CDU counter theft: %.2f%%\n", theft)
		fmt.Fprintf(&b, "- idle: %.2f%%\n", idle)
		fmt.Fprintf(&b, "- peak occupancy: cores %d/8, vacs %d/5\n", res.MaxCores, res.MaxVACs)
		fmt.Fprintf(&b, "- restarts: %d\n", res.Restarts)
	}
	return b.String()
}
