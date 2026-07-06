package mysql

// ENTERPRISE smoke axes for the MySQL declarative (SDL) migration path. The real-world
// suite (sdl_realworld_test.go) proved the path on mid-size production schemas; this file
// scales verification to ENTERPRISE corpora and adds the axes an enterprise rollout
// exercises hardest:
//
//	corpus     — Zabbix 7.0.0 (203 tables, 272 FKs, 65 changelog triggers), PrestaShop
//	             (243 tables), OpenEMR (283 tables, INSERTs stripped), and the stock
//	             MySQL `sys` schema (scaffolded, gated on an in-flight omni fix).
//	baseline   — every corpus loads live, and its canonical dump self-diffs EMPTY
//	             (determinism) and no-ops through the production schema.SDLMigration.
//	A1         — single-object CRUD (create / modify / drop) for every object kind, on a
//	             20-table Zabbix slice.
//	A2         — dependent objects: FK chains, trigger→table deps (the Zabbix changelog
//	             pattern), 3-deep view stacks, function-used-by-view, circular FK pairs.
//	A3         — special types: AUTO_INCREMENT, generated columns, ENUM/SET members,
//	             charset/collation, temporal precision, DECIMAL, TEXT/BLOB families,
//	             signedness, prefix-indexed TEXT, partition ops.
//	A5         — sequential chain (with one-shot endpoint equality), apply-back,
//	             combined release, scale timing guard, multi-file export round-trip.
//	A4 (fuzz)  — lives in sdl_enterprise_fuzz_test.go.
//
// The per-case oracle protocol (entOracle) is the realworld suite's, made reusable:
//
//	(1) load base B into an entsdl_-prefixed scratch DB, sync → canonical current SDL C;
//	(2) DDL = mysqlDiffSDLMigration(C, target T, version);
//	(3) apply DDL to the scratch DB — clean execution (dependency ordering) is under test;
//	(4) re-sync → C'; assert Diff(C', T) == "" (convergence) and Diff(C', C') == ""
//	    (idempotence);
//	(5) minimality asserts (exact statement counts) where cheap.
//
// Shared helpers (liveServers, createLiveMySQLDriver, newLiveDatabase, dumpSDL, applyDDL,
// statementCount, normalizeDelimiters, syncMetaForDB, objectCounts, addColumnToTable,
// addIndexToTable, addColumnToView, dropObjectBlock, dropTrigger, findObjectSegment,
// removeSegment, mustReplace, concatMultiFile) come from the sibling _test.go files.

import (
	"context"
	_ "embed"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
)

// ----------------------------------------------------------------------------
// Embedded enterprise corpora.
//
// Preprocessing applied when the corpus was lifted into testdata:
//   - zabbix:     Zabbix 7.0.0 create/mysql/schema.sql verbatim, minus the single
//                 INSERT INTO dbversion row (DDL-only testdata). Keeps the DELIMITER $$
//                 changelog-trigger block (normalizeDelimiters handles it), the 301
//                 standalone CREATE INDEX statements, and the 272 trailing
//                 ALTER TABLE ... ADD CONSTRAINT foreign keys.
//   - prestashop: db_structure.sql with the installer placeholders substituted
//                 (PREFIX_ -> ps_, ENGINE_TYPE -> InnoDB, ...). Leads with
//                 SET SESSION sql_mode='' (two DEFAULT '0000-00-00 00:00:00' columns
//                 need non-strict mode at load; the driver executes the corpus on a
//                 single connection so the session setting holds).
//   - openemr:    database.sql with all 5,821 INSERT statements stripped by a
//                 quote-aware scanner (283 tables, DDL-only).
//   - sys:        the stock MySQL 8.0 sys schema, single-file form. Formerly gated
//                 on two omni parser gaps it exposed (adjacent string literals #360,
//                 paren-subquery operand continuation #366) — both fixed and pinned.
// ----------------------------------------------------------------------------

//go:embed testdata/enterprise/zabbix.sql
var entZabbixSQL string

//go:embed testdata/enterprise/prestashop.sql
var entPrestashopSQL string

//go:embed testdata/enterprise/openemr.sql
var entOpenemrSQL string

//go:embed testdata/enterprise/sys.sql
var entSysSQL string

// entCorpus is one embedded enterprise schema.
type entCorpus struct {
	name string
	ddl  string
	// gate, when non-empty, skips every leg for this corpus with the given reason.
	gate string
	// gate57, when non-empty, skips only the 5.7 legs with the given reason.
	gate57 string
	// canonical marks ddl as a canonical SDL dump (alphabetical object order) rather
	// than upstream creation-ordered DDL. Canonical dumps are loaded by applying the
	// engine's own empty→schema plan, whose statements are dependency-ordered —
	// sequential client apply of the raw dump would break on view-on-view forward
	// references (sys: `host_summary` reads `x$...` views that sort later).
	canonical bool
}

func entCorpora() []entCorpus {
	return []entCorpus{
		{name: "zabbix", ddl: entZabbixSQL},
		{name: "prestashop", ddl: entPrestashopSQL},
		{name: "openemr", ddl: entOpenemrSQL},
		{name: "sys", ddl: entSysSQL, canonical: true,
			gate57: "sys corpus is the stock MySQL 8.0 sys schema; its views read performance_schema tables that 5.7.25 does not have"},
	}
}

// entLoadCorpus loads one corpus into a fresh scratch database, honoring per-version
// gates. Canonical dumps go through the engine's own dependency-ordered create plan
// (dogfooding: every corpus load exercises the empty→schema ordering guarantees).
func entLoadCorpus(ctx context.Context, t *testing.T, srv liveServer, prefix string, corpus entCorpus) string {
	t.Helper()
	if corpus.gate != "" {
		t.Skipf("[%s/%s] %s", srv.name, corpus.name, corpus.gate)
	}
	if corpus.gate57 != "" && srv.name == "mysql57" {
		t.Skipf("[%s/%s] %s", srv.name, corpus.name, corpus.gate57)
	}
	if !corpus.canonical {
		return entLoadDDL(ctx, t, srv, prefix+corpus.name, corpus.ddl)
	}
	dbName := newLiveDatabase(ctx, t, srv, prefix+corpus.name)
	plan, err := mysqlDiffSDLMigration("", corpus.ddl, srv.version)
	require.NoError(t, err, "[%s/%s] empty→corpus create plan", srv.name, corpus.name)
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, plan, db.ExecuteOptions{})
	require.NoError(t, err, "[%s/%s] apply engine create plan", srv.name, corpus.name)
	return dbName
}

