package mysql

// Live round-trip proof for the MySQL multi-file SDL export (GetMultiFileDatabaseDefinition).
//
// The correctness bar is the declarative round-trip: the exported multi-file set must be
// re-importable via the declarative path, which CONCATENATES the files
// (action/command/file.go: lexical sort of paths, joined with "\n"). So the proof loads a
// real, rich schema (sakila: 16 tables, plain views, 3 functions, 3 procedures, 3 triggers)
// into live MySQL, syncs it, generates the multi-file set, then asserts:
//
//	concat(multi-file) ≡ single-file dump   (mysqlDiffSDLMigration(concat, single) empty)
//	concat(multi-file) is idempotent         (mysqlDiffSDLMigration(concat, concat) empty)
//
// proving the multi-file export is a faithful, re-importable representation identical to the
// single-file dump. It also asserts the expected per-object files exist (correct dirs, valid
// non-empty SDL) and that the API ZIP construction (mirroring database_service.getMultiFileSDL)
// yields one entry per file.
//
// Shared helpers (liveServers, loadRealWorld, dumpSDL, statementCount, realWorldSchemas,
// objectCounts) come from the sibling _test.go files in this package.

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"io"
	"slices"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

// concatMultiFile reproduces the declarative rollout concatenation from
// action/command/file.go: sort files by path lexically, then join contents with a single
// "\n" separator between files. This is the exact byte stream the declarative re-import
// path feeds back to the SDL loader.
func concatMultiFile(result *schema.MultiFileSchemaResult) string {
	files := make([]schema.File, len(result.Files))
	copy(files, result.Files)
	slices.SortFunc(files, func(a, b schema.File) int { return cmp.Compare(a.Name, b.Name) })
	var b strings.Builder
	for i, f := range files {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(f.Content)
	}
	return b.String()
}

