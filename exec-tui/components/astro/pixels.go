package astro

import "github.com/theprimeagen/apollo-11/exec-tui/components/sprite"

// The astronaut, drawn by hand on the classic 16×16 side-scroller
// envelope. Letters are palette IDs; '.' is empty sky.
//
//	W suit white   H shade grey   V visor gold
//	D dark grey (boots, gloves, pack floor, chest box)   R red accent
//
// Anatomy, shared across poses: a seven-row helmet dome — white crown,
// gold visor across the middle looking right — the life-support pack
// riding the left edge (his back; he only ever travels left to right),
// a red flag patch and dark chest box on the torso, and two-row moon
// boots. Feet stand on the last pixel row; the jump and the pole grips
// leave it empty because they are airborne.
//
// The run is the classic three-frame stride: run1 is the full contact
// split (front boot planted, trailing boot kicked up, lead fist punched
// forward), run2 is the passing tuck (legs gathered under the hips,
// boot folded mid-air, fist dropping), run3 is the second, shorter
// contact on the other beat (fist swung back low).

var grids = map[sprite.Heading][]string{
	PoseStand: {
		//        0123456789012345
		/*  0 */ ".....WWWWWW.....",
		/*  1 */ "....WWWWWWWW....",
		/*  2 */ "...WWWWWWWWWW...",
		/*  3 */ "...WWWWVVVVVV...",
		/*  4 */ ".HHWWWWVVVVVV...",
		/*  5 */ "HHHWWWWVVVVVW...",
		/*  6 */ "HHH.WWWWWWWW....",
		/*  7 */ "HHH.WWRRWWWWW...",
		/*  8 */ "HHH.WWWWWDDWW...",
		/*  9 */ "DDD.WWWWWWWWW...",
		/* 10 */ "....DWWWWWWWD...",
		/* 11 */ ".....WWWWWWW....",
		/* 12 */ ".....WWW.WWW....",
		/* 13 */ ".....WWW.WWW....",
		/* 14 */ "....DDD..DDD....",
		/* 15 */ "....DDD..DDDD...",
	},
	PoseRun1: {
		//        0123456789012345
		/*  0 */ ".....WWWWWW.....",
		/*  1 */ "....WWWWWWWW....",
		/*  2 */ "...WWWWWWWWWW...",
		/*  3 */ "...WWWWVVVVVV...",
		/*  4 */ ".HHWWWWVVVVVV...",
		/*  5 */ "HHHWWWWVVVVVW...",
		/*  6 */ "HHH.WWWWWWWW....",
		/*  7 */ "HHH.WWRRWWWWW...",
		/*  8 */ "HHH.WWWWWDDWWW..",
		/*  9 */ "DDD.WWWWWWWWWDD.",
		/* 10 */ ".....WWWWWWW....",
		/* 11 */ "....WWW..WWW....",
		/* 12 */ "...WWW....WWW...",
		/* 13 */ "..DDD......WW...",
		/* 14 */ "..DD.......DDD..",
		/* 15 */ "...........DDDD.",
	},
	PoseRun2: {
		//        0123456789012345
		/*  0 */ ".....WWWWWW.....",
		/*  1 */ "....WWWWWWWW....",
		/*  2 */ "...WWWWWWWWWW...",
		/*  3 */ "...WWWWVVVVVV...",
		/*  4 */ ".HHWWWWVVVVVV...",
		/*  5 */ "HHHWWWWVVVVVW...",
		/*  6 */ "HHH.WWWWWWWW....",
		/*  7 */ "HHH.WWRRWWWWW...",
		/*  8 */ "HHH.WWWWWDDWW...",
		/*  9 */ "DDD.WWWWWWWWW...",
		/* 10 */ ".....WWWWWWWDD..",
		/* 11 */ ".....WWWWWWW....",
		/* 12 */ ".......WWWWWW...",
		/* 13 */ ".......WW.DDD...",
		/* 14 */ ".......WW.......",
		/* 15 */ "......DDDD......",
	},
	PoseRun3: {
		//        0123456789012345
		/*  0 */ ".....WWWWWW.....",
		/*  1 */ "....WWWWWWWW....",
		/*  2 */ "...WWWWWWWWWW...",
		/*  3 */ "...WWWWVVVVVV...",
		/*  4 */ ".HHWWWWVVVVVV...",
		/*  5 */ "HHHWWWWVVVVVW...",
		/*  6 */ "HHH.WWWWWWWW....",
		/*  7 */ "HHH.WWRRWWWWW...",
		/*  8 */ "HHH.WWWWWDDWW...",
		/*  9 */ "DDD.WWWWWWWWW...",
		/* 10 */ "...DDWWWWWWW....",
		/* 11 */ ".....WWW.WWW....",
		/* 12 */ "....WWW..WWW....",
		/* 13 */ "...DDD....WW....",
		/* 14 */ "..........DDD...",
		/* 15 */ "..........DDDD..",
	},
	PoseJump: {
		//        0123456789012345
		/*  0 */ ".....WWWWWW.....",
		/*  1 */ "....WWWWWWWW....",
		/*  2 */ "...WWWWWWWWWW...",
		/*  3 */ "...WWWWVVVVVV...",
		/*  4 */ ".HHWWWWVVVVVV...",
		/*  5 */ "HHHWWWWVVVVVWDD.",
		/*  6 */ "HHH.WWWWWWWW.W..",
		/*  7 */ "HHH.WWRRWWWWWW..",
		/*  8 */ "HHH.WWWWWDDWW...",
		/*  9 */ "DDD.WWWWWWWWW...",
		/* 10 */ ".....WWWWWWW....",
		/* 11 */ ".....WW..WWWW...",
		/* 12 */ "....WW....DDDD..",
		/* 13 */ "...WW...........",
		/* 14 */ ".DDD............",
		/* 15 */ "................",
	},
	PosePole1: {
		//        0123456789012345
		/*  0 */ ".....WWWWWW.....",
		/*  1 */ "....WWWWWWWW....",
		/*  2 */ "...WWWWWWWWWW...",
		/*  3 */ "...WWWWVVVVVV...",
		/*  4 */ ".HHWWWWVVVVVVDD.",
		/*  5 */ "HHHWWWWVVVVVWW..",
		/*  6 */ "HHH.WWWWWWWW.W..",
		/*  7 */ "HHH.WWRRWWWWW...",
		/*  8 */ "HHH.WWWWWDDWWDD.",
		/*  9 */ "DDD.WWWWWWWWW...",
		/* 10 */ ".....WWWWWWW....",
		/* 11 */ "......WWWWWW....",
		/* 12 */ "........WWWW....",
		/* 13 */ ".........DDDD...",
		/* 14 */ ".........DDDD...",
		/* 15 */ "................",
	},
	PosePole2: {
		//        0123456789012345
		/*  0 */ ".....WWWWWW.....",
		/*  1 */ "....WWWWWWWW....",
		/*  2 */ "...WWWWWWWWWWDD.",
		/*  3 */ "...WWWWVVVVVVW..",
		/*  4 */ ".HHWWWWVVVVVVW..",
		/*  5 */ "HHHWWWWVVVVVWW..",
		/*  6 */ "HHH.WWWWWWWW.W..",
		/*  7 */ "HHH.WWRRWWWWW...",
		/*  8 */ "HHH.WWWWWDDWW...",
		/*  9 */ "DDD.WWWWWWWWWDD.",
		/* 10 */ ".....WWWWWWW....",
		/* 11 */ "......WWWWWW....",
		/* 12 */ "........WWW.....",
		/* 13 */ "........DDDD....",
		/* 14 */ "........DDDD....",
		/* 15 */ "................",
	},
}
