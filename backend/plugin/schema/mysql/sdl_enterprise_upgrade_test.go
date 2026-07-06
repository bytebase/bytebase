package mysql

// A7 of the ENTERPRISE smoke axes (see sdl_enterprise_test.go): CROSS-VERSION upgrade —
// the migration-during-upgrade scenario where a schema is authored/dumped on one MySQL
// version and the target is applied on the other.
//
// Legs, per corpus (zabbix / prestashop / openemr; sys stays behind its entCorpora gates):
//
//	upgrade_57_dump_targets_80_db  — load the corpus on 5.7, take its canonical dump D57,
//	    load the SAME original DDL on 8.0, then use D57 as the TARGET against the 8.0
//	    database (version-aware diff as 8.0.32). The plan must apply cleanly and
//	    converge; 5.7-isms (integer display widths) must normalize away — asserted by the
//	    no-width-in-ALTER guard — and nothing may phantom-drop. Genuine semantic
//	    differences (the server-default charset a version applied at load time) are
//	    LEGITIMATE plan content: a 5.7-authored dump of a charset-less corpus says
//	    latin1, and declarative semantics honor it.
//	minimal_mutation_after_upgrade — (zabbix) one semantic edit (add column + index) on
//	    top of D57, targeted at the upgraded 8.0 database → exactly the minimal DDL, no
//	    cross-version noise re-emerging.
//	fresh_80_from_57_dump          — D57 against an EMPTY 8.0 database (fresh create from
//	    a 5.7-authored dump) → apply → re-dump → converges to D57 and self-diffs empty.
//	downgrade_80_dump_targets_57_db — the reverse: D80 as target on the 5.7 database.
//	    Probed: where an 8.0-ism blocks the 5.7 apply (utf8mb4_0900_ai_ci — 8.0's default
//	    utf8mb4 collation does not exist on 5.7), the leg skips with the named reason.
//	aligned_zabbix_utf8mb4_general_ci — the distilled asymmetry probe: both databases
//	    created with the SAME 5.7-legal default (utf8mb4/utf8mb4_general_ci) before
//	    loading zabbix, so the only cross-dump differences left are pure version
//	    renderings (display widths). Both cross-diffs must then be EMPTY — any residual
//	    is a phantom diff by definition.
//
// prestashop note: the corpus itself leads with SET SESSION sql_mode='' (two
// '0000-00-00 00:00:00' defaults need non-strict mode). Generated plans cannot carry
// session state, so plan APPLICATION for that corpus is prefixed with the corpus's own
// preamble — the exact concession the corpus already requires at load time.
//
// Shared helpers (liveServers, entLoadDDL, dumpSDL, applyDDL, entPlanStatementCount,
// entNormalizeDelimiters, newLiveDatabase, createLiveMySQLDriver, addColumnToTable,
// addIndexToTable, entCorpora + embedded corpora) come from the sibling files.

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

// Full server versions are threaded into the diff for this axis (the release path passes
// the synced server version string, e.g. "8.0.32", not a bare major.minor).
const (
	entUpg57Version = "5.7.25"
	entUpg80Version = "8.0.32"
)

type entUpgCorpus struct {
	name string
	ddl  string
	// preamble is prepended to every generated plan before application (see the
	// prestashop note above). It mirrors the corpus's own leading session setup.
	preamble string
	// gate, when non-empty, skips the corpus with the given reason.
	gate string
}

// entUpgCorpora derives the cross-version corpus list from the shared entCorpora().
// The axis authors the SAME original DDL on BOTH versions, so a corpus gated on either
// version (sys: gate + gate57) is gated here too.
//
// openemr note: the corpus's uuid columns (`uuid` binary(16) NOT NULL DEFAULT ”) once
// gated it out of this axis — the 8.0 sync stored the information_schema hex NOTATION
// ('0x…') verbatim and every cross-version leg looped on a phantom
// `MODIFY COLUMN ... DEFAULT '0x…'` residual. The sync now canonicalizes binary-family
// defaults (see TestSDLEnterpriseUpgradeBinaryDefaultCanonicalForm), so the corpus runs
// un-gated.
func entUpgCorpora() []entUpgCorpus {
	var out []entUpgCorpus
	for _, c := range entCorpora() {
		u := entUpgCorpus{name: c.name, ddl: c.ddl, gate: c.gate}
		if u.gate == "" && c.gate57 != "" {
			u.gate = c.gate57
		}
		if u.gate == "" && c.canonical {
			u.gate = "corpus ships as a canonical dump requiring the engine create-plan loader; A7 authors from raw upstream DDL"
		}
		if c.name == "prestashop" {
			u.preamble = "SET SESSION sql_mode='';\n"
		}
		out = append(out, u)
	}
	return out
}

