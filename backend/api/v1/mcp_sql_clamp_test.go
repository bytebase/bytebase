package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"

	// The clamp asks the parser registry what it can classify and how it
	// splits, so this test binary has to register what the server registers.
	// The set is the parser half of backend/server/ultimate.go, verbatim:
	// naming a subset would make a row's verdict depend on which packages this
	// file happens to import rather than on the engine it says it is about,
	// and a missing splitter reads as "no splitter" without saying so.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/bigquery"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/cassandra"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/cosmosdb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/doris"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/elasticsearch"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mariadb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mongodb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/partiql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/redis"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/spanner"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/starrocks"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/trino"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

// unclassifiedEngineWrites are the engines with no registered read-only
// classifier, each with a statement that writes on it. Every one fails closed
// under the clamp rather than gaining a validator in this change.
//
// One map rather than a list beside a lookup table: two copies of this set
// could disagree about which engines the test is even about.
var unclassifiedEngineWrites = map[storepb.Engine]string{
	storepb.Engine_MONGODB:       `db.employee.insertOne({name: "Bytebase"})`,
	storepb.Engine_ELASTICSEARCH: "POST /employee/_doc\n{\"name\":\"Bytebase\"}",
	storepb.Engine_DATABRICKS:    "INSERT INTO employee VALUES ('Bytebase')",
}

