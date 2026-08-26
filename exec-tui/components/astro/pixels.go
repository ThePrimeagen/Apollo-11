package astro

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

// The astronaut, drawn by hand on the classic 16×16 side-scroller
// envelope. Letters are palette IDs; '.' is empty sky.
//
//	W suit white   H shade grey (life-support pack)   V visor gold
//	D dark grey (boots, gloves, pack floor, chest box)   R red accent
//
// Anatomy, shared across poses: a seven-row helmet dome with a 4×2
// gold visor window looking right, the pack hugging his back (left
// edge — he only travels left to right) with no sky slit between pack
// and ribs, a red flag patch and dark chest box, and chunky two-row
// moon boots. Grounded poses stand on the last pixel row; airborne
// poses (jump, both pole grips) leave the bottom rows empty.
//
// The run is the classic three-frame contact → passing → contact
// loop, and the planted foot sweeps backward through the body the way
// a real stride does: run1 is the double-support V-split (lead boot
// planted far forward, trailing toe still down), run2 is the passing
// tuck (support boot under the hips, free boot folded aft, body
// dipped one row for the bob), run3 is the opposite contact (left
// boot planted at the back, right knee driving forward). All three
// run frames and the airborne poses lean one column forward off the
// stand so the sprint reads as motion, not a stamped cutout.

var grids = map[sprite.Heading][]string{
	PoseStand: {
		//        0123456789012345
		/*  0 */ ".....WWWWWW.....",
		/*  1 */ "....WWWWWWWW....",
		/*  2 */ "...WWWWWWWWWW...",
		/*  3 */ ".HHWWWWWVVVVW...",
		/*  4 */ ".HHWWWWWVVVVW...",
		/*  5 */ ".HHWWWWWWWWWW...",
		/*  6 */ ".HHHWWWWWWWW....",
		/*  7 */ ".HHHWWRRWWWWW...",
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
		/*  2 */ "....WWWWWWWWWW..",
		/*  3 */ "..HHWWWWWVVVVW..",
		/*  4 */ "..HHWWWWWVVVVW..",
		/*  5 */ "..HHWWWWWWWWWW..",
		/*  6 */ "..HHHWWWWWWWW...",
		/*  7 */ "..HHHWWRRWWWWW..",
		/*  8 */ "..HHHWWWWWDDWWW.",
		/*  9 */ "..DDDWWWWWWWWWDD",
		/* 10 */ "......WWWWWWW...",
		/* 11 */ "....WWW...WWW...",
		/* 12 */ "...WWW.....WWW..",
		/* 13 */ "..WWW.......WW..",
		/* 14 */ "..DDD.......DDD.",
		/* 15 */ "..DDD.......DDDD",
	},
	PoseRun2: {
		//        0123456789012345
		/*  0 */ "................",
		/*  1 */ "......WWWWWW....",
		/*  2 */ ".....WWWWWWWW...",
		/*  3 */ "....WWWWWWWWWW..",
		/*  4 */ "..HHWWWWWVVVVW..",
		/*  5 */ "..HHWWWWWVVVVW..",
		/*  6 */ "..HHWWWWWWWWWW..",
		/*  7 */ "..HHHWWWWWWWW...",
		/*  8 */ "..HHHWWRRWWWWW..",
		/*  9 */ "..HHHWWWWWDDWW..",
		/* 10 */ "..DDDWWWWWWWWDD.",
		/* 11 */ "......WWWWWWW...",
		/* 12 */ "....WWW.WWW.....",
		/* 13 */ "...DDD..WW......",
		/* 14 */ "........WW......",
		/* 15 */ ".......DDDD.....",
	},
	PoseRun3: {
		//        0123456789012345
		/*  0 */ "......WWWWWW....",
		/*  1 */ ".....WWWWWWWW...",
		/*  2 */ "....WWWWWWWWWW..",
		/*  3 */ "..HHWWWWWVVVVW..",
		/*  4 */ "..HHWWWWWVVVVW..",
		/*  5 */ "..HHWWWWWWWWWW..",
		/*  6 */ "..HHHWWWWWWWW...",
		/*  7 */ "..HHHWWRRWWWWW..",
		/*  8 */ "..HHHWWWWWDDWW..",
		/*  9 */ "..DDDWWWWWWWWW..",
		/* 10 */ "......WWWWWWW...",
		/* 11 */ "...DD.WWWWWWW...",
		/* 12 */ ".....WWW..DDDD..",
		/* 13 */ "....WWW.........",
		/* 14 */ "...DDD..........",
		/* 15 */ "..DDDD..........",
	},
	PoseJump: {
		//        0123456789012345
		/*  0 */ "......WWWWWW....",
		/*  1 */ ".....WWWWWWWW...",
		/*  2 */ "....WWWWWWWWWW..",
		/*  3 */ "..HHWWWWWVVVVW..",
		/*  4 */ "..HHWWWWWVVVVW..",
		/*  5 */ "..HHWWWWWWWWWW..",
		/*  6 */ "..HHHWWWWWWWWWDD",
		/*  7 */ "..HHHWWRRWWWWWW.",
		/*  8 */ "..HHHWWWWWDDWW..",
		/*  9 */ "..DDDWWWWWWWWW..",
		/* 10 */ "......WWWWWWW...",
		/* 11 */ ".....WWW.WWWW...",
		/* 12 */ "....DDD...DDDD..",
		/* 13 */ "....DDD...DDD...",
		/* 14 */ "................",
		/* 15 */ "................",
	},
	PosePole1: {
		//        0123456789012345
		/*  0 */ "......WWWWWW....",
		/*  1 */ ".....WWWWWWWW...",
		/*  2 */ "....WWWWWWWWWW..",
		/*  3 */ "..HHWWWWWVVVVW..",
		/*  4 */ "..HHWWWWWVVVVWDD",
		/*  5 */ "..HHWWWWWWWWWWW.",
		/*  6 */ "..HHHWWWWWWWW.W.",
		/*  7 */ "..HHHWWRRWWWWW..",
		/*  8 */ "..HHHWWWWWDDWWDD",
		/*  9 */ "..DDDWWWWWWWWW..",
		/* 10 */ "......WWWWWWW...",
		/* 11 */ ".......WWWWWW...",
		/* 12 */ ".........WWWW...",
		/* 13 */ "..........DDDD..",
		/* 14 */ ".........DDDD...",
		/* 15 */ "................",
	},
	PosePole2: {
		//        0123456789012345
		/*  0 */ "......WWWWWW....",
		/*  1 */ ".....WWWWWWWW...",
		/*  2 */ "....WWWWWWWWWW..",
		/*  3 */ "..HHWWWWWVVVVW..",
		/*  4 */ "..HHWWWWWVVVVW..",
		/*  5 */ "..HHWWWWWWWWWWDD",
		/*  6 */ "..HHHWWWWWWWW.W.",
		/*  7 */ "..HHHWWRRWWWWW..",
		/*  8 */ "..HHHWWWWWDDWW..",
		/*  9 */ "..DDDWWWWWWWWWDD",
		/* 10 */ "......WWWWWWW...",
		/* 11 */ ".......WWWWWW...",
		/* 12 */ "........DDDD....",
		/* 13 */ ".........DDDD...",
		/* 14 */ "................",
		/* 15 */ "................",
	},
}