// entNormalizeDelimiters rewrites a multi-DELIMITER script into the no-DELIMITER form
// the production split path handles. It extends the realworld suite's normalizeDelimiters
// for the Zabbix style, where the custom delimiter sits on its OWN line and most trigger
// bodies are ALREADY ';'-terminated — naively rewriting every delimiter line to ";" left
// bare ";" statements that MySQL rejects (Error 1064 near ';'). Here a delimiter (inline
// or standalone) only yields a ";" when the statement is not already terminated.
func entNormalizeDelimiters(ddl string) string {
	if !strings.Contains(ddl, "DELIMITER") {
		return ddl
	}
	lines := strings.Split(ddl, "\n")
	out := make([]string, 0, len(lines))
	delim := ";"
	terminated := func() bool {
		for i := len(out) - 1; i >= 0; i-- {
			trimmed := strings.TrimSpace(out[i])
			if trimmed == "" {
				continue
			}
			return strings.HasSuffix(trimmed, ";")
		}
		return true
	}
	for _, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(strings.ToUpper(trimmed), "DELIMITER ") {
			delim = strings.TrimSpace(trimmed[len("DELIMITER "):])
			if delim == "" {
				delim = ";"
			}
			continue
		}
		if delim == ";" {
			out = append(out, ln)
			continue
		}
		if trimmed == delim {
			if !terminated() {
				out = append(out, ";")
			}
			continue
		}
		if strings.HasSuffix(trimmed, delim) {
			idx := strings.LastIndex(ln, delim)
			body := strings.TrimRight(ln[:idx], " \t")
			if !strings.HasSuffix(strings.TrimSpace(body), ";") {
				body += ";"
			}
			out = append(out, body)
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}

// entLoadDDL creates a fresh entsdl_-prefixed scratch database on srv (dropped in
// cleanup by newLiveDatabase) and applies ddl through the production driver path.
func entLoadDDL(ctx context.Context, t *testing.T, srv liveServer, prefix, ddl string) string {
	t.Helper()
	dbName := newLiveDatabase(ctx, t, srv, prefix)
	driver, err := createLiveMySQLDriver(ctx, srv, dbName)
	require.NoError(t, err)
	defer driver.Close(ctx)
	_, err = driver.Execute(ctx, entNormalizeDelimiters(ddl), db.ExecuteOptions{})
	require.NoError(t, err, "[%s] apply base DDL", srv.name)
	return dbName
}

// entPlanStatementCount counts plan statements compound-aware: routine/trigger bodies
// carry internal ';' that the naive statementCount (string split) over-counts, so the
// minimality asserts split with the production splitter instead.
func entPlanStatementCount(t *testing.T, plan string) int {
	t.Helper()
	stmts, err := mysqlparser.SplitSQL(plan)
	require.NoError(t, err, "split plan for statement count:\n%s", plan)
	n := 0
	for _, s := range stmts {
		if text := strings.TrimSpace(s.Text); text != "" && text != ";" {
			n++
		}
	}
	return n
}

// entOracle runs steps (2)-(4) of the oracle protocol against an already-loaded scratch
// database: dump the canonical current SDL, diff to target, apply the generated DDL, and
// prove convergence + idempotence. Returns the generated plan for minimality asserts.
func entOracle(ctx context.Context, t *testing.T, srv liveServer, dbName, target, label string) string {
	t.Helper()
	source := dumpSDL(ctx, t, srv, dbName)
	require.NotEqual(t, source, target, "[%s] target must differ from source", label)

	plan, err := mysqlDiffSDLMigration(source, target, srv.version)
	require.NoError(t, err, "[%s] diff source->target", label)
	require.NotEmpty(t, plan, "[%s] expected a non-empty migration plan", label)
	t.Logf("[%s] plan (%d stmts):\n%s", label, statementCount(plan), plan)

	applyErr := applyDDL(ctx, t, srv, dbName, plan)
	require.NoError(t, applyErr, "[%s] generated plan failed to apply:\n%s", label, plan)

	after := dumpSDL(ctx, t, srv, dbName)
	converge, err := mysqlDiffSDLMigration(after, target, srv.version)
	require.NoError(t, err, "[%s] converge diff", label)
	require.Empty(t, converge, "[%s] did not converge; residual:\n%s\nplan was:\n%s", label, converge, plan)

	self, err := mysqlDiffSDLMigration(after, after, srv.version)
	require.NoError(t, err, "[%s] idempotence diff", label)
	require.Empty(t, self, "[%s] post-apply dump not idempotent:\n%s", label, self)
	return plan
}

// ----------------------------------------------------------------------------
// Zabbix slice extraction.
//
// The corpus declares tables, then standalone CREATE [UNIQUE] INDEX statements, then a
// DELIMITER block of changelog triggers, then trailing ALTER TABLE ... ADD CONSTRAINT
// foreign keys. A slice keeps the CREATE TABLE + CREATE INDEX statements of the included
// tables, the triggers ON included tables (their bodies write `changelog`, so changelog
// must be included), and only the FKs whose BOTH endpoints are included — yielding a
// self-consistent sub-schema in the corpus's own statement order.
// ----------------------------------------------------------------------------

type entZbxKind int

const (
	entZbxTable entZbxKind = iota
	entZbxIndex
	entZbxFK
	entZbxTrigger
	entZbxOther
)

type entZbxStmt struct {
	text     string
	kind     entZbxKind
	table    string
	refTable string
}

var (
	entReTable   = regexp.MustCompile("(?i)^CREATE TABLE `(\\w+)`")
	entReIndex   = regexp.MustCompile("(?i)^CREATE (?:UNIQUE )?INDEX `\\w+` ON `(\\w+)`")
	entReFK      = regexp.MustCompile("(?is)^ALTER TABLE `(\\w+)` ADD CONSTRAINT `\\w+` FOREIGN KEY.*?REFERENCES `(\\w+)`")
	entReTrigger = regexp.MustCompile(`(?is)^create\s+trigger\s+\w+\s+(?:before|after)\s+(?:insert|update|delete)\s+on\s+` + "`?(\\w+)`?")

	entZbxOnce  sync.Once
	entZbxStmts []entZbxStmt
	entZbxErr   error
)

// entZabbixStatements splits the normalized zabbix corpus once and classifies every
// statement for slicing.
func entZabbixStatements() ([]entZbxStmt, error) {
	entZbxOnce.Do(func() {
		stmts, err := mysqlparser.SplitSQL(entNormalizeDelimiters(entZabbixSQL))
		if err != nil {
			entZbxErr = err
			return
		}
		for _, s := range stmts {
			text := strings.TrimSpace(s.Text)
			if text == "" || text == ";" {
				continue
			}
			classified := entZbxStmt{text: text, kind: entZbxOther}
			if m := entReTable.FindStringSubmatch(text); m != nil {
				classified.kind, classified.table = entZbxTable, m[1]
			} else if m := entReIndex.FindStringSubmatch(text); m != nil {
				classified.kind, classified.table = entZbxIndex, m[1]
			} else if m := entReFK.FindStringSubmatch(text); m != nil {
				classified.kind, classified.table, classified.refTable = entZbxFK, m[1], m[2]
			} else if m := entReTrigger.FindStringSubmatch(text); m != nil {
				classified.kind, classified.table = entZbxTrigger, m[1]
			}
			entZbxStmts = append(entZbxStmts, classified)
		}
	})
	return entZbxStmts, entZbxErr
}

// entZabbixSlice extracts a self-consistent slice of the zabbix corpus containing the
// given tables (which must include changelog — the trigger bodies write to it).
func entZabbixSlice(t *testing.T, tables ...string) string {
	t.Helper()
	include := make(map[string]bool, len(tables))
	for _, tbl := range tables {
		include[tbl] = true
	}
	require.True(t, include["changelog"], "zabbix slices must include changelog (trigger bodies write to it)")

	stmts, err := entZabbixStatements()
	require.NoError(t, err, "split zabbix corpus")

	var b strings.Builder
	found := map[string]bool{}
	for _, s := range stmts {
		keep := false
		switch s.kind {
		case entZbxTable, entZbxIndex, entZbxTrigger:
			keep = include[s.table]
			if s.kind == entZbxTable && keep {
				found[s.table] = true
			}
		case entZbxFK:
			keep = include[s.table] && include[s.refTable]
		case entZbxOther:
			keep = false
		default:
			keep = false
		}
		if keep {
			b.WriteString(s.text)
			if !strings.HasSuffix(strings.TrimSpace(s.text), ";") {
				b.WriteString(";")
			}
			b.WriteString("\n")
		}
	}
	for _, tbl := range tables {
		require.True(t, found[tbl], "zabbix slice: table %q not found in corpus", tbl)
	}
	return b.String()
}

// entSliceCoreTables is the ~20-table Zabbix slice used by A1/A2/A5: a real 4-deep FK
// chain (item_tag -> items -> hosts -> proxy -> proxy_group), the changelog trigger
// pattern on 8 of the tables (hosts alone carries 5 triggers, including a BEGIN/END
// body), self-referencing FKs (hosts.templateid, items.templateid/master_itemid), and
// composite unique indexes (hosts_groups).
var entSliceCoreTables = []string{
	"changelog",
	"role", "users", "media_type", "media",
	"proxy_group", "proxy", "proxy_rtdata",
	"maintenances", "hosts", "host_rtdata",
	"hstgrp", "hosts_groups",
	"interface", "valuemap", "items", "item_tag", "item_preproc",
	"drules", "dchecks",
	"connector", "connector_tag",
}

// ----------------------------------------------------------------------------
// Targeted string-surgery helpers on the canonical dump (ent-prefixed; the shared
// realworld helpers cover the untargeted forms).
// ----------------------------------------------------------------------------

// entTableSegment returns the [start,end) range of the CREATE TABLE statement for table
// in source (located via the SDL splitter, so partition comments stay in-segment).
func entTableSegment(t *testing.T, source, table string) (int, int) {
	t.Helper()
	header := "CREATE TABLE `" + table + "`"
	start, end := findObjectSegment(t, source, func(stmt string) bool {
		return strings.HasPrefix(stmt, header)
	})
	require.GreaterOrEqual(t, start, 0, "table %q not found in source", table)
	return start, end
}

// entReplaceInTable replaces old with new exactly once, scoped to table's CREATE block.
func entReplaceInTable(t *testing.T, source, table, old, replacement string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	require.Contains(t, seg, old, "table %q block must contain %q", table, old)
	return source[:start] + strings.Replace(seg, old, replacement, 1) + source[end:]
}

// entReplaceAllInTable replaces every occurrence of old within table's CREATE block.
func entReplaceAllInTable(t *testing.T, source, table, old, replacement string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	require.Contains(t, seg, old, "table %q block must contain %q", table, old)
	return source[:start] + strings.ReplaceAll(seg, old, replacement) + source[end:]
}

// entDropTableBlock removes the whole CREATE TABLE statement for table.
func entDropTableBlock(t *testing.T, source, table string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	return removeSegment(source, start, end)
}

// entDropObjectNamed removes the CREATE statement whose text contains `<KEYWORD>
// `+"`name`"+“ (EVENT and other kinds the shared dropObjectBlock does not cover).
func entDropObjectNamed(t *testing.T, source, keyword, name string) string {
	t.Helper()
	needle := keyword + " `" + strings.ToUpper(name) + "`"
	start, end := findObjectSegment(t, source, func(stmt string) bool {
		u := strings.ToUpper(stmt)
		return strings.HasPrefix(u, "CREATE ") && strings.Contains(u, needle)
	})
	require.GreaterOrEqual(t, start, 0, "%s %q not found in source", keyword, name)
	return removeSegment(source, start, end)
}

// entDropLineInTable removes the single body line containing marker from table's CREATE
// block, fixing the dangling comma when the removed line was the last body element.
func entDropLineInTable(t *testing.T, source, table, marker string) string {
	t.Helper()
	return entEditLineInTable(t, source, table, marker, "")
}

// entReplaceLineInTable swaps the single body line containing marker for newLine
// (indentation and trailing comma are managed here).
func entReplaceLineInTable(t *testing.T, source, table, marker, newLine string) string {
	t.Helper()
	return entEditLineInTable(t, source, table, marker, newLine)
}

// entEditLineInTable is the shared core of drop/replace-line: newLine == "" drops the
// marker line, otherwise the line is replaced by newLine (re-indented, comma preserved).
func entEditLineInTable(t *testing.T, source, table, marker, newLine string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	lines := strings.Split(seg, "\n")
	idx := -1
	for i, ln := range lines {
		if i == 0 {
			continue // never the CREATE TABLE header
		}
		if strings.Contains(ln, marker) {
			idx = i
			break
		}
	}
	require.GreaterOrEqual(t, idx, 0, "table %q has no body line containing %q:\n%s", table, marker, seg)
	hadComma := strings.HasSuffix(strings.TrimSpace(lines[idx]), ",")
	if newLine == "" {
		lines = append(lines[:idx], lines[idx+1:]...)
		if !hadComma {
			// Removed the last body element: strip the now-dangling comma above it.
			for j := idx - 1; j > 0; j-- {
				trimmed := strings.TrimRight(lines[j], " \t")
				if strings.HasSuffix(trimmed, ",") {
					lines[j] = strings.TrimSuffix(trimmed, ",")
					break
				}
				if strings.TrimSpace(trimmed) != "" {
					break
				}
			}
		}
	} else {
		replaced := "  " + strings.TrimSpace(newLine)
		if hadComma {
			replaced += ","
		}
		lines[idx] = replaced
	}
	return source[:start] + strings.Join(lines, "\n") + source[end:]
}

// entSetPartitionClause replaces table's whole partition clause (the dumper's
// /*!NNNNN PARTITION BY ... */ executable comment, or a plain clause) with newClause
// (plain form; pass "" to departition). The clause runs from the comment opener (or
// PARTITION BY) to the end of the statement, so it is rebuilt rather than patched.
func entSetPartitionClause(t *testing.T, source, table, newClause string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	pIdx := strings.Index(strings.ToUpper(seg), "PARTITION BY")
	require.GreaterOrEqual(t, pIdx, 0, "table %q has no PARTITION BY clause:\n%s", table, seg)
	clauseStart := pIdx
	if c := strings.LastIndex(seg[:pIdx], "/*!"); c >= 0 {
		clauseStart = c
	}
	// The clause (with its optional comment close) runs to the statement end; the
	// trailing ";" (when in-segment) is preserved.
	tail := ""
	rest := strings.TrimRight(seg[clauseStart:], " \t\n")
	if strings.HasSuffix(rest, ";") {
		tail = ";"
	}
	prefix := strings.TrimRight(seg[:clauseStart], " \t\n")
	if newClause == "" {
		return source[:start] + prefix + tail + source[end:]
	}
	return source[:start] + prefix + "\n" + newClause + tail + source[end:]
}

// entAppendTableClause appends clause (e.g. a plain PARTITION BY) to the end of table's
// CREATE statement, before the trailing semicolon.
func entAppendTableClause(t *testing.T, source, table, clause string) string {
	t.Helper()
	start, end := entTableSegment(t, source, table)
	seg := source[start:end]
	trimmed := strings.TrimRight(seg, " \t\n")
	require.True(t, strings.HasSuffix(trimmed, ";"), "table %q statement must end with ';':\n%s", table, seg)
	body := strings.TrimSuffix(trimmed, ";")
	return source[:start] + body + "\n" + clause + ";" + source[end:]
}

// entUintType returns the canonical dump spelling of INT UNSIGNED on srv. The sync
// driver's columnTypeCanonicalSynonyms map folds the SIGNED default display widths
// (int(11) -> int) but not the unsigned ones, so a 5.7 dump renders `int(10) unsigned`
// verbatim while a plain int renders `int` on both versions (an asymmetry the omni
// canonicalizer absorbs — recorded as an observation in the campaign findings).
func entUintType(srv liveServer) string {
	if srv.version == "5.7" {
		return "int(10) unsigned"
	}
	return "int unsigned"
}

// ----------------------------------------------------------------------------
// Baseline: every corpus loads on every version, and its canonical dump is a fixed
// point — self-diff empty AND the production schema.SDLMigration no-op empty.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseBaseline(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, corpus := range entCorpora() {
				corpus := corpus
				t.Run(corpus.name, func(t *testing.T) {
					dbName := entLoadCorpus(ctx, t, srv, "entsdl_bl_", corpus)
					meta := syncMetaForDB(ctx, t, srv, dbName)
					tbl, vw, fn, pr, tg := objectCounts(meta)
					t.Logf("[BASELINE %s/%s] loaded: tables=%d views=%d functions=%d procedures=%d triggers=%d (db=%s)",
						srv.name, corpus.name, tbl, vw, fn, pr, tg, dbName)

					source := dumpSDL(ctx, t, srv, dbName)
					require.NotEmpty(t, source, "[%s/%s] MetadataToSDL produced empty SDL", srv.name, corpus.name)

					selfDiff, err := mysqlDiffSDLMigration(source, source, srv.version)
					require.NoError(t, err)
					require.Empty(t, selfDiff, "[%s/%s] canonical dump must self-diff empty, got:\n%s",
						srv.name, corpus.name, selfDiff)

					noop, err := schema.SDLMigration(storepb.Engine_MYSQL, source, meta, srv.version)
					require.NoError(t, err)
					require.Empty(t, noop, "[%s/%s] production-path no-op must be empty, got:\n%s",
						srv.name, corpus.name, noop)
				})
			}
		})
	}
}