// TestMCPClampRefusesWhatItCannotShowIsARead is the clamp's whole rule: a
// request is served only when every statement the driver would run classifies
// as a read.
func TestMCPClampRefusesWhatItCannotShowIsARead(t *testing.T) {
	rows := []struct {
		name      string
		engine    storepb.Engine
		statement string
		refused   bool
		reason    string
	}{
		{
			name:      "a plain read is served",
			engine:    storepb.Engine_POSTGRES,
			statement: "SELECT * FROM employee",
		},
		{
			name:      "an all-read batch is served whole",
			engine:    storepb.Engine_POSTGRES,
			statement: "SELECT 1; SELECT 2; SELECT 3;",
		},
		{
			name:      "a write is refused",
			engine:    storepb.Engine_POSTGRES,
			statement: "INSERT INTO employee (name) VALUES ('a')",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			name:      "one write refuses the whole batch",
			engine:    storepb.Engine_POSTGRES,
			statement: "SELECT 1; UPDATE employee SET name = 'a'; SELECT 2;",
			refused:   true,
			reason:    "statement 2 of 3 is not a read",
		},
		{
			name:      "a write last in the batch refuses it too",
			engine:    storepb.Engine_MYSQL,
			statement: "SELECT 1; SELECT 2; DELETE FROM employee;",
			refused:   true,
			reason:    "statement 3 of 3 is not a read",
		},
		{
			name:      "a statement that will not parse is refused",
			engine:    storepb.Engine_POSTGRES,
			statement: "SELECT FROM WHERE (((",
			refused:   true,
			reason:    "could not be parsed",
		},
		{
			name:      "DDL is refused",
			engine:    storepb.Engine_MYSQL,
			statement: "CREATE TABLE t (id INT)",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			name:      "a data-modifying CTE is refused",
			engine:    storepb.Engine_POSTGRES,
			statement: "WITH moved AS (DELETE FROM employee RETURNING *) SELECT * FROM moved",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			// Nothing to refuse: an empty request carries no statement, so it
			// reaches the driver as one and writes nothing. The clamp has no
			// opinion on it, and query_database rejects it before this point.
			name:      "an empty request has no statement to refuse",
			engine:    storepb.Engine_POSTGRES,
			statement: "",
		},
		{
			// The engines whose validator classifies by leading keyword over
			// whatever string it is handed. Passing the request whole would
			// read this as a SELECT; classifying the statements the driver
			// will run is what catches the tail.
			name:      "a leading-keyword engine still has its batch tail classified",
			engine:    storepb.Engine_CASSANDRA,
			statement: "SELECT * FROM employee; DROP TABLE employee",
			refused:   true,
			reason:    "statement 2 of 2 is not a read",
		},
		{
			name:      "the same on Spanner",
			engine:    storepb.Engine_SPANNER,
			statement: "SELECT 1; DELETE FROM employee WHERE true",
			refused:   true,
			reason:    "statement 2 of 2 is not a read",
		},
		{
			name:      "a statement that rewrites the session is refused",
			engine:    storepb.Engine_POSTGRES,
			statement: "SET default_transaction_read_only = off",
			refused:   true,
			reason:    "rewrites the session it runs on",
		},
		{
			// The disarm this rule exists for. Both statements classify as
			// reads, every unit of a request runs on one connection, and
			// Postgres applies the changed default to the next transaction —
			// so admitting the first would let the second write.
			name:      "a session rewrite ahead of a read is refused before either runs",
			engine:    storepb.Engine_POSTGRES,
			statement: "SET default_transaction_read_only = off; SELECT nextval('s')",
			refused:   true,
			reason:    "statement 1 of 2 returns no data",
		},
		{
			name:      "the same on MySQL",
			engine:    storepb.Engine_MYSQL,
			statement: "SET SESSION TRANSACTION READ WRITE; SELECT 1",
			refused:   true,
			reason:    "rewrites the session it runs on",
		},
		{
			// A single well-formed statement the server executes as a
			// server-side file write. The MySQL family has no read-only
			// driver session, so the classifier is the whole guarantee here.
			name:      "SELECT INTO OUTFILE is a write, not a read",
			engine:    storepb.Engine_MYSQL,
			statement: "SELECT * FROM employee INTO OUTFILE '/tmp/employee.txt'",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			name:      "so is INTO DUMPFILE",
			engine:    storepb.Engine_MYSQL,
			statement: "SELECT * FROM employee INTO DUMPFILE '/tmp/employee.bin'",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			name:      "and on MariaDB, which shares the validator",
			engine:    storepb.Engine_MARIADB,
			statement: "SELECT * FROM employee INTO OUTFILE '/tmp/employee.txt'",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			// TiDB carries its own validator and its own AST. It rejects
			// DUMPFILE at parse time, so OUTFILE is the form to pin.
			name:      "and on TiDB, which carries its own",
			engine:    storepb.Engine_TIDB,
			statement: "SELECT * FROM employee INTO OUTFILE '/tmp/employee.txt'",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			// The forms that put INTO below the root. The parser attaches it
			// to an arm of a set operation or to a parenthesized query, so a
			// check on the statement it hands back misses all three — pg
			// walks its arms for the same reason.
			name:      "INTO OUTFILE on a set operation is still a write",
			engine:    storepb.Engine_MYSQL,
			statement: "SELECT 1 UNION SELECT 2 INTO OUTFILE '/tmp/u.txt'",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			name:      "and on the leading arm of one",
			engine:    storepb.Engine_MYSQL,
			statement: "SELECT * FROM employee INTO OUTFILE '/tmp/x.txt' UNION SELECT 1",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			name:      "and inside a parenthesized query",
			engine:    storepb.Engine_MYSQL,
			statement: "(SELECT * FROM employee INTO OUTFILE '/tmp/p.txt')",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			// INTO a variable assigns session state and returns no rows. It
			// is refused, but as the "returns no data" case rather than as a
			// write, because that is what it is.
			name:      "INTO a variable is session state, not a file write",
			engine:    storepb.Engine_MYSQL, // TiDB rejects this form at parse time.
			statement: "SELECT id FROM employee INTO @found",
			refused:   true,
			reason:    "returns no data",
		},
		{
			// The clause form decides, not the path it names: these fields
			// hold the decoded filename, so an empty one is still a write.
			name:      "an empty OUTFILE target is still a file write",
			engine:    storepb.Engine_MYSQL,
			statement: "SELECT 1 INTO OUTFILE ''",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			// StarRocks and Doris export straight out of the database, to S3
			// or HDFS, which leaves before Bytebase can mask a returned row.
			name:      "StarRocks INTO OUTFILE exports out of the database",
			engine:    storepb.Engine_STARROCKS,
			statement: `SELECT * FROM employee INTO OUTFILE "s3://bucket/export/" FORMAT AS PARQUET PROPERTIES("a" = "b")`,
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			name:      "and so does Doris",
			engine:    storepb.Engine_DORIS,
			statement: `SELECT * FROM employee INTO OUTFILE "s3://bucket/export/" FORMAT AS PARQUET PROPERTIES("a" = "b")`,
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			// USE rebinds the connection's catalog and schema, so an
			// unqualified read after it resolves somewhere the caller never
			// named and the span ACL never authorized.
			name:      "Trino USE rebinds the connection",
			engine:    storepb.Engine_TRINO,
			statement: "USE other_catalog.secret_schema; SELECT * FROM t",
			refused:   true,
			reason:    "returns no data",
		},
		{
			name:      "so does SET SESSION AUTHORIZATION",
			engine:    storepb.Engine_TRINO,
			statement: "SET SESSION AUTHORIZATION alice",
			refused:   true,
			reason:    "returns no data",
		},
		{
			// Redis SELECT switches the connection's logical database, so the
			// GET after it reads a database the caller did not ask for.
			name:      "Redis SELECT switches the logical database",
			engine:    storepb.Engine_REDIS,
			statement: "SELECT 1\nGET secret",
			refused:   true,
			reason:    "returns no data",
		},
		{
			name:      "an unseparated batch is refused rather than read on its first statement",
			engine:    storepb.Engine_CLICKHOUSE,
			statement: "SELECT 1; DROP TABLE employee",
			refused:   true,
			reason:    "the statement is not a read",
		},
		{
			name:      "a read on an engine with no splitter is still classified",
			engine:    storepb.Engine_REDIS,
			statement: "GET k",
		},
		{
			name:      "a write on an engine with no splitter is refused",
			engine:    storepb.Engine_REDIS,
			statement: "GET k\nSET k v",
			refused:   true,
			reason:    "the statement is not a read",
		},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			err := refuseNonReadOnlyStatement(row.engine, row.statement)
			if !row.refused {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err),
				"a clamp refusal is a policy denial, the same kind the ceiling gate returns")
			require.Contains(t, err.Error(), row.reason)
			require.Contains(t, err.Error(), "READ_ONLY",
				"the denial must name the ceiling that refused")
			// query_database suppresses its own "request the SQL Editor role"
			// suggestion when it sees this phrase in the body, because no
			// project role lifts a workspace ceiling. The two live in
			// different processes and meet only over HTTP, so the producer
			// pins the phrase here and executeQuery (backend/api/mcp) keys on
			// the same one.
			require.Contains(t, err.Error(), "MCP capability ceiling",
				"query_database keys its suggestion suppression on this phrase")
			require.Contains(t, err.Error(), "raise the MCP ceiling",
				"the denial must name the way out")
			require.Contains(t, err.Error(), "Bytebase console",
				"the denial must name where the human can do it instead")
		})
	}
}

