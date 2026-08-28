package msim

import (
	"fmt"
	"strconv"
	"strings"
)

// The SERVICER instruction array: one complete P63 braking-phase guidance
// pass transcribed from the Luminary099 listing, one entry per interpretive
// instruction (packed opcode pairs like "VLOAD VXSC" are two entries;
// STODL/STOVL/STCALL split into their two packed operations) plus short
// basic-language blocks with explicit costs.
//
// Costs: opCost below. Relative costs follow instruction complexity; the
// absolute scale is pinned by two documented anchors —
//   1. an interpretive vector op (VXV/MXV/UNIT) runs ≈ 5 ms (Eyles, Tales;
//      also exec-tui/RESEARCH.md "Interpretive vector cross product ≈ 5 ms"),
//   2. the whole P63 pass costs 1.30-1.45 s: 65-72% of the 2 s cycle
//      (Cherry's job table; Eyles' duty-cycle margins).
// Between any two entries the engine checks NEWJOB — the DANZIG boundary
// (INTERPRETER.agc L74-L82).

// Every interpretive instruction's cost includes its share of the DANZIG
// dispatch (decode, bank restore, operand address arithmetic) — the reason
// interpretive code ran an order of magnitude slower than basic. Values are
// the calibrated model documented in msim/RESEARCH.md.
var opCost = map[string]Nanos{
	// scalar / double-precision
	"DLOAD": 4300 * Microsecond,
	"SLOAD": 3800 * Microsecond,
	"DAD":   4400 * Microsecond,
	"DSU":   4400 * Microsecond,
	"DMP":   4800 * Microsecond,
	"DDV":   5000 * Microsecond,
	"BDDV":  5000 * Microsecond,
	"BDSU":  4400 * Microsecond,
	"DSQ":   4400 * Microsecond,
	"SL":    3800 * Microsecond, // all scalar shift forms
	"SL1":   3600 * Microsecond,
	"SL3":   3600 * Microsecond,
	"SLR":   3600 * Microsecond,
	"SR4":   3600 * Microsecond,
	"ABS":   3200 * Microsecond,
	"SSP":   3200 * Microsecond,
	"SETPD": 2800 * Microsecond,
	"PUSH":  3200 * Microsecond,
	"PDDL":  4100 * Microsecond,
	"PDVL":  4300 * Microsecond,
	"STORE": 4300 * Microsecond,
	"STADR": 3600 * Microsecond,
	"STQ":   3000 * Microsecond,
	"EXIT":  2000 * Microsecond,
	"RVQ":   2400 * Microsecond,
	"CALL":  3200 * Microsecond,
	"GOTO":  2400 * Microsecond,
	"RTB":   3000 * Microsecond, // dispatch only; the routine is its own entry
	"BON":   2800 * Microsecond,
	"BOV":   2800 * Microsecond,
	"BOVB":  2800 * Microsecond,
	"BMN":   2800 * Microsecond,
	"BPL":   2800 * Microsecond,
	"BHIZ":  2800 * Microsecond,
	"VDEF":  3800 * Microsecond,

	// vector
	"VLOAD": 4500 * Microsecond,
	"VAD":   4900 * Microsecond,
	"VSU":   4900 * Microsecond,
	"BVSU":  4900 * Microsecond,
	"VXSC":  4900 * Microsecond,
	"V/SC":  5000 * Microsecond,
	"VCOMP": 4000 * Microsecond,
	"VSL1":  4800 * Microsecond, // all vector shift forms
	"VSL2":  4800 * Microsecond,
	"VSR2":  4800 * Microsecond,
	"DOT":   4900 * Microsecond,

	// the ≈5 ms anchor (Eyles): a vector cross product
	"VXV": 5000 * Microsecond,
}

// Ops that exceed the 5 ms DANZIG grain are decomposed into their real
// sub-phases (the interpreter computes MXV/VXM as three row/column dots;
// UNIT is ABVAL then a scale-divide; NORMUNIT pre-normalizes first; SQRT
// normalizes then iterates).
var opExpand = map[string][]struct {
	suffix string
	cost   Nanos
}{
	"MXV":      {{".row1", 4000 * Microsecond}, {".row2", 4000 * Microsecond}, {".row3", 4000 * Microsecond}},
	"VXM":      {{".col1", 4000 * Microsecond}, {".col2", 4000 * Microsecond}, {".col3", 4000 * Microsecond}},
	"UNIT":     {{".abval", 4800 * Microsecond}, {".scale", 4000 * Microsecond}},
	"ABVAL":    {{".sumsq", 4000 * Microsecond}, {".sqrt", 4000 * Microsecond}},
	"NORMUNIT": {{".norm", 3200 * Microsecond}, {".abval", 4800 * Microsecond}, {".scale", 4000 * Microsecond}},
	"SQRT":     {{".norm", 4000 * Microsecond}, {".iter", 4000 * Microsecond}},
}

