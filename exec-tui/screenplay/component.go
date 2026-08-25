package screenplay

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

// Component is one performer inside a scene. It never touches the
// screen: every frame it hands the scene a sprite — its stage-sized
// slab of pixels — and the scene composites the cast in its own order.
//
//	Start(w, h)  allocate everything for a w×h stage: the curtain rises
//	Update(dt)   advance internal clocks by dt seconds; no render data
//	Render()     this instant's pixels, as a stage-sized sprite
//	Stop()       free everything Start built; Start may come again
//
// The lifecycle loops: Start, update, render, update, render, …, Stop
// — and a later Start re-allocates from scratch, so a stopped
// component holds nothing for the collector to keep. A resize is a
// Stop followed by a Start at the new size.
type Component interface {
	Start(width, height int)
	Update(dt float64)
	Render() sprite.Sprite
	Stop()
}
