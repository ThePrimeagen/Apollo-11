package screenplay

// Bill is one screenplay's worth of scenes, in playing order — the
// composable unit. Each show exports its bill, and the final product
// is bills added together into one big screenplay.
type Bill []Entry

// Compose flattens bills, in order, into one screenplay. Nil and
// empty bills compose to nothing, so a missing show never breaks the
// house.
func Compose(bills ...Bill) *Screenplay {
	var all []Entry
	for _, b := range bills {
		all = append(all, b...)
	}
	return New(all...)
}
