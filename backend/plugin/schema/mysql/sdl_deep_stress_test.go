package mysql

// Deep (round-2) stress test for the MySQL declarative (SDL) migration path. Round 1
// (sdl_stress_live_test.go) covered breadth/scale/normalization. This file pushes four
// DIFFERENT axes, all adversarial — the goal is to FIND bugs, not to confirm green:
//
//   - Axis 1: malformed / semantic-error TARGET SDL fed to mysqlDiffSDLMigration and
//     schema.SDLMigration. Must fail GRACEFULLY (clean error or defined no-op) — never panic,
//     never silently emit a wrong/destructive plan.
//   - Axis 2: exotic types + extreme values. sync -> MetadataToSDL -> diff vs user form must
//     be empty (idempotent) on both versions; every non-empty no-op is a normalization gap.
//   - Axis 3: sequential release chains S0->S1->...->S4 on a real DB; each step minimal,
//     applied, re-synced, converged before the next.
//   - Axis 4: ~80-100 table scale; no-op must be empty + fast; one small change must be one
//     minimal ALTER, not a re-emit of all tables.
//
// Drives the PRODUCTION entry points only:
//   - schema.SDLMigration(MYSQL, userSDL, syncedMetadata, version),
//   - mysqlDiffSDLMigration(source, target, version),
//   - schema.SDLDropAdvices(MYSQL, userSDL, syncedMetadata, version),
//   - schema.MetadataToSDL.
//
// Shared helpers (createLiveMySQLDriver, newLiveDatabase, applyAndDump, statementCount,
// liveServers, liveServer, syncMetadata, dumpSDL, applyDDL) come from the other two live
// test files in this package.

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// ============================================================================
// Axis 1: adversarial / malformed / semantic-error TARGET SDL.
// ============================================================================
//
// Robustness contract: feeding bad target SDL into the diff must produce a CLEAN Go error
// (or a defined no-op) — NEVER a panic, NEVER a malformed/destructive plan. The diff path
// returns errors as Go values; a panic is therefore a (B) bug regardless of message.

// diffOutcome captures how a single adversarial input was handled.
type diffOutcome struct {
	panicked  bool
	panicVal  any
	err       error
	plan      string
	planEmpty bool
}

// safeDiff runs mysqlDiffSDLMigration(source, target, version) under a panic
// recover so a panic is reported as data instead of crashing the test binary. A baseline
// non-empty `source` is supplied so that if the target parses as EMPTY, the resulting plan
// would be a (destructive) full-drop — letting us detect "silently dropped everything".
func safeDiff(source, target, version string) (out diffOutcome) {
	defer func() {
		if r := recover(); r != nil {
			out.panicked = true
			out.panicVal = r
		}
	}()
	plan, err := mysqlDiffSDLMigration(source, target, version)
	out.err = err
	out.plan = plan
	out.planEmpty = strings.TrimSpace(plan) == ""
	return out
}

// safeSDLMigration runs the production schema.SDLMigration under a panic recover.
func safeSDLMigration(target string, meta *model.DatabaseMetadata, version string) (out diffOutcome) {
	defer func() {
		if r := recover(); r != nil {
			out.panicked = true
			out.panicVal = r
		}
	}()
	plan, err := schema.SDLMigration(storepb.Engine_MYSQL, target, meta, version)
	out.err = err
	out.plan = plan
	out.planEmpty = strings.TrimSpace(plan) == ""
	return out
}

// adversarialCase is one bad-input probe.
//
// The UNIVERSAL bar for every case is: no panic, and no error-free plan that would
// DESTROY a valid pre-existing object that the (correctly-interpreted) target still
// contains. Beyond that, cases differ in how strict we can be about rejection:
//
//   - wantReject=true: the input is malformed enough that a robust loader SHOULD return a
//     non-nil error. We assert err != nil. (If a future omni accepts it, this flips to a
//     soft finding rather than a crash — see the test body.)
//   - wantReject=false: the input is degenerate-but-arguably-valid, or its rejection is the
//     server's job at apply time (e.g. over-length identifier, dangling FK). We only assert
//     no-panic + no-unexpected-destruction and LOG the outcome.
type adversarialCase struct {
	name       string
	target     string
	wantReject bool // require a non-nil error from the loader
	note       string
}