// ----------------------------------------------------------------------------
// A1: single-object CRUD — for each object kind, one create, one semantic modify, and
// one drop, chained on a freshly loaded Zabbix core slice (each phase runs the full
// oracle protocol from a fresh canonical dump).
// ----------------------------------------------------------------------------

// entA1Aux seeds the slice with one object of each kind that needs a pre-existing
// instance to modify/drop, plus a RANGE-partitioned table for the partition kind.
const entA1Aux = `
CREATE TABLE ent_ranked (
	id bigint unsigned NOT NULL,
	score integer DEFAULT '0' NOT NULL,
	ratio decimal(5,2) DEFAULT '0.00' NOT NULL,
	owner_userid bigint unsigned NULL,
	PRIMARY KEY (id)
) ENGINE=InnoDB;
CREATE TABLE ent_part_log (
	id bigint unsigned NOT NULL,
	bucket integer NOT NULL,
	note varchar(64) DEFAULT '' NOT NULL,
	PRIMARY KEY (id,bucket)
) ENGINE=InnoDB
PARTITION BY RANGE (bucket) (
	PARTITION p0 VALUES LESS THAN (100),
	PARTITION p1 VALUES LESS THAN (200)
);
CREATE VIEW ent_v_users AS SELECT userid, username, name FROM users;
CREATE FUNCTION ent_f_host_count() RETURNS INT READS SQL DATA RETURN (SELECT COUNT(*) FROM hosts);
CREATE PROCEDURE ent_p_touch_user(IN uid BIGINT UNSIGNED) BEGIN UPDATE users SET name = name WHERE userid = uid; END;
CREATE TRIGGER ent_trg_role_bi BEFORE INSERT ON role FOR EACH ROW SET NEW.name = TRIM(NEW.name);
CREATE EVENT ent_ev_hk ON SCHEDULE EVERY 1 DAY DO DELETE FROM changelog WHERE clock < 0;
`

// entCRUDPhase is one oracle round (create / modify / drop) within a kind.
type entCRUDPhase struct {
	name   string
	mutate func(t *testing.T, srv liveServer, source string) string
	// want are uppercased substrings the plan must contain.
	want []string
	// exactStmts, when > 0, asserts the plan is exactly this many statements
	// (single-object minimality).
	exactStmts int
}

// entCRUDKind is the create+modify+drop triple for one object kind.
type entCRUDKind struct {
	name   string
	skip57 string
	phases []entCRUDPhase
}

