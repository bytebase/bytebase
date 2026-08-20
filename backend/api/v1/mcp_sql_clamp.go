package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// SQLService/Query is classified READ, but on the EngineSupportQueryNewACL
// engines it authorizes DML and DDL per statement against the caller's own
// RBAC — so its class depends on its argument, and a read-only session held by
// someone who may write would write. The clamp is the point that reads the
// argument.
//
// Ceiling: classifier-enforced, not proven. A structurally-reading statement
// can still have an effect, and a classifier can be wrong about a grammar it
// does not fully model; closing that is the conformance lane's work.

// mcpCeilingContextKey carries the ceiling the gate already resolved, so the
// clamp holds the request against that same read rather than a second one the
// gate could disagree with.
type mcpCeilingContextKey struct{}

func withMCPCeiling(ctx context.Context, ceiling storepb.WorkspaceProfileSetting_MCPCapability) context.Context {
	return context.WithValue(ctx, mcpCeilingContextKey{}, ceiling)
}

func mcpCeilingFromContext(ctx context.Context) (storepb.WorkspaceProfileSetting_MCPCapability, bool) {
	ceiling, ok := ctx.Value(mcpCeilingContextKey{}).(storepb.WorkspaceProfileSetting_MCPCapability)
	return ceiling, ok
}

// mcpReadOnlyClampApplies reports whether this request must be held to
// read-only statements.
//
// Keyed on the grant's presence, never a field value: a legacy or
// scope-omitting session leaves every field empty, so no value marks MCP
// origin. An MCP request with no stamped ceiling means the gate never ran, and
// an unresolved ceiling is not a read-write one.
func mcpReadOnlyClampApplies(ctx context.Context) (bool, error) {
	authCtx, ok := common.GetAuthContextFromContext(ctx)
	if !ok || authCtx.DelegatedGrant == nil {
		return false, nil
	}
	ceiling, ok := mcpCeilingFromContext(ctx)
	if !ok {
		return false, connect.NewError(connect.CodeInternal, errors.New(
			"this MCP request cannot be checked against the workspace MCP capability ceiling, so it fails closed"))
	}
	return ceiling == storepb.WorkspaceProfileSetting_READ_ONLY, nil
}

// refuseNonReadOnlyStatement classifies a request fail-closed. Nothing here
// calls base.ValidateSQLForEditor bare: it answers "read-only" for an engine
// with no registered validator, which suits a caller that only routes on the
// verdict and admits every write for one that must refuse a write.
//
// A statement that returns no data is refused as well as one that is not a
// read. Every unit of a request runs on one connection, and Postgres applies a
// changed default_transaction_read_only to the next transaction, so
// "SET default_transaction_read_only = off" followed by a classifier-admitted
// read disarms the depth layer and writes (verified, PG 17). The same bool
// carries the other rebinding statements — Trino USE and SET ROLE, Redis
// SELECT — which repoint the connection so a later read resolves somewhere the
// caller never named.
//
// Not closed: a structurally-reading statement can still call a function that
// rewrites the same setting (set_config, BOT-88).
func refuseNonReadOnlyStatement(engine storepb.Engine, statement string) error {
	if !parserbase.HasQueryValidator(engine) {
		return refuseClampedStatement(fmt.Sprintf(
			"Bytebase has no read-only classifier for %v, so no statement on this engine can be shown to be a read", engine))
	}
	units := mcpClampUnits(engine, statement)
	for i, unit := range units {
		readOnly, allQuery, err := parserbase.ValidateSQLForEditor(engine, unit)
		if err != nil {
			return refuseClampedStatement(fmt.Sprintf(
				"%s could not be parsed, so it cannot be shown to be a read: %v", describeClampUnit(i, len(units)), err))
		}
		if !readOnly {
			return refuseClampedStatement(fmt.Sprintf("%s is not a read", describeClampUnit(i, len(units))))
		}
		if !allQuery {
			return refuseClampedStatement(fmt.Sprintf(
				"%s returns no data, so it either rewrites the session it runs on, which can switch off the read-only session the rest of the request depends on, or runs the query to measure it. A read-only ceiling serves statements that only read",
				describeClampUnit(i, len(units))))
		}
	}
	return nil
}

// mcpClampUnits splits a request the way queryRetryStopOnError does, including
// its whole-request fallback for an engine with no splitter (Redis) or a failed
// split, so the clamp classifies the text that would execute.
//
// A splitter can also succeed and still return the request whole: ClickHouse
// and Hive break on newlines rather than terminators (BOT-86). Their validator
// refuses a unit carrying a terminator it did not end with, which is what keeps
// that from being read on its leading statement.
//
// MSSQL differs in the safe direction: split for analysis, sent to the driver
// whole to keep variable scope, so the clamp is stricter than the executor.
func mcpClampUnits(engine storepb.Engine, statement string) []string {
	statements, err := parserbase.SplitMultiSQL(engine, statement)
	if err != nil {
		return []string{statement}
	}
	var units []string
	for _, s := range statements {
		if s.Empty {
			continue
		}
		units = append(units, s.Text)
	}
	if len(units) == 0 {
		return []string{statement}
	}
	return units
}

// describeClampUnit names the offending statement the way a denial should read
// for a request holding one statement and for a batch alike.
func describeClampUnit(index, total int) string {
	if total <= 1 {
		return "the statement"
	}
	return fmt.Sprintf("statement %d of %d", index+1, total)
}

// refuseClampedStatement wraps a reason in the denial the gate set the shape
// for: what refused, why, and the two ways out.
func refuseClampedStatement(reason string) error {
	return connect.NewError(connect.CodePermissionDenied, errors.Errorf(
		"this workspace's MCP capability ceiling is READ_ONLY, so an MCP session may only run statements Bytebase can show are reads: %s. "+
			"Ask a workspace admin to raise the MCP ceiling in the workspace settings, "+
			"or run this statement signed in to the Bytebase console instead", reason))
}