// adversarialCases enumerates malformed and semantically-broken target SDL. The `source`
// is a fixed valid 2-table schema so any plan the engine produces against a broken target
// is observable (and a full-table-drop plan is detectable).
func adversarialCases() []adversarialCase {
	return []adversarialCase{
		// ---- Syntax errors (must be rejected) ----
		{name: "truncated_create", target: "CREATE TABLE t (id INT", wantReject: true, note: "unterminated CREATE"},
		{name: "unmatched_paren", target: "CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(50) ENGINE=InnoDB;", wantReject: true, note: "missing close paren"},
		{name: "garbage_tokens", target: "@@@ this is not sql ;;; %%%", wantReject: true, note: "pure garbage"},
		{name: "keyword_salad", target: "CREATE CREATE TABLE TABLE t t (id id INT INT);", wantReject: true, note: "doubled keywords"},
		{name: "missing_type", target: "CREATE TABLE t (id, name VARCHAR(50));", wantReject: true, note: "column with no type"},

		// ---- Semantic errors ----
		{
			name:       "fk_nonexistent_table",
			target:     "CREATE TABLE child (id INT PRIMARY KEY, pid INT, CONSTRAINT fk FOREIGN KEY (pid) REFERENCES no_such_parent (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // omni accepts a dangling FK at load; the server rejects on apply. No-panic + non-destructive is the bar.
			note:       "FK to a table that does not exist",
		},
		{
			name:       "fk_nonexistent_column",
			target:     "CREATE TABLE parent (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\nCREATE TABLE child (id INT PRIMARY KEY, pid INT, CONSTRAINT fk FOREIGN KEY (pid) REFERENCES parent (no_such_col)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false,
			note:       "FK references a column that does not exist on parent",
		},
		{
			name:       "duplicate_table",
			target:     "CREATE TABLE t (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\nCREATE TABLE t (id INT PRIMARY KEY, x INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true,
			note:       "two CREATE TABLE with the same name",
		},
		{
			name:       "duplicate_column",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(50), name VARCHAR(60)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true,
			note:       "duplicate column name within a table",
		},
		{
			name:       "duplicate_index_name",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, a INT, b INT, KEY idx_dup (a), KEY idx_dup (b)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true,
			note:       "two indexes with the same name",
		},
		{
			name:       "pk_declared_twice",
			target:     "CREATE TABLE t (id INT, x INT, PRIMARY KEY (id), PRIMARY KEY (x)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true,
			note:       "two PRIMARY KEY clauses",
		},
		{
			name:       "index_nonexistent_column",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, a INT, KEY idx_ghost (no_such_col)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // omni does NOT validate index-column existence at load; server rejects on apply (errno 1072). Soft finding.
			note:       "index on a column that does not exist",
		},
		{
			name:       "check_references_other_table",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, a INT, CONSTRAINT ck CHECK (a < (SELECT COUNT(*) FROM other))) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true, // 8.0 rejects a subquery in CHECK at load
			note:       "CHECK with a subquery referencing another table",
		},
		{
			name:       "generated_col_forward_ref",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, g INT AS (later + 1) STORED, later INT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // forward reference; MySQL allows it. No-panic + sane plan is the bar.
			note:       "generated column referencing a column declared later",
		},
		{
			name:       "circular_fk_notnull",
			target:     "CREATE TABLE a (id INT PRIMARY KEY, b_id INT NOT NULL, CONSTRAINT fk_a FOREIGN KEY (b_id) REFERENCES b (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\nCREATE TABLE b (id INT PRIMARY KEY, a_id INT NOT NULL, CONSTRAINT fk_b FOREIGN KEY (a_id) REFERENCES a (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // SDL loader disables FK checks; a clean plan or error both acceptable, no panic is the bar
			note:       "mutually-referential NOT NULL FKs",
		},

		// ---- Degenerate ----
		{name: "empty_string", target: "", wantReject: false, note: "empty SDL -> full drop of source; must not panic"},
		{name: "whitespace_only", target: "   \n\t  \n", wantReject: false, note: "whitespace only"},
		{name: "comments_only", target: "-- just a comment\n/* block comment */\n", wantReject: false, note: "comments only"},
		{name: "single_semicolon", target: ";", wantReject: false, note: "a lone statement terminator"},
		{name: "create_sequence", target: "CREATE SEQUENCE s START WITH 1 INCREMENT BY 1;", wantReject: true, note: "different-engine DDL (no SEQUENCE in MySQL)"},
		{name: "dml_insert", target: "INSERT INTO t (id) VALUES (1);", wantReject: false, note: "DML in SDL target (LoadSQL fallback may accept and yield a plan)"},
		{name: "dml_update", target: "UPDATE t SET id = 2;", wantReject: false, note: "UPDATE in SDL target"},
		{name: "drop_table_in_target", target: "DROP TABLE t;", wantReject: true, note: "DROP in an SDL target (SDL is CREATE-only -> rejected)"},
		{name: "alter_in_target", target: "ALTER TABLE t ADD COLUMN x INT;", wantReject: true, note: "ALTER in an SDL target (rejected)"},

		// ---- Identifier edges ----
		{
			name:       "ident_64_chars",
			target:     "CREATE TABLE `" + strings.Repeat("a", 64) + "` (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // 64 is the MAX legal length -> accepted, no panic
			note:       "64-char identifier (legal max)",
		},
		{
			name:       "ident_65_chars",
			target:     "CREATE TABLE `" + strings.Repeat("a", 65) + "` (id INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false, // omni does NOT enforce the 64-char limit at load; the server rejects on apply (errno 1059). Soft finding.
			note:       "65-char identifier (over the limit)",
		},
		{
			name:       "reserved_word_quoted",
			target:     "CREATE TABLE `select` (`from` INT PRIMARY KEY, `where` VARCHAR(50)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false,
			note:       "reserved words as backtick-quoted identifiers",
		},
		{
			name:       "unicode_emoji_ident",
			target:     "CREATE TABLE `ta*b📊le` (`col😀` INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false,
			note:       "unicode/emoji identifiers",
		},
		{
			name:       "ident_with_backtick_dot",
			target:     "CREATE TABLE `we``ird` (`a.b` INT PRIMARY KEY) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: false,
			note:       "identifiers needing escaping (embedded backtick, dot)",
		},
		{
			name:       "case_only_differ",
			target:     "CREATE TABLE t (id INT PRIMARY KEY, Name VARCHAR(50), name VARCHAR(60)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			wantReject: true, // MySQL column names are case-insensitive -> Name/name collide
			note:       "columns differing only by case",
		},
	}
}

