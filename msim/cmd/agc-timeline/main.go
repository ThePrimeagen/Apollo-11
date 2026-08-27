// agc-timeline generates the two P63 timelines — baseline and V16N68 —
// from the millisecond simulator, as markdown reports.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	msim "github.com/theprimeagen/apollo-11/msim"
)

func main() {
	out := flag.String("out", "timelines", "output directory")
	ms := flag.Int("ms", 100_000, "window length in simulated milliseconds")
	flag.Parse()

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	base := msim.RunBaselineP63(*ms)
	mon := msim.RunMonitor1668(*ms)

	write := func(name string, res *msim.Result) {
		path := filepath.Join(*out, name)
		if err := os.WriteFile(path, []byte(msim.RenderTimeline(res, res.Title)), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%s: %d alarms, %d restarts, peak cores %d/8 vacs %d/5\n",
			path, len(res.Alarms), res.Restarts, res.MaxCores, res.MaxVACs)
		for _, a := range res.Alarms {
			fmt.Printf("  alarm %d at t=%.3fs (requester %s, cores %d/8, vacs %d/5)\n",
				a.Code, float64(a.At)/1e9, a.Requester, a.CoresHeld, a.VACsHeld)
		}
	}
	write("p63-baseline.md", base)
	write("p63-monitor-1668.md", mon)
}
