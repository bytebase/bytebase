package mysql

// A8 of the ENTERPRISE smoke axes (see sdl_enterprise_test.go): CROSS-VERSION STATEFUL
// FUZZ — the composition of A6 (same-version multi-round stateful fuzz) and A7
// (single-shot cross-version upgrade). Real teams run MIXED fleets: a schema authored
// and dumped on 5.7 drives targets applied to 8.0 (and vice versa during migrations),
// repeatedly, while the schema keeps evolving. A6 and A7 both pass in isolation; this
// axis hunts what only their composition shows — version-normalization drift
// COMPOUNDING across rounds (a residual one round becomes the authored text of the
// next, so a per-round phantom that a single-shot test shrugs off snowballs here).
//
// Protocol per seed, on the 5.7-legal A6 base (30-table Zabbix slice + fuzz aux +
// stateful aux; the 8.0-only aux tables are excluded — the SAME logical schema must
// load on BOTH versions), both databases collation-aligned (utf8mb4/utf8mb4_general_ci,
// the A7 aligned-probe posture) so the fleets start logically identical:
//
//	round 0: the two fresh dumps must already cross-diff EMPTY (all four
//	         direction x version-normalization combos).
//	for round r in 1..4:
//	    T_r = mutate(authored side's dump_{r-1}, K ∈ [2,4] from the 5.7-LEGAL menu)
//	    each side s ∈ {5.7, 8.0}: plan_s = Diff(dump_s, T_r, full server version) must
//	        be non-empty, apply cleanly, converge (re-dump diffs empty against T_r
//	        under s's version), self-diff empty, reload through STRICT LoadSDL, and
//	        match the mutation ledger's object counts (per-side ledgers: baselines may
//	        differ, deltas are shared);
//	    plan_80 must carry no 5.7 integer display width in any ALTER (entUpgAssert-
//	        NoWidthInAlters — a leaked width is a phantom column MODIFY);
//	    CROSS-CHECK: the two fresh dumps must cross-diff EMPTY in all four combos —
//	        the fleets stay logically identical despite version-different stored forms;
//	    ACCRETION (A6's guards, on BOTH sides): each side's dump stays under its own
//	        dump_0 length + the cumulative added-object budget, and every statement not
//	        named by this round's mutations is byte-identical to its round-start form.
//
// forward_57_authored seeds mutate the 5.7 dump; reverse_80_authored seeds mutate the
// 8.0 dump (the authored text then carries NO display widths while the 5.7 side's
// stored form does — the opposite normalization direction). Both directions use the
// 5.7-legal menu: every target applies to both fleets, so 5.7-illegal mutations (CHECK,
// SRID spatial, INVISIBLE) are structurally excluded. Additionally, child-side FK
// member columns are protected from the generated-column kinds — both servers refuse
// a STORED generated column based on a cascading-FK member (see
// entA8ProtectFKMemberColumns for the empirical matrix this axis established).
//
// Determinism: the mutation stream is a pure function of the seed (in the subtest
// name). Failures ddmin-minimize over a cross-version trial (both round-start dumps
// reloaded into fresh aligned scratch databases) and report seed + round + the failing
// check (apply/converge/idempotence/reload/counts per side, or the cross combo) + the
// minimized ledger. Accretion-guard failures are terminal requires with the same
// seed/round/check labeling. Scratch databases are entsdl_-prefixed and dropped.
//
// ≥8 forward seeds + ≥4 reverse seeds.
//
// Shared machinery: entSfBaseDDL, entSfParse, entSfMenu, entSfMutationsFromMenu,
// entSfRoundToTarget, entSfMinimizeWith, entSfScratchDB, entSfCountObjects,
// entSfAssertUntouchedStable, entSfDescs, entSfLedger, entSfRoundSlack (A6);
// entUpgServers, entUpgLoadAligned, entUpgAssertNoWidthInAlters, entUpgTrim,
// entUpg57Version, entUpg80Version (A7); liveServers, dumpSDL, syncMetaForDB,
// newLiveDatabase, createLiveMySQLDriver, entNormalizeDelimiters (siblings).

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
)

const (
	entA8Rounds       = 4
	entA8ForwardSeeds = 8
	entA8ReverseSeeds = 4
)