// TestMultiFileSDLExportRoundTrip is the headline proof across the real-world corpus and
// both live versions. sakila is the priority (rich views/functions/procedures/triggers) on
// 8.0; roundcube / employees / employees_part / mediawiki carry the 5.7 leg (sakila's
// actor_info INVOKER view does not CREATE on 5.7). For each loadable fixture × version:
// generate the multi-file export, assert one valid non-empty file per synced object in the
// correct flat dir, then prove the declarative round-trip — concat(files) ≡ single-file
// dump AND concat(files) is idempotent.
//
//nolint:tparallel
func TestMultiFileSDLExportRoundTrip(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	for _, srv := range liveServers {
		srv := srv
		t.Run(srv.name, func(t *testing.T) {
			for _, rw := range realWorldSchemas() {
				rw := rw
				t.Run(rw.name, func(t *testing.T) {
					if srv.version == "5.7" && rw.only57Skip != "" {
						t.Skipf("[%s/%s] skipped on 5.7: %s", srv.name, rw.name, rw.only57Skip)
					}
					if rw.viewParsePending != "" {
						t.Skipf("[%s/%s] SDL re-import pending: %s", srv.name, rw.name, rw.viewParsePending)
					}

					meta, dbName := loadRealWorld(ctx, t, srv, rw)
					tbl, vw, fn, pr, tg := objectCounts(meta)
					t.Logf("[%s/%s] loaded: tables=%d views=%d functions=%d procedures=%d triggers=%d (db=%s)",
						srv.name, rw.name, tbl, vw, fn, pr, tg, dbName)

					metaProto := meta.GetProto()
					require.NotNil(t, metaProto, "synced metadata proto must be non-nil")

					// --- Generate the multi-file export. ---
					result, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{
						SkipBackupSchema: true,
					}, metaProto)
					require.NoError(t, err)
					require.NotEmpty(t, result.Files, "[%s/%s] multi-file export produced no files", srv.name, rw.name)

					// --- Assert the expected per-object files: correct dirs, one per object,
					// valid non-empty SDL, unique names. ---
					byDir := map[string][]schema.File{}
					seen := map[string]bool{}
					for _, f := range result.Files {
						require.False(t, seen[f.Name], "[%s/%s] duplicate file name %q", srv.name, rw.name, f.Name)
						seen[f.Name] = true
						require.True(t, strings.HasSuffix(f.Name, ".sql"), "[%s/%s] file %q must end .sql", srv.name, rw.name, f.Name)
						require.NotEmpty(t, strings.TrimSpace(f.Content), "[%s/%s] file %q has empty content", srv.name, rw.name, f.Name)
						dir, _, ok := strings.Cut(f.Name, "/")
						require.True(t, ok, "[%s/%s] file %q must be under a directory", srv.name, rw.name, f.Name)
						byDir[dir] = append(byDir[dir], f)

						// Spot-check the CREATE keyword matches the directory. A routine/trigger/event
						// file may be preceded by the BYT-9832 concat-safe session-context SET framing
						// (SET @saved_sql_mode = …; SET sql_mode = '…'; …), so the file need only CONTAIN
						// the CREATE, not lead with it. Tables never carry session context.
						upper := strings.ToUpper(strings.TrimSpace(f.Content))
						switch dir {
						case "tables":
							require.True(t, strings.HasPrefix(upper, "CREATE TABLE "), "[%s/%s] %q: %.40s", srv.name, rw.name, f.Name, upper)
						case "views":
							require.Contains(t, upper, " VIEW `", "[%s/%s] %q must be a CREATE VIEW", srv.name, rw.name, f.Name)
						case "functions":
							require.Contains(t, upper, "FUNCTION", "[%s/%s] %q must be a CREATE FUNCTION", srv.name, rw.name, f.Name)
						case "procedures":
							require.Contains(t, upper, "PROCEDURE", "[%s/%s] %q must be a CREATE PROCEDURE", srv.name, rw.name, f.Name)
						case "triggers":
							require.Contains(t, upper, "CREATE TRIGGER ", "[%s/%s] %q must be a CREATE TRIGGER", srv.name, rw.name, f.Name)
						case "events":
							require.Contains(t, upper, "EVENT", "[%s/%s] %q must be a CREATE EVENT", srv.name, rw.name, f.Name)
						default:
							t.Fatalf("[%s/%s] unexpected directory %q from file %q", srv.name, rw.name, dir, f.Name)
						}
					}

					// One file per synced object, per type.
					require.Len(t, byDir["tables"], tbl, "[%s/%s] table file count", srv.name, rw.name)
					require.Len(t, byDir["views"], vw, "[%s/%s] view file count", srv.name, rw.name)
					require.Len(t, byDir["functions"], fn, "[%s/%s] function file count", srv.name, rw.name)
					require.Len(t, byDir["procedures"], pr, "[%s/%s] procedure file count", srv.name, rw.name)
					require.Len(t, byDir["triggers"], tg, "[%s/%s] trigger file count", srv.name, rw.name)

					t.Logf("[%s/%s] multi-file layout: %d files (tables=%d views=%d functions=%d procedures=%d triggers=%d events=%d)",
						srv.name, rw.name, len(result.Files), len(byDir["tables"]), len(byDir["views"]),
						len(byDir["functions"]), len(byDir["procedures"]), len(byDir["triggers"]), len(byDir["events"]))

					// --- The round-trip proof. ---
					single := dumpSDL(ctx, t, srv, dbName)
					require.NotEmpty(t, single, "[%s/%s] single-file dump empty", srv.name, rw.name)
					concat := concatMultiFile(result)
					require.NotEmpty(t, concat, "[%s/%s] concatenated multi-file empty", srv.name, rw.name)

					// (a) concat ≡ single-file: diffing the concatenated multi-file export against
					// the single-file dump must be a no-op (both describe the identical schema).
					diffVsSingle, err := mysqlDiffSDLMigration(concat, single, srv.version)
					require.NoError(t, err)
					if diffVsSingle != "" {
						t.Logf("[%s/%s] NON-EMPTY concat-vs-single (%d stmts):\n%s",
							srv.name, rw.name, statementCount(diffVsSingle), diffVsSingle)
					}
					require.Empty(t, diffVsSingle, "[%s/%s] concat(multi-file) must equal single-file dump", srv.name, rw.name)

					// (b) concat is idempotent: re-importing the concatenated export against itself
					// is a no-op, proving it is a self-consistent, re-importable SDL document.
					selfDiff, err := mysqlDiffSDLMigration(concat, concat, srv.version)
					require.NoError(t, err)
					require.Empty(t, selfDiff, "[%s/%s] concat(multi-file) must be idempotent, got:\n%s", srv.name, rw.name, selfDiff)
				})
			}
		})
	}
}

