package sprite

import "sort"

// FrameNames lists the frames present at a size: compass headings in
// canonical order first, then any extra named frames (animation poses
// like "run1") sorted by name, so callers walk a stable order.
func (a *Atlas) FrameNames(sz Size) []Heading {
	if a == nil {
		return nil
	}
	byH := a.frames[sz]
	if len(byH) == 0 {
		return nil
	}
	canon := map[Heading]bool{}
	out := make([]Heading, 0, len(byH))
	for _, h := range Headings {
		canon[h] = true
		if _, ok := byH[h]; ok {
			out = append(out, h)
		}
	}
	extra := make([]Heading, 0, len(byH))
	for h := range byH {
		if !canon[h] {
			extra = append(extra, h)
		}
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i] < extra[j] })
	return append(out, extra...)
}