func entUpgServers(t *testing.T) (srv57, srv80 liveServer) {
	t.Helper()
	found57, found80 := false, false
	for _, srv := range liveServers {
		switch srv.version {
		case "5.7":
			srv57, found57 = srv, true
		case "8.0":
			srv80, found80 = srv, true
		default:
		}
	}
	require.True(t, found57 && found80, "cross-version axis needs both a 5.7 and an 8.0 live server")
	return srv57, srv80
}

// entUpgTrim caps logged plan/residual text so a 250-table plan stays readable.
func entUpgTrim(s string) string {
	const limit = 6000
	if len(s) <= limit {
		return s
	}
	return s[:limit] + fmt.Sprintf("\n... (%d more bytes)", len(s)-limit)
}

// entUpgReDrop matches destructive operations that can NEVER be legitimate when both
// databases were loaded from the same original DDL — any hit is a phantom diff.
var entUpgReDrop = regexp.MustCompile(`(?i)\bDROP\s+(TABLE|COLUMN|TRIGGER|VIEW|FUNCTION|PROCEDURE|INDEX|KEY|CHECK|CONSTRAINT|FOREIGN)`)

func entUpgAssertNoDrops(t *testing.T, label, plan string) {
	t.Helper()
	if m := entUpgReDrop.FindString(plan); m != "" {
		require.Failf(t, "phantom destructive operation in cross-version plan",
			"[%s] plan contains %q — with identical source DDL on both versions nothing may be dropped:\n%s",
			label, m, entUpgTrim(plan))
	}
}

// entUpgReIntWidth matches integer display widths — the 5.7 stored form 8.0 must
// normalize away. tinyint(1) is exempt (the BOOLEAN spelling, canonical on BOTH
// versions).
var entUpgReIntWidth = regexp.MustCompile(`(?i)\b(?:tiny|small|medium|big)?int\(\d+\)`)

// entUpgAssertNoWidthInAlters proves no ALTER statement carries a 5.7 display width —
// a width inside an ALTER means the 5.7 rendering leaked through normalization and
// phantom-modified a column. CREATE TABLE statements are not judged: a pass-through
// width in a fresh CREATE is cosmetic, converges, and is not a phantom diff.
func entUpgAssertNoWidthInAlters(t *testing.T, label, plan string) {
	t.Helper()
	if strings.TrimSpace(plan) == "" {
		return
	}
	stmts, err := mysqlparser.SplitSQL(plan)
	require.NoError(t, err, "[%s] split plan for width guard", label)
	for _, s := range stmts {
		text := strings.TrimSpace(s.Text)
		if !strings.HasPrefix(strings.ToUpper(text), "ALTER TABLE") {
			continue
		}
		for _, m := range entUpgReIntWidth.FindAllString(text, -1) {
			if strings.EqualFold(m, "tinyint(1)") {
				continue
			}
			require.Failf(t, "integer display width leaked into a cross-version ALTER",
				"[%s] ALTER carries %q (5.7 display width not normalized away):\n%s", label, m, text)
		}
	}
}

// entUpgConverge diffs the live database against target (as version), applies the plan
// (with the corpus preamble), and proves convergence + idempotence. Returns the plan
// ("" when the schemas were already equivalent).
func entUpgConverge(ctx context.Context, t *testing.T, srv liveServer, dbName, target, version, preamble, label string) string {
	t.Helper()
	source := dumpSDL(ctx, t, srv, dbName)
	plan, err := mysqlDiffSDLMigration(source, target, version)
	require.NoError(t, err, "[%s] diff", label)
	if strings.TrimSpace(plan) == "" {
		t.Logf("[%s] empty plan — schemas already equivalent", label)
	} else {
		t.Logf("[%s] plan: %d statements, %d bytes:\n%s", label, entPlanStatementCount(t, plan), len(plan), entUpgTrim(plan))
		require.NoError(t, applyDDL(ctx, t, srv, dbName, preamble+plan),
			"[%s] generated plan failed to apply:\n%s", label, entUpgTrim(plan))
	}
	after := dumpSDL(ctx, t, srv, dbName)
	converge, err := mysqlDiffSDLMigration(after, target, version)
	require.NoError(t, err, "[%s] converge diff", label)
	require.Empty(t, converge, "[%s] did not converge; residual:\n%s", label, entUpgTrim(converge))
	self, err := mysqlDiffSDLMigration(after, after, version)
	require.NoError(t, err, "[%s] idempotence diff", label)
	require.Empty(t, self, "[%s] post-apply dump not idempotent:\n%s", label, entUpgTrim(self))
	return plan
}

