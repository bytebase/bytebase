package v1

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// The SQL clamp is the third point that holds an MCP session to the workspace
// ceiling, and the only one that reads what the request carries rather than
// which method it is.
//
// SQLService/Query is classified READ, so the method gate serves it under a
// read-only ceiling. But what Query does is decided by its statement: on every
// engine in EngineSupportQueryNewACL the handler classifies each statement and
// authorizes DML with bb.sql.dml and DDL with bb.sql.ddl, against the caller's
// own RBAC. A read-only session held by someone who may write would therefore
// write. The clamp closes that: under a READ_ONLY ceiling every statement in
// the request must classify as a read, or the whole request is refused.
//
// What it claims is classifier-enforced read-only, reinforced by a read-only
// driver session where the engine has one. It is not a proof that nothing can
// write: a statement that reads structurally can still call a side-effecting
// function, and a classifier can be wrong about a grammar it does not fully
// model. Those are the conformance lane's business, not this gate's.

// mcpCeilingContextKey carries the ceiling the MCP gate resolved for this
// request. The gate reads the workspace's ceiling once per request; stamping it
// here is what lets the clamp hold the same request against the same value,
// rather than reading the setting a second time and possibly disagreeing with
// the gate that already admitted the call.
//
// It travels inward — gate to handler — so it is a plain context value, not the
// callback shape SetMCPPolicyDenied needs to reach the audit interceptor
// wrapping the gate from outside.
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
// It keys on the presence of the delegated grant, never on any of its values:
// a legacy or scope-omitting session leaves every grant field empty, so no
// value is a sound proxy for MCP origin. Public-chain requests carry no grant
// at all and are untouched.
//
// An MCP request that carries no ceiling means the gate did not run, which can
// only happen if the internal chain was reordered. The clamp refuses rather
// than guess: an unresolved ceiling is not a read-write one.
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

// refuseNonReadOnlyStatement is the fail-closed classifier the clamp owns.
//
// It exists because base.ValidateSQLForEditor answers "read-only" for an engine
// with no registered validator: that default suits a caller who only routes or
// formats with the verdict, and admits every write for a caller who must refuse
// one. Nothing here calls it bare.
//
// A request is refused when the engine has no classifier, when a statement will
// not parse, when any statement classifies as anything but a read, and when any
// statement changes session state — so a batch is served only if every
// statement in it reads and none of it rewrites the session it runs on.
//
// "Returns data" is the second bool the validators already report, and a
// statement that returns none is either a session change or a statement that
// runs the query to measure it. Refusing both is what stops the depth layer
// being switched off by a statement this same rule admits: every unit of a
// request runs on one connection, so "SET default_transaction_read_only = off"
// followed by anything the classifier calls a read would leave the read-only
// session disarmed and the second statement free to write — verified against
// Postgres 17, where the two land as separate transactions and the write
// succeeds.
//
// It closes neither class completely, and both gaps are recorded rather than
// papered over. The bool reports the SET family on postgres, cockroachdb, the
// mysql family, tidb, snowflake and redshift, but not on Trino, whose
// validator classifies SET SESSION as a data-returning read (BOT-91). And a
// statement that reads structurally can still call a function that rewrites
// the same setting — set_config on Postgres — which no classifier catches
// (BOT-88). See the type comment on QueryResponse.ReadOnlyEnforcement for what
// the depth may and may not be said to guarantee.
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

// mcpClampUnits splits a request the way queryRetryStopOnError does, so the
// clamp classifies the same text that would execute — including the
// whole-request fallback that function takes when an engine has no splitter
// (Redis) or the split fails. Classifying a different unit than the executor
// runs is how a batch gets past a per-statement rule: on the engines whose
// validator classifies by leading keyword, handing it the whole request reads
// "SELECT 1; DROP TABLE t" as a SELECT.
//
// A splitter can also return the request whole without failing: the ClickHouse
// and Hive splitters break on newlines rather than statement terminators, so a
// one-line batch arrives here as one unit and their leading-keyword validator
// reads it as a SELECT. That is BOT-86, and it is why this comment enumerates
// three ways to end up judging the whole request rather than two.
//
// MSSQL is the one engine where the units differ, and in the safe direction:
// it is split for analysis but sent to the driver whole, to keep variable
// scope across the batch, so the clamp judges it one statement at a time and
// is stricter than the executor rather than looser.
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

// mcpReadOnlyDepth reports what a clamped request actually got, for the
// response to disclose. Unclamped requests disclose nothing.
func mcpReadOnlyDepth(clamped bool, engine storepb.Engine, datashare bool) v1pb.QueryResponse_ReadOnlyEnforcement {
	if !clamped {
		return v1pb.QueryResponse_READ_ONLY_ENFORCEMENT_UNSPECIFIED
	}
	if readOnlyDriverSession(engine, datashare) {
		return v1pb.QueryResponse_STATEMENT_CLASSIFICATION_AND_READ_ONLY_SESSION
	}
	return v1pb.QueryResponse_STATEMENT_CLASSIFICATION
}

// readOnlyDriverSession reports whether the driver turns
// ConnectionContext.ReadOnly into a read-only database session. Three do, all
// by setting default_transaction_read_only on the connection they open:
// postgres (plugin/db/pg), cockroachdb (plugin/db/cockroachdb), and redshift
// (plugin/db/redshift), which skips it on a datashare database because a
// datashare cannot run a read-only transaction. Every other driver ignores the
// flag, so a clamped request there gets classification alone.
//
// This mirrors the drivers rather than asking them, because the answer is
// needed before a connection is opened. A driver that starts honoring the flag
// without a row here understates its own depth, which is the safe way for this
// to be wrong.
func readOnlyDriverSession(engine storepb.Engine, datashare bool) bool {
	switch engine {
	case storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB:
		return true
	case storepb.Engine_REDSHIFT:
		return !datashare
	default:
		return false
	}
}

// refuseClampedStatement wraps a reason in the denial the gate set the shape
// for: what refused, why, and the two ways out.
func refuseClampedStatement(reason string) error {
	return connect.NewError(connect.CodePermissionDenied, errors.Errorf(
		"this workspace's MCP capability ceiling is READ_ONLY, so an MCP session may only run statements Bytebase can show are reads: %s. "+
			"Ask a workspace admin to raise the MCP ceiling in the workspace settings, "+
			"or run this statement signed in to the Bytebase console instead", reason))
}
