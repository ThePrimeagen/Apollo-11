package msim

import (
	"fmt"
	"os"
	"testing"
)

// TestDumpSlotDance is a diagnostic, not an assertion: AGC_DUMP=1 go test
// -run DumpSlotDance -v prints the slot-level dance of the 1668 run so the
// leak geometry can be inspected against the flight record.
func TestDumpSlotDance(t *testing.T) {
	if os.Getenv("AGC_DUMP") == "" {
		t.Skip("set AGC_DUMP=1 to print the slot dance")
	}
	d := newDescent(true)
	e := d.e
	keyV16N68(e, Monitor1EntrMS*Millisecond, "first")
	for step := 0; step < 400; step++ {
		e.RunMS(100)
		if e.Now() < 14*Second {
			continue
		}
		var slots string
		for i, r := range e.exec.slots {
			if r == nil {
				slots += fmt.Sprintf(" %d:----------", i)
				continue
			}
			st := "W"
			switch {
			case r.dormant:
				st = "Z"
			case i == 0:
				st = "R"
			case r.started:
				st = "P"
			}
			slots += fmt.Sprintf(" %d:%-8s%s%d", i, r.name, st, r.prio)
		}
		fmt.Printf("t=%6.1fs cores=%d vacs=%d%s\n",
			float64(e.Now())/1e9, e.CoresHeld(), e.VACsHeld(), slots)
		if len(e.Alarms()) > 0 {
			a := e.Alarms()[0]
			fmt.Printf("ALARM %d at %.3fs req=%s cores=%d vacs=%d\n",
				a.Code, float64(a.At)/1e9, a.Requester, a.CoresHeld, a.VACsHeld)
			break
		}
	}
}