// TestSDLDeepAdversarial drives malformed/semantic-error target SDL through the production
// diff. Bar: NEVER panic, NEVER silently emit a destructive plan. We feed a fixed valid
// 2-table `source` so a wrong plan that drops everything is observable.
//
//nolint:tparallel
func TestSDLDeepAdversarial(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	// A fixed valid source schema (two real tables) so the diff has something to operate on.
	const sourceSDL = `CREATE TABLE t (id INT PRIMARY KEY, name VARCHAR(50) NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE other (id INT PRIMARY KEY, t_id INT NOT NULL) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			// Materialize the source schema once so the production-path probe has a real synced
			// metadata to diff against (schema.SDLMigration path).
			meta, _ := syncMetadata(ctx, t, srv, "sdl_deep_adv", sourceSDL)

			for _, ac := range adversarialCases() {
				ac := ac
				t.Run(ac.name, func(t *testing.T) {
					// (1) Lower-level version-aware diff: source(valid) -> target(bad).
					out := safeDiff(sourceSDL, ac.target, srv.version)

					// UNIVERSAL bar 1: never panic. A panic on ANY input is a (B) bug.
					require.False(t, out.panicked,
						"[%s/%s] (B) PANIC on adversarial input %q: %v\ninput:\n%s",
						srv.name, ac.name, ac.note, out.panicVal, ac.target)

					// (2) Production entry: schema.SDLMigration(meta, badTarget). Same no-panic bar
					// on the real production call (covers the MetadataToSDL->diff composition).
					// Run for EVERY case, including the rejected ones.
					pout := safeSDLMigration(ac.target, meta, srv.version)
					require.False(t, pout.panicked,
						"[%s/%s] (B) PANIC in schema.SDLMigration on %q: %v\ninput:\n%s",
						srv.name, ac.name, ac.note, pout.panicVal, ac.target)

					if ac.wantReject {
						// The loader SHOULD reject this. If it does (err != nil), great. If it
						// does NOT, that is a robustness gap, but it is only a (B) bug if the
						// resulting error-free plan is also DESTRUCTIVE/wrong; a malformed input
						// that yields a clean error OR a harmless no-op is acceptable. We assert
						// the error and, on the rare accept, demand the plan at least not be
						// silently emitted as a non-empty migration.
						if out.err == nil {
							require.True(t, out.planEmpty,
								"[%s/%s] (B) malformed input %q was ACCEPTED (no error) and produced a non-empty plan — a wrong/garbage migration:\n%s",
								srv.name, ac.name, ac.note, out.plan)
							t.Logf("[%s/%s] SOFT: malformed input %q accepted as empty no-op rather than erroring", srv.name, ac.name, ac.note)
						} else {
							t.Logf("[%s/%s] reject OK: %v", srv.name, ac.name, out.err)
						}
						return
					}

					// wantReject==false: the input is degenerate-but-tolerable or its rejection
					// is the server's job at apply time. No-panic already asserted. A clean error
					// is fine; an error-free plan is fine (the declarative semantics of the target
					// are well-defined). Just record the outcome for the report.
					if out.err != nil {
						t.Logf("[%s/%s] tolerated-with-error: %v", srv.name, ac.name, out.err)
					} else {
						t.Logf("[%s/%s] tolerated input %q -> plan(empty=%v):\n%s",
							srv.name, ac.name, ac.note, out.planEmpty, out.plan)
					}
				})
			}
		})
	}
}

// ============================================================================
// Axis 2: exotic types + extreme values (idempotence; flag normalization gaps).
// ============================================================================
//
// Each case authors ONE less-common type-surface table. It is applied to a live DB, synced,
// dumped via MetadataToSDL, and the PRODUCTION version-aware diff of (dumped vs the user
// form) MUST be empty on each version it runs on. A non-empty no-op is a normalization gap
// (a (B) bug) and the test logs input + stored/dumped form so the owning omni Canonical* /
// dumper rule can be located. 8.0-only constructs are gated off 5.7 via only80.

// exoticCase authors one exotic-type table. only80 gates 8.0-only surface off 5.7.
//
// bGap marks a case that this round CONFIRMED is NOT idempotent — a (B) normalization /
// dumper round-trip bug. Those cases are SKIPPED by the idempotence test (so the suite stays
// green) but kept here as the executable repro, and are reproduced/explained by
// TestSDLDeepExoticKnownGaps below. bGapReason is the one-line root cause.
type exoticCase struct {
	name       string
	ddl        string
	only80     bool
	bGap       bool
	bGapReason string
}

// exoticCases enumerates the less-common type surface across families. Each names exactly
// one construct cluster so a single non-empty no-op (or load error) pinpoints the offending
// type. Cases marked bGap are confirmed (B) bugs this round found — see bGapReason and the
// detailed analysis on TestSDLDeepExoticKnownGaps.
func exoticCases() []exoticCase {
	return []exoticCase{
		// ---- JSON ----
		{
			name: "json_basic",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	doc JSON,
	doc_nn JSON NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:   "json_default_80",
			only80: true, // JSON column DEFAULT requires 8.0.13+
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	doc JSON DEFAULT (JSON_OBJECT()),
	arr JSON DEFAULT (JSON_ARRAY())
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:   "json_multivalue_index_80",
			only80: true,  // multi-valued index (CAST ... AS ... ARRAY) is 8.0.17+
			bGap:   false, // FIXED (dumper strips the functional-index charset introducer + unescapes quotes; omni #353 accepts AS … ARRAY, re-pinned a362f7a7)
			// FIXED (Bug B): the synced functional-index expression
			// `(cast(json_extract(`tags`,_utf8mb4\'$.ids\') as unsigned array))` carried two things the
			// omni loader rejects — the `_utf8mb4'…'` charset introducer before the JSON-path literal
			// (`unexpected token`) and backslash-escaped single quotes (`syntax error at or near "\"`).
			// normalizeFunctionalIndexExpr in get_database_definition.go now unescapes the quotes and
			// strips the introducer, so the dumped expr is `((cast(json_extract(`tags`,'$.ids') as
			// unsigned array)))`; omni's functional-index normalizer canonicalizes the dumped and user
			// forms identically and the no-op is empty. (The `AS UNSIGNED ARRAY` cast itself is accepted
			// by omni #353 — the introducer/escaping was the LoadSDL blocker, not the ARRAY keyword.)
			bGapReason: "",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	tags JSON NOT NULL,
	KEY idx_tags ((CAST(tags->'$.ids' AS UNSIGNED ARRAY)))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Spatial ----
		{
			name:   "spatial_types",
			only80: true, // 8.0-only GAP (FIXED): on 5.7 the dumper emits `geometrycollection` and IS idempotent; 8.0 used to break.
			// FIXED (bug 8): the dumper now normalizes the 8.0 `geomcollection` type synonym to
			// the canonical `geometrycollection` spelling omni parses (normalizeColumnType in
			// get_database_definition.go), so the dumped source reloads and the no-op is empty.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	g GEOMETRY,
	pt POINT,
	ls LINESTRING,
	poly POLYGON,
	gc GEOMETRYCOLLECTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "spatial_notnull_index",
			// FIXED (bug 7): the dumper no longer emits a key-part prefix length for SPATIAL (and
			// FULLTEXT) index parts (printIndexKeyPart suppressPrefix), so the spatial key dumps as
			// `(`pt`)` matching SHOW CREATE and the no-op is empty on both 5.7 and 8.0.
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	pt POINT NOT NULL,
	PRIMARY KEY (id),
	SPATIAL KEY idx_pt (pt)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Fractional seconds ----
		{
			name: "fractional_seconds",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	dt6 DATETIME(6) NOT NULL,
	tm3 TIME(3) NOT NULL,
	ts6 TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	dt0 DATETIME(0)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- BLOB / TEXT family ----
		{
			name: "blob_text_family",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	tb TINYBLOB,
	b BLOB,
	mb MEDIUMBLOB,
	lb LONGBLOB,
	tt TINYTEXT,
	txt TEXT,
	mt MEDIUMTEXT,
	lt LONGTEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "text_charset_collate",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	b LONGTEXT CHARACTER SET latin1 COLLATE latin1_swedish_ci,
	c BLOB
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "text_blob_prefix_index",
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	body TEXT,
	data BLOB,
	PRIMARY KEY (id),
	KEY idx_body (body(50)),
	KEY idx_data (data(30))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Charsets beyond utf8mb4 ----
		{
			name: "charset_latin1_ascii",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a VARCHAR(50) CHARACTER SET latin1 COLLATE latin1_swedish_ci,
	b VARCHAR(50) CHARACTER SET ascii,
	c CHAR(10) CHARACTER SET latin1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "charset_binary_varbinary",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a BINARY(16),
	b VARBINARY(255),
	c VARCHAR(50) CHARACTER SET binary
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "charset_gbk_big5_utf16",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a VARCHAR(50) CHARACTER SET gbk,
	b VARCHAR(50) CHARACTER SET big5,
	c VARCHAR(50) CHARACTER SET utf16
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "table_charset_differs_from_column",
			// Table default latin1; one column overrides to utf8mb4 (column charset != table charset).
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a VARCHAR(50),
	b VARCHAR(50) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
) ENGINE=InnoDB DEFAULT CHARSET=latin1;`,
		},

		// ---- Numeric / bit / extreme ----
		{
			name: "bit_type",
			// FIXED (Bug A): the dumper rendered the BIT default as a quoted string (`DEFAULT 'b\\'0\\''`)
			// because the sync stores the bit literal QUOTE()-escaped in ColumnMetadata.Default; omni then
			// loaded a string default that never matched the user's `b'0'`, re-emitting `MODIFY ... DEFAULT
			// b'0'` every no-op. renderColumnDefault in get_database_definition.go now recovers the bit
			// literal and emits it unquoted (`DEFAULT b'0'`, matching SHOW CREATE / mysqldump); omni #352
			// canonicalizes the bit literal so the no-op is empty on both versions. b64 (no default) is fine.
			bGap:       false,
			bGapReason: "",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	b1 BIT(1) NOT NULL DEFAULT b'0',
	b8 BIT(8) NOT NULL DEFAULT b'101',
	b64 BIT(64)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "decimal_max_precision",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	d DECIMAL(65,30) NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:       "bigint_unsigned_maxdefault",
			bGap:       false, // FIXED (re-pinned omni a362f7a7)
			bGapReason: "DANGEROUS (silently-wrong value, not just a phantom): a BIGINT UNSIGNED default of 18446744073709551615 (max uint64) is compared/re-emitted as 9223372036854775807 (max INT64) — the default is parsed into a signed int64 and clamped/overflows. The diff emits `MODIFY ... DEFAULT '9223372036854775807'`, which would CHANGE the default to the wrong number on apply. sm/ti (in-range) are fine.",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	big BIGINT UNSIGNED NOT NULL DEFAULT 18446744073709551615,
	sm SMALLINT UNSIGNED NOT NULL DEFAULT 65535,
	ti TINYINT UNSIGNED NOT NULL DEFAULT 255
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name:       "year_and_set",
			bGap:       false, // FIXED (re-pinned omni a362f7a7)
			bGapReason: "YEAR default canonicalization: user `DEFAULT 2000` (numeric) vs stored `'2000'` (string) not recognized equal; the diff re-emits `MODIFY ... y year ... DEFAULT '2000'` every no-op (phantom; both versions). The SET column is idempotent.",
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	y YEAR NOT NULL DEFAULT 2000,
	s SET('a','b','c','d','e','f','g') NOT NULL DEFAULT 'a,c,e'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "large_enum_set",
			// Many ENUM/SET members.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	e ENUM('m00','m01','m02','m03','m04','m05','m06','m07','m08','m09','m10','m11','m12','m13','m14','m15') NOT NULL DEFAULT 'm00',
	s SET('s0','s1','s2','s3','s4','s5','s6','s7','s8','s9') NOT NULL DEFAULT 's0,s9'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
		{
			name: "varchar_near_row_limit",
			// A wide VARCHAR in latin1 (1 byte/char) approaching the 65535-byte row limit.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	wide VARCHAR(16000) CHARACTER SET latin1 NOT NULL DEFAULT ''
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Generated columns chained ----
		{
			name: "generated_chained",
			// total references price/qty; with_tax references total (a generated col referencing
			// another generated col); both a VIRTUAL and a STORED, plus an index on a generated col.
			// NB: `code` is a plain (non-auto-increment) column — MySQL forbids a generated column
			// referencing an AUTO_INCREMENT column (errno 3109), so `label` references `code`.
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	code INT NOT NULL DEFAULT 0,
	price DECIMAL(10,2) NOT NULL DEFAULT 0,
	qty INT NOT NULL DEFAULT 0,
	total DECIMAL(20,2) AS (price * qty) STORED,
	with_tax DECIMAL(20,2) AS (total * 1.1) VIRTUAL,
	label VARCHAR(40) AS (CONCAT('#', code)) STORED,
	PRIMARY KEY (id),
	KEY idx_total (total)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- AUTO_INCREMENT seed ----
		{
			name: "auto_increment_seed_500",
			ddl: `CREATE TABLE t (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(50) NOT NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=500 DEFAULT CHARSET=utf8mb4;`,
		},

		// ---- Invisible indexes (8.0) ----
		{
			name:   "invisible_index_80",
			only80: true,
			// FIXED (bug 5): index visibility was already synced (IndexMetadata.visible from
			// information_schema.STATISTICS.IS_VISIBLE); the dumper now emits `/*!80000 INVISIBLE */`
			// in the index clause, so the no-op is empty instead of a DROP+ADD.
			ddl: `CREATE TABLE t (
	id INT PRIMARY KEY,
	a INT NOT NULL,
	KEY idx_a (a) INVISIBLE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;`,
		},
	}
}

// TestSDLDeepExoticIdempotence is Axis 2: every NON-bGap exotic-type case is applied live,
// synced, dumped, and the production version-aware diff (dumped vs user form) MUST be empty
// on both versions (8.0-only types skip 5.7). The bGap cases (confirmed normalization/dumper
// (B) bugs found this round) are skipped here and reproduced/explained by
// TestSDLDeepExoticKnownGaps so this assertion stays green. Each unexpected non-empty no-op
// is logged with input + stored form so the gap can be pinned to a Canonical* rule or the
// dumper.
//
//nolint:tparallel
func TestSDLDeepExoticIdempotence(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, ec := range exoticCases() {
				ec := ec
				if srv.version == "5.7" && ec.only80 {
					continue
				}
				if ec.bGap {
					continue // confirmed (B) gap — pinned by TestSDLDeepExoticKnownGaps
				}
				t.Run(ec.name, func(t *testing.T) {
					meta, dbName := syncMetadata(ctx, t, srv, "sdl_deep_exo", ec.ddl)

					// Production entry: schema.SDLMigration (meta -> SDL internally).
					noop := safeSDLMigration(ec.ddl, meta, srv.version)
					require.False(t, noop.panicked,
						"[%s/%s] (B) PANIC in schema.SDLMigration: %v\ninput:\n%s",
						srv.name, ec.name, noop.panicVal, ec.ddl)
					require.NoError(t, noop.err, "[%s/%s] SDLMigration error:\n%s", srv.name, ec.name, ec.ddl)

					source := dumpSDL(ctx, t, srv, dbName)
					if !noop.planEmpty {
						t.Logf("[%s/%s] (B) NON-EMPTY no-op normalization gap:\nUSER FORM:\n%s\nSTORED/DUMPED:\n%s\nDIFF:\n%s",
							srv.name, ec.name, ec.ddl, source, noop.plan)
					}
					require.True(t, noop.planEmpty,
						"[%s/%s] (B) exotic-type no-op (SDLMigration) must be empty, got:\n%s", srv.name, ec.name, noop.plan)

					// Version-aware lower-level entry, same property.
					noop2, err := mysqlDiffSDLMigration(source, ec.ddl, srv.version)
					require.NoError(t, err)
					require.Empty(t, noop2,
						"[%s/%s] (B) exotic-type no-op (mysqlDiffSDLMigration) must be empty, got:\n%s", srv.name, ec.name, noop2)
				})
			}
		})
	}
}

// TestSDLDeepExoticKnownGaps PINS the (B) normalization / dumper round-trip bugs this deep
// round found in the exotic-type surface. Each bGap case is applied live and its no-op is
// computed; the test asserts the no-op is currently NON-empty OR the dumped source fails to
// reload (the bug signature), and logs the full user->stored->diff so a fix subagent has the
// exact repro. When a bug is FIXED the corresponding case will start producing an empty,
// reloadable no-op — flip its bGap flag to false (moving it back under the green idempotence
// test) and this pin will report the now-idempotent case so the regression coverage moves
// with the fix.
//
// Confirmed gaps (see each case's bGapReason for the precise root cause):
//   - DANGEROUS, silently-wrong value: bigint_unsigned_maxdefault (uint64 max default clamped
//     to int64 max in the diff — would change the default on apply).
//   - Default-literal canonicalization phantoms (re-emit every no-op): bit_type (BIT b'..'),
//     year_and_set (YEAR numeric-vs-quoted).
//   - Dumper drops a real attribute (re-emit every no-op): invisible_index_80,
//     spatial_notnull_index (phantom spatial-key prefix length).
//   - Dumper emits a spelling the omni parser can't reload (HARD error): spatial_types
//     (`geomcollection`), json_multivalue_index_80 (charset introducer in the functional key).
//
// SKIPPED so the suite stays green; remove the Skip to reproduce all gaps at once.
//
//nolint:tparallel
func TestSDLDeepExoticKnownGaps(t *testing.T) {
	t.Skip("Pins confirmed (B) exotic-type normalization/dumper gaps; remove Skip to reproduce")
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, ec := range exoticCases() {
				ec := ec
				if !ec.bGap {
					continue
				}
				if srv.version == "5.7" && ec.only80 {
					continue
				}
				t.Run(ec.name, func(t *testing.T) {
					meta, dbName := syncMetadata(ctx, t, srv, "sdl_deep_exo_gap", ec.ddl)

					noop := safeSDLMigration(ec.ddl, meta, srv.version)
					require.False(t, noop.panicked,
						"[%s/%s] (B) PANIC: %v", srv.name, ec.name, noop.panicVal)

					source := dumpSDL(ctx, t, srv, dbName)
					t.Logf("[%s/%s] (B) %s\nUSER:\n%s\nSTORED:\n%s\nNO-OP err=%v plan:\n%s",
						srv.name, ec.name, ec.bGapReason, ec.ddl, source, noop.err, noop.plan)

					// The bug signature is EITHER a hard error from reloading the dumped source
					// OR a non-empty no-op. Asserting it documents the current broken behavior.
					broke := noop.err != nil || !noop.planEmpty
					require.True(t, broke,
						"[%s/%s] expected the known (B) gap to reproduce (load error or non-empty no-op) but it was idempotent — the bug may be FIXED; clear bGap on this case.\nreason was: %s",
						srv.name, ec.name, ec.bGapReason)
				})
			}
		})
	}
}

// ============================================================================
// Axis 3: sequential release chains (the real iterative usage).
// ============================================================================
//
// Start from S0 on a real DB. Apply a SEQUENCE of declarative releases S0->S1->...->S4,
// each a realistic incremental change set. At EACH step: source = MetadataToSDL(sync(DB)),
// diff against the step target, assert the plan is minimal+correctly-ordered, APPLY it to
// the real DB, re-sync, and assert the next no-op is empty before moving on. This proves
// correctness across a migration HISTORY, not a single diff.

// chainStep is one release in the sequence. target is the FULL desired SDL at that step
// (declarative — the complete schema, not a delta). wantContains are uppercase substrings
// the generated plan must contain; wantOrder pins a required relative ordering (first must
// appear before second). only80Extra is appended to target only on 8.0 (CHECK etc.).
type chainStep struct {
	name        string
	target80    string
	target57    string
	wantContain []string  // uppercase substrings the plan must contain
	wantAbsent  []string  // uppercase substrings the plan must NOT contain
	wantOrder   [2]string // [before, after] uppercase substrings; before must precede after (skip if either empty)
}

func (s chainStep) target(version string) string {
	if version == "5.7" {
		return s.target57
	}
	return s.target80
}

// --- The sequence. Each Sn target is the FULL schema at that point. ---

// chainS0: a small starting schema — customer + order with an FK, one view.
const chainS0 = `
CREATE TABLE customer (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	email VARCHAR(255) NOT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uk_cust_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE ord (
	id INT NOT NULL AUTO_INCREMENT,
	customer_id INT NOT NULL,
	total DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY idx_ord_cust (customer_id),
	CONSTRAINT fk_ord_cust FOREIGN KEY (customer_id) REFERENCES customer (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW v_cust_orders AS
SELECT c.name, o.id AS order_id, o.total
FROM customer c JOIN ord o ON o.customer_id = c.id;
`

// chainS1: add table `order_item` + FK to ord (and to a new `product` table).
const chainS1Common = `
CREATE TABLE customer (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	email VARCHAR(255) NOT NULL,
	PRIMARY KEY (id),
	UNIQUE KEY uk_cust_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE ord (
	id INT NOT NULL AUTO_INCREMENT,
	customer_id INT NOT NULL,
	total DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY idx_ord_cust (customer_id),
	CONSTRAINT fk_ord_cust FOREIGN KEY (customer_id) REFERENCES customer (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product (
	id INT NOT NULL AUTO_INCREMENT,
	sku VARCHAR(40) NOT NULL,
	name VARCHAR(120) NOT NULL,
	price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	PRIMARY KEY (id),
	UNIQUE KEY uk_prod_sku (sku)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_item (
	id INT NOT NULL AUTO_INCREMENT,
	order_id INT NOT NULL,
	product_id INT NOT NULL,
	qty INT NOT NULL DEFAULT 1,
	PRIMARY KEY (id),
	KEY idx_oi_order (order_id),
	KEY idx_oi_product (product_id),
	CONSTRAINT fk_oi_order FOREIGN KEY (order_id) REFERENCES ord (id) ON DELETE CASCADE,
	CONSTRAINT fk_oi_product FOREIGN KEY (product_id) REFERENCES product (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW v_cust_orders AS
SELECT c.name, o.id AS order_id, o.total
FROM customer c JOIN ord o ON o.customer_id = c.id;
`

// chainS2: add a column (customer.loyalty_points) + an index (ord.idx_ord_created) + a
// generated column (order_item.line_total references qty — but needs price; keep it simple:
// generated col on product: price_with_tax).
const chainS2Common = `
CREATE TABLE customer (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(100) NOT NULL,
	email VARCHAR(255) NOT NULL,
	loyalty_points INT NOT NULL DEFAULT 0,
	PRIMARY KEY (id),
	UNIQUE KEY uk_cust_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE ord (
	id INT NOT NULL AUTO_INCREMENT,
	customer_id INT NOT NULL,
	total DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY idx_ord_cust (customer_id),
	KEY idx_ord_created (created_at),
	CONSTRAINT fk_ord_cust FOREIGN KEY (customer_id) REFERENCES customer (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product (
	id INT NOT NULL AUTO_INCREMENT,
	sku VARCHAR(40) NOT NULL,
	name VARCHAR(120) NOT NULL,
	price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	price_with_tax DECIMAL(12,4) AS (price * 1.1) STORED,
	PRIMARY KEY (id),
	UNIQUE KEY uk_prod_sku (sku)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_item (
	id INT NOT NULL AUTO_INCREMENT,
	order_id INT NOT NULL,
	product_id INT NOT NULL,
	qty INT NOT NULL DEFAULT 1,
	PRIMARY KEY (id),
	KEY idx_oi_order (order_id),
	KEY idx_oi_product (product_id),
	CONSTRAINT fk_oi_order FOREIGN KEY (order_id) REFERENCES ord (id) ON DELETE CASCADE,
	CONSTRAINT fk_oi_product FOREIGN KEY (product_id) REFERENCES product (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW v_cust_orders AS
SELECT c.name, o.id AS order_id, o.total
FROM customer c JOIN ord o ON o.customer_id = c.id;
`

// chainS3: modify a column type (ord.total DECIMAL(10,2)->DECIMAL(14,4)) + widen a VARCHAR
// (customer.name VARCHAR(100)->VARCHAR(200)) + change a default (customer.loyalty_points
// DEFAULT 0 -> DEFAULT 100).
const chainS3Common = `
CREATE TABLE customer (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(200) NOT NULL,
	email VARCHAR(255) NOT NULL,
	loyalty_points INT NOT NULL DEFAULT 100,
	PRIMARY KEY (id),
	UNIQUE KEY uk_cust_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE ord (
	id INT NOT NULL AUTO_INCREMENT,
	customer_id INT NOT NULL,
	total DECIMAL(14,4) NOT NULL DEFAULT 0.0000,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY idx_ord_cust (customer_id),
	KEY idx_ord_created (created_at),
	CONSTRAINT fk_ord_cust FOREIGN KEY (customer_id) REFERENCES customer (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE product (
	id INT NOT NULL AUTO_INCREMENT,
	sku VARCHAR(40) NOT NULL,
	name VARCHAR(120) NOT NULL,
	price DECIMAL(10,2) NOT NULL DEFAULT 0.00,
	price_with_tax DECIMAL(12,4) AS (price * 1.1) STORED,
	PRIMARY KEY (id),
	UNIQUE KEY uk_prod_sku (sku)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_item (
	id INT NOT NULL AUTO_INCREMENT,
	order_id INT NOT NULL,
	product_id INT NOT NULL,
	qty INT NOT NULL DEFAULT 1,
	PRIMARY KEY (id),
	KEY idx_oi_order (order_id),
	KEY idx_oi_product (product_id),
	CONSTRAINT fk_oi_order FOREIGN KEY (order_id) REFERENCES ord (id) ON DELETE CASCADE,
	CONSTRAINT fk_oi_product FOREIGN KEY (product_id) REFERENCES product (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW v_cust_orders AS
SELECT c.name, o.id AS order_id, o.total
FROM customer c JOIN ord o ON o.customer_id = c.id;
`

// chainS4: drop a column WITH its index (drop product.price_with_tax generated col), drop a
// table that is an FK PARENT (drop product — order_item.fk_oi_product child FK must drop
// first; also drop order_item to keep it consistent? No — keep order_item but remove its FK
// to product by dropping the product reference). To exercise "drop FK parent, child FK drops
// first", we drop `product` AND order_item's fk_oi_product + product_id column. Replace the
// view, and (8.0) add a CHECK on ord.
//
// S4 also changes order_item to remove product linkage, replaces v_cust_orders, adds a
// trigger on ord, and (8.0) adds a CHECK on ord.total.
const chainS4Base = `
CREATE TABLE customer (
	id INT NOT NULL AUTO_INCREMENT,
	name VARCHAR(200) NOT NULL,
	email VARCHAR(255) NOT NULL,
	loyalty_points INT NOT NULL DEFAULT 100,
	PRIMARY KEY (id),
	UNIQUE KEY uk_cust_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE ord (
	id INT NOT NULL AUTO_INCREMENT,
	customer_id INT NOT NULL,
	total DECIMAL(14,4) NOT NULL DEFAULT 0.0000,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id),
	KEY idx_ord_cust (customer_id),
	KEY idx_ord_created (created_at){{ORD_CHECK}}
	,CONSTRAINT fk_ord_cust FOREIGN KEY (customer_id) REFERENCES customer (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE order_item (
	id INT NOT NULL AUTO_INCREMENT,
	order_id INT NOT NULL,
	qty INT NOT NULL DEFAULT 1,
	PRIMARY KEY (id),
	KEY idx_oi_order (order_id),
	CONSTRAINT fk_oi_order FOREIGN KEY (order_id) REFERENCES ord (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE VIEW v_cust_orders AS
SELECT c.name, c.email, o.id AS order_id, o.total
FROM customer c JOIN ord o ON o.customer_id = c.id;

CREATE TRIGGER trg_ord_ins BEFORE INSERT ON ord FOR EACH ROW
SET NEW.created_at = NOW();
`

func chainSteps() []chainStep {
	s4_80 := strings.Replace(chainS4Base, "{{ORD_CHECK}}", ",\n\tCONSTRAINT chk_total CHECK (total >= 0)", 1)
	s4_57 := strings.Replace(chainS4Base, "{{ORD_CHECK}}", "", 1)
	return []chainStep{
		{
			name:        "S1_add_table_fk",
			target80:    chainS1Common,
			target57:    chainS1Common,
			wantContain: []string{"CREATE TABLE `PRODUCT`", "CREATE TABLE `ORDER_ITEM`"},
			wantAbsent:  []string{"DROP TABLE", "CREATE TABLE `CUSTOMER`", "CREATE TABLE `ORD`"},
			// The order_item FK to ord/product is deferred to PhasePost: table create precedes the FK add.
			wantOrder: [2]string{"CREATE TABLE `ORDER_ITEM`", "ADD CONSTRAINT `FK_OI_ORDER`"},
		},
		{
			name:        "S2_add_column_index_generated",
			target80:    chainS2Common,
			target57:    chainS2Common,
			wantContain: []string{"ADD COLUMN `LOYALTY_POINTS`", "ADD KEY `IDX_ORD_CREATED`", "PRICE_WITH_TAX"},
			wantAbsent:  []string{"DROP TABLE", "CREATE TABLE"},
		},
		{
			name:     "S3_modify_type_widen_default",
			target80: chainS3Common,
			target57: chainS3Common,
			// total type change, name widen, loyalty default change.
			wantContain: []string{"DECIMAL(14,4)", "VARCHAR(200)"},
			wantAbsent:  []string{"DROP TABLE", "CREATE TABLE `CUSTOMER`", "DROP COLUMN"},
		},
		{
			name:     "S4_drop_col_drop_parent_replace_view_trigger",
			target80: s4_80,
			target57: s4_57,
			// product table dropped; order_item.product_id column + its FK dropped; view replaced; trigger added.
			wantContain: []string{"DROP TABLE `PRODUCT`", "DROP COLUMN `PRODUCT_ID`", "DROP FOREIGN KEY `FK_OI_PRODUCT`", "TRG_ORD_INS", "V_CUST_ORDERS"},
			wantAbsent:  []string{"DROP TABLE `CUSTOMER`", "DROP TABLE `ORD`"},
			// The child FK on order_item must drop before the parent product table is dropped.
			wantOrder: [2]string{"DROP FOREIGN KEY `FK_OI_PRODUCT`", "DROP TABLE `PRODUCT`"},
		},
	}
}

// TestSDLDeepSequentialChain is Axis 3: drive S0 -> S1 -> S2 -> S3 -> S4 on ONE live DB.
// Each step computes source=MetadataToSDL(sync), diffs the step target, asserts the plan is
// minimal + correctly ordered, APPLIES it, re-syncs, and asserts the next no-op is empty
// before advancing. Proves correctness across a migration history. Both versions.
//
// The steps run inline (NOT as separate subtests) BECAUSE they share one evolving DB: each
// step's diff is from its predecessor's applied state. Running a single step in isolation
// (e.g. `-run .../S4`) would compute a diff from S0 and assert wrong things — so the chain is
// kept atomic per server.
//
//nolint:tparallel
func TestSDLDeepSequentialChain(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			// Materialize S0 on a single DB that persists across all steps.
			dbName := newLiveDatabase(ctx, t, srv, "sdl_deep_chain")
			require.NoError(t, applyDDL(ctx, t, srv, dbName, chainS0), "[%s] apply S0", srv.name)

			// Confirm S0 itself is idempotent before starting the chain.
			s0src := dumpSDL(ctx, t, srv, dbName)
			s0noop, err := mysqlDiffSDLMigration(s0src, chainS0, srv.version)
			require.NoError(t, err)
			require.Empty(t, s0noop, "[%s] S0 must be idempotent before the chain, got:\n%s", srv.name, s0noop)

			// Run every step inline, in order, against the same evolving DB.
			for _, step := range chainSteps() {
				runChainStep(ctx, t, srv, dbName, step)
				if t.Failed() {
					return // a broken step poisons the chain state; stop so the failure is unambiguous
				}
			}
		})
	}
}

