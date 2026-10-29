package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"github.com/bytebase/bytebase/backend/api/auth"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

func TestMCPStatementIsRead(t *testing.T) {
	for _, row := range []struct {
		name      string
		engine    storepb.Engine
		statement string
		read      bool
	}{
		{name: "a read", engine: storepb.Engine_POSTGRES, statement: "SELECT id FROM employee", read: true},
		{name: "a write", engine: storepb.Engine_POSTGRES, statement: "UPDATE employee SET name = 'x'"},
		{name: "a read and a write", engine: storepb.Engine_POSTGRES, statement: "SELECT id FROM employee; DELETE FROM employee"},
		{name: "a statement that returns no data", engine: storepb.Engine_POSTGRES, statement: "SET search_path TO app"},
		{name: "a statement that does not parse", engine: storepb.Engine_POSTGRES, statement: "SELEC id FROM employee"},
		{name: "a MongoDB read", engine: storepb.Engine_MONGODB, statement: `db.employee.find({})`, read: true},
		{name: "a MongoDB write", engine: storepb.Engine_MONGODB, statement: unclassifiedEngineWrites[storepb.Engine_MONGODB]},
		{name: "an Elasticsearch read", engine: storepb.Engine_ELASTICSEARCH, statement: "GET /employee/_search\n{\"query\":{\"match_all\":{}}}", read: true},
		{name: "an Elasticsearch write", engine: storepb.Engine_ELASTICSEARCH, statement: unclassifiedEngineWrites[storepb.Engine_ELASTICSEARCH]},
		{name: "a Databricks read", engine: storepb.Engine_DATABRICKS, statement: "SELECT 1"},
	} {
		t.Run(row.name, func(t *testing.T) {
			require.Equal(t, row.read, mcpStatementIsRead(context.Background(), row.engine, row.statement))
		})
	}
}

func TestMCPWriteMarked(t *testing.T) {
	const database = "instances/prod/databases/hr"
	mark := func(edit func(*auth.MCPConfirmedDecision)) context.Context {
		decision := auth.MCPConfirmedDecision{
			Procedure:     v1connect.SQLServiceQueryProcedure,
			Resource:      database,
			WithoutAsking: true,
		}
		edit(&decision)
		return auth.WithMCPConfirmedDecision(context.Background(), decision)
	}
	for _, row := range []struct {
		name   string
		ctx    context.Context
		marked bool
	}{
		{name: "no mark", ctx: context.Background()},
		{name: "query_database's mark", ctx: mark(func(*auth.MCPConfirmedDecision) {}), marked: true},
		{name: "a mark for another database", ctx: mark(func(d *auth.MCPConfirmedDecision) { d.Resource = "instances/prod/databases/payroll" })},
		{name: "a mark for another procedure", ctx: mark(func(d *auth.MCPConfirmedDecision) { d.Procedure = v1connect.IssueServiceApproveIssueProcedure })},
		{name: "a decision that does not run unasked", ctx: mark(func(d *auth.MCPConfirmedDecision) { d.WithoutAsking = false })},
	} {
		t.Run(row.name, func(t *testing.T) {
			require.Equal(t, row.marked, mcpWriteMarked(row.ctx, database))
		})
	}
}

func TestMCPWriteDataSourceID(t *testing.T) {
	admin := &storepb.DataSource{Id: "admin", Type: storepb.DataSourceType_ADMIN}
	readOnly := &storepb.DataSource{Id: "ro", Type: storepb.DataSourceType_READ_ONLY}
	readOnly2 := &storepb.DataSource{Id: "ro-2", Type: storepb.DataSourceType_READ_ONLY}
	for _, row := range []struct {
		name        string
		dataSources []*storepb.DataSource
		picked      string
		allowAdmin  bool
		want        string
	}{
		{name: "the policy allows ADMIN", dataSources: []*storepb.DataSource{admin, readOnly}, picked: "ro", allowAdmin: true, want: "admin"},
		{name: "the policy keeps writes off ADMIN", dataSources: []*storepb.DataSource{admin, readOnly}, picked: "ro", want: "ro"},
		{name: "no read-only data source", dataSources: []*storepb.DataSource{admin}, picked: "admin", want: "admin"},
		{name: "several read-only data sources keep query_database's pick", dataSources: []*storepb.DataSource{admin, readOnly, readOnly2}, picked: "ro-2", want: "ro-2"},
	} {
		t.Run(row.name, func(t *testing.T) {
			instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES, DataSources: row.dataSources}}
			got, err := mcpWriteDataSourceID(instance, row.picked, row.allowAdmin)
			require.NoError(t, err)
			require.Equal(t, row.want, got)
		})
	}

	t.Run("several read-only data sources and no pick", func(t *testing.T) {
		instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES, DataSources: []*storepb.DataSource{readOnly, readOnly2}}}
		_, err := mcpWriteDataSourceID(instance, "", false)
		require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	})

	// The SQL Editor's routing reads a statement it cannot parse as a read, and
	// a person there can still pick ADMIN. An MCP write is routed by the verdict.
	t.Run("a write the SQL Editor's routing cannot parse", func(t *testing.T) {
		instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES, DataSources: []*storepb.DataSource{admin, readOnly}}}
		const statement = "UPSERT INTO employee VALUES (1)"
		require.False(t, mcpStatementIsRead(context.Background(), storepb.Engine_POSTGRES, statement))

		editor, err := resolveDataSourceID(context.Background(), instance, "", statement, true)
		require.NoError(t, err)
		require.Equal(t, "ro", editor)
		routed, err := mcpWriteDataSourceID(instance, "ro", true)
		require.NoError(t, err)
		require.Equal(t, "admin", routed)
	})
}
