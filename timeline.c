/*
 * timeline.c — C telling of Apollo-11/timeline.md
 *
 * Same causal chain as the MEMORY_LEAK1..9 annotations, abstracted so you can
 * talk through the freeze without reading AGC assembly.
 *
 *   gcc -o timeline timeline.c && ./timeline
 *
 * What is real vs invented:
 *   REAL:     8 core sets, 5 VAC areas, prio 20 SERVICER, fixed 2.00 s re-arm,
 *             FINDVAC claims a NEW pair, ENDOFJOB is the only release,
 *             1201 = no VAC, 1202 = no core set, radar steals time not memory.
 *   INVENTED: function bodies for nav/guidance/math; wall-clock bookkeeping.
 */

#include <stdio.h>
#include <stdbool.h>
#include <string.h>

/* ---------- the pool (erasable scratchpad) ---------- */

#define N_CORES 8
#define N_VACS  5
#define PRIO_SERVICER 20

/* Wall-clock model (seconds). Radar steals ~15% of CPU; does NOT allocate. */
#define PERIOD_S          2.00
#define SERVICER_CPU_NEED 1.80   /* work this copy needs if it got the whole CPU */

/*
 * One 2.00s period — all actors (Cherry job table + RR ECDU).
 * Seconds are illustrative but sum to PERIOD_S and keep SERVICER short of its ~1.80s need.
 */
typedef struct {
    const char *name;
    int         prio;      /* 0 = hardware */
    const char *kind;      /* HW / NOVAC / VAC */
    const char *cadence;
    double      cpu_s;
} LandingJob;

static const LandingJob LANDING_JOBS[] = {
    { "RR ECDU",  0,  "HW",    "PINC/MINC up to 12.8k/s @ CDUT+CDUS", 0.300 },
    { "LRHJOB",   32, "NOVAC", "~1ms run, ~80ms sleep (LR altitude)",  0.080 },
    { "LRVJOB",   32, "NOVAC", "short run / ~500ms sleep (LR vel)",    0.050 },
    { "HIGATJOB", 32, "VAC",   "once near High Gate (antenna pos2)",  0.010 },
    { "CHARIN",   30, "NOVAC", "each DSKY keystroke",                 0.040 },
    { "MONDO",    30, "NOVAC", "while V16 N68 monitor is up (tip)",    0.120 },
    { "1/GYRO",   21, "NOVAC", "IMU gyro compensation",               0.100 },
    { "MAKEPLAY", 20, "VAC",   "display job set up by SERVICER",      0.100 },
    { "SERVICER", 20, "VAC",   "every 2.00s via READACCS",            1.200 },
};
static const int N_LANDING_JOBS = (int)(sizeof LANDING_JOBS / sizeof LANDING_JOBS[0]);

static double servicer_leftover_s(void)
{
    return LANDING_JOBS[N_LANDING_JOBS - 1].cpu_s; /* SERVICER row */
}

static void print_landing_cast(void)
{
    printf("\n######## LANDING CAST (one 2.00s period) ########\n");
    printf("%-8s %4s %-5s %6s  %s\n", "JOB", "PRIO", "KIND", "CPU", "CADENCE");
    double sum = 0;
    for (int i = 0; i < N_LANDING_JOBS; i++) {
        const LandingJob *j = &LANDING_JOBS[i];
        if (j->prio)
            printf("%-8s %4d %-5s %5.2fs  %s\n",
                   j->name, j->prio, j->kind, j->cpu_s, j->cadence);
        else
            printf("%-8s %4s %-5s %5.2fs  %s\n",
                   j->name, "HW", j->kind, j->cpu_s, j->cadence);
        sum += j->cpu_s;
    }
    printf("-------- ---- ----- ------\n");
    printf("%-8s %4s %-5s %5.2fs\n", "TOTAL", "", "", sum);
    printf("SERVICER needs %.2fs but only gets %.2fs leftovers "
           "→ misses ENDOFJOB\n",
           SERVICER_CPU_NEED, servicer_leftover_s());
}

static void run_higher_prio_jobs(void)
{
    /* Invented stand-ins — preempt SERVICER; short or chunky as noted. */
    printf("  [prio 32] LRHJOB   altitude sample\n");
    printf("  [prio 32] LRVJOB   velocity sample\n");
    printf("  [prio 32] HIGATJOB (idle / brief if not High Gate)\n");
    printf("  [prio 30] CHARIN   keystroke\n");
    printf("  [prio 30] MONDO    V16 N68 monitor refresh\n");
    printf("  [prio 21] 1/GYRO   gyro compensation\n");
    printf("  [prio 20] MAKEPLAY display\n");
}