// TestMCPClampBatchTailIsClassifiedOnEveryEngine walks one adversarial batch —
// a leading read with a write behind it — across every engine that has a
// classifier, and pins which ones the clamp catches it on.
//
// It exists because the rule is only as good as the split, and two engines
// cannot make it: the ClickHouse and Hive splitters break on newlines rather
// than statement terminators, so a one-line batch reaches the leading-keyword
// validator whole (BOT-86). That validator now refuses a statement still
// carrying a terminator it did not end with, which is what keeps the exception
// set below empty.
//
// The set is kept rather than deleted: an engine that starts serving such a
// batch fails this test instead of passing quietly.
func TestMCPClampBatchTailIsClassifiedOnEveryEngine(t *testing.T) {
	const batch = "SELECT 1; DROP TABLE employee"

	// Empty, and it must stay that way. The ClickHouse and Hive splitters
	// still break on newlines rather than terminators (BOT-86), but their
	// validator now refuses a statement that carries a terminator which is not
	// its own, so an unseparated batch fails closed instead of being read on
	// its leading SELECT.
	servesTheWholeBatch := map[storepb.Engine]bool{}

	values := storepb.Engine(0).Descriptor().Values()
	for i := range values.Len() {
		engine := storepb.Engine(values.Get(i).Number())
		if engine == storepb.Engine_ENGINE_UNSPECIFIED || !parserbase.HasQueryValidator(engine) {
			continue
		}
		// Redis takes commands, not SQL, and separates them by newline, so
		// this probe is not a batch there at all — it is one malformed SELECT
		// command. Its batch rule is covered by the newline rows in
		// TestMCPClampRefusesWhatItCannotShowIsARead instead.
		if engine == storepb.Engine_REDIS {
			continue
		}
		t.Run(engine.String(), func(t *testing.T) {
			err := refuseNonReadOnlyStatement(engine, batch)
			if servesTheWholeBatch[engine] {
				require.NoError(t, err,
					"%v is a recorded BOT-86 exception; if it now refuses the batch, delete its row here", engine)
				return
			}
			require.Error(t, err, "%v must not serve a batch whose second statement writes", engine)
		})
	}
}