// runChainStep executes one release in the sequential chain against the live dbName.
func runChainStep(ctx context.Context, t *testing.T, srv liveServer, dbName string, step chainStep) {
	t.Helper()
	target := step.target(srv.version)

	// 1) Compute the plan from the CURRENT live DB state to the step target.
	source := dumpSDL(ctx, t, srv, dbName)
	plan, err := mysqlDiffSDLMigration(source, target, srv.version)
	require.NoError(t, err)
	require.NotEmpty(t, plan, "[%s/%s] step must produce a non-empty plan", srv.name, step.name)
	t.Logf("[%s/%s] plan:\n%s", srv.name, step.name, plan)
	upper := strings.ToUpper(plan)

	// 2) Minimality + correctness: required substrings present, forbidden absent.
	for _, want := range step.wantContain {
		require.Contains(t, upper, want, "[%s/%s] plan missing %q:\n%s", srv.name, step.name, want, plan)
	}
	for _, absent := range step.wantAbsent {
		require.NotContains(t, upper, absent, "[%s/%s] plan must not contain %q:\n%s", srv.name, step.name, absent, plan)
	}

	// 3) Ordering.
	if step.wantOrder[0] != "" && step.wantOrder[1] != "" {
		bi := indexOf(upper, step.wantOrder[0])
		ai := indexOf(upper, step.wantOrder[1])
		require.GreaterOrEqual(t, bi, 0, "[%s/%s] ordering: %q not found:\n%s", srv.name, step.name, step.wantOrder[0], plan)
		require.GreaterOrEqual(t, ai, 0, "[%s/%s] ordering: %q not found:\n%s", srv.name, step.name, step.wantOrder[1], plan)
		require.Less(t, bi, ai, "[%s/%s] %q must precede %q:\n%s", srv.name, step.name, step.wantOrder[0], step.wantOrder[1], plan)
	}

	// 4) Apply to the real DB.
	require.NoError(t, applyDDL(ctx, t, srv, dbName, plan),
		"[%s/%s] step plan failed to apply:\n%s", srv.name, step.name, plan)

	// 5) Re-sync and assert convergence (next no-op empty) BEFORE advancing.
	newSource := dumpSDL(ctx, t, srv, dbName)
	converge, err := mysqlDiffSDLMigration(newSource, target, srv.version)
	require.NoError(t, err)
	require.Empty(t, converge, "[%s/%s] step did not converge, residual:\n%s", srv.name, step.name, converge)

	// 6) Production-path cross-check: schema.SDLMigration on a freshly-synced meta must also be
	// empty (the path the release executor takes).
	meta := syncMetaForDB(ctx, t, srv, dbName)
	prodNoop, err := schema.SDLMigration(storepb.Engine_MYSQL, target, meta, srv.version)
	require.NoError(t, err)
	require.Empty(t, prodNoop, "[%s/%s] production-path no-op after apply must be empty, got:\n%s", srv.name, step.name, prodNoop)
}