func entCRUDKinds() []entCRUDKind {
	return []entCRUDKind{
		{
			name: "table",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE TABLE ent_crud_t (\n  id bigint unsigned NOT NULL,\n  label varchar(40) DEFAULT '' NOT NULL,\n  PRIMARY KEY (id)\n) ENGINE=InnoDB;\n"
					},
					want:       []string{"CREATE TABLE", "ENT_CRUD_T"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceInTable(t, source, "ent_crud_t", ") ENGINE=InnoDB", ") ENGINE=InnoDB COMMENT='ent crud table'")
					},
					want:       []string{"ENT_CRUD_T", "COMMENT"},
					exactStmts: 1,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropTableBlock(t, source, "ent_crud_t")
					},
					want:       []string{"DROP TABLE", "ENT_CRUD_T"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "column",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addColumnToTable(t, source, "users", "`ent_crud_col` varchar(50) DEFAULT NULL")
					},
					want:       []string{"ALTER TABLE", "ENT_CRUD_COL"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceInTable(t, source, "users", "`ent_crud_col` varchar(50)", "`ent_crud_col` varchar(120)")
					},
					want:       []string{"ENT_CRUD_COL", "VARCHAR(120)"},
					exactStmts: 1,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "users", "`ent_crud_col`")
					},
					want:       []string{"DROP COLUMN", "ENT_CRUD_COL"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "index_plain",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "users", "KEY `ent_idx_theme` (`theme`)")
					},
					want:       []string{"ENT_IDX_THEME"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "users", "`ent_idx_theme`", "KEY `ent_idx_theme` (`theme`,`lang`)")
					},
					want: []string{"ENT_IDX_THEME"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "users", "`ent_idx_theme`")
					},
					want:       []string{"ENT_IDX_THEME"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "index_unique",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "users", "UNIQUE KEY `ent_uk_passwd` (`passwd`)")
					},
					want:       []string{"ENT_UK_PASSWD", "UNIQUE"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "users", "`ent_uk_passwd`", "UNIQUE KEY `ent_uk_passwd` (`passwd`,`lang`)")
					},
					want: []string{"ENT_UK_PASSWD"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "users", "`ent_uk_passwd`")
					},
					want:       []string{"ENT_UK_PASSWD"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "index_prefix",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "hosts", "KEY `ent_idx_desc` (`description`(32))")
					},
					want:       []string{"ENT_IDX_DESC", "(32)"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "hosts", "`ent_idx_desc`", "KEY `ent_idx_desc` (`description`(64))")
					},
					want: []string{"ENT_IDX_DESC", "(64)"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "hosts", "`ent_idx_desc`")
					},
					want:       []string{"ENT_IDX_DESC"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "index_composite",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "maintenances", "KEY `ent_idx_window` (`active_till`,`maintenance_type`)")
					},
					want:       []string{"ENT_IDX_WINDOW"},
					exactStmts: 1,
				},
				{
					name: "modify", // reorder the key parts — a semantic index change
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "maintenances", "`ent_idx_window`", "KEY `ent_idx_window` (`maintenance_type`,`active_till`)")
					},
					want: []string{"ENT_IDX_WINDOW"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "maintenances", "`ent_idx_window`")
					},
					want:       []string{"ENT_IDX_WINDOW"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "foreign_key",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "ent_ranked", "CONSTRAINT `ent_fk_owner` FOREIGN KEY (`owner_userid`) REFERENCES `users` (`userid`) ON DELETE SET NULL")
					},
					want: []string{"ENT_FK_OWNER", "FOREIGN KEY"},
				},
				{
					name: "modify", // referential-action change
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceInTable(t, source, "ent_ranked", "ON DELETE SET NULL", "ON DELETE CASCADE")
					},
					want: []string{"ENT_FK_OWNER"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "ent_ranked", "CONSTRAINT `ent_fk_owner`")
					},
					want: []string{"ENT_FK_OWNER"},
				},
			},
		},
		{
			name:   "check",
			skip57: "5.7 parses-and-ignores CHECK constraints (nothing syncs back)",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addIndexToTable(t, source, "ent_ranked", "CONSTRAINT `ent_chk_score` CHECK ((`score` >= 0))")
					},
					want:       []string{"ENT_CHK_SCORE", "CHECK"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entReplaceLineInTable(t, source, "ent_ranked", "`ent_chk_score`", "CONSTRAINT `ent_chk_score` CHECK ((`score` <= 1000000))")
					},
					want: []string{"ENT_CHK_SCORE"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropLineInTable(t, source, "ent_ranked", "`ent_chk_score`")
					},
					want:       []string{"ENT_CHK_SCORE"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "view",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE VIEW ent_crud_v AS SELECT roleid, name FROM role;\n"
					},
					want:       []string{"ENT_CRUD_V"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return addColumnToView(t, source, "ent_crud_v", "`role`.`type` AS `type`")
					},
					want:       []string{"ENT_CRUD_V"},
					exactStmts: 1,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return dropObjectBlock(t, source, "VIEW", "ent_crud_v")
					},
					want:       []string{"DROP VIEW", "ENT_CRUD_V"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "function",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE FUNCTION ent_crud_f() RETURNS INT DETERMINISTIC RETURN 41;\n"
					},
					want:       []string{"ENT_CRUD_F"},
					exactStmts: 1,
				},
				{
					name: "modify", // body change — this omni build renders it as DROP + CREATE
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return mustReplace(t, source, "RETURN 41", "RETURN 42")
					},
					want:       []string{"ENT_CRUD_F"},
					exactStmts: 2,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return dropObjectBlock(t, source, "FUNCTION", "ent_crud_f")
					},
					want:       []string{"DROP FUNCTION", "ENT_CRUD_F"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "procedure",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE PROCEDURE ent_crud_p(IN rid BIGINT UNSIGNED) BEGIN UPDATE role SET readonly = readonly WHERE roleid = rid; END;\n"
					},
					want:       []string{"ENT_CRUD_P"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return mustReplace(t, source, "SET readonly = readonly", "SET readonly = 0")
					},
					want:       []string{"ENT_CRUD_P"},
					exactStmts: 2,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return dropObjectBlock(t, source, "PROCEDURE", "ent_crud_p")
					},
					want:       []string{"DROP PROCEDURE", "ENT_CRUD_P"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "trigger",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE TRIGGER ent_crud_trg BEFORE UPDATE ON media_type FOR EACH ROW SET NEW.name = NEW.name;\n"
					},
					want:       []string{"ENT_CRUD_TRG"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return mustReplace(t, source, "SET NEW.name = NEW.name", "SET NEW.name = TRIM(NEW.name)")
					},
					want:       []string{"ENT_CRUD_TRG"},
					exactStmts: 2,
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return dropTrigger(t, source, "ent_crud_trg")
					},
					want:       []string{"DROP TRIGGER", "ENT_CRUD_TRG"},
					exactStmts: 1,
				},
			},
		},
		{
			name: "event",
			phases: []entCRUDPhase{
				{
					name: "create",
					mutate: func(_ *testing.T, _ liveServer, source string) string {
						return source + "\nCREATE EVENT ent_crud_ev ON SCHEDULE EVERY 12 HOUR DO DELETE FROM changelog WHERE clock < 100;\n"
					},
					want:       []string{"ENT_CRUD_EV"},
					exactStmts: 1,
				},
				{
					name: "modify",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return mustReplace(t, source, "EVERY 12 HOUR", "EVERY 6 HOUR")
					},
					want: []string{"ENT_CRUD_EV"},
				},
				{
					name: "drop",
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entDropObjectNamed(t, source, "EVENT", "ent_crud_ev")
					},
					want:       []string{"DROP EVENT", "ENT_CRUD_EV"},
					exactStmts: 1,
				},
			},
		},
		{
			// NOTE (observation, not a failure): for every partition change the differ
			// emits a full `ALTER TABLE ... PARTITION BY` REPARTITION statement rather
			// than the targeted ADD/DROP/REORGANIZE PARTITION — semantically correct and
			// converging, but a whole-table rebuild on large enterprise tables. See the
			// campaign findings.
			name: "partition",
			phases: []entCRUDPhase{
				{
					name: "create", // add a partition
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entSetPartitionClause(t, source, "ent_part_log",
							"PARTITION BY RANGE (`bucket`)\n(PARTITION p0 VALUES LESS THAN (100),\n PARTITION p1 VALUES LESS THAN (200),\n PARTITION p2 VALUES LESS THAN (300))")
					},
					want: []string{"PARTITION BY", "P2"},
				},
				{
					name: "modify", // reorganize: merge p0+p1 into p01
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entSetPartitionClause(t, source, "ent_part_log",
							"PARTITION BY RANGE (`bucket`)\n(PARTITION p01 VALUES LESS THAN (200),\n PARTITION p2 VALUES LESS THAN (300))")
					},
					want: []string{"PARTITION BY", "P01"},
				},
				{
					name: "drop", // drop the p2 partition (the repartition plan re-lists survivors)
					mutate: func(t *testing.T, _ liveServer, source string) string {
						return entSetPartitionClause(t, source, "ent_part_log",
							"PARTITION BY RANGE (`bucket`)\n(PARTITION p01 VALUES LESS THAN (200))")
					},
					want: []string{"PARTITION BY", "P01"},
				},
			},
		},
	}
}