typedef struct {
    bool busy;
    int  owner;      /* SERVICER copy id, or -1 */
    int  priority;   /* 0 = free (−0 in the real machine) */
    int  vac_index;  /* which VAC is packed into this job, or -1 */
} CoreSet;

typedef struct {
    bool busy;       /* VACnUSE == 0 in the real machine */
    int  owner;
} VacArea;

typedef struct {
    CoreSet cores[N_CORES];
    VacArea vacs[N_VACS];
    int     next_copy_id;   /* S0, S1, S2… */
    double  wall_time;
    int     alarm;          /* 0, 1201, or 1202 */
} Executive;

/* One running/stub SERVICER instance. */
typedef struct {
    int    id;
    int    core;
    int    vac;
    double cpu_done;     /* how much of SERVICER_CPU_NEED finished */
    bool   finished;
} ServicerJob;

#define MAX_JOBS 16
static ServicerJob jobs[MAX_JOBS];
static int         n_jobs;

/* ---------- board printer (matches timeline.md ASCII) ---------- */

static void print_board(const Executive *e, const char *caption)
{
    printf("\n--- %s ---\n", caption);
    printf("CORE  ");
    for (int i = 0; i < N_CORES; i++) {
        if (!e->cores[i].busy) printf(" · ");
        else                   printf("S%-2d", e->cores[i].owner);
    }
    printf("\nVAC   ");
    for (int i = 0; i < N_VACS; i++) {
        if (!e->vacs[i].busy) printf(" · ");
        else                  printf("S%-2d", e->vacs[i].owner);
    }
    printf("\n");
}

/* ---------- MEMORY_LEAK1 — GOREADAX: re-arm for exactly 2.00 s ---------- */
/*
 * Real: CA 2SECS / TC VARDELAY. Never asks "is the old SERVICER done?"
 * Memory moved: none.
 */
static double goreadax_schedule_next_readaccs(double now)
{
    printf("\n[MEMORY_LEAK1] GOREADAX: arm READACCS at t+%.2fs "
           "(no check that old SERVICER finished)\n", PERIOD_S);
    return now + PERIOD_S;   /* unconditional */
}

/* ---------- MEMORY_LEAK2 — T3RUPT: timer fires on time ---------- */
/*
 * Real: TIME3 is a hardware counter. Radar stole instruction cycles at
 * CDUT/CDUS, but did not slow this clock. Demand stays one cycle / 2.00 s
 * while supply is only ~85% CPU.
 * Memory moved: none in the core/VAC pools.
 */
static void t3rupt_dispatch(double due_at, double wall)
{
    printf("\n[MEMORY_LEAK2] T3RUPT: TIME3 overflow at t=%.2fs "
           "(punctual; radar theft does not delay the timer)\n", due_at);
    printf("               RR ECDU still stealing ~%.2fs/period; "
           "demand still 1 cycle / %.2fs\n",
           LANDING_JOBS[0].cpu_s, PERIOD_S);
    (void)wall;
}

/* ---------- MEMORY_LEAK3 — READACCS: short task, start a cycle ---------- */
/*
 * Real: read PIPAs, then ask Executive for a new SERVICER.
 * Memory moved: none yet (this is a task, not a VAC job).
 */
static void readaccs_read_pipas(void)
{
    printf("\n[MEMORY_LEAK3] READACCS: read accelerometer counters (PIPASR)\n");
    /* invent: capture delta-V into a buffer the job will use */
}

/* ---------- MEMORY_LEAK5 — find_vac: claim or 1201 ---------- */
static int find_vac(Executive *e, int owner)
{
    /* Same shape as the real CCS VAC1USE … VAC5USE fall-through. */
    if (!e->vacs[0].busy) { e->vacs[0].busy = true; e->vacs[0].owner = owner;
        printf("[MEMORY_LEAK5] vac_1 ← S%d\n", owner); return 0; }
    if (!e->vacs[1].busy) { e->vacs[1].busy = true; e->vacs[1].owner = owner;
        printf("[MEMORY_LEAK5] vac_2 ← S%d\n", owner); return 1; }
    if (!e->vacs[2].busy) { e->vacs[2].busy = true; e->vacs[2].owner = owner;
        printf("[MEMORY_LEAK5] vac_3 ← S%d\n", owner); return 2; }
    if (!e->vacs[3].busy) { e->vacs[3].busy = true; e->vacs[3].owner = owner;
        printf("[MEMORY_LEAK5] vac_4 ← S%d\n", owner); return 3; }
    if (!e->vacs[4].busy) { e->vacs[4].busy = true; e->vacs[4].owner = owner;
        printf("[MEMORY_LEAK5] vac_5 ← S%d\n", owner); return 4; }
    return 1201;
}

