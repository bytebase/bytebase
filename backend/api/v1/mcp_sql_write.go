package v1

import (
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/api/auth"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/generated-go/v1/v1connect"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store"
)

// mcpStatementIsRead reports whether a Read-write MCP session may run statement
// as a read. MongoDB and Elasticsearch have no validator, so they use the span
// check access grants use; other engines with no validator have neither (BOT-87).
func mcpStatementIsRead(ctx context.Context, engine storepb.Engine, statement string) bool {
	if parserbase.HasQueryValidator(engine) {
		return nonReadReason(engine, statement) == ""
	}
	if !shouldClassifyStatementByQuerySpan(engine) {
		return false
	}
	read, err := isReadOnlyStatementForAccessGrant(ctx, engine, statement)
	return err == nil && read
}

// mcpWriteMarked reports whether query_database marked this request to
// database. call_api passes its context on unchanged, so it cannot.
func mcpWriteMarked(ctx context.Context, database string) bool {
	marker, ok := auth.MCPConfirmedDecisionFromContext(ctx)
	return ok && marker.WithoutAsking && marker.Procedure == v1connect.SQLServiceQueryProcedure && marker.Resource == database
}

func mcpWriteOutsideQueryDatabase() error {
	return connect.NewError(connect.CodeFailedPrecondition, errors.New(
		"this statement may write, and an MCP session runs a statement that may write only through the query_database tool; run it there"))
}

func mcpExportOfPossibleWrite() error {
	return connect.NewError(connect.CodePermissionDenied, errors.New(
		"an MCP session exports only statements Bytebase can show are reads; run a statement that may write with the query_database tool"))
}

var errMultipleReadOnlyDataSources = errors.New("instance has multiple read-only data sources, please specify data_source_id explicitly")

// mcpWriteDataSourceID routes an MCP write as the SQL Editor routes a write that
// names no data source. No MCP call can name ADMIN, so a write that
// requiresAdminDataSource reads as a read would never reach it. Where the SQL
// Editor asks the user to pick among read-only data sources, picked stands.
func mcpWriteDataSourceID(instance *store.InstanceMessage, picked string, allowAdminDataSource bool) (string, error) {
	id, err := routeDataSource(instance, allowAdminDataSource, func() bool { return true })
	if errors.Is(err, errMultipleReadOnlyDataSources) && picked != "" {
		return picked, nil
	}
	return id, err
}