// entUpgLoadAligned creates a database whose DEFAULT charset/collation is pinned to the
// 5.7-legal utf8mb4/utf8mb4_general_ci on either server, then loads ddl — removing the
// server-default charset divergence so cross-dumps differ only by version renderings.
func entUpgLoadAligned(ctx context.Context, t *testing.T, srv liveServer, prefix, ddl string) string {
	t.Helper()
	dbName := newLiveDatabase(ctx, t, srv, prefix)
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, "ALTER DATABASE `"+dbName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", db.ExecuteOptions{})
	require.NoError(t, err, "[%s] align database default collation", srv.name)
	_, err = driver.Execute(ctx, entNormalizeDelimiters(ddl), db.ExecuteOptions{})
	require.NoError(t, err, "[%s] apply aligned base DDL", srv.name)
	return dbName
}

//nolint:tparallel
func TestSDLEnterpriseCrossVersionUpgrade(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	srv57, srv80 := entUpgServers(t)

	for _, corpus := range entUpgCorpora() {
		corpus := corpus
		t.Run(corpus.name, func(t *testing.T) {
			if corpus.gate != "" {
				t.Skipf("[A7/%s] %s", corpus.name, corpus.gate)
			}

			// Author on 5.7 and on 8.0 from the SAME original DDL; capture both
			// canonical dumps up front so later legs judge fixed texts.
			db57 := entLoadDDL(ctx, t, srv57, "entsdl_up57_"+corpus.name, corpus.ddl)
			d57 := dumpSDL(ctx, t, srv57, db57)
			db80 := entLoadDDL(ctx, t, srv80, "entsdl_up80_"+corpus.name, corpus.ddl)
			d80 := dumpSDL(ctx, t, srv80, db80)
			t.Logf("[A7/%s] D57 %d bytes, D80 %d bytes", corpus.name, len(d57), len(d80))

			t.Run("upgrade_57_dump_targets_80_db", func(t *testing.T) {
				label := "A7/" + corpus.name + "/upgrade_57_to_80"
				plan := entUpgConverge(ctx, t, srv80, db80, d57, entUpg80Version, corpus.preamble, label)
				entUpgAssertNoDrops(t, label, plan)
				entUpgAssertNoWidthInAlters(t, label, plan)
			})

			if corpus.name == "zabbix" {
				t.Run("minimal_mutation_after_upgrade", func(t *testing.T) {
					label := "A7/" + corpus.name + "/minimal_mutation_after_upgrade"
					// Self-sufficient: land the database on D57 first (a no-op when the
					// upgrade leg just ran).
					entUpgConverge(ctx, t, srv80, db80, d57, entUpg80Version, corpus.preamble, label+"/pre")

					// One semantic edit on top of the 5.7-authored dump.
					target := addColumnToTable(t, d57, "users", "`ent_up_col` varchar(50) DEFAULT NULL")
					target = addIndexToTable(t, target, "users", "KEY `ent_up_idx` (`ent_up_col`)")

					source := dumpSDL(ctx, t, srv80, db80)
					plan, err := mysqlDiffSDLMigration(source, target, entUpg80Version)
					require.NoError(t, err, "[%s] diff", label)
					require.NotEmpty(t, plan, "[%s] the semantic edit must produce a plan", label)
					t.Logf("[%s] plan:\n%s", label, plan)

					n := entPlanStatementCount(t, plan)
					require.LessOrEqual(t, n, 2, "[%s] plan must be the minimal DDL (add column + add index), got %d statements:\n%s", label, n, plan)
					stmts, err := mysqlparser.SplitSQL(plan)
					require.NoError(t, err)
					for _, s := range stmts {
						text := strings.TrimSpace(s.Text)
						if text == "" || text == ";" {
							continue
						}
						require.True(t, strings.HasPrefix(strings.ToUpper(text), "ALTER TABLE `USERS`"),
							"[%s] plan statement strays off the edited table (cross-version noise re-emerged):\n%s", label, text)
					}
					upper := strings.ToUpper(plan)
					require.Contains(t, upper, "ENT_UP_COL", "[%s] plan missing the added column:\n%s", label, plan)
					require.Contains(t, upper, "ENT_UP_IDX", "[%s] plan missing the added index:\n%s", label, plan)

					require.NoError(t, applyDDL(ctx, t, srv80, db80, plan), "[%s] minimal plan failed to apply:\n%s", label, plan)
					after := dumpSDL(ctx, t, srv80, db80)
					converge, err := mysqlDiffSDLMigration(after, target, entUpg80Version)
					require.NoError(t, err)
					require.Empty(t, converge, "[%s] minimal mutation did not converge:\n%s", label, converge)
				})
			}

			t.Run("fresh_80_from_57_dump", func(t *testing.T) {
				label := "A7/" + corpus.name + "/fresh_80_from_57_dump"
				dbEmpty := newLiveDatabase(ctx, t, srv80, "entsdl_upe80_"+corpus.name)
				plan := entUpgConverge(ctx, t, srv80, dbEmpty, d57, entUpg80Version, corpus.preamble, label)
				require.NotEmpty(t, plan, "[%s] creating from a 5.7-authored dump on an empty database must emit DDL", label)
				entUpgAssertNoWidthInAlters(t, label, plan)
			})

			t.Run("downgrade_80_dump_targets_57_db", func(t *testing.T) {
				label := "A7/" + corpus.name + "/downgrade_80_to_57"
				source := dumpSDL(ctx, t, srv57, db57)
				plan, err := mysqlDiffSDLMigration(source, d80, entUpg57Version)
				require.NoError(t, err, "[%s] diff", label)
				if strings.TrimSpace(plan) != "" {
					t.Logf("[%s] plan: %d statements, %d bytes:\n%s", label, entPlanStatementCount(t, plan), len(plan), entUpgTrim(plan))
					if applyErr := applyDDL(ctx, t, srv57, db57, corpus.preamble+plan); applyErr != nil {
						// Probe outcome: an 8.0-authored dump can carry 8.0-isms no 5.7
						// server accepts. The only one these corpora produce is 8.0's
						// utf8mb4 default collation.
						if strings.Contains(plan, "utf8mb4_0900_ai_ci") {
							t.Skipf("[%s] 8.0-authored dump is not 5.7-legal: plan carries utf8mb4_0900_ai_ci "+
								"(8.0's utf8mb4 default collation, unknown to 5.7); apply failed as expected: %v", label, applyErr)
						}
						require.NoErrorf(t, applyErr, "[%s] downgrade plan failed to apply and no known 8.0-ism explains it:\n%s",
							label, entUpgTrim(plan))
					}
				}
				after := dumpSDL(ctx, t, srv57, db57)
				converge, err := mysqlDiffSDLMigration(after, d80, entUpg57Version)
				require.NoError(t, err, "[%s] converge diff", label)
				require.Empty(t, converge, "[%s] did not converge; residual:\n%s", label, entUpgTrim(converge))
				self, err := mysqlDiffSDLMigration(after, after, entUpg57Version)
				require.NoError(t, err)
				require.Empty(t, self, "[%s] post-apply dump not idempotent:\n%s", label, entUpgTrim(self))
			})
		})
	}

	// The distilled asymmetry probe: with the server-default charset divergence removed,
	// BOTH cross-diffs must be empty — the remaining differences are exactly the
	// version renderings (5.7 display widths) the normalizer must absorb.
	t.Run("aligned_zabbix_utf8mb4_general_ci", func(t *testing.T) {
		db57 := entUpgLoadAligned(ctx, t, srv57, "entsdl_upal57", entZabbixSQL)
		db80 := entUpgLoadAligned(ctx, t, srv80, "entsdl_upal80", entZabbixSQL)
		d57 := dumpSDL(ctx, t, srv57, db57)
		d80 := dumpSDL(ctx, t, srv80, db80)

		t.Run("57_dump_targets_80_db", func(t *testing.T) {
			cur := dumpSDL(ctx, t, srv80, db80)
			plan, err := mysqlDiffSDLMigration(cur, d57, entUpg80Version)
			require.NoError(t, err, "aligned 57->80 diff")
			require.Empty(t, plan,
				"collation-aligned 5.7 dump phantom-diffs an equivalent 8.0 database (widths/collation defaults did not normalize away):\n%s",
				entUpgTrim(plan))
		})
		t.Run("80_dump_targets_57_db", func(t *testing.T) {
			cur := dumpSDL(ctx, t, srv57, db57)
			plan, err := mysqlDiffSDLMigration(cur, d80, entUpg57Version)
			require.NoError(t, err, "aligned 80->57 diff")
			require.Empty(t, plan,
				"collation-aligned 8.0 dump phantom-diffs an equivalent 5.7 database (widths/collation defaults did not normalize away):\n%s",
				entUpgTrim(plan))
		})
	})
}

