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

	// The clamp asks the parser registry which engines it can classify, so the
	// registry has to be populated the way the server populates it. Naming the
	// packages here rather than relying on what the rest of the package happens
	// to import keeps every row below about the engine it says it is about.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/cassandra"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/redis"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/spanner"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/standard"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
)

// unclassifiedEngines are the engines with no registered read-only classifier.
// Every one of them fails closed under the clamp rather than gaining a
// validator in this change.
var unclassifiedEngines = []storepb.Engine{
	storepb.Engine_MONGODB,
	storepb.Engine_ELASTICSEARCH,
	storepb.Engine_DATABRICKS,
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
			require.Contains(t, err.Error(), "raise the MCP ceiling",
				"the denial must name the way out")
			require.Contains(t, err.Error(), "Bytebase console",
				"the denial must name where the human can do it instead")
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
	writes := map[storepb.Engine]string{
		storepb.Engine_MONGODB:       `db.employee.insertOne({name: "Bytebase"})`,
		storepb.Engine_ELASTICSEARCH: "POST /employee/_doc\n{\"name\":\"Bytebase\"}",
		storepb.Engine_DATABRICKS:    "INSERT INTO employee VALUES ('Bytebase')",
	}

	for _, engine := range unclassifiedEngines {
		t.Run(engine.String(), func(t *testing.T) {
			statement := writes[engine]
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

// TestMCPClampDisclosesTheDepthItApplied pins what the response tells the
// caller. The depth is whatever the connection actually got, so an agent does
// not have to know which drivers open a read-only session to know whether this
// one did.
func TestMCPClampDisclosesTheDepthItApplied(t *testing.T) {
	const withSession = v1pb.QueryResponse_STATEMENT_CLASSIFICATION_AND_READ_ONLY_SESSION
	const classificationOnly = v1pb.QueryResponse_STATEMENT_CLASSIFICATION

	rows := []struct {
		name      string
		clamped   bool
		engine    storepb.Engine
		datashare bool
		want      v1pb.QueryResponse_ReadOnlyEnforcement
	}{
		{
			name:   "an unclamped request discloses nothing",
			engine: storepb.Engine_POSTGRES,
			want:   v1pb.QueryResponse_READ_ONLY_ENFORCEMENT_UNSPECIFIED,
		},
		{name: "postgres gets the session too", clamped: true, engine: storepb.Engine_POSTGRES, want: withSession},
		{name: "cockroachdb gets the session too", clamped: true, engine: storepb.Engine_COCKROACHDB, want: withSession},
		{name: "redshift gets the session too", clamped: true, engine: storepb.Engine_REDSHIFT, want: withSession},
		{
			// The redshift driver skips the read-only parameter here, so the
			// disclosure has to skip it as well.
			name:      "a redshift datashare gets classification alone",
			clamped:   true,
			engine:    storepb.Engine_REDSHIFT,
			datashare: true,
			want:      classificationOnly,
		},
		{name: "mysql gets classification alone", clamped: true, engine: storepb.Engine_MYSQL, want: classificationOnly},
		{name: "tidb gets classification alone", clamped: true, engine: storepb.Engine_TIDB, want: classificationOnly},
		{name: "mssql gets classification alone", clamped: true, engine: storepb.Engine_MSSQL, want: classificationOnly},
		{name: "oracle gets classification alone", clamped: true, engine: storepb.Engine_ORACLE, want: classificationOnly},
		{name: "clickhouse gets classification alone", clamped: true, engine: storepb.Engine_CLICKHOUSE, want: classificationOnly},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			require.Equal(t, row.want, mcpReadOnlyDepth(row.clamped, row.engine, row.datashare))
		})
	}
}

func contextWithAuth(authCtx *common.AuthContext) context.Context {
	return context.WithValue(context.Background(), common.AuthContextKey, authCtx)
}
