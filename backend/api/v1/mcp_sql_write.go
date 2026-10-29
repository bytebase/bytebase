package v1

import (
	"context"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	"github.com/bytebase/bytebase/backend/store"
)

// mcpStatementIsRead reports whether a Read-write MCP session may run statement
// as a read. MongoDB and Elasticsearch have no validator, so they use the span
// check access grants use; on other engines with no validator nothing is a read
// (BOT-87).
func mcpStatementIsRead(ctx context.Context, engine storepb.Engine, statement string) bool {
	if shouldClassifyStatementByQuerySpan(engine) {
		read, err := isReadOnlyStatementForAccessGrant(ctx, engine, statement)
		return err == nil && read
	}
	return nonReadReason(engine, statement) == ""
}

// queryDatabaseRequest reports whether query_database sent this request to
// database. call_api passes its context on unchanged, so it cannot mark one.
func queryDatabaseRequest(ctx context.Context, database string) bool {
	marker, ok := auth.MCPConfirmedDecisionFromContext(ctx)
	return ok && marker.Procedure == v1connect.SQLServiceQueryProcedure && marker.Resource == database
}

// mcpWriteDataSourceID routes a query_database request that may write as the SQL
// Editor routes a write that names no data source. picked is the tool's own
// choice: the first read-only data source, or ADMIN when there is none. It skips
// requiresAdminDataSource, which reads every Databricks statement as a read.
// Where the SQL Editor asks for a pick among read-only data sources, picked
// stands.
func mcpWriteDataSourceID(instance *store.InstanceMessage, picked string, allowAdminDataSource bool) (string, error) {
	id, err := routeDataSource(instance, allowAdminDataSource, func() bool { return true })
	if errors.Is(err, errMultipleReadOnlyDataSources) && picked != "" {
		return picked, nil
	}
	return id, err
}
