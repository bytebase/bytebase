package v1

import (
	"bytes"
	"context"

	"connectrpc.com/connect"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	"github.com/bytebase/bytebase/backend/api/auth"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
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

// mcpWriteConfirmed reports whether query_database confirmed this exact request.
// call_api passes its context on unchanged, so it cannot.
func mcpWriteConfirmed(ctx context.Context, request *v1pb.QueryRequest) bool {
	marker, ok := auth.MCPConfirmedDecisionFromContext(ctx)
	return ok && marker.Procedure == v1connect.SQLServiceQueryProcedure &&
		bytes.Equal(marker.Digest, auth.MCPStatementDigest(request.Statement, request.Name, request.GetSchema()))
}

// mcpConfirmationRequired refuses an MCP request that may write and that the
// user has not confirmed. Its ErrorInfo carries what the form shows.
func mcpConfirmationRequired(database *store.DatabaseMessage, schema string) error {
	err := connect.NewError(connect.CodeFailedPrecondition, errors.New(
		"this statement may write, and call_api cannot confirm it: run it with the query_database tool"))
	environment := ""
	if database.EffectiveEnvironmentID != nil {
		environment = *database.EffectiveEnvironmentID
	}
	if detail, detailErr := connect.NewErrorDetail(&errdetails.ErrorInfo{
		Reason:   auth.MCPConfirmationReason,
		Domain:   auth.MCPConfirmationDomain,
		Metadata: map[string]string{"environment": environment, "schema": schema},
	}); detailErr == nil {
		err.AddDetail(detail)
	}
	return err
}

// mcpWriteDataSourceID routes an MCP request that may write as the SQL Editor
// routes a write that names no data source. Only query_database's requests run,
// and the data source one names is the tool's own choice: the first read-only
// data source, or ADMIN when there is none. It skips requiresAdminDataSource,
// which reads every Databricks statement as a read.
func mcpWriteDataSourceID(instance *store.InstanceMessage, picked string, allowAdminDataSource bool) (string, error) {
	id, err := routeDataSource(instance, allowAdminDataSource, func() bool { return true })
	if errors.Is(err, errMultipleReadOnlyDataSources) && picked != "" {
		return picked, nil
	}
	return id, err
}