// ----------------------------------------------------------------------------
// REGRESSION (formerly the pinned bug this axis found, openemr corpus): binary-family
// column defaults.
//
// MySQL reports binary/varbinary literal defaults in information_schema.COLUMNS in
// version-specific encodings: 8.0 as hex NOTATION text built from the value truncated
// at its first NUL byte ("0x" for DEFAULT '', "0x6162" for DEFAULT 'ab'), 5.7 as the
// RAW BYTES, NUL-padded to the declared width for binary(N). The sync used to store
// the 8.0 notation verbatim and the dumper re-quoted it as a STRING literal
// (DEFAULT '0x6162') — value-unfaithful dumps, cross-version phantom MODIFY loops (the
// openemr uuid columns that once gated that corpus out of the A7 legs), and round-trip
// double-encoding ('0x6162' -> '0x307836313632').
//
// The sync now decodes both encodings into one canonical form — '' when the value is
// empty (binary(N) padding stripped), a plain quoted string for clean text, an
// unquoted hex literal otherwise; see canonicalBinaryDefault in
// backend/plugin/db/mysql/sync.go. This test pins the fixed behavior: value-faithful
// dumps on both versions, EMPTY cross-version diffs (the DDL pins the table charset so
// the dumps may differ by nothing at all), and a clean round-trip through a fresh 8.0
// database. The per-version legs then cover the hex-literal fallback for values no
// quoted SDL string can carry — probed on the version whose information_schema reports
// them faithfully (0xFF61 on 8.0; a significant trailing NUL, 0x6100, on 5.7).
// ----------------------------------------------------------------------------