//nolint:tparallel
func TestSDLEnterpriseSingleObjectCRUD(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, kind := range entCRUDKinds() {
				kind := kind
				t.Run(kind.name, func(t *testing.T) {
					if srv.version == "5.7" && kind.skip57 != "" {
						t.Skipf("[%s/%s] skipped on 5.7: %s", srv.name, kind.name, kind.skip57)
					}
					base := entZabbixSlice(t, entSliceCoreTables...) + entA1Aux
					dbName := entLoadDDL(ctx, t, srv, "entsdl_a1", base)

					for _, phase := range kind.phases {
						label := srv.name + "/A1/" + kind.name + "/" + phase.name
						source := dumpSDL(ctx, t, srv, dbName)
						target := phase.mutate(t, srv, source)
						plan := entOracle(ctx, t, srv, dbName, target, label)

						upper := strings.ToUpper(plan)
						for _, want := range phase.want {
							require.Contains(t, upper, want, "[%s] plan missing %q:\n%s", label, want, plan)
						}
						if phase.exactStmts > 0 {
							require.Equal(t, phase.exactStmts, entPlanStatementCount(t, plan),
								"[%s] plan not minimal (%d stmts expected):\n%s", label, phase.exactStmts, plan)
						}
					}
				})
			}
		})
	}
}

// ----------------------------------------------------------------------------
// A2: dependent objects — the enterprise differentiator. FK chains, trigger→table
// dependencies (Zabbix changelog pattern), view-on-view stacks, function-used-by-view,
// and a circular FK pair, each as one or more oracle rounds on the core slice.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseDependentObjects(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			t.Run("fk_chain_extend", func(t *testing.T) {
				// Extend the real item_tag -> items -> hosts -> proxy -> proxy_group chain
				// with two new tables in ONE release: ent_item_note -> items and
				// ent_note_tag -> ent_item_note. Creation must be topological.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := source + `
CREATE TABLE ent_item_note (
  noteid bigint unsigned NOT NULL,
  itemid bigint unsigned NOT NULL,
  note varchar(255) DEFAULT '' NOT NULL,
  PRIMARY KEY (noteid),
  KEY ent_item_note_1 (itemid),
  CONSTRAINT ent_c_item_note_1 FOREIGN KEY (itemid) REFERENCES items (itemid) ON DELETE CASCADE
) ENGINE=InnoDB;
CREATE TABLE ent_note_tag (
  notetagid bigint unsigned NOT NULL,
  noteid bigint unsigned NOT NULL,
  tag varchar(64) DEFAULT '' NOT NULL,
  PRIMARY KEY (notetagid),
  KEY ent_note_tag_1 (noteid),
  CONSTRAINT ent_c_note_tag_1 FOREIGN KEY (noteid) REFERENCES ent_item_note (noteid) ON DELETE CASCADE
) ENGINE=InnoDB;
`
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/fk_chain_extend")
				upper := strings.ToUpper(plan)
				noteIdx := strings.Index(upper, "CREATE TABLE `ENT_ITEM_NOTE`")
				tagIdx := strings.Index(upper, "CREATE TABLE `ENT_NOTE_TAG`")
				require.GreaterOrEqual(t, noteIdx, 0, "plan must create ent_item_note:\n%s", plan)
				require.GreaterOrEqual(t, tagIdx, 0, "plan must create ent_note_tag:\n%s", plan)
			})

			t.Run("fk_chain_drop_mid", func(t *testing.T) {
				// Drop the middle of the chain — items — together with its dependents
				// item_tag and item_preproc (each carrying 3 changelog triggers) in ONE
				// release. The plan must drop FKs/tables in dependency order and must not
				// drop a trigger after its table is already gone.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := source
				for _, trg := range []string{
					"items_insert", "items_update", "items_delete",
					"item_tag_insert", "item_tag_update", "item_tag_delete",
					"item_preproc_insert", "item_preproc_update", "item_preproc_delete",
				} {
					target = dropTrigger(t, target, trg)
				}
				for _, tbl := range []string{"item_tag", "item_preproc", "items"} {
					target = entDropTableBlock(t, target, tbl)
				}
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/fk_chain_drop_mid")
				upper := strings.ToUpper(plan)
				for _, want := range []string{"`ITEMS`", "`ITEM_TAG`", "`ITEM_PREPROC`"} {
					require.Contains(t, upper, want, "plan must drop %s:\n%s", want, plan)
				}
			})

			t.Run("trigger_table_widen", func(t *testing.T) {
				// The Zabbix changelog pattern: hosts carries 5 triggers, two of which
				// (hosts_name_upper_*) read/write hosts.name and hosts.name_upper. Widen
				// BOTH columns in one release; the triggers must survive the ALTERs.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := entReplaceInTable(t, source, "hosts", "`name` varchar(128)", "`name` varchar(190)")
				target = entReplaceInTable(t, target, "hosts", "`name_upper` varchar(128)", "`name_upper` varchar(190)")
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/trigger_table_widen")
				upper := strings.ToUpper(plan)
				require.Contains(t, upper, "VARCHAR(190)", "plan must widen the columns:\n%s", plan)
				require.NotContains(t, upper, "DROP TABLE", "widening must not recreate hosts:\n%s", plan)
			})

			t.Run("trigger_modify_one_of_five", func(t *testing.T) {
				// Redefine ONE of hosts' five triggers (hosts_update) and prove the other
				// four are untouched by the plan.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := mustReplace(t, source, "values (1,old.hostid,2,", "values (1,new.hostid,2,")
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/trigger_modify_one_of_five")
				upper := strings.ToUpper(plan)
				require.Contains(t, upper, "HOSTS_UPDATE", "plan must redefine hosts_update:\n%s", plan)
				for _, untouched := range []string{"HOSTS_INSERT", "HOSTS_DELETE", "HOSTS_NAME_UPPER_INSERT", "HOSTS_NAME_UPPER_UPDATE"} {
					require.NotContains(t, upper, untouched, "plan must not touch %s:\n%s", untouched, plan)
				}
				require.Equal(t, 2, entPlanStatementCount(t, plan), "trigger redefine should be DROP+CREATE:\n%s", plan)
			})

			t.Run("view_stack", func(t *testing.T) {
				// 3-deep view stack v3 -> v2 -> v1 -> users: create the whole stack in one
				// release, modify the MIDDLE view, then drop the whole stack.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))

				source := dumpSDL(ctx, t, srv, dbName)
				target := source + `
CREATE VIEW ent_vs1 AS SELECT userid, username, surname FROM users;
CREATE VIEW ent_vs2 AS SELECT userid, username FROM ent_vs1;
CREATE VIEW ent_vs3 AS SELECT userid FROM ent_vs2;
`
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/view_stack/create")
				upper := strings.ToUpper(plan)
				v1 := strings.Index(upper, "`ENT_VS1`")
				v2 := strings.Index(upper, "`ENT_VS2`")
				v3 := strings.Index(upper, "`ENT_VS3`")
				require.True(t, v1 >= 0 && v2 >= 0 && v3 >= 0, "plan must create all three views:\n%s", plan)
				require.True(t, v1 < v2 && v2 < v3, "views must be created base-first (v1<v2<v3):\n%s", plan)

				source = dumpSDL(ctx, t, srv, dbName)
				target = addColumnToView(t, source, "ent_vs2", "`ent_vs1`.`surname` AS `surname`")
				plan = entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/view_stack/modify_middle")
				upper = strings.ToUpper(plan)
				require.Contains(t, upper, "VIEW `ENT_VS2`")
				require.NotContains(t, upper, "ENT_VS3", "modifying v2 must not churn v3:\n%s", plan)
				// v2's body legitimately references ent_vs1 in FROM; minimality is proven
				// by the single-statement plan (v1 and v3 are untouched).
				require.Equal(t, 1, entPlanStatementCount(t, plan), "modifying v2 must be a single statement:\n%s", plan)

				source = dumpSDL(ctx, t, srv, dbName)
				target = dropObjectBlock(t, source, "VIEW", "ent_vs3")
				target = dropObjectBlock(t, target, "VIEW", "ent_vs2")
				target = dropObjectBlock(t, target, "VIEW", "ent_vs1")
				plan = entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/view_stack/drop")
				require.Equal(t, 3, entPlanStatementCount(t, plan), "stack drop should be exactly 3 DROP VIEW:\n%s", plan)
			})

			t.Run("function_used_by_view", func(t *testing.T) {
				// Create a function and a view CALLING it in one release (MySQL rejects a
				// view over a missing function, so creation order is enforced by apply);
				// then drop both in one release.
				//
				// Was PINNED-BUG (view created before the function it calls → Error 1305):
				// fixed by omni #361 (routine creates precede view creates; drops inverse).
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))

				source := dumpSDL(ctx, t, srv, dbName)
				target := source + `