// sec expands a whitespace-separated "OP:line OP:line ..." transcription
// into instruction entries for one section of one source file.
func sec(section, file, listing string) Script {
	var s Script
	for _, tok := range strings.Fields(listing) {
		parts := strings.SplitN(tok, ":", 2)
		op, line := parts[0], parts[1]
		if exp, ok := opExpand[op]; ok {
			for _, p := range exp {
				s = append(s, Instr{
					Section: section,
					Op:      op + p.suffix,
					Ref:     file + ":" + line,
					Cost:    p.cost,
				})
			}
			continue
		}
		cost, ok := opCost[op]
		if !ok {
			panic(fmt.Sprintf("no cost for opcode %q (%s %s:%s)", op, section, file, line))
		}
		s = append(s, Instr{
			Section: section,
			Op:      op,
			Ref:     file + ":" + line,
			Cost:    cost,
		})
	}
	return s
}

// bas is one basic-language block with an explicit cost.
func bas(section, label, file string, line int, us int) Instr {
	return Instr{
		Section: section,
		Op:      "BASIC/" + label,
		Ref:     file + ":" + strconv.Itoa(line),
		Cost:    Nanos(us) * Microsecond,
	}
}

const fSERV = "SERVICER.agc"
const fLLGE = "LUNAR_LANDING_GUIDANCE_EQUATIONS.agc"
const fTHROT = "THROTTLE_CONTROL_ROUTINES.agc"
const fCDUW = "FINDCDUW--GUIDAP_INTERFACE.agc"

// mungrav is the lunar gravity subroutine (SERVICER.agc L1121-L1131) —
// called once by RVBOTH (for the CSM state) and once by MUNRVG (for the LM).
func mungrav(section string) Script {
	return sec(section, fSERV, `
		UNIT:1121 STORE:1122 DLOAD:1122 SL:1124 BDDV:1124
		DMP:1127 VXSC:1127 STORE:1130 RVQ:1131`)
}

