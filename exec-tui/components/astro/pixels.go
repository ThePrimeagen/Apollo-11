package astro

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

// The astronaut, drawn by hand on the classic 16×16 side-scroller
// envelope. Letters are palette IDs; '.' is empty sky.
//
//	W suit white   H shade grey (life-support pack)   V visor gold
//	D dark grey (boots, gloves, pack floor, chest box)   R red accent
//
// Anatomy, shared across poses: a seven-row helmet dome with a 4×2
// gold visor window high on the face looking right, the pack hugging
// his back (left edge — he only travels left to right), a red flag
// patch on the jaw row (clear of the 8px tile seams), a dark chest
// box, and chunky moon boots. Grounded poses stand on the last pixel
// row; airborne poses leave the bottom row clear.
//
// The run is the classic three-frame contact → passing → contact
// loop. The planted foot sweeps backward through the body (front
// plant → under the hips → rear plant), the contacts lean one column
// forward while the passing frame sits at the neutral column and dips
// one row for the bob, and the near arm pumps through a five-row arc:
// punched up-forward at full extension (run1), dropped tight to the
// ribs on the pass (run2), dragged low and back on the drive (run3).
// The far arm stays behind the pack — at this size a second swinging
// arm reads as a third leg, so it never leaves the silhouette.
//
// The jump throws the lead glove a full row above the helmet crown,
// arm hugging the dome, the far glove counter-thrown low behind, and
// both boots folded into a knees-out ball with two clear rows below.
// The pole pose is a fireman slide: two fists stacked on the pole
// line with a wrist break between them, both boots clamped on that
// same line below, and the only difference between the two frames is
// a one-pixel hip wobble — sliding friction, never hand-over-hand.