CREATE FUNCTION ent_ucount() RETURNS INT READS SQL DATA RETURN (SELECT COUNT(*) FROM users);
CREATE VIEW ent_v_fn AS SELECT ent_ucount() AS n;
`
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/function_used_by_view/create")
				upper := strings.ToUpper(plan)
				require.Contains(t, upper, "ENT_UCOUNT")
				require.Contains(t, upper, "ENT_V_FN")

				source = dumpSDL(ctx, t, srv, dbName)
				target = dropObjectBlock(t, source, "VIEW", "ent_v_fn")
				target = dropObjectBlock(t, target, "FUNCTION", "ent_ucount")
				plan = entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/function_used_by_view/drop")
				require.Equal(t, 2, entPlanStatementCount(t, plan), "should be exactly DROP VIEW + DROP FUNCTION:\n%s", plan)
			})

			t.Run("circular_fk_pair", func(t *testing.T) {
				// Create two mutually-referencing tables in ONE release. The generated DDL
				// must break the cycle itself (deferred ALTER or FK-checks toggling) — the
				// apply connection runs with default settings.
				dbName := entLoadDDL(ctx, t, srv, "entsdl_a2", entZabbixSlice(t, entSliceCoreTables...))
				source := dumpSDL(ctx, t, srv, dbName)
				target := source + `
CREATE TABLE ent_circ_a (
  aid bigint unsigned NOT NULL,
  bid bigint unsigned NULL,
  PRIMARY KEY (aid),
  KEY ent_circ_a_1 (bid),
  CONSTRAINT ent_c_circ_a FOREIGN KEY (bid) REFERENCES ent_circ_b (bid) ON DELETE SET NULL
) ENGINE=InnoDB;
CREATE TABLE ent_circ_b (
  bid bigint unsigned NOT NULL,
  aid bigint unsigned NULL,
  PRIMARY KEY (bid),
  KEY ent_circ_b_1 (aid),
  CONSTRAINT ent_c_circ_b FOREIGN KEY (aid) REFERENCES ent_circ_a (aid) ON DELETE SET NULL
) ENGINE=InnoDB;
`
				plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A2/circular_fk_pair")
				upper := strings.ToUpper(plan)
				require.Contains(t, upper, "ENT_CIRC_A")
				require.Contains(t, upper, "ENT_CIRC_B")
			})
		})
	}
}

// ----------------------------------------------------------------------------
// A3: special types — AUTO_INCREMENT, generated columns, ENUM/SET members,
// charset/collation, temporal precision, DECIMAL, TEXT/BLOB families, signedness,
// prefix-indexed TEXT, and partition operations. Small synthetic bases, one oracle
// round each.
// ----------------------------------------------------------------------------

type entTypeCase struct {
	name   string
	base   string
	skip57 string
	// pinned, when non-empty, skips the case with a PINNED-BUG classification (a
	// confirmed engine bug being fixed upstream; the case re-arms when the skip drops).
	pinned string
	mutate func(t *testing.T, srv liveServer, source string) string
	want   []string
	// exactStmts, when > 0, asserts the plan statement count.
	exactStmts int
}

func entTypeCases() []entTypeCase {
	return []entTypeCase{
		{
			// REPRO: add `seq bigint unsigned NOT NULL AUTO_INCREMENT` + UNIQUE KEY uk_seq
			// (seq) in one release. The plan emits TWO statements — ADD COLUMN, then ADD
			// UNIQUE KEY — and MySQL rejects the first with Error 1075 (an auto column
			// must be defined as a key). The clauses must be grouped into one ALTER.
			name: "ai_add_column",
			base: "CREATE TABLE t3 (id int NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			// Was PINNED-BUG (AI column and its key split into separate ALTERs →
			// Error 1075): fixed by omni #364 (mergeAutoIncrementKeyOps grouping).
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := addColumnToTable(t, source, "t3", "`seq` bigint unsigned NOT NULL AUTO_INCREMENT")
				return addIndexToTable(t, s, "t3", "UNIQUE KEY `uk_seq` (`seq`)")
			},
			want: []string{"SEQ", "AUTO_INCREMENT"},
		},
		{
			name: "ai_drop_attribute",
			base: "CREATE TABLE t3 (id int NOT NULL AUTO_INCREMENT, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3",
					"`id` int NOT NULL AUTO_INCREMENT", "`id` int NOT NULL")
			},
			want:       []string{"`ID`"},
			exactStmts: 1,
		},
		{
			name: "ai_composite_pk_member",
			base: "CREATE TABLE t3 (a int NOT NULL, b int NOT NULL AUTO_INCREMENT, PRIMARY KEY (a,b), KEY idx_b (b)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3",
					"`b` int NOT NULL AUTO_INCREMENT", "`b` bigint NOT NULL AUTO_INCREMENT")
			},
			want:       []string{"BIGINT", "AUTO_INCREMENT"},
			exactStmts: 1,
		},
		{
			name: "generated_stored_add",
			base: "CREATE TABLE t3 (id int NOT NULL, price decimal(8,2) NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return addColumnToTable(t, source, "t3", "`total` decimal(10,2) GENERATED ALWAYS AS ((`price` * 1.1)) STORED")
			},
			want:       []string{"TOTAL", "STORED"},
			exactStmts: 1,
		},
		{
			name: "generated_virtual_add",
			base: "CREATE TABLE t3 (id int NOT NULL, price decimal(8,2) NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return addColumnToTable(t, source, "t3", "`total_v` decimal(10,2) GENERATED ALWAYS AS ((`price` * 2)) VIRTUAL")
			},
			want:       []string{"TOTAL_V"},
			exactStmts: 1,
		},
		{
			name: "generated_dependent_alter",
			// Widen the BASE column and change the generated column depending on it in the
			// SAME release — the plan must sequence the two ALTERs so MySQL accepts them.
			base: "CREATE TABLE t3 (id int NOT NULL, price decimal(8,2) NOT NULL, total decimal(10,2) GENERATED ALWAYS AS (price * 2) STORED, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "`price` decimal(8,2)", "`price` decimal(10,2)")
				return entReplaceInTable(t, s, "t3", "* 2", "* 3")
			},
			want: []string{"PRICE"},
		},
		{
			name: "enum_member_add",
			base: "CREATE TABLE t3 (id int NOT NULL, kind enum('a','b') NOT NULL DEFAULT 'a', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "enum('a','b')", "enum('a','b','c')")
			},
			want:       []string{"ENUM"},
			exactStmts: 1,
		},
		{
			name: "enum_member_remove",
			base: "CREATE TABLE t3 (id int NOT NULL, kind enum('a','b','c') NOT NULL DEFAULT 'a', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "enum('a','b','c')", "enum('a','b')")
			},
			want:       []string{"ENUM"},
			exactStmts: 1,
		},
		{
			name: "enum_member_reorder",
			base: "CREATE TABLE t3 (id int NOT NULL, kind enum('a','b','c') NOT NULL DEFAULT 'a', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "enum('a','b','c')", "enum('c','a','b')")
			},
			want:       []string{"ENUM"},
			exactStmts: 1,
		},
		{
			name: "set_member_add_remove",
			base: "CREATE TABLE t3 (id int NOT NULL, tags set('x','y') NOT NULL DEFAULT '', flags set('p','q','r') NOT NULL DEFAULT '', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "set('x','y')", "set('x','y','z')")
				return entReplaceInTable(t, s, "t3", "set('p','q','r')", "set('p','q')")
			},
			want: []string{"SET("},
		},
		{
			// utf8mb4_unicode_ci is non-default on BOTH versions (8.0 default 0900_ai_ci,
			// 5.7 default general_ci), so the dump renders the explicit column COLLATE on
			// both — a stable anchor. When the collation matches the table default the
			// dump omits CHARACTER SET/COLLATE entirely.
			name: "charset_column_collation",
			base: "CREATE TABLE t3 (id int NOT NULL, label varchar(80) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3",
					"`label` varchar(80) COLLATE utf8mb4_unicode_ci",
					"`label` varchar(80) COLLATE utf8mb4_bin")
			},
			want:       []string{"UTF8MB4_BIN"},
			exactStmts: 1,
		},
		{
			name: "charset_table_level",
			base: "CREATE TABLE t3 (id int NOT NULL, title varchar(60) NOT NULL, body varchar(200) DEFAULT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=latin1;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				// Replace the whole table-options line so the (version-dependent) COLLATE
				// suffix can never survive as an invalid latin1/utf8mb4 hybrid. The options
				// line is the segment's last line, so everything from ') ENGINE=InnoDB' to
				// the segment end is rewritten.
				start, end := entTableSegment(t, source, "t3")
				seg := source[start:end]
				optIdx := strings.Index(seg, ") ENGINE=InnoDB")
				require.GreaterOrEqual(t, optIdx, 0, "t3 options line not found:\n%s", seg)
				tail := ""
				if nl := strings.Index(seg[optIdx:], "\n"); nl >= 0 {
					tail = seg[optIdx+nl:]
				}
				newSeg := seg[:optIdx] + ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;" + tail
				return source[:start] + newSeg + source[end:]
			},
			want: []string{"UTF8MB4"},
		},
		{
			name: "temporal_precision",
			base: "CREATE TABLE t3 (id int NOT NULL, ts timestamp(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3), dt datetime NOT NULL, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceAllInTable(t, source, "t3", "timestamp(3)", "timestamp(6)")
				s = entReplaceAllInTable(t, s, "t3", "CURRENT_TIMESTAMP(3)", "CURRENT_TIMESTAMP(6)")
				return entReplaceInTable(t, s, "t3", "`dt` datetime ", "`dt` datetime(3) ")
			},
			want: []string{"TIMESTAMP(6)", "DATETIME(3)"},
		},
		{
			name: "decimal_widen",
			base: "CREATE TABLE t3 (id int NOT NULL, amount decimal(10,2) NOT NULL DEFAULT '0.00', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "decimal(10,2)", "decimal(12,4)")
			},
			want:       []string{"DECIMAL(12,4)"},
			exactStmts: 1,
		},
		{
			name: "decimal_narrow",
			base: "CREATE TABLE t3 (id int NOT NULL, amount decimal(12,4) NOT NULL DEFAULT '0.0000', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "decimal(12,4)", "decimal(10,2)")
				return entReplaceInTable(t, s, "t3", "'0.0000'", "'0.00'")
			},
			want:       []string{"DECIMAL(10,2)"},
			exactStmts: 1,
		},
		{
			name: "text_blob_family",
			base: "CREATE TABLE t3 (id int NOT NULL, body text, payload blob, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "`body` text", "`body` mediumtext")
				return entReplaceInTable(t, s, "t3", "`payload` blob", "`payload` longblob")
			},
			want: []string{"MEDIUMTEXT", "LONGBLOB"},
		},
		{
			name: "unsigned_signed_flip",
			base: "CREATE TABLE t3 (id int NOT NULL, cnt int unsigned NOT NULL DEFAULT '0', pos int NOT NULL DEFAULT '0', PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, srv liveServer, source string) string {
				s := entReplaceInTable(t, source, "t3", "`cnt` "+entUintType(srv), "`cnt` int")
				return entReplaceInTable(t, s, "t3", "`pos` int", "`pos` "+entUintType(srv))
			},
			want: []string{"`CNT`", "`POS`"},
		},
		{
			name: "prefix_indexed_text_type_change",
			base: "CREATE TABLE t3 (id int NOT NULL, body text NOT NULL, PRIMARY KEY (id), KEY idx_body (body(24))) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entReplaceInTable(t, source, "t3", "`body` text", "`body` mediumtext")
			},
			want: []string{"MEDIUMTEXT"},
		},
		{
			name: "partition_add",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 PARTITION BY RANGE (bucket) (PARTITION p0 VALUES LESS THAN (100), PARTITION p1 VALUES LESS THAN (200));",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entSetPartitionClause(t, source, "t3",
					"PARTITION BY RANGE (`bucket`)\n(PARTITION p0 VALUES LESS THAN (100),\n PARTITION p1 VALUES LESS THAN (200),\n PARTITION p2 VALUES LESS THAN (300))")
			},
			want: []string{"P2"},
		},
		{
			name: "partition_drop",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 PARTITION BY RANGE (bucket) (PARTITION p0 VALUES LESS THAN (100), PARTITION p1 VALUES LESS THAN (200), PARTITION p2 VALUES LESS THAN (300));",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entSetPartitionClause(t, source, "t3",
					"PARTITION BY RANGE (`bucket`)\n(PARTITION p0 VALUES LESS THAN (100),\n PARTITION p1 VALUES LESS THAN (200))")
			},
			// The differ emits a full repartition listing only the survivors (see the
			// partition-minimality observation in the campaign findings).
			want: []string{"PARTITION BY", "P0"},
		},
		{
			name: "partition_reorganize",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 PARTITION BY RANGE (bucket) (PARTITION p0 VALUES LESS THAN (100), PARTITION p1 VALUES LESS THAN (200), PARTITION pmax VALUES LESS THAN MAXVALUE);",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entSetPartitionClause(t, source, "t3",
					"PARTITION BY RANGE (`bucket`)\n(PARTITION p01 VALUES LESS THAN (200),\n PARTITION pmax VALUES LESS THAN MAXVALUE)")
			},
			want: []string{"P01"},
		},
		{
			name: "partition_existing_table",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entAppendTableClause(t, source, "t3", "PARTITION BY HASH (`bucket`)\nPARTITIONS 4")
			},
			want: []string{"PARTITION BY", "HASH"},
		},
		{
			name: "departition",
			base: "CREATE TABLE t3 (id int NOT NULL, bucket int NOT NULL, PRIMARY KEY (id,bucket)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 PARTITION BY HASH (bucket) PARTITIONS 4;",
			mutate: func(t *testing.T, _ liveServer, source string) string {
				return entSetPartitionClause(t, source, "t3", "")
			},
			want: []string{"T3"},
		},
	}
}

//nolint:tparallel
func TestSDLEnterpriseSpecialTypes(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, tc := range entTypeCases() {
				tc := tc
				t.Run(tc.name, func(t *testing.T) {
					if srv.version == "5.7" && tc.skip57 != "" {
						t.Skipf("[%s/%s] skipped on 5.7: %s", srv.name, tc.name, tc.skip57)
					}
					if tc.pinned != "" {
						t.Skip(tc.pinned)
					}
					dbName := entLoadDDL(ctx, t, srv, "entsdl_a3", tc.base)
					source := dumpSDL(ctx, t, srv, dbName)
					target := tc.mutate(t, srv, source)
					plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A3/"+tc.name)

					upper := strings.ToUpper(plan)
					for _, want := range tc.want {
						require.Contains(t, upper, want, "[%s/A3/%s] plan missing %q:\n%s", srv.name, tc.name, want, plan)
					}
					if tc.exactStmts > 0 {
						require.Equal(t, tc.exactStmts, entPlanStatementCount(t, plan),
							"[%s/A3/%s] plan not minimal:\n%s", srv.name, tc.name, plan)
					}
				})
			}
		})
	}
}

// ----------------------------------------------------------------------------
// A5(a): sequential chain S0 -> S1 -> S2 -> S3 -> S4 on the core slice. Every step must
// converge, and the endpoint must be REACHABLE FROM S0 IN ONE SHOT on a second database
// — both databases must dump to the same canonical schema.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseSequentialChain(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			base := entZabbixSlice(t, entSliceCoreTables...)
			chainDB := entLoadDDL(ctx, t, srv, "entsdl_a5chain", base)

			// The chain targets are built by cumulative TEXTUAL mutation of the canonical
			// S0 dump, so the final text is identical for the chained and one-shot legs.
			s0 := dumpSDL(ctx, t, srv, chainDB)

			// The appended blocks use backticked names so the later chain steps can locate
			// and mutate them with the same segment helpers that work on canonical dumps.
			s1 := addColumnToTable(t, s0, "users", "`ent_ch_flag` int NOT NULL DEFAULT '0'")
			s1 += `