// TestMultiFileSDLExportZip proves the API ZIP path: the MultiFileSchemaResult produces a
// valid ZIP with exactly one entry per file whose bytes match the file content — mirroring
// backend/api/v1/database_service.go getMultiFileSDL. Exercised on 8.0 (sakila's load target).
//
//nolint:tparallel
func TestMultiFileSDLExportZip(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()

	var sakila realWorldSchema
	for _, rw := range realWorldSchemas() {
		if rw.name == "sakila" {
			sakila = rw
		}
	}

	srv := liveServers[0] // mysql80
	require.Equal(t, "8.0", srv.version)

	meta, _ := loadRealWorld(ctx, t, srv, sakila)
	metaProto := meta.GetProto()
	require.NotNil(t, metaProto)

	result, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{
		SkipBackupSchema: true,
	}, metaProto)
	require.NoError(t, err)
	require.NotEmpty(t, result.Files)

	// Build the ZIP exactly as getMultiFileSDL does.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range result.Files {
		w, err := zw.Create(f.Name)
		require.NoError(t, err)
		_, err = w.Write([]byte(f.Content))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())

	// Read it back and assert entry-for-entry fidelity.
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, zr.File, len(result.Files), "ZIP entry count must equal file count")

	want := map[string]string{}
	for _, f := range result.Files {
		want[f.Name] = f.Content
	}
	for _, entry := range zr.File {
		content, ok := want[entry.Name]
		require.True(t, ok, "unexpected ZIP entry %q", entry.Name)
		rc, err := entry.Open()
		require.NoError(t, err)
		got, err := io.ReadAll(rc)
		rc.Close()
		require.NoError(t, err)
		require.Equal(t, content, string(got), "ZIP entry %q content mismatch", entry.Name)
	}
	t.Logf("[%s/sakila] ZIP has %d entries, all matching", srv.name, len(zr.File))
}