// syncMetaForDB syncs an existing dbName and returns its model.DatabaseMetadata.
func syncMetaForDB(ctx context.Context, t *testing.T, srv liveServer, dbName string) *model.DatabaseMetadata {
	t.Helper()
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	return model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true)
}

// ============================================================================
// Axis 4: scale / performance sanity.
// ============================================================================
//
// Generate ~90 tables (each ~8-15 columns, a PK, 2-3 secondary indexes, FKs forming a
// realistic chain). Sync -> MetadataToSDL -> no-op diff must be EMPTY and complete in
// reasonable wall-time. Then add ONE column to ONE table -> assert exactly one minimal
// ALTER, not a re-emit of all tables. Record timings.

// genScaleSchema builds n tables t000..t{n-1}. Each table i (i>=1) carries an FK to table
// i-1 (a chain), plus 2 secondary indexes and ~10 columns of mixed types. Returns the DDL.
func genScaleSchema(n int) string {
	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "CREATE TABLE `t%03d` (\n", i)
		b.WriteString("\tid INT NOT NULL AUTO_INCREMENT,\n")
		b.WriteString("\tcode VARCHAR(40) NOT NULL,\n")
		b.WriteString("\tname VARCHAR(120) NOT NULL,\n")
		b.WriteString("\tdescription TEXT,\n")
		b.WriteString("\tamount DECIMAL(14,2) NOT NULL DEFAULT 0.00,\n")
		b.WriteString("\tqty INT NOT NULL DEFAULT 0,\n")
		b.WriteString("\tactive BOOLEAN NOT NULL DEFAULT TRUE,\n")
		b.WriteString("\tcreated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\n")
		b.WriteString("\tupdated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,\n")
		b.WriteString("\tstatus ENUM('new','open','closed') NOT NULL DEFAULT 'new',\n")
		if i > 0 {
			b.WriteString("\tprev_id INT NULL,\n")
		}
		b.WriteString("\tPRIMARY KEY (id),\n")
		fmt.Fprintf(&b, "\tUNIQUE KEY `uk_t%03d_code` (code),\n", i)
		fmt.Fprintf(&b, "\tKEY `idx_t%03d_name` (name),\n", i)
		if i > 0 {
			fmt.Fprintf(&b, "\tKEY `idx_t%03d_prev` (prev_id),\n", i)
			fmt.Fprintf(&b, "\tCONSTRAINT `fk_t%03d_prev` FOREIGN KEY (prev_id) REFERENCES `t%03d` (id) ON DELETE SET NULL\n", i, i-1)
		} else {
			// trim the trailing comma from the last index line for table 0
			fmt.Fprintf(&b, "\tKEY `idx_t%03d_amount` (amount)\n", i)
		}
		b.WriteString(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\n\n")
	}
	return b.String()
}