// servicerPass builds the pass. locked adds the landing-radar nav-frame
// conversion (UPDATCHK/POSUPDAT through STORE DELTAH), the ~2% that arrived
// with "data good" (Eyles; the outline's margin 15% → 13%). approach adds
// the REDESIG landing-site perturbation equations — the P64 phase's
// unsheddable extra guidance (Eyles: margin < 10%).
func servicerPass(locked, approach bool) Script {
	var s Script

	// --- job entry, ABVAL of accumulated delta-V, mass update, gimbal trig
	// SERVICER.agc L206-L263
	s = append(s,
		bas("ENTRY", "PHASCHNG", fSERV, 206, 350),
		bas("ENTRY", "1/PIPA-X", fSERV, 215, 2200), // PIPA compensation, per axis
		bas("ENTRY", "1/PIPA-Y", fSERV, 215, 2200),
		bas("ENTRY", "1/PIPA-GCOMP", fSERV, 216, 2200), // the 1/GYRO gate
	)
	s = append(s, sec("ENTRY", fSERV, `VLOAD:219 ABVAL:219 EXIT:221`)...)
	s = append(s,
		bas("ENTRY", "ABDELV-MASSMON", fSERV, 222, 900),
		bas("ENTRY", "MOONSPOT-SHORTMP", fSERV, 250, 700),
		bas("ENTRY", "QUICTRIG-SIN", fSERV, 258, 2600), // sin/cos per gimbal pair
		bas("ENTRY", "QUICTRIG-COS", fSERV, 258, 2600),
		bas("ENTRY", "QUICTRIG-TRIM", fSERV, 258, 2600),
		bas("ENTRY", "FLESHPOT-XNB-A", fSERV, 261, 3600), // XNB matrix build
		bas("ENTRY", "FLESHPOT-XNB-B", fSERV, 261, 3600),
	)

	// --- AVERAGEG: BON MUNFLAG → RVBOTH: advance the CSM state, then fall
	// into MUNRVG for the LM. SERVICER.agc L265, L1058-L1084
	s = append(s, sec("AVERAGEG", fSERV, `BON:265`)...)
	s = append(s, sec("RVBOTH", fSERV, `
		VLOAD:1058 PUSH:1058 VAD:1060 PDDL:1060 DDV:1063 VXSC:1063
		VAD:1065 STORE:1067 CALL:1067`)...)
	s = append(s, mungrav("RVBOTH")...)
	s = append(s, sec("RVBOTH", fSERV, `
		VAD:1069 VAD:1069 STADR:1071 STORE:1072 EXIT:1073`)...)
	s = append(s, bas("RVBOTH", "QUIKFAZ5", fSERV, 1074, 250))
	s = append(s, sec("RVBOTH", fSERV, `
		VLOAD:1076 STORE:1078 VLOAD:1078 STORE:1080 VLOAD:1080 STORE:1082 EXIT:1083`)...)

	// --- MUNRVG: LM average-G about the Moon. SERVICER.agc L1086-L1120
	s = append(s, bas("MUNRVG", "QUIKFAZ5", fSERV, 1084, 250))
	s = append(s, sec("MUNRVG", fSERV, `
		VLOAD:1086 VXSC:1086 PUSH:1089 VAD:1089 PUSH:1091 VAD:1091
		PDDL:1093 DDV:1093 VXSC:1096 VAD:1097 STORE:1099 CALL:1099`)...)
	s = append(s, mungrav("MUNRVG")...)
	s = append(s, sec("MUNRVG", fSERV, `
		VAD:1102 VAD:1102 VAD:1103 STORE:1105 ABVAL:1106
		STORE:1107 VLOAD:1107 DOT:1109 SL1:1109 STORE:1111 VLOAD:1111
		VXV:1113 VSL2:1113 STORE:1115 DLOAD:1115 DSU:1117 STORE:1119 CALL:1119`)...)

	// --- R12, entered from MUNRETRN: the landing-radar nav-frame conversion
	// (radar locked only) — the HBEAM body→SM transform and the DELTAH
	// computation. SERVICER.agc L762-L821 gates, L1146-L1188 body. The READLR
	// gate opens at the 50,000 ft ALTCRIT (35KCHK, L948), so this runs at the
	// flight's ~34,000 ft. (The reasonableness test is skipped before HIGATE
	// — L1190-L1193 — and the weighted incorporation with its MUNGRAV
	// re-call waits for V57.)
	if locked {
		s = append(s, bas("LR-CONVERT", "UPDATCHK", fSERV, 1146, 300))
		s = append(s, sec("LR-CONVERT", fSERV, `
			VLOAD:1159 VXM:1159 PDVL:1162 VSL2:1162 VAD:1164 DOT:1164
			DMP:1167 EXIT:1167`)...)
		s = append(s, bas("LR-CONVERT", "ALTSCALE", fSERV, 1170, 300))
		s = append(s, sec("LR-CONVERT", fSERV, `
			DAD:1179 SL:1179 DMP:1182 VXSC:1182 DOT:1184 DSU:1184
			STORE:1187 EXIT:1188`)...)
	}

	// --- CONTSERV → copy cycle → thrust monitor → 1/ACCS → AVGEXIT
	// (SERVICER.agc L822-L829, L530-L542, L279-L320, L359-L372; the
	// NOR29NOW state rebuild L855-L917 is absorbed in the EXEC residue)
	s = append(s,
		bas("SERVOUT", "COPYCYC", fSERV, 530, 800),
		bas("SERVOUT", "DVMON", fSERV, 293, 500),
		bas("SERVOUT", "1/ACCS-A", fSERV, 361, 4500), // DAP accel/jet parameters
		bas("SERVOUT", "1/ACCS-B", fSERV, 361, 4000),
		bas("SERVOUT", "AVGEXIT", fSERV, 370, 200),
	)

	// --- LUNLAND: guidance entry + GUILDENSTERN auto-modes monitor (R13)
	// LUNAR_LANDING_GUIDANCE_EQUATIONS.agc L117-L246
	s = append(s,
		bas("GUIDANCE", "PHASCHNG-G5", fLLGE, 117, 350),
		bas("GUIDANCE", "PHASCHNG-G3", fLLGE, 119, 350),
		bas("GUIDANCE", "GUILDEN-R13", fLLGE, 134, 300),
		bas("GUIDANCE", "GUILDRET-TPIP", fLLGE, 224, 500),
		bas("GUIDANCE", "FASTCHNG", fLLGE, 232, 250),
	)

	// --- TTFINCR: TTF/8 increment, landing-site rotation, range display
	// LLGE L288-L329
	s = append(s, sec("GUIDANCE", fLLGE, `
		DLOAD:289 DSU:289 SLR:292 PUSH:292 VXSC:294 VXV:294
		BVSU:297 RTB:297 NORMUNIT:299 VXSC:300 VSL1:300 STORE:302 DLOAD:302 EXIT:303`)...)
	s = append(s,
		bas("GUIDANCE", "TTF/8TMP-DAS", fLLGE, 305, 500),
		bas("GUIDANCE", "FASTCHNG", fLLGE, 308, 250),
		bas("GUIDANCE", "LAND-COPY", fLLGE, 314, 400),
		bas("GUIDANCE", "TDISPSET", fLLGE, 325, 900),
		bas("GUIDANCE", "FASTCHNG", fLLGE, 326, 250),
	)

	// --- REDESIG: the landing-site perturbation equations (approach phase
	// only). The WCHPHASE dispatch after TTFINCR (BRSPOT2, L328-L329) sends
	// the approach quadratic through REDESIG — the APPRQUAD table entry at
	// L59 — before falling into RGVGCALC (L408). Every P64 pass pays it:
	// fetch and clear the ELINCR/AZINCR increments, perturb LAND along the
	// line of sight, guard the horizon depression, renormalize |LAND|, and
	// copy LANDTEMP back. LLGE L335-L408.
	if approach {
		s = append(s,
			bas("REDESIG", "REDFLAG-TREDES", fLLGE, 335, 500),
			bas("REDESIG", "ELINCR-FETCH", fLLGE, 344, 700),
		)
		s = append(s, sec("REDESIG", fLLGE, `
			VLOAD:361 VSU:361 RTB:364 NORMUNIT:365 VXV:366 VSL1:366
			VXSC:368 PDDL:368 VXSC:371 VSU:371 VAD:373 PUSH:373
			DLOAD:377 DSU:377 BMN:380 DLOAD:380 STORE:383
			DLOAD:384 DSU:384 DDV:387 VXSC:387 VAD:389 UNIT:389
			VXSC:391 VSL1:391 STORE:393 EXIT:394`)...)
		s = append(s, bas("REDESIG", "FASTCHNG-LANDCOPY", fLLGE, 396, 900))
	}

	// --- RGVGCALC: state into guidance coordinates (P63 skips REDESIG)
	// LLGE L442-L480
	s = append(s, sec("RGVGCALC", fLLGE, `
		VLOAD:443 VXV:443 VAD:446 VSR2:446 STORE:448 MXV:449
		STORE:451 PDDL:452 VDEF:452 ABVAL:454 SL3:454 STORE:455 VLOAD:455
		VSU:457 PUSH:457 MXV:459 VSL1:459 STORE:461 ABVAL:462
		STORE:463 VLOAD:463 RTB:464 NORMUNIT:465 DOT:464 EXIT:467`)...)
	s = append(s,
		bas("RGVGCALC", "PUSHLOC-RESET", fLLGE, 469, 200),
		bas("RGVGCALC", "SPARCSIN-LOOKANGL", fLLGE, 473, 900),
	)

	// --- TTF/8 update: cubic coefficient table + ROOTPSRS Newton root
	// LLGE L489-L527
	s = append(s, bas("TTF/8", "INTPRETX", fLLGE, 489, 200))
	s = append(s, sec("TTF/8", fLLGE, `
		DLOAD:490 STORE:492 DLOAD:492 STORE:494 DLOAD:494 DMP:496 DAD:496
		STORE:499 DLOAD:499 DSU:501 DMP:501 STORE:504 EXIT:505`)...)
	s = append(s,
		bas("TTF/8", "TABLE-SETUP", fLLGE, 507, 400),
		bas("TTF/8", "ROOTPSRS-ITER1", fLLGE, 516, 3900), // Newton on the cubic
		bas("TTF/8", "ROOTPSRS-ITER2", fLLGE, 516, 3900),
		bas("TTF/8", "ROOTPSRS-ITER3", fLLGE, 516, 3900),
		bas("TTF/8", "TDISPSET", fLLGE, 525, 900),
	)

	// --- QUADGUID: the quadratic guidance acceleration command + AFC
	// LLGE L546-L627
	s = append(s, bas("QUADGUID", "COEFFS", fLLGE, 546, 900))
	s = append(s, sec("QUADGUID", fLLGE, `
		VXSC:581 PDDL:581 VXSC:584 PDVL:584 VSU:587 V/SC:587
		VSR2:590 VXSC:590 VAD:592 VAD:592 V/SC:593 VXSC:593
		PDDL:596 VXSC:596 VAD:599
		VXM:600 VSL1:600 PDVL:602 V/SC:602 BVSU:605 STADR:605 STORE:606 ABVAL:607
		STORE:608 DLOAD:608 DSQ:610 PDDL:610 DSQ:612 PDDL:612
		DDV:614 DSQ:614 DSU:616 DSU:616 BPL:617 DLOAD:617
		SQRT:620 DAD:620 BPL:623 BDSU:623 STORE:626 EXIT:627`)...)
	s = append(s, bas("QUADGUID", "FASTCHNG-FLPASS0", fLLGE, 628, 400))

	// --- CGCALC: erect the guidance-stable member matrix (braking exit)
	// LLGE L641-L682
	s = append(s, bas("CGCALC", "EBANK-TTFTEST", fLLGE, 641, 400))
	s = append(s, sec("CGCALC", fLLGE, `
		VLOAD:661 UNIT:661 STORE:663 DLOAD:663 DMP:665 VXSC:665
		VAD:668 VSU:670 RTB:670 NORMUNIT:672 VXV:674 RTB:674 NORMUNIT:676
		STORE:677 VLOAD:677 VXV:679 VSL1:679 STORE:681 EXIT:682`)...)

	// --- EXTLOGIC + EXBRAK + STEER? — the braking-phase exit
	// LLGE L692-L820
	s = append(s, bas("EXITLOGIC", "EXTLOGIC", fLLGE, 692, 300))
	s = append(s, sec("EXITLOGIC", fLLGE, `VLOAD:741 STORE:744 EXIT:745`)...)
	s = append(s, bas("EXITLOGIC", "STEER?", fLLGE, 799, 150))

	// --- THROTTLE: FP/FC computation, region logic, PIF out, FWEIGHT
	// THROTTLE_CONTROL_ROUTINES.agc L37-L187 (basic throughout)
	s = append(s,
		bas("THROTTLE", "FP-FC-MASSMULT", fTHROT, 37, 2400), // two MASSMULT double-multiplies
		bas("THROTTLE", "WHERETO-DOPIF", fTHROT, 67, 700),
		bas("THROTTLE", "DOIT-CHAN14", fTHROT, 124, 400),
		bas("THROTTLE", "FWCOMP", fTHROT, 146, 900),
	)

	// --- FINDCDUW: thrust/window commands into gimbal increments + rates
	// FINDCDUW--GUIDAP_INTERFACE.agc L116-L455
	s = append(s, sec("FINDCDUW", fCDUW, `VLOAD:116 BOV:118 SETPD:118 STQ:121 EXIT:121`)...)
	s = append(s, bas("FINDCDUW", "HAUSKEEPING", fCDUW, 125, 400))
	s = append(s, bas("FINDCDUW", "FETCH-CDUD", fCDUW, 143, 400))
	s = append(s, sec("FINDCDUW", fCDUW, `
		RTB:175 NORMUNIT:177 STORE:178 VLOAD:178 RTB:180 NORMUNIT:181
		RTB:180 EXIT:193`)...)
	s = append(s,
		bas("FINDCDUW", "QUICTRIG-SIN", fCDUW, 182, 2600),
		bas("FINDCDUW", "QUICTRIG-COS", fCDUW, 182, 2600),
		bas("FINDCDUW", "QUICTRIG-TRIM", fCDUW, 182, 2600),
	)
	// MXV:189 is the *SMNB* call operand — the SM↔NB transform charged as
	// the matrix op it dispatches
	s = append(s, sec("FINDCDUW", fCDUW, `
		STORE:183 VLOAD:183 BOVB:185 UNIT:185 BOV:187 CALL:187 MXV:189 EXIT:193`)...)
	s = append(s, bas("FINDCDUW", "FLTRSUB-Y", fCDUW, 195, 700))
	s = append(s, bas("FINDCDUW", "FLTRSUB-Z", fCDUW, 200, 700))
	// AFTRFLTR: X-axis override permitted in P63 → FETCHZNB path + UNWCTEST
	s = append(s, sec("FINDCDUW", fCDUW, `
		SLOAD:210 BHIZ:210 VLOAD:217 STORE:219 CALL:219
		DOT:478 DSQ:478 DSU:480 BMN:480 RVQ:483`)...)
	// DCMCL: the direction cosine matrix
	s = append(s, sec("FINDCDUW", fCDUW, `
		VLOAD:228 VXV:228 UNIT:231 PUSH:231 VXV:232 VSL1:232
		STORE:234 VXSC:235 PDVL:235 VXSC:237 BVSU:237 VSL1:239 VAD:239
		UNIT:241 STORE:242 VXV:243 VSL1:243 STORE:245 VCOMP:246 VXV:246
		VSL1:248 STORE:249 CALL:254`)...)
	// NB2CDUSP: required gimbal angles (SQRT + three ARCTRGSP)
	s = append(s, sec("FINDCDUW", fCDUW, `
		DLOAD:492 DSQ:492 BDSU:494 BPL:494 SQRT:499 EXIT:499`)...)
	s = append(s,
		bas("FINDCDUW", "ARCTRGSP-CDUZ", fCDUW, 512, 900),
		bas("FINDCDUW", "DVBYCOSM+ARCTRGSP-CDUY", fCDUW, 515, 1300),
		bas("FINDCDUW", "DVBYCOSM+ARCTRGSP-CDUX", fCDUW, 523, 1300),
		bas("FINDCDUW", "MGA-LIMIT", fCDUW, 260, 400),
		bas("FINDCDUW", "DELGMBLP-3PASS", fCDUW, 275, 800),
		bas("FINDCDUW", "ATT-LIMIT", fCDUW, 324, 1000),
		bas("FINDCDUW", "OMEGA-RATES", fCDUW, 375, 900),
		bas("FINDCDUW", "CDUWXFR-3PASS", fCDUW, 415, 1300),
	)
	s = append(s, sec("FINDCDUW", fCDUW, `SETPD:452 GOTO:452`)...)

	// --- phase-table and executive bookkeeping spread across the pass:
	// QUIKFAZ5/GNUFAZ5 restart-protection updates, bank switches, and the
	// interpreter's pushdown housekeeping (calibration residue; see
	// msim/RESEARCH.md)
	for left := execResidueUS; left > 0; left -= 4000 {
		c := left
		if c > 4000 {
			c = 4000
		}
		s = append(s, bas("EXEC", "PHASE-BOOKKEEPING", fSERV, 270, c))
	}

	// --- DISPEXIT: kill group 3, spawn the P63 display job (V06N63,
	// static → NOVAC MAKEPLAY), then ENDOFJOB. LLGE L835-L854
	s = append(s,
		bas("DISPEXIT", "GROUP3-KILL", fLLGE, 835, 250),
		bas("DISPEXIT", "FLUNDISP-TEST", fLLGE, 839, 150),
		bas("DISPEXIT", "P63DISPS-REGODSPR", fLLGE, 850, 600),
	)

	return s
}