const entUpgBinaryDefaultDDL = `
CREATE TABLE ent_up_bin (
  id int NOT NULL,
  u binary(16) NOT NULL DEFAULT '',
  v varbinary(24) NOT NULL DEFAULT 'ab',
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
`

//nolint:tparallel
func TestSDLEnterpriseUpgradeBinaryDefaultCanonicalForm(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	srv57, srv80 := entUpgServers(t)

	// Both versions dump the same value-faithful canonical form.
	db80 := entLoadDDL(ctx, t, srv80, "entsdl_upbin80", entUpgBinaryDefaultDDL)
	d80 := dumpSDL(ctx, t, srv80, db80)
	db57 := entLoadDDL(ctx, t, srv57, "entsdl_upbin57", entUpgBinaryDefaultDDL)
	d57 := dumpSDL(ctx, t, srv57, db57)
	for _, d := range []struct{ version, dump string }{{"8.0", d80}, {"5.7", d57}} {
		require.Contains(t, d.dump, "`u` binary(16) NOT NULL DEFAULT ''",
			"%s dump must render the empty binary(16) default as '' (not the I_S encoding):\n%s", d.version, d.dump)
		require.Contains(t, d.dump, "`v` varbinary(24) NOT NULL DEFAULT 'ab'",
			"%s dump must render the varbinary default value-faithfully:\n%s", d.version, d.dump)
		require.NotContains(t, d.dump, "0x",
			"%s dump leaked an information_schema hex encoding for a clean-text default:\n%s", d.version, d.dump)
	}

	// Cross-version: with the table charset pinned, the dumps describe the same schema
	// and BOTH cross-diffs must be empty — the shape that once looped on
	// MODIFY COLUMN ... DEFAULT '0x…'.
	cross80, err := mysqlDiffSDLMigration(d80, d57, entUpg80Version)
	require.NoError(t, err)
	require.Empty(t, cross80, "5.7 dump phantom-diffs the equivalent 8.0 database:\n%s", cross80)
	cross57, err := mysqlDiffSDLMigration(d57, d80, entUpg57Version)
	require.NoError(t, err)
	require.Empty(t, cross57, "8.0 dump phantom-diffs the equivalent 5.7 database:\n%s", cross57)

	// Round-trip the 8.0 dump through a FRESH database: the create plan must apply, the
	// re-dump must converge to the dump (no double-encoding), and self-diff empty.
	dbFresh := newLiveDatabase(ctx, t, srv80, "entsdl_upbinfresh")
	empty := dumpSDL(ctx, t, srv80, dbFresh)
	plan, err := mysqlDiffSDLMigration(empty, d80, entUpg80Version)
	require.NoError(t, err)
	require.NotEmpty(t, plan)
	require.NoError(t, applyDDL(ctx, t, srv80, dbFresh, plan), "create plan from the 8.0 dump failed to apply:\n%s", plan)
	redump := dumpSDL(ctx, t, srv80, dbFresh)
	require.NotContains(t, redump, "0x3078",
		"the 8.0 dump round-trip double-encodes the default again ('0x6162' -> '0x307836313632'):\n%s", redump)
	converge, err := mysqlDiffSDLMigration(redump, d80, entUpg80Version)
	require.NoError(t, err)
	require.Empty(t, converge, "8.0 dump did not converge through a fresh database; residual:\n%s", converge)

	// Values no quoted SDL string can carry fall back to an unquoted hex literal, each
	// probed on the version whose information_schema reports the value faithfully (8.0
	// truncates at NUL bytes; 5.7 truncates at non-UTF8 bytes).
	t.Run("hex_literal_fallback_nonutf8_80", func(t *testing.T) {
		ddl := "CREATE TABLE ent_up_binhex (id int NOT NULL, w varbinary(8) NOT NULL DEFAULT 0xFF61, PRIMARY KEY (id)) ENGINE=InnoDB;"
		dbName := entLoadDDL(ctx, t, srv80, "entsdl_upbinhex80", ddl)
		dump := dumpSDL(ctx, t, srv80, dbName)
		require.Contains(t, dump, "`w` varbinary(8) NOT NULL DEFAULT 0xff61",
			"8.0 dump must keep the non-UTF8 default as an unquoted hex literal:\n%s", dump)
		self, err := mysqlDiffSDLMigration(dump, dump, entUpg80Version)
		require.NoError(t, err)
		require.Empty(t, self, "hex-literal dump not idempotent:\n%s", self)

		fresh := newLiveDatabase(ctx, t, srv80, "entsdl_upbinhexf80")
		plan, err := mysqlDiffSDLMigration(dumpSDL(ctx, t, srv80, fresh), dump, entUpg80Version)
		require.NoError(t, err)
		require.NoError(t, applyDDL(ctx, t, srv80, fresh, plan), "hex-literal create plan failed to apply:\n%s", plan)
		require.Equal(t, dump, dumpSDL(ctx, t, srv80, fresh), "hex-literal default did not round-trip byte-identically")
	})
	t.Run("hex_literal_fallback_trailnul_57", func(t *testing.T) {
		ddl := "CREATE TABLE ent_up_binnul (id int NOT NULL, w varbinary(8) NOT NULL DEFAULT 0x6100, PRIMARY KEY (id)) ENGINE=InnoDB;"
		dbName := entLoadDDL(ctx, t, srv57, "entsdl_upbinnul57", ddl)
		dump := dumpSDL(ctx, t, srv57, dbName)
		require.Contains(t, dump, "`w` varbinary(8) NOT NULL DEFAULT 0x6100",
			"5.7 dump must keep the trailing-NUL default as an unquoted hex literal (the NUL is significant on varbinary):\n%s", dump)
		self, err := mysqlDiffSDLMigration(dump, dump, entUpg57Version)
		require.NoError(t, err)
		require.Empty(t, self, "hex-literal dump not idempotent:\n%s", self)

		fresh := newLiveDatabase(ctx, t, srv57, "entsdl_upbinnulf57")
		plan, err := mysqlDiffSDLMigration(dumpSDL(ctx, t, srv57, fresh), dump, entUpg57Version)
		require.NoError(t, err)
		require.NoError(t, applyDDL(ctx, t, srv57, fresh, plan), "hex-literal create plan failed to apply:\n%s", plan)
		require.Equal(t, dump, dumpSDL(ctx, t, srv57, fresh), "hex-literal default did not round-trip byte-identically")
	})
}