// entA8Fleet is one side of the mixed-version fleet.
type entA8Fleet struct {
	srv liveServer
	// version is the FULL server version threaded into every diff for this side (the
	// release path passes the synced version string, not a bare major.minor).
	version string
	dbName  string
	prev    string // canonical dump at the current round boundary
	dump0   int    // round-0 dump length (text-accretion baseline)
	budget  int    // cumulative added-object text budget
	expect  entSfCounts
}

// entA8CrossCheck proves the two fleets are logically identical: all four
// (direction x version-normalization) cross-diffs of the two canonical dumps must be
// empty. The two "apply" pairings mirror A7's aligned probe — the plan a fleet would
// receive to be driven to the OTHER fleet's schema; the remaining two probe the
// canonicalizer's version symmetry (the same pair re-judged under the other version's
// stored form).
func entA8CrossCheck(d57, d80 string) error {
	for _, c := range []struct{ label, source, target, version string }{
		{"80db<=57dump(as 8.0.32)", d80, d57, entUpg80Version},
		{"57db<=80dump(as 5.7.25)", d57, d80, entUpg57Version},
		{"57src->80tgt(as 8.0.32)", d57, d80, entUpg80Version},
		{"80src->57tgt(as 5.7.25)", d80, d57, entUpg57Version},
	} {
		plan, err := mysqlDiffSDLMigration(c.source, c.target, c.version)
		if err != nil {
			return errors.Wrapf(err, "check=cross %s: diff failed", c.label)
		}
		if strings.TrimSpace(plan) != "" {
			return errors.Errorf("check=cross %s: fleets drifted apart; residual plan:\n%s", c.label, entUpgTrim(plan))
		}
	}
	return nil
}

// entA8ProtectFKMemberColumns marks child-side foreign-key member columns as protected
// in the mutation-planning model, keeping the A8 menu off a server-refused shape this
// axis found empirically: BOTH oracle servers reject a STORED generated column whose
// base column participates in a foreign key with a cascading referential action —
// 8.0.32 with a clean 1215 "Cannot add foreign key constraint" at DDL time, 5.7.25
// with an opaque errno-150 failure at ALGORITHM=COPY rename (the shape seed 18 of a
// widened sweep minimized to: ADD COLUMN ... GENERATED ALWAYS AS ((groupid + 7)) STORED
// on hosts_groups, whose groupid carries zabbix's usual ON DELETE CASCADE FK).
// Parent-side referenced columns, VIRTUAL generated columns, and RESTRICT/NO ACTION
// FKs are all accepted by both servers and stay eligible. The guard is
// action-agnostic (membership, not action): zabbix FKs are almost all
// ON DELETE CASCADE and the add_fk mutation plants ON DELETE SET NULL, so an
// action-aware refinement would buy nothing here.
func entA8ProtectFKMemberColumns(m *entSfSchema) {
	for _, tbl := range m.base.tables {
		for _, fk := range tbl.fks {
			for _, c := range fk.cols {
				m.protected[tbl.name+"."+c] = true
			}
		}
	}
}

// entA8Round drives one shared target through both fleets and cross-checks the
// results: each side runs the full A6 round oracle (diff -> apply -> converge ->
// idempotence -> strict LoadSDL reload -> ledger counts) under its own version, then
// the two fresh dumps must cross-diff empty. Protocol violations return stage-tagged
// errors (for ddmin); infrastructure failures fail t hard. On success both fleets'
// prev/expect/budget are advanced and the 8.0-side plan is returned for the width
// guard (the 5.7-side plan may carry display widths legitimately — they ARE its
// stored form — so nothing judges it beyond the round oracle).
func entA8Round(ctx context.Context, t *testing.T, f57, f80 *entA8Fleet, target string, muts []entSfMutation) (string, error) {
	t.Helper()
	want57, want80 := f57.expect, f80.expect
	for _, m := range muts {
		want57 = want57.plus(m.delta)
		want80 = want80.plus(m.delta)
	}
	after57, _, err := entSfRoundToTarget(ctx, t, f57.srv, f57.dbName, f57.prev, target, f57.version, want57)
	if err != nil {
		return "", errors.Wrapf(err, "check=side %s", f57.srv.name)
	}
	after80, plan80, err := entSfRoundToTarget(ctx, t, f80.srv, f80.dbName, f80.prev, target, f80.version, want80)
	if err != nil {
		return "", errors.Wrapf(err, "check=side %s", f80.srv.name)
	}
	if err := entA8CrossCheck(after57, after80); err != nil {
		return "", err
	}
	for _, m := range muts {
		f57.budget += m.budget
		f80.budget += m.budget
	}
	f57.prev, f57.expect = after57, want57
	f80.prev, f80.expect = after80, want80
	return plan80, nil
}