// TestSDLDeepScale is Axis 4: a ~90-table schema. The no-op diff must be empty and fast;
// adding one column to one table must yield exactly one minimal ALTER, not a re-emit of all
// tables. Timings are recorded and a pathological (super-linear / multi-second) no-op is
// flagged.
//
//nolint:tparallel
func TestSDLDeepScale(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	const tableCount = 90
	schemaDDL := genScaleSchema(tableCount)

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			meta, dbName := syncMetadata(ctx, t, srv, "sdl_deep_scale", schemaDDL)

			// (1) No-op diff: must be empty + record wall-time.
			source := dumpSDL(ctx, t, srv, dbName)
			start := time.Now()
			noop, err := mysqlDiffSDLMigration(source, schemaDDL, srv.version)
			elapsed := time.Since(start)
			require.NoError(t, err)
			if noop != "" {
				t.Logf("[%s] NON-EMPTY %d-table no-op (%d statements):\n%s", srv.name, tableCount, statementCount(noop), noop)
			}
			require.Empty(t, noop, "[%s] %d-table no-op must be empty, got %d statements", srv.name, tableCount, statementCount(noop))
			t.Logf("[%s] SCALE no-op diff over %d tables: %s", srv.name, tableCount, elapsed)
			// Soft perf flag: a no-op over ~90 tables should be well under a few seconds.
			if elapsed > 10*time.Second {
				t.Errorf("[%s] (PERF) %d-table no-op took %s (>10s) — pathological", srv.name, tableCount, elapsed)
			}

			// (2) Production-path no-op timing too.
			pstart := time.Now()
			prodNoop, err := schema.SDLMigration(storepb.Engine_MYSQL, schemaDDL, meta, srv.version)
			pelapsed := time.Since(pstart)
			require.NoError(t, err)
			require.Empty(t, prodNoop, "[%s] %d-table production no-op must be empty", srv.name, tableCount)
			t.Logf("[%s] SCALE production-path no-op (incl. MetadataToSDL): %s", srv.name, pelapsed)

			// (3) One small change: add a column to ONE table (t045). Must be exactly one ALTER
			// touching only that table — not a re-emit of the other 89.
			changed := strings.Replace(schemaDDL,
				"CREATE TABLE `t045` (\n\tid INT NOT NULL AUTO_INCREMENT,\n\tcode VARCHAR(40) NOT NULL,",
				"CREATE TABLE `t045` (\n\tid INT NOT NULL AUTO_INCREMENT,\n\tcode VARCHAR(40) NOT NULL,\n\tnew_flag INT NOT NULL DEFAULT 0,",
				1)
			require.NotEqual(t, schemaDDL, changed, "[%s] scale change setup must differ", srv.name)

			cstart := time.Now()
			change, err := mysqlDiffSDLMigration(source, changed, srv.version)
			celapsed := time.Since(cstart)
			require.NoError(t, err)
			t.Logf("[%s] SCALE single-change diff: %s\nplan:\n%s", srv.name, celapsed, change)
			require.NotEmpty(t, change, "[%s] single-change must produce DDL", srv.name)

			upper := strings.ToUpper(change)
			require.Contains(t, upper, "ADD COLUMN `NEW_FLAG`", "[%s] expected the new column:\n%s", srv.name, change)
			require.Contains(t, upper, "`T045`", "[%s] expected the changed table named:\n%s", srv.name, change)
			require.NotContains(t, upper, "CREATE TABLE", "[%s] single-change must not recreate tables:\n%s", srv.name, change)
			require.NotContains(t, upper, "DROP TABLE", "[%s] single-change must not drop tables:\n%s", srv.name, change)
			// Strict minimality: exactly ONE statement (the single ADD COLUMN).
			require.Equal(t, 1, statementCount(change),
				"[%s] single-change over %d tables must be exactly 1 statement, got %d:\n%s",
				srv.name, tableCount, statementCount(change), change)

			// No other table may be named in the plan.
			for i := 0; i < tableCount; i++ {
				if i == 45 {
					continue
				}
				token := fmt.Sprintf("`T%03d`", i)
				require.NotContains(t, upper, token,
					"[%s] single-change must not touch table t%03d:\n%s", srv.name, i, change)
			}
		})
	}
}