// TestMCPClampRefusesEngineItCannotClassify is the reason the clamp owns a
// wrapper instead of calling base.ValidateSQLForEditor the way the rest of the
// file does.
//
// The first half is the fail-open itself, pinned: for an engine with no
// registered validator the bare call answers "read-only" for a statement that
// writes. Substitute that call for the wrapper in the clamp and this engine's
// writes run under a read-only ceiling. The second half is the wrapper
// refusing the same statements.
func TestMCPClampRefusesEngineItCannotClassify(t *testing.T) {
	for engine, statement := range unclassifiedEngineWrites {
		t.Run(engine.String(), func(t *testing.T) {
			require.False(t, parserbase.HasQueryValidator(engine),
				"this test is about the engines with no classifier; %v has gained one", engine)

			readOnly, _, err := parserbase.ValidateSQLForEditor(engine, statement)
			require.NoError(t, err)
			require.True(t, readOnly,
				"the bare call answers read-only here, which is the fail-open the wrapper exists to close")

			err = refuseNonReadOnlyStatement(engine, statement)
			require.Error(t, err, "the wrapper must refuse an engine it cannot classify")
			require.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
			require.Contains(t, err.Error(), "has no read-only classifier")

			require.Error(t, refuseNonReadOnlyStatement(engine, "SELECT 1"),
				"a statement that looks like a read is refused too: the engine, not the statement, is what cannot be shown")
		})
	}
}

// TestMCPClampAppliesOnlyToReadOnlyMCPSessions pins what turns the clamp on.
// The marker is the presence of the delegated grant, never a field of it, and
// the ceiling is the one the gate already resolved for this request.
func TestMCPClampAppliesOnlyToReadOnlyMCPSessions(t *testing.T) {
	withGrant := &common.AuthContext{DelegatedGrant: &common.DelegatedGrant{}}

	rows := []struct {
		name    string
		ctx     context.Context
		applies bool
		failure bool
	}{
		{
			name: "a human request carries no auth context here",
			ctx:  context.Background(),
		},
		{
			name: "a public-chain request carries no delegated grant",
			ctx:  contextWithAuth(&common.AuthContext{}),
		},
		{
			name: "an MCP session under a read-write ceiling is not clamped",
			ctx: withMCPCeiling(contextWithAuth(withGrant),
				storepb.WorkspaceProfileSetting_READ_WRITE),
		},
		{
			name: "an MCP session under a read-only ceiling is clamped",
			ctx: withMCPCeiling(contextWithAuth(withGrant),
				storepb.WorkspaceProfileSetting_READ_ONLY),
			applies: true,
		},
		{
			name: "an MCP session whose ceiling never resolved fails closed",
			ctx:  contextWithAuth(withGrant),

			failure: true,
		},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			applies, err := mcpReadOnlyClampApplies(row.ctx)
			if row.failure {
				require.Error(t, err, "an MCP request the gate never held must not run unclamped")
				require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
				return
			}
			require.NoError(t, err)
			require.Equal(t, row.applies, applies)
		})
	}
}

// TestMCPClampReadsTheCeilingTheGateResolved is the agreement the three
// enforcement points rest on. The gate resolves the workspace ceiling once,
// per request, and stamps it; the clamp holds the same request against that
// same value rather than reading the setting again. One resolution per request
// is what makes it impossible for the gate to admit a call the clamp then
// judges under a different ceiling.
func TestMCPClampReadsTheCeilingTheGateResolved(t *testing.T) {
	for _, row := range []struct {
		name    string
		stores  mcpGateStore
		clamped bool
	}{
		{name: "read-only", stores: readOnlyCeiling(), clamped: true},
		{name: "read-write", stores: readWriteCeiling(), clamped: false},
	} {
		t.Run(row.name, func(t *testing.T) {
			got := invokeMCPGate(t, row.stores, &common.AuthContext{
				MCPMethodClass: v1pb.MCPMethodClass_READ,
				DelegatedGrant: &common.DelegatedGrant{},
			}, "/bytebase.v1.SQLService/Query", connect.NewRequest(&v1pb.QueryRequest{}))

			require.NoError(t, got.err)
			require.True(t, got.dispatched)

			ceiling, ok := mcpCeilingFromContext(got.dispatchedCtx)
			require.True(t, ok, "the gate must hand the handler the ceiling it resolved")
			require.Equal(t, row.stores.ceiling, ceiling)

			clamped, err := mcpReadOnlyClampApplies(got.dispatchedCtx)
			require.NoError(t, err)
			require.Equal(t, row.clamped, clamped)
		})
	}
}

func contextWithAuth(authCtx *common.AuthContext) context.Context {
	return context.WithValue(context.Background(), common.AuthContextKey, authCtx)
}