// entA8ScratchFromDump reloads a canonical dump into a fresh, eagerly-droppable,
// collation-aligned scratch database. The dump text executes with FOREIGN_KEY_CHECKS
// off for the session (canonical dumps order tables alphabetically, so inline foreign
// keys may forward-reference; the driver runs the whole batch on one connection, so
// the session setting holds).
func entA8ScratchFromDump(ctx context.Context, t *testing.T, srv liveServer, dump string) (string, func()) {
	t.Helper()
	dbName, drop := entSfScratchDB(ctx, t, srv, "entsdl_a8t")
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, "ALTER DATABASE `"+dbName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", db.ExecuteOptions{})
	require.NoError(t, err, "[%s] align trial database collation", srv.name)
	_, err = driver.Execute(ctx, "SET FOREIGN_KEY_CHECKS=0;\n"+entNormalizeDelimiters(dump), db.ExecuteOptions{})
	require.NoError(t, err, "[%s] reload round-start dump into trial database", srv.name)
	return dbName, drop
}

// entA8Trial reproduces one cross-version round from scratch for ddmin: both
// round-start dumps reload into fresh aligned scratch databases, the mutation subset
// re-authors the target from the authored side's dump, and the full round runs.
func entA8Trial(ctx context.Context, t *testing.T, srv57, srv80 liveServer, prev57, prev80 string, authored80 bool, muts []entSfMutation) error {
	t.Helper()
	db57, drop57 := entA8ScratchFromDump(ctx, t, srv57, prev57)
	defer drop57()
	db80, drop80 := entA8ScratchFromDump(ctx, t, srv80, prev80)
	defer drop80()

	f57 := &entA8Fleet{srv: srv57, version: entUpg57Version, dbName: db57}
	f80 := &entA8Fleet{srv: srv80, version: entUpg80Version, dbName: db80}
	for _, f := range []*entA8Fleet{f57, f80} {
		f.prev = dumpSDL(ctx, t, f.srv, f.dbName)
		f.expect = entSfCountObjects(syncMetaForDB(ctx, t, f.srv, f.dbName))
	}
	authored := f57.prev
	if authored80 {
		authored = f80.prev
	}
	target := authored
	for _, m := range muts {
		target = m.apply(t, target)
	}
	_, err := entA8Round(ctx, t, f57, f80, target, muts)
	return err
}

