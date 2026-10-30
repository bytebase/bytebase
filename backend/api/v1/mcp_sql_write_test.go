package v1

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/bytebase/bytebase/backend/api/auth"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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

func TestMCPWriteConfirmed(t *testing.T) {
	const database = "instances/prod/databases/hr"
	request := &v1pb.QueryRequest{Name: database, Statement: "DELETE FROM employee", Schema: new("app")}
	digest := auth.MCPStatementDigest("DELETE FROM employee", database, "app")
	mark := func(procedure, resource string, digest []byte) context.Context {
		return auth.WithMCPConfirmedDecision(context.Background(), auth.MCPConfirmedDecision{Procedure: procedure, Resource: resource, Digest: digest})
	}
	for _, row := range []struct {
		name      string
		ctx       context.Context
		confirmed bool
	}{
		{name: "no mark", ctx: context.Background()},
		{name: "the confirmed request", ctx: mark(v1connect.SQLServiceQueryProcedure, database, digest), confirmed: true},
		{name: "a mark with no digest", ctx: mark(v1connect.SQLServiceQueryProcedure, database, nil)},
		{name: "another statement", ctx: mark(v1connect.SQLServiceQueryProcedure, database, auth.MCPStatementDigest("DELETE FROM salary", database, "app"))},
		{name: "another schema", ctx: mark(v1connect.SQLServiceQueryProcedure, database, auth.MCPStatementDigest("DELETE FROM employee", database, ""))},
		{name: "another database", ctx: mark(v1connect.SQLServiceQueryProcedure, "instances/prod/databases/payroll", digest)},
		{name: "another procedure", ctx: mark(v1connect.IssueServiceApproveIssueProcedure, database, digest)},
	} {
		t.Run(row.name, func(t *testing.T) {
			require.Equal(t, row.confirmed, mcpWriteConfirmed(row.ctx, request))
		})
	}
}

func TestMCPConfirmationRequired(t *testing.T) {
	err := mcpConfirmationRequired(&store.DatabaseMessage{EffectiveEnvironmentID: new("prod")}, "app")
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "query_database")
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Len(t, connectErr.Details(), 1)
	value, valueErr := connectErr.Details()[0].Value()
	require.NoError(t, valueErr)
	info, ok := value.(*errdetails.ErrorInfo)
	require.True(t, ok)
	require.Equal(t, auth.MCPConfirmationReason, info.GetReason())
	require.Equal(t, auth.MCPConfirmationDomain, info.GetDomain())
	require.Equal(t, map[string]string{"environment": "prod", "schema": "app"}, info.GetMetadata())
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
}