// execResidueUS is the pass's phase-table/bank-switch residue in µs — the
// calibration constant that pins the pass total inside Cherry's band
// (see msim/RESEARCH.md).
var execResidueUS = 36_000

// ServicerScript returns the per-cycle SERVICER instruction array for a
// descent phase.
func ServicerScript(p Phase) (Script, error) {
	switch p {
	case P63Prelock:
		return servicerPass(false, false), nil
	case P63Locked:
		return servicerPass(true, false), nil
	case P64Approach:
		return servicerPass(true, true), nil
	default:
		return nil, fmt.Errorf("msim: no SERVICER transcription for phase %d", int(p))
	}
}

// Hooks are the scenario's spawn effects, attached at their real positions
// in the pass: Pipa at the ENTRY 1/PIPA gyro-compensation gate, LRV at the
// end of the R12 radar section (VMEASCHK/VALTCHK, SERVICER.agc L1251-L1255,
// launch the velocity read from within SERVICER — under a backlog this
// window drifts with the pass), Dispexit at the P63DISPS display request.
type Hooks struct {
	Pipa     func(*Engine)
	LRV      func(*Engine)
	Dispexit func(*Engine)
}

// WithHooks copies the script, attaching the scenario's spawn effects.
func (s Script) WithHooks(h Hooks) Script {
	out := make(Script, len(s))
	copy(out, s)
	lastLR := -1
	for i := range out {
		if out[i].Section == "LR-CONVERT" {
			lastLR = i
		}
		switch {
		case strings.HasSuffix(out[i].Op, "1/PIPA-GCOMP"):
			out[i].Then = h.Pipa
		case strings.HasSuffix(out[i].Op, "P63DISPS-REGODSPR"):
			out[i].Then = h.Dispexit
		}
	}
	if lastLR >= 0 {
		out[lastLR].Then = h.LRV
	}
	return out
}