var grids = map[sprite.Heading][]string{
	PoseStand: {
		//        0123456789012345
		/*  0 */ ".....WWWWWW.....",
		/*  1 */ "....WWWWWWWW....",
		/*  2 */ "...WWWWWVVVVW...",
		/*  3 */ ".HHWWWWWVVVVW...",
		/*  4 */ ".HHWWWWWWWWWW...",
		/*  5 */ ".HHWWWWWWWWWW...",
		/*  6 */ ".HHHWRRWWWWW....",
		/*  7 */ ".HHHWWWWWWWWW...",
		/*  8 */ ".HHHWWWWWDDWW...",
		/*  9 */ ".DDDWWWWWWWWW...",
		/* 10 */ "....DWWWWWWWD...",
		/* 11 */ ".....WWWWWWW....",
		/* 12 */ ".....WWW.WWW....",
		/* 13 */ ".....WWW.WWW....",
		/* 14 */ "....DDD..DDD....",
		/* 15 */ "....DDD..DDDD...",
	},
	PoseRun1: {
		//        0123456789012345
		/*  0 */ "......WWWWWW....",
		/*  1 */ ".....WWWWWWWW...",
		/*  2 */ "....WWWWWVVVVW..",
		/*  3 */ "..HHWWWWWVVVVW..",
		/*  4 */ "..HHWWWWWWWWWW..",
		/*  5 */ "..HHWWWWWWWWW...",
		/*  6 */ "..HHHWRRWWWWW.DD",
		/*  7 */ "..HHHWWWWWWWWWDD",
		/*  8 */ "..HHHWWWWWDDWW..",
		/*  9 */ "..DDDWWWWWWWWW..",
		/* 10 */ "......WWWWWWW...",
		/* 11 */ "....WWW...WWW...",
		/* 12 */ "...WWW.....WWW..",
		/* 13 */ "...DDD......WW..",
		/* 14 */ "...DDD.....DDD..",
		/* 15 */ "...........DDDD.",
	},
	PoseRun2: {
		//        0123456789012345
		/*  0 */ "................",
		/*  1 */ ".....WWWWWW.....",
		/*  2 */ "....WWWWWWWW....",
		/*  3 */ "...WWWWWVVVVW...",
		/*  4 */ ".HHWWWWWVVVVW...",
		/*  5 */ ".HHWWWWWWWWWW...",
		/*  6 */ ".HHWWWWWWWWWW...",
		/*  7 */ ".HHHWRRWWWWW....",
		/*  8 */ ".HHHWWWWWWWWW...",
		/*  9 */ ".HHHWWWWWDDWW.DD",
		/* 10 */ ".DDDWWWWWWWWWWDD",
		/* 11 */ "...WWWWWWWWW....",
		/* 12 */ ".....DD..WWW....",
		/* 13 */ ".....DD..WW.....",
		/* 14 */ "........WW......",
		/* 15 */ ".......DDDD.....",
	},
	PoseRun3: {
		//        0123456789012345
		/*  0 */ "......WWWWWW....",
		/*  1 */ ".....WWWWWWWW...",
		/*  2 */ "....WWWWWVVVVW..",
		/*  3 */ "..HHWWWWWVVVVW..",
		/*  4 */ "..HHWWWWWWWWWW..",
		/*  5 */ "..HHWWWWWWWWWW..",
		/*  6 */ "..HHHWRRWWWWW...",
		/*  7 */ "..HHHWWWWWWWWW..",
		/*  8 */ "..HHHWWWWWDDWW..",
		/*  9 */ "..DDDWWWWWWWWW..",
		/* 10 */ "......WWWWWWWDD.",
		/* 11 */ ".....WWW..WW.DD.",
		/* 12 */ "....WWW....W....",
		/* 13 */ "....WWW....DDD..",
		/* 14 */ "...DDD.....DDD..",
		/* 15 */ "..DDDD..........",
	},
	PoseJump: {
		//        0123456789012345
		/*  0 */ "..............DD",
		/*  1 */ "......WWWWWW..DD",
		/*  2 */ ".....WWWWWWWW.W.",
		/*  3 */ "....WWWWWVVVVWW.",
		/*  4 */ "..HHWWWWWVVVVWW.",
		/*  5 */ "..HHWWWWWWWWWWW.",
		/*  6 */ "..HHWWWWWWWWWWW.",
		/*  7 */ "..HHHWRRWWWWWWW.",
		/*  8 */ "..HHHWWWWWWWWWW.",
		/*  9 */ "..HHHWWWWWDDWW..",
		/* 10 */ "..DDDWWWWWWWWW..",
		/* 11 */ "..WW..WWWWWWWW..",
		/* 12 */ "..DD.DDDDWWWWW..",
		/* 13 */ "..DD..DDD.......",
		/* 14 */ "................",
		/* 15 */ "................",
	},
	PosePole1: {
		//        0123456789012345
		/*  0 */ "......WWWWWW..DD",
		/*  1 */ ".....WWWWWWWW.DD",
		/*  2 */ "....WWWWWVVVVWW.",
		/*  3 */ "..HHWWWWWVVVVWDD",
		/*  4 */ "..HHWWWWWWWWWWDD",
		/*  5 */ "..HHWWWWWWWWWWW.",
		/*  6 */ "..HHHWRRWWWWWWW.",
		/*  7 */ "..HHHWWWWWWWWW..",
		/*  8 */ "..HHHWWWWWDDWW..",
		/*  9 */ "..DDDWWWWWWWWW..",
		/* 10 */ "......WWWWWWW...",
		/* 11 */ ".......WWWWWW.DD",
		/* 12 */ ".........WWWWWDD",
		/* 13 */ "..........WWWWWW",
		/* 14 */ "..............DD",
		/* 15 */ "................",
	},
	PosePole2: {
		//        0123456789012345
		/*  0 */ "......WWWWWW..DD",
		/*  1 */ ".....WWWWWWWW.DD",
		/*  2 */ "....WWWWWVVVVWW.",
		/*  3 */ "..HHWWWWWVVVVWDD",
		/*  4 */ "..HHWWWWWWWWWWDD",
		/*  5 */ "..HHWWWWWWWWWWW.",
		/*  6 */ "..HHHWRRWWWWWWW.",
		/*  7 */ "..HHHWWWWWWWWW..",
		/*  8 */ "..HHHWWWWWDDWW..",
		/*  9 */ "..DDDWWWWWWWWW..",
		/* 10 */ ".....WWWWWWW....",
		/* 11 */ ".......WWWWWW.DD",
		/* 12 */ ".........WWWWWDD",
		/* 13 */ "..........WWWWWW",
		/* 14 */ "..............DD",
		/* 15 */ "................",
	},
}
