// Package transition is a background crossfade: two full-stage
// components keep painting, and every cell's floor walks From's
// background ink toward To's through RGB. A sky can become a flag
// (or any floor become any other) without a hard cut, and the
// result stays a floor so an eagle, a shotgun, a blast can sit on
// top. Delay holds From; Over is how long the walk takes. A fade
// of zero or less snaps to To the moment the delay elapses. Stack
// paints several components in order as one layer.
package transition
