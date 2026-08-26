// Package danzig is the Executive job-picker as a handful of lines of
// pseudocode: FINDVAC packs class|VAC into the PRIORITY word, DANZIG
// swaps to NEWJOB between Interpreter opcodes, EJSCAN rescans for the
// largest PRIORITY word. Two "priority 20" SERVICERs are therefore
// not equal — the newer copy's higher VAC address wins, and the old
// copy starves on its still-allocated core set and VAC.
package danzig

// Source is the picker: jobs live in core sets, the PRIORITY word is
// class|VAC-addr, NEWJOB is the winning core-set offset. Assembly
// mnemonics stay out of it.
const Source = `# PRIORITY word = class | VAC-addr
# lives in a core set (0 12 24 ... 84)
# NEWJOB = that offset. core 0 = CPU
# S0/VAC1=20401  S1/VAC2=20455

FINDVAC(job, class):
    vac  = first free VAC        # 1201
    core = first free core set   # 1202
    job.PRIORITY = class | vac
    SETLOC(job)

SETLOC(job):
    if job.PRIORITY > cores[NEWJOB].PRIORITY:
        NEWJOB = job.core        # whole word

HASNEWJOB:                       # DANZIG
    if NEWJOB != 0:
        swap cores[0], cores[NEWJOB]
        run cores[0]

EJSCAN:                          # ENDOFJOB
    best = 0
    for off in 12,24,36,48,60,72,84:
        w = cores[off].PRIORITY
        if w <= 0: continue      # free / asleep
        if w > best:
            best, NEWJOB = w, off
    HASNEWJOB`

// Title sits on the card above the source.
const Title = "HOW THE EXEC PICKS A JOB"

// Kind is one syntax class the highlighter paints.
type Kind int

const (
	KindSpace Kind = iota
	KindComment
	KindKeyword
	KindLabel
	KindIdent
	KindNumber
	KindOp
)

// Token is one lossless piece of a source line.
type Token struct {
	Kind Kind
	Text string
}

// Rose Pine (https://rosepinetheme.com) as truecolor and the nearest
// xterm-256 indexes the sprite card can actually paint.
const (
	BaseHex    = "#191724"
	MutedHex   = "#6e6a86"
	TextHex    = "#e0def4"
	GoldHex    = "#f6c177"
	FoamHex    = "#9ccfd8"
	IrisHex    = "#c4a7e7"
	RoseHex    = "#ebbcba"
	OverlayHex = "#26233a"

	Base256    = 235
	Muted256   = 103
	Text256    = 189
	Gold256    = 222
	Foam256    = 152
	Iris256    = 183
	Rose256    = 181
	Overlay256 = 237
)