/* ---------- MEMORY_LEAK6 — find_core: claim or 1202 ---------- */
static int find_core(Executive *e, int owner, int vac, int prio)
{
    for (int i = 0; i < N_CORES; i++) {
        if (!e->cores[i].busy) {
            e->cores[i].busy = true;
            e->cores[i].owner = owner;
            e->cores[i].priority = prio;
            e->cores[i].vac_index = vac;
            printf("[MEMORY_LEAK6] core_%d ← S%d\n", i + 1, owner);
            return i;
        }
    }
    return 1202;
}

/* ---------- MEMORY_LEAK4 — start a brand-new SERVICER ---------- */
static int start_servicer(Executive *e)
{
    int id = e->next_copy_id++;
    printf("\n[MEMORY_LEAK4] start_job(SERVICER) → S%d  (new copy)\n", id);

    int vac = find_vac(e, id);
    if (vac == 1201) {
        e->alarm = 1201;
        printf("*** ALARM 1201 — no VAC areas ***\n");
        return -1;
    }

    int core = find_core(e, id, vac, PRIO_SERVICER);
    if (core == 1202) {
        e->vacs[vac].busy = false;
        e->vacs[vac].owner = -1;
        e->alarm = 1202;
        printf("*** ALARM 1202 — no core sets ***\n");
        return -1;
    }

    if (n_jobs >= MAX_JOBS) return -1;
    jobs[n_jobs++] = (ServicerJob){
        .id = id, .core = core, .vac = vac,
        .cpu_done = 0.0, .finished = false,
    };
    return id;
}

/* Invented stand-ins for the real Interpretive / guidance work. */
static void average_g_navigation(void) { /* update RN, VN from PIPAs */ }
static void guidance_equations(void)   { /* P63/P64 aiming */ }
static void throttle_and_dap(void)     { /* engine + attitude cmds */ }
static void update_displays(void)      { /* DSKY nouns */ }

/* ---------- MEMORY_LEAK8 — finish line ---------- */
static bool servexit_ready(const ServicerJob *j)
{
    bool ready = j->cpu_done >= SERVICER_CPU_NEED;
    printf("\n[MEMORY_LEAK8] S%d  %.2f / %.2fs%s\n",
           j->id, j->cpu_done, SERVICER_CPU_NEED,
           ready ? " → end_of_job()"
                 : " → still running (keeps vac+core)");
    return ready;
}

/* ---------- MEMORY_LEAK9 — the only release ---------- */
static void end_of_job(Executive *e, ServicerJob *j)
{
    printf("\n[MEMORY_LEAK9] end_of_job(S%d): core_%d + vac_%d free\n",
           j->id, j->core + 1, j->vac + 1);

    e->cores[j->core].busy = false;
    e->cores[j->core].owner = -1;
    e->cores[j->core].priority = 0;
    e->cores[j->core].vac_index = -1;

    e->vacs[j->vac].busy = false;
    e->vacs[j->vac].owner = -1;

    j->finished = true;
}

/* ---------- restart (what the crew felt as the brief freeze) ---------- */

static void bailout_restart(Executive *e)
{
    printf("\n[RESTART] BAILOUT → PROG light, FAILREG=%d, wipe stubs, "
           "rebuild one READACCS + one SERVICER\n", e->alarm);
    memset(e->cores, 0, sizeof e->cores);
    memset(e->vacs, 0, sizeof e->vacs);
    for (int i = 0; i < N_CORES; i++) e->cores[i].owner = -1;
    for (int i = 0; i < N_VACS;  i++) e->vacs[i].owner  = -1;
    n_jobs = 0;
    e->next_copy_id = 0;
    e->alarm = 0;
    /* Invented: navigation / engine / PIPAs kept counting through this. */
}

/* ---------- drive the timeline ---------- */

static void init_exec(Executive *e)
{
    memset(e, 0, sizeof *e);
    for (int i = 0; i < N_CORES; i++) e->cores[i].owner = -1;
    for (int i = 0; i < N_VACS;  i++) e->vacs[i].owner  = -1;
    n_jobs = 0;
}

/* Advance unfinished jobs by one period's worth of wall time.
 *
 * Important: all stubs share the leftover CPU. As more copies pile up, each
 * one gets less done — so under overload they never reach ENDOFJOB before
 * the next FINDVAC. That is the whole leak.
 */