// entA8RunSeed runs one full cross-version stateful seed: 4 rounds of
// mutate-on-the-authored-side -> apply-to-both-fleets -> cross-check -> accretion
// guards on both sides.
func entA8RunSeed(ctx context.Context, t *testing.T, srv57, srv80 liveServer, baseDDL string, seed int64, authored80 bool) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed)) //nolint:gosec // deterministic fuzz, not crypto

	f57 := &entA8Fleet{srv: srv57, version: entUpg57Version}
	f80 := &entA8Fleet{srv: srv80, version: entUpg80Version}
	f57.dbName = entUpgLoadAligned(ctx, t, srv57, "entsdl_a8m57", baseDDL)
	f80.dbName = entUpgLoadAligned(ctx, t, srv80, "entsdl_a8m80", baseDDL)
	for _, f := range []*entA8Fleet{f57, f80} {
		f.prev = dumpSDL(ctx, t, f.srv, f.dbName)
		f.dump0 = len(f.prev)
		f.expect = entSfCountObjects(syncMetaForDB(ctx, t, f.srv, f.dbName))
	}
	dir, author := "fwd57", f57
	if authored80 {
		dir, author = "rev80", f80
	}

	// Round 0: freshly loaded from the same DDL under aligned defaults, the fleets must
	// already be logically identical — a failure here is a base cross-version phantom,
	// not mutation drift.
	require.NoError(t, entA8CrossCheck(f57.prev, f80.prev),
		"[A8 %s seed=%d round=0] fleets diverge before any mutation (base cross-check)", dir, seed)

	ledger := &entSfLedger{viewModified: map[string]bool{}}
	var history []string
	for r := 1; r <= entA8Rounds; r++ {
		m := entSfParse(t, author.prev)
		entA8ProtectFKMemberColumns(m)
		k := 2 + rng.Intn(3) // K ∈ [2,4]
		muts := entSfMutationsFromMenu(t, m, rng, entSfMenu("5.7"), ledger, r, k)
		for _, mut := range muts {
			history = append(history, fmt.Sprintf("r%d: %s", r, mut.desc))
		}
		t.Logf("[A8 %s seed=%d round=%d] K=%d mutations:\n  %s",
			dir, seed, r, k, strings.Join(entSfDescs(muts), "\n  "))

		target := author.prev
		for _, mut := range muts {
			target = mut.apply(t, target)
		}
		prev57, prev80 := f57.prev, f80.prev
		plan80, err := entA8Round(ctx, t, f57, f80, target, muts)
		if err != nil {
			minimized, minErr := entSfMinimizeWith(muts, err, func(trial []entSfMutation) error {
				return entA8Trial(ctx, t, srv57, srv80, prev57, prev80, authored80, trial)
			})
			t.Fatalf("[A8 %s seed=%d round=%d] FAILED\nminimized mutations (%d of %d):\n  %s\nerror:\n%v\nseed history:\n  %s",
				dir, seed, r, len(minimized), len(muts), strings.Join(entSfDescs(minimized), "\n  "),
				minErr, strings.Join(history, "\n  "))
		}

		// A leaked 5.7 display width inside an 8.0-side ALTER is a phantom column MODIFY
		// (the 5.7-side plan may carry widths legitimately — they ARE its stored form).
		entUpgAssertNoWidthInAlters(t, fmt.Sprintf("A8 %s seed=%d round=%d plan80", dir, seed, r), plan80)

		// Accretion guards on BOTH sides' dumps (each against its own round-0 baseline).
		for _, s := range []struct {
			f    *entA8Fleet
			prev string
		}{{f57, prev57}, {f80, prev80}} {
			require.Less(t, len(s.f.prev), s.f.dump0+s.f.budget+entSfRoundSlack*r,
				"[A8 %s seed=%d round=%d side=%s] check=accretion: dump grew past the added-object budget: len(dump_%d)=%d, len(dump_0)=%d, budget=%d — text accretion suspected.\nseed history:\n  %s",
				dir, seed, r, s.f.srv.name, r, len(s.f.prev), s.f.dump0, s.f.budget+entSfRoundSlack*r, strings.Join(history, "\n  "))
			entSfAssertUntouchedStable(t, s.f.srv, seed, r, s.prev, s.f.prev, muts)
		}

		t.Logf("[A8 %s seed=%d round=%d] ok: 57 %d -> %d bytes, 80 %d -> %d bytes (budget 57=%d 80=%d), counts 57=%+v 80=%+v",
			dir, seed, r, len(prev57), len(f57.prev), len(prev80), len(f80.prev),
			f57.dump0+f57.budget+entSfRoundSlack*r, f80.dump0+f80.budget+entSfRoundSlack*r, f57.expect, f80.expect)
	}
}

//nolint:tparallel
func TestSDLEnterpriseCrossVersionStatefulFuzz(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	srv57, srv80 := entUpgServers(t)
	// The fleet schema must load on BOTH versions, so the base is the 5.7 shape of the
	// A6 stateful-fuzz base (entSfAux80's SRID/INVISIBLE/CHECK tables are 8.0-only).
	baseDDL := entSfBaseDDL(t, srv57)

	t.Run("forward_57_authored", func(t *testing.T) {
		for seed := int64(1); seed <= entA8ForwardSeeds; seed++ {
			seed := seed
			t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
				entA8RunSeed(ctx, t, srv57, srv80, baseDDL, seed, false)
			})
		}
	})
	// Reverse-authored seeds use a disjoint seed range: the two dumps' candidate models
	// are logically identical, so reusing 1..N would largely replay the forward streams.
	t.Run("reverse_80_authored", func(t *testing.T) {
		for seed := int64(101); seed < 101+entA8ReverseSeeds; seed++ {
			seed := seed
			t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
				entA8RunSeed(ctx, t, srv57, srv80, baseDDL, seed, true)
			})
		}
	})
}