CREATE TABLE ` + "`ent_ch_audit`" + ` (
  auditid bigint unsigned NOT NULL,
  userid bigint unsigned NULL,
  note varchar(60) DEFAULT '' NOT NULL,
  PRIMARY KEY (auditid)
) ENGINE=InnoDB;
`
			s2 := addIndexToTable(t, s1, "users", "KEY `ent_ch_idx` (`ent_ch_flag`)")
			s2 += `
CREATE VIEW ` + "`ent_ch_v`" + ` AS SELECT auditid, note FROM ent_ch_audit;
`
			s3 := addIndexToTable(t, s2, "ent_ch_audit", "KEY `ent_ch_fk` (`userid`)")
			s3 = addIndexToTable(t, s3, "ent_ch_audit", "CONSTRAINT `ent_ch_fk` FOREIGN KEY (`userid`) REFERENCES `users` (`userid`) ON DELETE SET NULL")
			s3 = entReplaceInTable(t, s3, "maintenances", "`name` varchar(128)", "`name` varchar(190)")
			s3 += `
CREATE TRIGGER ent_ch_trg BEFORE INSERT ON ent_ch_audit FOR EACH ROW SET NEW.note = TRIM(NEW.note);
`
			s4 := entDropLineInTable(t, s3, "users", "`ent_ch_idx`")
			s4 = addColumnToView(t, s4, "ent_ch_v", "`ent_ch_audit`.`userid` AS `userid`")
			s4 += `