// TestMultiFileSDLEventOrderingLive is the direct answer to the review claim that the
// multi-file layout — which writes events/ (sorting alphabetically BEFORE tables/) — can
// feed the SDL loader an event-before-table order that fails, because an event body may
// reference a table. It builds a schema with a table AND an event whose body DELETEs from
// that table, exports it multi-file, concatenates the files in the EXACT alphabetical path
// order the declarative rollout uses (action/command/file.go: events/*.sql then
// tables/*.sql), and proves the loader handles it: LoadSDL is dependency-aware and layers
// CREATE EVENT last regardless of text order (omni sdl.go sdlPriority: event=6, after
// table=1), and event bodies are parsed opaquely (no dependency edge, no forward-reference
// error). So:
//
//	(1) the concatenation really is event-before-table (the layout the claim describes);
//	(2) concat(multi-file) is idempotent and equals the single-file dump; and
//	(3) the realistic path — source=empty, target=that alphabetical concat -> generate plan
//	    -> APPLY to a fresh live 8.0 database -> converges — succeeds.
//
//nolint:tparallel
func TestMultiFileSDLEventOrderingLive(t *testing.T) {
	skipUnlessLiveOracle(t)
	ctx := context.Background()
	srv := liveServers[0] // 8.0 — events are the relevant surface and the scheduler is on.
	require.Equal(t, "8.0", srv.version)

	// A table plus an event whose body references it. In the flat multi-file layout the
	// event lands in events/purge_audit.sql and the table in tables/audit_log.sql, so the
	// action's lexical path sort places the event's CREATE ahead of the table's.
	const ddl = `
CREATE TABLE audit_log (
	id INT PRIMARY KEY AUTO_INCREMENT,
	msg VARCHAR(200) NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE EVENT purge_audit
	ON SCHEDULE EVERY 1 DAY
	DO DELETE FROM audit_log WHERE created_at < (NOW() - INTERVAL 30 DAY);
`

	dbSrc := newLiveDatabase(ctx, t, srv, "sdl_evt_src")
	require.NoError(t, applyDDL(ctx, t, srv, dbSrc, ddl))

	driver, err := createLiveMySQLDriver(ctx, srv, dbSrc)
	require.NoError(t, err)
	defer driver.Close(ctx)
	metadata, err := driver.SyncDBSchema(ctx)
	require.NoError(t, err)
	metaProto := model.NewDatabaseMetadata(metadata, nil, nil, storepb.Engine_MYSQL, true).GetProto()
	require.NotNil(t, metaProto)

	result, err := schema.GetMultiFileDatabaseDefinition(storepb.Engine_MYSQL, schema.GetDefinitionContext{
		SkipBackupSchema: true,
	}, metaProto)
	require.NoError(t, err)
	require.NotEmpty(t, result.Files)

	// (1) The event and table files exist, and — sorted lexically as the action does — the
	// event file really does sort before the table file. This is the ordering the claim is
	// about; the rest of the test proves the loader tolerates it.
	var eventPath, tablePath string
	for _, f := range result.Files {
		switch {
		case strings.HasPrefix(f.Name, "events/"):
			eventPath = f.Name
		case strings.HasPrefix(f.Name, "tables/"):
			tablePath = f.Name
		default:
		}
	}
	require.NotEmpty(t, eventPath, "expected an events/ file")
	require.NotEmpty(t, tablePath, "expected a tables/ file")
	require.Less(t, eventPath, tablePath, "the action sorts %q BEFORE %q (event before table)", eventPath, tablePath)

	// (2) concat in alphabetical path order (event body precedes its table in the text) is
	// both idempotent and equal to the single-file dump — the loader topo-sorts the event
	// after the table regardless of this text order.
	concat := concatMultiFile(result)
	require.NotEmpty(t, concat)
	require.Less(t, strings.Index(concat, "CREATE EVENT"), strings.Index(concat, "CREATE TABLE"),
		"concat must present the event body before the table (the order under test)")

	selfDiff, err := mysqlDiffSDLMigration(concat, concat, srv.version)
	require.NoError(t, err, "LoadSDL must accept the event-before-table concat without a forward-reference error")
	require.Empty(t, selfDiff, "concat(multi-file) with event-before-table must be idempotent")

	single := dumpSDL(ctx, t, srv, dbSrc)
	require.NotEmpty(t, single)
	diffVsSingle, err := mysqlDiffSDLMigration(concat, single, srv.version)
	require.NoError(t, err)
	require.Empty(t, diffVsSingle, "concat(multi-file) must equal the single-file dump")

	// (3) The realistic apply: source=empty, target=the alphabetical concat. Generate the
	// plan and APPLY it to a fresh live 8.0 database, then prove convergence (re-dump ==
	// single). If event-before-table ordering were a real problem this apply would fail.
	plan, err := mysqlDiffSDLMigration("", concat, srv.version)
	require.NoError(t, err)
	require.NotEmpty(t, plan, "empty -> target must produce a non-empty plan")
	require.Contains(t, plan, "CREATE TABLE", "plan must create the table")
	require.Contains(t, plan, "EVENT", "plan must create the event")

	dbDst := newLiveDatabase(ctx, t, srv, "sdl_evt_dst")
	require.NoError(t, applyDDL(ctx, t, srv, dbDst, plan), "applying the generated plan to live 8.0 must succeed")

	redump := dumpSDL(ctx, t, srv, dbDst)
	converge, err := mysqlDiffSDLMigration(redump, single, srv.version)
	require.NoError(t, err)
	require.Empty(t, converge, "applied schema must converge with the source dump")
	t.Logf("[%s] event-before-table concat applied and converged; plan:\n%s", srv.name, plan)
}