static void work_unfinished(Executive *e, double wall_budget)
{
    int alive = 0;
    for (int i = 0; i < n_jobs; i++)
        if (!jobs[i].finished) alive++;
    if (alive == 0) return;

    /* One shared leftover slice = what SERVICER actually gets in the cast. */
    double total_for_servicers = servicer_leftover_s() *
        (wall_budget / PERIOD_S);
    double each = total_for_servicers / alive;

    printf("\n[MEMORY_LEAK7] higher-prio cast runs first, then SERVICER leftovers\n");
    run_higher_prio_jobs();
    printf("  [HW]      RR ECDU continuous PINC/MINC (~%.2fs this period)\n",
           LANDING_JOBS[0].cpu_s * (wall_budget / PERIOD_S));
    printf("  %d unfinished SERVICER(s) share ~%.2fs leftover "
           "(~%.2fs each) — need %.2fs each to finish\n",
           alive, total_for_servicers, each, SERVICER_CPU_NEED);

    for (int i = 0; i < n_jobs; i++) {
        if (jobs[i].finished) continue;

        printf("  S%d: avg-G → guidance → throttle/DAP → displays\n",
               jobs[i].id);
        average_g_navigation();
        guidance_equations();
        throttle_and_dap();
        update_displays();

        jobs[i].cpu_done += each;
        if (servexit_ready(&jobs[i]))
            end_of_job(e, &jobs[i]);
    }
}

int main(void)
{
    Executive e;
    init_exec(&e);

    printf("====================================================\n");
    printf(" Apollo 11 LGC — MEMORY_LEAK timeline (C sketch)\n");
    printf(" Companion to timeline.md\n");
    printf("====================================================\n");
    printf("Pool: %d core sets × 12 words, %d VAC areas × 44 words\n",
           N_CORES, N_VACS);
    printf("Each unfinished SERVICER holds ~55 words until ENDOFJOB.\n");

    print_landing_cast();

    /* ---- T=0 baseline: one healthy cycle that WOULD finish if alone ---- */
    printf("\n######## T=0 BASELINE ########\n");
    start_servicer(&e);                 /* S0 */
    print_board(&e, "baseline: S0 owns one core + one VAC");

    /*
     * OVERLOAD MODE: each 2s period we:
     *   1) re-arm (LEAK1)
     *   2) timer fires (LEAK2)
     *   3) READACCS (LEAK3)
     *   4-6) allocate a NEW copy (LEAK4-6) while old ones still hold memory
     *   7-8) try to run; often miss ENDOFJOB (LEAK7-8)
     *   9) release only if finished (LEAK9) — usually loses the race
     *
     * Give each unfinished job only a fraction of PERIOD_S of wall time
     * before the next allocation — enough that they cannot finish.
     */
    printf("\n######## OVERLOAD: V16N68 + radar theft → demand > 2.00s ########\n");
    printf("SERVICER needs ~%.2fs of CPU; each period only yields ~%.2fs to it.\n",
           SERVICER_CPU_NEED, servicer_leftover_s());

    double next_due = goreadax_schedule_next_readaccs(0.0);

    for (int cycle = 0; cycle < 8; cycle++) {
        e.wall_time = next_due;
        printf("\n======== cycle %d  wall t=%.2fs ========\n", cycle + 1, e.wall_time);

        /* LEAK2 + LEAK3 */
        t3rupt_dispatch(next_due, e.wall_time);
        readaccs_read_pipas();

        /* LEAK4 → LEAK5 → LEAK6: brand-new copy while stubs still hold slots */
        int id = start_servicer(&e);
        if (id < 0) {
            print_board(&e, "pool exhausted — freeze / alarm");
            bailout_restart(&e);
            print_board(&e, "after restart: desk cleared");
            break;
        }
        print_board(&e, "after FINDVAC (old stubs still hold their pairs)");

        /*
         * Give unfinished jobs the rest of the period — shared across ALL
         * stubs. With several copies alive, nobody reaches SERVICER_CPU_NEED
         * before the next GOREADAX. MEMORY_LEAK8 loses; MEMORY_LEAK9 never runs.
         */
        work_unfinished(&e, PERIOD_S);
        print_board(&e, "after partial run (finish line often missed)");

        /* LEAK1: arm the next cycle regardless */
        next_due = goreadax_schedule_next_readaccs(e.wall_time);
    }

    printf("\n======== one-page causal chain ========\n");
    printf("radar steals TIME @ CDUT/CDUS\n");
    printf("  → TIME3 still fires every 2.00s (LEAK2)\n");
    printf("  → READACCS always FINDVACs a NEW SERVICER (LEAK4)\n");
    printf("  → VACFOUND + CORFOUND claim another 55 words (LEAK5/6)\n");
    printf("  → old copy still below SERVEXIT (LEAK8)\n");
    printf("  → ENDOFJOB never runs for stubs (LEAK9 skipped)\n");
    printf("  → cores/VACs fill → 1201/1202 → restart\n");

    return 0;
}