CREATE PROCEDURE ent_ch_p(IN aid BIGINT UNSIGNED) BEGIN DELETE FROM ent_ch_audit WHERE auditid = aid; END;
`
			for i, step := range []string{s1, s2, s3, s4} {
				entOracle(ctx, t, srv, chainDB, step, srv.name+"/A5/chain/S"+string(rune('1'+i)))
			}

			// One-shot leg: a second fresh database goes S0 -> S4 in a single release.
			oneshotDB := entLoadDDL(ctx, t, srv, "entsdl_a5one", base)
			entOracle(ctx, t, srv, oneshotDB, s4, srv.name+"/A5/chain/oneshot")

			// Same endpoint: the two databases' canonical dumps must be equivalent.
			chainDump := dumpSDL(ctx, t, srv, chainDB)
			oneshotDump := dumpSDL(ctx, t, srv, oneshotDB)
			diffAB, err := mysqlDiffSDLMigration(chainDump, oneshotDump, srv.version)
			require.NoError(t, err)
			require.Empty(t, diffAB, "chained and one-shot endpoints diverge (chain->oneshot):\n%s", diffAB)
			diffBA, err := mysqlDiffSDLMigration(oneshotDump, chainDump, srv.version)
			require.NoError(t, err)
			require.Empty(t, diffBA, "chained and one-shot endpoints diverge (oneshot->chain):\n%s", diffBA)
		})
	}
}

// ----------------------------------------------------------------------------
// A5(b): apply-back — migrate B -> T (destructive + additive mix), then T -> B using the
// ORIGINAL canonical dump as the target; the database must land exactly back on B.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseApplyBack(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			dbName := entLoadDDL(ctx, t, srv, "entsdl_a5back", entZabbixSlice(t, entSliceCoreTables...))
			c0 := dumpSDL(ctx, t, srv, dbName)

			// Forward: drop a plain index, drop an unindexed column, widen a column, and
			// add a table — a realistic mixed release.
			target := entDropLineInTable(t, c0, "users", "`users_2`")
			target = entDropLineInTable(t, target, "users", "`attempt_ip`")
			target = entReplaceInTable(t, target, "media_type", "`smtp_helo` varchar(255)", "`smtp_helo` varchar(300)")
			target += `
CREATE TABLE ent_ab_t (
  id bigint unsigned NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;
`
			entOracle(ctx, t, srv, dbName, target, srv.name+"/A5/apply_back/forward")

			// Backward: the original canonical dump is the target again.
			entOracle(ctx, t, srv, dbName, c0, srv.name+"/A5/apply_back/backward")

			// The database must be exactly B again (canonical-dump equivalence).
			final := dumpSDL(ctx, t, srv, dbName)
			backDiff, err := mysqlDiffSDLMigration(final, c0, srv.version)
			require.NoError(t, err)
			require.Empty(t, backDiff, "apply-back did not restore B:\n%s", backDiff)
		})
	}
}

// ----------------------------------------------------------------------------
// A5(c): combined release — table + column + index + FK + view + trigger + routine all
// changed in ONE diff.
// ----------------------------------------------------------------------------

// entA5cAux seeds the view/function the combined release modifies.
const entA5cAux = `
CREATE VIEW ent_cmb_v AS SELECT roleid, name FROM role;
CREATE FUNCTION ent_cmb_f() RETURNS INT DETERMINISTIC RETURN 7;
`

//nolint:tparallel
func TestSDLEnterpriseCombinedRelease(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			base := entZabbixSlice(t, entSliceCoreTables...) + entA5cAux
			dbName := entLoadDDL(ctx, t, srv, "entsdl_a5cmb", base)
			source := dumpSDL(ctx, t, srv, dbName)

			// One release, seven object kinds:
			target := addColumnToTable(t, source, "media_type", "`ent_cmb_col` varchar(32) DEFAULT NULL") // column
			target = addIndexToTable(t, target, "users", "KEY `ent_cmb_idx` (`theme`)")                   // index
			target = addColumnToTable(t, target, "dchecks", "`ent_cmb_ref` bigint unsigned NULL")         // FK column +
			target = addIndexToTable(t, target, "dchecks", "KEY `ent_cmb_fk` (`ent_cmb_ref`)")
			target = addIndexToTable(t, target, "dchecks", "CONSTRAINT `ent_cmb_fk` FOREIGN KEY (`ent_cmb_ref`) REFERENCES `connector` (`connectorid`) ON DELETE SET NULL") // FK
			target = addColumnToView(t, target, "ent_cmb_v", "`role`.`type` AS `type`")                                                                                     // view
			target = mustReplace(t, target, "RETURN 7", "RETURN 8")                                                                                                         // routine
			target += `
CREATE TABLE ent_cmb_t (
  id bigint unsigned NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB;
CREATE TRIGGER ent_cmb_trg BEFORE UPDATE ON maintenances FOR EACH ROW SET NEW.name = TRIM(NEW.name);
`
			plan := entOracle(ctx, t, srv, dbName, target, srv.name+"/A5/combined_release")
			upper := strings.ToUpper(plan)
			for _, want := range []string{
				"ENT_CMB_COL", "ENT_CMB_IDX", "ENT_CMB_REF", "ENT_CMB_FK",
				"ENT_CMB_V", "ENT_CMB_F", "ENT_CMB_T", "ENT_CMB_TRG",
			} {
				require.Contains(t, upper, want, "combined release plan missing %q:\n%s", want, plan)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// A5(e): scale timing guard on the largest corpus (OpenEMR, 283 tables) — the no-op
// diff must stay under 30s, and a single-column change must be exactly one ALTER,
// also computed under 30s.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseScaleGuard(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	const budget = 30 * time.Second

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			dbName := entLoadDDL(ctx, t, srv, "entsdl_scale", entOpenemrSQL)
			source := dumpSDL(ctx, t, srv, dbName)

			startNoop := time.Now()
			selfDiff, err := mysqlDiffSDLMigration(source, source, srv.version)
			noopDur := time.Since(startNoop)
			require.NoError(t, err)
			require.Empty(t, selfDiff, "openemr no-op diff must be empty:\n%s", selfDiff)
			t.Logf("[SCALE %s] openemr no-op diff: %v (budget %v)", srv.name, noopDur, budget)
			require.Less(t, noopDur, budget, "openemr no-op diff exceeded the %v budget", budget)

			// Single-column change: exactly one ALTER, computed within budget.
			target := addColumnToTable(t, source, "patient_data", "`ent_scale_col` varchar(40) DEFAULT NULL")
			startOne := time.Now()
			plan, err := mysqlDiffSDLMigration(source, target, srv.version)
			oneDur := time.Since(startOne)
			require.NoError(t, err)
			t.Logf("[SCALE %s] openemr single-column diff: %v, plan:\n%s", srv.name, oneDur, plan)
			require.Less(t, oneDur, budget, "openemr single-column diff exceeded the %v budget", budget)
			require.Equal(t, 1, entPlanStatementCount(t, plan), "single-column change must be exactly 1 statement:\n%s", plan)
			upper := strings.ToUpper(plan)
			require.Contains(t, upper, "ALTER TABLE")
			require.Contains(t, upper, "ENT_SCALE_COL")

			// Close the loop: apply + converge (the full oracle contract at scale).
			require.NoError(t, applyDDL(ctx, t, srv, dbName, plan), "scale plan failed to apply:\n%s", plan)
			after := dumpSDL(ctx, t, srv, dbName)
			converge, err := mysqlDiffSDLMigration(after, target, srv.version)
			require.NoError(t, err)
			require.Empty(t, converge, "openemr single-column change did not converge:\n%s", converge)
		})
	}
}

// ----------------------------------------------------------------------------
// A5(f): multi-file export round-trip at enterprise scale — for every corpus, the
// concatenated GetMultiFileDatabaseDefinition output must describe the identical schema
// as the single-file dump (diff empty both ways) and must be self-idempotent.
// ----------------------------------------------------------------------------

//nolint:tparallel
func TestSDLEnterpriseMultiFileRoundTrip(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, corpus := range entCorpora() {
				corpus := corpus
				t.Run(corpus.name, func(t *testing.T) {
					dbName := entLoadCorpus(ctx, t, srv, "entsdl_mf_", corpus)
					meta := syncMetaForDB(ctx, t, srv, dbName)

					result, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{
						SkipBackupSchema: true,
					}, meta.GetProto())
					require.NoError(t, err)
					require.NotEmpty(t, result.Files, "[%s/%s] multi-file export produced no files", srv.name, corpus.name)

					single := dumpSDL(ctx, t, srv, dbName)
					concat := concatMultiFile(result)
					t.Logf("[MULTIFILE %s/%s] %d files, concat %d bytes, single %d bytes",
						srv.name, corpus.name, len(result.Files), len(concat), len(single))

					forward, err := mysqlDiffSDLMigration(concat, single, srv.version)
					require.NoError(t, err)
					require.Empty(t, forward, "[%s/%s] concat(multi-file) != single dump (concat->single):\n%s",
						srv.name, corpus.name, forward)
					backward, err := mysqlDiffSDLMigration(single, concat, srv.version)
					require.NoError(t, err)
					require.Empty(t, backward, "[%s/%s] concat(multi-file) != single dump (single->concat):\n%s",
						srv.name, corpus.name, backward)
					self, err := mysqlDiffSDLMigration(concat, concat, srv.version)
					require.NoError(t, err)
					require.Empty(t, self, "[%s/%s] concat(multi-file) not idempotent:\n%s", srv.name, corpus.name, self)
				})
			}
		})
	}
}
